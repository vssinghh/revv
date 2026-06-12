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
	"sync"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/jsonmessage"
	"github.com/docker/docker/pkg/stdcopy"

	"github.com/vipinsingh/revv/internal/runner"
)

// Sandbox manages Docker containers for isolated test execution.
// After Build(), each call to Exec() creates a fresh container from the image.
type Sandbox struct {
	client   *client.Client
	imageTag string
	workDir  string
	repoDir  string
	envVars  []string

	mu         sync.Mutex
	containers []string // track all containers for cleanup
}

// New creates a new Sandbox by initializing a Docker client.
func New() (*Sandbox, error) {
	ensureDockerHost()
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("docker: failed to create client: %w", err)
	}

	return &Sandbox{
		client:  cli,
		workDir: "/workspace",
	}, nil
}

// SetEnv sets environment variables to pass into containers.
func (s *Sandbox) SetEnv(env []string) {
	s.envVars = env
}

// CheckAvailable verifies the Docker daemon is reachable.
func CheckAvailable(ctx context.Context) error {
	ensureDockerHost()
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

// ensureDockerHost sets DOCKER_HOST if not already set, by discovering
// common socket locations (Colima, Rancher, etc.).
func ensureDockerHost() {
	if os.Getenv("DOCKER_HOST") != "" {
		return
	}
	if _, err := os.Stat("/var/run/docker.sock"); err == nil {
		return
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	candidates := []string{
		filepath.Join(home, ".colima", "default", "docker.sock"),
		filepath.Join(home, ".colima", "docker.sock"),
		filepath.Join(home, ".rd", "docker.sock"),
	}
	for _, sock := range candidates {
		if _, err := os.Stat(sock); err == nil {
			os.Setenv("DOCKER_HOST", "unix://"+sock)
			return
		}
	}
}

// Build builds a Docker image from a Dockerfile in the given directory.
func (s *Sandbox) Build(ctx context.Context, contextDir, dockerfilePath string, verbose bool) error {
	buildContext, err := createTarArchive(contextDir)
	if err != nil {
		return fmt.Errorf("docker: create build context: %w", err)
	}
	defer buildContext.Close()

	s.imageTag = fmt.Sprintf("revv-sandbox-%d", time.Now().UnixNano())

	absRepoDir, err := filepath.Abs(contextDir)
	if err != nil {
		return fmt.Errorf("docker: resolve repo path: %w", err)
	}
	s.repoDir = absRepoDir

	options := types.ImageBuildOptions{
		Dockerfile:  dockerfilePath,
		Tags:        []string{s.imageTag},
		Remove:      true,
		ForceRemove: true,
	}

	res, err := s.client.ImageBuild(ctx, buildContext, options)
	if err != nil {
		return fmt.Errorf("docker: image build request: %w", err)
	}
	defer res.Body.Close()

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

// Exec creates a fresh container, runs the command, captures output, and removes the container.
// This provides full isolation — each test gets a clean filesystem from the image.
func (s *Sandbox) Exec(ctx context.Context, cmd []string) (*runner.ExecResult, error) {
	if s.imageTag == "" {
		return nil, fmt.Errorf("docker: no image built — call Build() first")
	}
	if len(cmd) == 0 {
		return nil, fmt.Errorf("docker: exec command is empty")
	}

	start := time.Now()

	// Create a fresh container from the cached image
	resp, err := s.client.ContainerCreate(ctx,
		&container.Config{
			Image:      s.imageTag,
			WorkingDir: s.workDir,
			Cmd:        cmd,
			Tty:        false,
			Env:        s.envVars,
		},
		&container.HostConfig{
			Resources: container.Resources{
				Memory:   512 * 1024 * 1024, // 512MB
				NanoCPUs: 2e9,               // 2 CPU cores
			},
			SecurityOpt: []string{"no-new-privileges"},
		},
		nil, nil, "",
	)
	if err != nil {
		return nil, fmt.Errorf("docker: create container: %w", err)
	}

	containerID := resp.ID
	s.trackContainer(containerID)

	// Attach to capture stdout/stderr before starting
	attachResp, err := s.client.ContainerAttach(ctx, containerID, container.AttachOptions{
		Stdout: true,
		Stderr: true,
		Stream: true,
	})
	if err != nil {
		s.removeContainer(containerID)
		return nil, fmt.Errorf("docker: attach container: %w", err)
	}

	// Start the container
	if err := s.client.ContainerStart(ctx, containerID, container.StartOptions{}); err != nil {
		attachResp.Close()
		s.removeContainer(containerID)
		return nil, fmt.Errorf("docker: start container: %w", err)
	}

	// Read output
	var stdout, stderr bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdout, &stderr, attachResp.Reader)
	attachResp.Close()

	// Wait for container to finish
	statusCh, errCh := s.client.ContainerWait(ctx, containerID, container.WaitConditionNotRunning)
	var exitCode int
	select {
	case err := <-errCh:
		if err != nil {
			if ctx.Err() != nil {
				exitCode = -1
			} else {
				s.removeContainer(containerID)
				return nil, fmt.Errorf("docker: wait for container: %w", err)
			}
		}
	case status := <-statusCh:
		exitCode = int(status.StatusCode)
	}

	// Clean up container
	s.removeContainer(containerID)

	return &runner.ExecResult{
		ExitCode: exitCode,
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		Duration: time.Since(start),
		TimedOut: ctx.Err() == context.DeadlineExceeded,
	}, nil
}

// Stop removes the built image and cleans up any remaining containers. Idempotent.
func (s *Sandbox) Stop(ctx context.Context) error {
	cleanupCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	s.mu.Lock()
	containers := make([]string, len(s.containers))
	copy(containers, s.containers)
	s.containers = nil
	s.mu.Unlock()

	for _, id := range containers {
		_ = s.client.ContainerRemove(cleanupCtx, id, container.RemoveOptions{
			Force:         true,
			RemoveVolumes: true,
		})
	}

	if s.imageTag != "" {
		_, _ = s.client.ImageRemove(cleanupCtx, s.imageTag, image.RemoveOptions{Force: true})
		s.imageTag = ""
	}

	if s.client != nil {
		s.client.Close()
	}

	return nil
}

func (s *Sandbox) trackContainer(id string) {
	s.mu.Lock()
	s.containers = append(s.containers, id)
	s.mu.Unlock()
}

func (s *Sandbox) removeContainer(id string) {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	_ = s.client.ContainerRemove(ctx, id, container.RemoveOptions{
		Force:         true,
		RemoveVolumes: true,
	})

	s.mu.Lock()
	for i, cid := range s.containers {
		if cid == id {
			s.containers = append(s.containers[:i], s.containers[i+1:]...)
			break
		}
	}
	s.mu.Unlock()
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
