package sandbox

import (
	"archive/tar"
	"bytes"
	"context"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/vipinsingh/revv/internal/runner"
)

// Sandbox manages a Docker container for isolated test execution.
type Sandbox struct {
	client      *client.Client
	containerID string
	imageTag    string
	workDir     string
	created     bool
}

// New creates a new Sandbox by initializing a Docker client.
func New() (*Sandbox, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker: failed to create client: %w", err)
	}

	return &Sandbox{
		client:  cli,
		workDir: "/workspace",
	}, nil
}

// CheckAvailable verifies the Docker daemon is reachable.
func CheckAvailable(ctx context.Context) error {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return fmt.Errorf("docker is not installed: %w\n\nInstall Docker: https://docs.docker.com/get-docker/", err)
	}
	defer cli.Close()

	_, err = cli.Ping(ctx)
	if err != nil {
		return fmt.Errorf("docker daemon is not running: %w\n\nPlease start Docker Desktop or the Docker daemon", err)
	}
	return nil
}

// Build builds a Docker image from a Dockerfile in the given directory.
// The contextDir should be the repo root, and dockerfilePath is relative to it.
func (s *Sandbox) Build(ctx context.Context, contextDir, dockerfilePath string, verbose bool) error {
	// Create tar archive of the build context
	buildContext, err := createTarArchive(contextDir)
	if err != nil {
		return fmt.Errorf("docker: create build context: %w", err)
	}
	defer buildContext.Close()

	s.imageTag = fmt.Sprintf("revv-sandbox-%d", time.Now().UnixNano())

	options := types.ImageBuildOptions{
		Dockerfile: dockerfilePath,
		Tags:       []string{s.imageTag},
		Remove:     true,
		ForceRemove: true,
	}

	res, err := s.client.ImageBuild(ctx, buildContext, options)
	if err != nil {
		return fmt.Errorf("docker: image build request: %w", err)
	}
	defer res.Body.Close()

	// Parse build output — errors are in the stream, not in the error above
	var output io.Writer = io.Discard
	if verbose {
		output = os.Stdout
	}

	err = jsonmessage.DisplayJSONMessagesStream(res.Body, output, 0, false, nil)
	if err != nil {
		return fmt.Errorf("docker: image build failed: %w", err)
	}

	return nil
}

// Start creates and starts a container from the built image, mounting repoDir as /workspace.
func (s *Sandbox) Start(ctx context.Context, repoDir string) error {
	if s.imageTag == "" {
		return fmt.Errorf("docker: no image built — call Build() first")
	}
	if s.created {
		return fmt.Errorf("docker: sandbox already started")
	}

	absRepoDir, err := filepath.Abs(repoDir)
	if err != nil {
		return fmt.Errorf("docker: resolve repo path: %w", err)
	}

	resp, err := s.client.ContainerCreate(ctx,
		&container.Config{
			Image:      s.imageTag,
			WorkingDir: s.workDir,
			Cmd:        []string{"sleep", "infinity"},
			Tty:        false,
		},
		&container.HostConfig{
			Mounts: []mount.Mount{
				{
					Type:     mount.TypeBind,
					Source:   absRepoDir,
					Target:   s.workDir,
					ReadOnly: false,
				},
			},
			Resources: container.Resources{
				Memory:   512 * 1024 * 1024, // 512MB
				NanoCPUs: 1e9,               // 1 CPU core
			},
			SecurityOpt: []string{"no-new-privileges"},
		},
		nil, nil, "",
	)
	if err != nil {
		return fmt.Errorf("docker: create container: %w", err)
	}

	s.containerID = resp.ID
	s.created = true

	if err := s.client.ContainerStart(ctx, s.containerID, container.StartOptions{}); err != nil {
		_ = s.client.ContainerRemove(ctx, s.containerID, container.RemoveOptions{Force: true})
		s.containerID = ""
		s.created = false
		return fmt.Errorf("docker: start container: %w", err)
	}

	return nil
}

