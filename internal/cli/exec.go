package cli

import (
	stdctx "context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"github.com/vipinsingh/revv/internal/runner"
	"github.com/vipinsingh/revv/internal/sandbox"
)

func newExecCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "exec",
		Short: "Execute tests from .revv/ inside Docker containers",
		Long:  `Builds a Docker image from .revv/Dockerfile, starts isolated containers, and executes all discovered tests in parallel. No LLM required — this is the pure execution engine.`,
		Args:  cobra.NoArgs,
		RunE:  runExec,
	}

	cmd.Flags().String("category", "", "Run only tests in a specific category (e.g., 'unit', 'build')")
	cmd.Flags().String("test", "", "Run a single test by path (e.g., 'unit/build_check')")
	cmd.Flags().Bool("json", false, "Output results as JSON (for IDE parsing)")

	return cmd
}

// jsonOutput is the structured output for --json mode.
type jsonOutput struct {
	Results []jsonResult `json:"results"`
	Summary jsonSummary  `json:"summary"`
}

type jsonResult struct {
	Category string  `json:"category"`
	Name     string  `json:"name"`
	Priority string  `json:"priority"`
	Passed   bool    `json:"passed"`
	Skipped  bool    `json:"skipped,omitempty"`
	Duration float64 `json:"duration"`
	Error    string  `json:"error,omitempty"`
	Output   string  `json:"output,omitempty"`
}

type jsonSummary struct {
	Passed        int `json:"passed"`
	Failed        int `json:"failed"`
	Skipped       int `json:"skipped"`
	BlockingPass  int `json:"blocking_passed"`
	BlockingTotal int `json:"blocking_total"`
}

func runExec(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	category, _ := cmd.Flags().GetString("category")
	testFilter, _ := cmd.Flags().GetString("test")
	jsonMode, _ := cmd.Flags().GetBool("json")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	revvDir := filepath.Join(cwd, ".revv")
	dockerfilePath := filepath.Join(revvDir, "Dockerfile")

	// Verify .revv/ exists
	if _, err := os.Stat(revvDir); os.IsNotExist(err) {
		return fmt.Errorf("no .revv/ directory found. Ask your IDE to run 'revv update' first")
	}

	// Verify Dockerfile exists
	if _, err := os.Stat(dockerfilePath); os.IsNotExist(err) {
		return fmt.Errorf("no .revv/Dockerfile found. Ask your IDE to run 'revv update' to generate one")
	}

	ctx, cancel := stdctx.WithTimeout(stdctx.Background(), timeout)
	defer cancel()

	if !jsonMode {
		fmt.Println("Checking Docker availability...")
	}
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

	if !jsonMode && len(envStatuses) > 0 {
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
	if !jsonMode {
		fmt.Println("Building sandbox from .revv/Dockerfile...")
	}
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

	if !jsonMode {
		fmt.Println("Running tests (parallel, isolated containers):")
	}
	results, err := runner.RunAll(ctx, sb, revvDir, filter)
	if err != nil {
		return fmt.Errorf("failed to run tests: %w", err)
	}

	// Output results
	if jsonMode {
		return outputJSON(results)
	}

	outputTable(results, verbose)

	if runner.HasBlockingFailure(results) {
		return fmt.Errorf("blocking tests failed")
	}

	return nil
}

func outputJSON(results []runner.TestResult) error {
	out := jsonOutput{}

	for _, r := range results {
		jr := jsonResult{
			Category: r.Category,
			Name:     r.Name,
			Priority: r.Priority,
			Passed:   r.Passed,
			Skipped:  r.Skipped,
			Duration: r.Duration.Seconds(),
		}
		if !r.Passed {
			jr.Error = r.Error
			jr.Output = r.Output
		}
		out.Results = append(out.Results, jr)

		if r.Skipped {
			out.Summary.Skipped++
		} else if r.Passed {
			out.Summary.Passed++
		} else {
			out.Summary.Failed++
		}
		if r.Priority == "blocking" {
			out.Summary.BlockingTotal++
			if r.Passed {
				out.Summary.BlockingPass++
			}
		}
	}

	enc := json.NewEncoder(os.Stdout)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func outputTable(results []runner.TestResult, verbose bool) {
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
	_ = time.Now() // ensure time import is used
}
