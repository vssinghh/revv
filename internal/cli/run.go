package cli

import (
	stdctx "context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vipinsingh/revv/internal/runner"
	"github.com/vipinsingh/revv/internal/sandbox"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run tests from .revv/ inside a Docker sandbox",
		Long:  `Builds a Docker image from .revv/Dockerfile, starts isolated containers, and executes all discovered tests in parallel.`,
		Args:  cobra.NoArgs,
		RunE:  runRun,
	}

	cmd.Flags().String("category", "", "Run only tests in a specific category (e.g., 'unit', 'build')")
	cmd.Flags().String("test", "", "Run a single test by path (e.g., 'unit/build_check')")
	cmd.Flags().String("model", "gemini-3.1-flash-lite", "Gemini model for analysis (reserved for future use)")

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	category, _ := cmd.Flags().GetString("category")
	testFilter, _ := cmd.Flags().GetString("test")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	revvDir := filepath.Join(cwd, ".revv")
	dockerfilePath := filepath.Join(revvDir, "Dockerfile")

	// Verify .revv/ exists
	if _, err := os.Stat(revvDir); os.IsNotExist(err) {
		return fmt.Errorf("no .revv/ directory found. Run 'revv init' first to generate configuration")
	}

	// Verify Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("no .revv/Dockerfile found. Run 'revv init' to generate one")
	}

	// Check Docker availability (auto-install if needed)
	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), timeout)
	defer cancel()

	fmt.Println("Checking Docker availability...")
	if err := sandbox.EnsureDocker(ctx, verbose); err != nil {
		return err
	}

	// Detect environment variables from test files
	testPaths, err := runner.DiscoverTests(revvDir)
	if err != nil {
		return fmt.Errorf("failed to discover tests: %w", err)
	}

	var testContents []string
	for _, tp := range testPaths {
		content, err := os.ReadFile(tp)
		if err == nil {
			testContents = append(testContents, string(content))
		}
	}

	// Look for .env files
	var envFiles []string
	for _, candidate := range []string{
		filepath.Join(cwd, ".env"),
		filepath.Join(revvDir, ".env"),
	} {
		if _, err := os.Stat(candidate); err == nil {
			envFiles = append(envFiles, candidate)
		}
	}

	envVars, envStatuses := runner.DetectEnvVars(testContents, envFiles)

	if len(envStatuses) > 0 {
		fmt.Println("\nEnvironment variables detected from tests:")
		for _, s := range envStatuses {
			if s.Set {
				fmt.Printf("  %-30s ✓ (%s)\n", s.Name, s.Source)
			} else {
				fmt.Printf("  %-30s ✗ (not set)\n", s.Name)
			}
		}
		fmt.Println()
	}

	// Build image
	fmt.Println("Building sandbox from .revv/Dockerfile...")
	sb, err := sandbox.New()
	if err != nil {
		return fmt.Errorf("failed to create sandbox: %w", err)
	}
	defer sb.Stop(ctx)

	sb.SetEnv(envVars)

	if err := sb.Build(ctx, cwd, ".revv/Dockerfile", verbose); err != nil {
		return fmt.Errorf("failed to build sandbox image: %w", err)
	}

	// Run tests (parallel, one container per test)
	filter := runner.FilterOpts{
		Category: category,
		Test:     testFilter,
	}

	fmt.Println("Running tests (parallel, isolated containers):")
	results, err := runner.RunAll(ctx, sb, revvDir, filter)
	if err != nil {
		return fmt.Errorf("failed to run tests: %w", err)
	}

	// Print results
	for _, r := range results {
		icon := "✓"
		status := "PASS"
		if r.Skipped {
			icon = "─"
			status = "SKIP"
		} else if !r.Passed {
			icon = "✗"
			status = "FAIL"
		}

		durStr := fmt.Sprintf("(%.1fs)", r.Duration.Seconds())
		if r.Skipped {
			durStr = "(no commands)"
		}

		fmt.Printf("  %s %-40s %-10s %-6s %s\n", icon, r.Category+"/"+r.Name, r.Priority, status, durStr)

		if verbose && !r.Passed && !r.Skipped {
			if r.Error != "" {
				fmt.Printf("    Error: %s\n", r.Error)
			}
			if r.Output != "" {
				lines := strings.Split(strings.TrimSpace(r.Output), "\n")
				for _, line := range lines {
					fmt.Printf("    │ %s\n", line)
				}
			}
		}
	}

	// Summary
	fmt.Print(runner.Summary(results))
	fmt.Println("\nSandbox cleaned up.")

	if runner.HasBlockingFailure(results) {
		return fmt.Errorf("blocking tests failed")
	}

	return nil
}