// Exec implements the runner.Executor interface — executes a command inside the container.
func (s *Sandbox) Exec(ctx context.Context, cmd []string) (*runner.ExecResult, error) {
	if s.containerID == "" {
		return nil, fmt.Errorf("docker: cannot exec: sandbox not started")
	}
	if len(cmd) == 0 {
		return nil, fmt.Errorf("docker: exec command is empty")
	}

	start := time.Now()

	execConfig := container.ExecOptions{
		Cmd:          cmd,
		WorkingDir:   s.workDir,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := s.client.ContainerExecCreate(ctx, s.containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("docker: exec create: %w", err)
	}

	hijack, err := s.client.ContainerExecAttach(ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("docker: exec attach: %w", err)
	}
	defer hijack.Close()

	var stdout, stderr bytes.Buffer
	_, err = stdcopy.StdCopy(&stdout, &stderr, hijack.Reader)
	if err != nil {
		if ctx.Err() != nil {
			return &runner.ExecResult{
				ExitCode: -1,
				Stdout:   stdout.String(),
				Stderr:   stderr.String(),
				Duration: time.Since(start),
				TimedOut: ctx.Err() == context.DeadlineExceeded,
			}, nil
		}
		return nil, fmt.Errorf("docker: read exec output: %w", err)
	}

	inspectResp, err := s.client.ContainerExecInspect(ctx, execResp.ID)
	if err != nil {
		return nil, fmt.Errorf("docker: exec inspect: %w", err)
	}

	return &runner.ExecResult{
		ExitCode: inspectResp.ExitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
		TimedOut: ctx.Err() == context.DeadlineExceeded,
	}, nil
}

// Stop stops and removes the container, and removes the built image. Idempotent.
func (s *Sandbox) Stop(ctx context.Context) error {
	// Use a fresh context for cleanup
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if s.containerID != "" {
		stopTimeout := 10
		_ = s.client.ContainerStop(cleanupCtx, s.containerID, container.StopOptions{Timeout: &stopTimeout})
		_ = s.client.ContainerRemove(cleanupCtx, s.containerID, container.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		})
		s.containerID = ""
		s.created = false
	}

	// Remove the built image
	if s.imageTag != "" {
		_, _ = s.client.ImageRemove(cleanupCtx, s.imageTag, image.RemoveOptions{Force: true})
		s.imageTag = ""
	}

	if s.client != nil {
		s.client.Close()
	}

	return nil
}

// createTarArchive creates a tar archive from a source directory for Docker build context.
func createTarArchive(srcDir string) (io.ReadCloser, error) {
	info, err := os.Stat(srcDir)
	if err != nil {
		return nil, fmt.Errorf("source directory: %w", err)
	}
	if !info.IsDir() {
		return nil, fmt.Errorf("not a directory: %s", srcDir)
	}

	pr, pw := io.Pipe()

	go func() {
		tw := tar.NewWriter(pw)
		err := filepath.WalkDir(srcDir, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}

			relPath, err := filepath.Rel(srcDir, path)
			if err != nil {
				return err
			}
			if relPath == "." {
				return nil
			}

			// Skip .git but keep .revv (needed for Dockerfile)
			if d.IsDir() && d.Name() == ".git" {
				return filepath.SkipDir
			}

			fi, err := d.Info()
			if err != nil {
				return err
			}

			header, err := tar.FileInfoHeader(fi, "")
			if err != nil {
				return err
			}
			header.Name = filepath.ToSlash(relPath)

			if err := tw.WriteHeader(header); err != nil {
				return err
			}

			if !d.IsDir() {
				f, err := os.Open(path)
				if err != nil {
					return err
				}
				if _, err := io.Copy(tw, f); err != nil {
					f.Close()
					return err
				}
				f.Close()
			}

			return nil
		})
		if closeErr := tw.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
		pw.CloseWithError(err)
	}()

	return pr, nil
}

// Compile-time check that Sandbox implements runner.Executor.
var _ runner.Executor = (*Sandbox)(nil)
