package cli

import (
	stdctx "context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"
	"github.com/vipinsingh/revv/internal/analysis"
	"github.com/vipinsingh/revv/internal/git"
	gh "github.com/vipinsingh/revv/internal/github"
	"github.com/vipinsingh/revv/internal/runner"
	"github.com/vipinsingh/revv/internal/sandbox"
)

func newRunCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "run",
		Short: "Run tests from .revv/ inside a Docker sandbox",
		Long:  `Builds a Docker image from .revv/Dockerfile, starts isolated containers, and executes all discovered tests in parallel. Use --pr to test a GitHub PR and post results as a comment.`,
		Args:  cobra.NoArgs,
		RunE:  runRun,
	}

	cmd.Flags().Int("pr", 0, "GitHub PR number to test (fetches branch, runs tests, posts results as PR comment)")
	cmd.Flags().String("category", "", "Run only tests in a specific category (e.g., 'unit', 'build')")
	cmd.Flags().String("test", "", "Run a single test by path (e.g., 'unit/build_check')")
	cmd.Flags().String("model", "gemini-3.5-flash", "LLM model for analysis (reserved for future use)")

	return cmd
}

func runRun(cmd *cobra.Command, args []string) error {
	verbose, _ := cmd.Flags().GetBool("verbose")
	timeout, _ := cmd.Flags().GetDuration("timeout")
	category, _ := cmd.Flags().GetString("category")
	testFilter, _ := cmd.Flags().GetString("test")
	prNumber, _ := cmd.Flags().GetInt("pr")

	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get working directory: %w", err)
	}

	// PR mode: fetch branch, run tests, post comment
	var ghClient *gh.Client
	var originalBranch string
	var prBranch string

	if prNumber > 0 {
		token := os.Getenv("GITHUB_TOKEN")
		if token == "" {
			return fmt.Errorf("GITHUB_TOKEN environment variable is required for --pr mode.\n\nSet it with: export GITHUB_TOKEN=<your-token>\nIn GitHub Actions, it's available automatically.")
		}

		// Create GitHub client from git remote
		ghClient, err = gh.NewFromRemote(token, cwd)
		if err != nil {
			return fmt.Errorf("failed to connect to GitHub: %w", err)
		}

		ctx, cancel := stdctx.WithTimeout(stdctx.Background(), timeout)
		defer cancel()

		// Get PR details
		fmt.Printf("Fetching PR #%d...\n", prNumber)
		pr, err := ghClient.GetPR(ctx, prNumber)
		if err != nil {
			return fmt.Errorf("failed to get PR: %w", err)
		}
		prBranch = pr.Branch
		fmt.Printf("PR #%d: %s (branch: %s)\n", pr.Number, pr.Title, pr.Branch)

		// Save current branch to restore later
		originalBranch, err = git.GetCurrentBranch(cwd)
		if err != nil {
			return fmt.Errorf("failed to get current branch: %w", err)
		}

		// Checkout PR branch
		fmt.Printf("Checking out branch %s...\n", pr.Branch)
		if err := git.FetchAndCheckout(cwd, pr.Branch); err != nil {
			return fmt.Errorf("failed to checkout PR branch: %w", err)
		}

		// Ensure we restore the original branch when done
		defer func() {
			fmt.Printf("\nRestoring branch %s...\n", originalBranch)
			if restoreErr := git.RestoreBranch(cwd, originalBranch); restoreErr != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to restore branch %s: %v\n", originalBranch, restoreErr)
			}
		}()
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

	// LLM Analysis (only in PR mode with API key)
	var analysisData *gh.AnalysisData
	if prNumber > 0 {
		apiKey := os.Getenv("GEMINI_API_KEY")
		modelName, _ := cmd.Flags().GetString("model")
		if apiKey != "" {
			// Get PR diff
			diff, diffErr := analysis.GetPRDiff(cwd, "main")
			if diffErr != nil {
				fmt.Fprintf(os.Stderr, "warning: could not get PR diff: %v\n", diffErr)
			}

			// Collect existing tests for context
			existingTests, _ := analysis.CollectExistingTests(revvDir)

			// Run analysis
			result, analysisErr := analysis.Analyze(ctx, apiKey, modelName, diff, existingTests, results, sb)
			if analysisErr != nil {
				fmt.Fprintf(os.Stderr, "warning: LLM analysis failed: %v\n", analysisErr)
			} else {
				// Print analysis to terminal
				analysis.PrintAnalysis(result)

				// Convert to comment format
				analysisData = &gh.AnalysisData{}
				for _, e := range result.FailureExplanations {
					analysisData.FailureExplanations = append(analysisData.FailureExplanations, gh.FailureExplanation{
						Category:    e.Category,
						Name:        e.Name,
						Explanation: e.Explanation,
						Suggestion:  e.Suggestion,
					})
				}
				for _, g := range result.CoverageGaps {
					analysisData.CoverageGaps = append(analysisData.CoverageGaps, gh.CoverageGap{
						File:        g.File,
						Description: g.Description,
						Severity:    g.Severity,
					})
				}
				for _, gt := range result.GeneratedTests {
					if gt.Result != nil {
						analysisData.GeneratedTests = append(analysisData.GeneratedTests, gh.GeneratedTestResult{
							Category: gt.Category,
							Name:     gt.Name,
							Passed:   gt.Result.Passed,
							Error:    gt.Result.Error,
							Duration: gt.Result.Duration,
						})
					}
				}
			}
		} else if verbose {
			fmt.Println("\nSkipping LLM analysis (GEMINI_API_KEY not set)")
		}
	}

	// Post PR comment if in PR mode
	if ghClient != nil && prNumber > 0 {
		fmt.Printf("\nPosting results to PR #%d...\n", prNumber)
		comment := gh.FormatCommentWithAnalysis(results, prNumber, prBranch, analysisData)
		if err := ghClient.UpsertComment(ctx, prNumber, comment); err != nil {
			fmt.Fprintf(os.Stderr, "warning: failed to post PR comment: %v\n", err)
		} else {
			fmt.Printf("✓ Results posted to PR #%d\n", prNumber)
		}
	}

	fmt.Println("\nSandbox cleaned up.")

	if runner.HasBlockingFailure(results) {
		return fmt.Errorf("blocking tests failed")
	}

	return nil
}
