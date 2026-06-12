package analysis

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/vipinsingh/revv/internal/adk"
	"github.com/vipinsingh/revv/internal/runner"
)

// Analysis holds the complete LLM analysis results.
type Analysis struct {
	FailureExplanations []FailureExplanation
	CoverageGaps        []CoverageGap
	GeneratedTests      []GeneratedTest
}

// FailureExplanation explains why a test failed.
type FailureExplanation struct {
	Category    string `json:"category"`
	Name        string `json:"name"`
	Explanation string `json:"explanation"`
	Suggestion  string `json:"suggestion"`
}

// CoverageGap identifies an untested change in the PR.
type CoverageGap struct {
	File        string `json:"file"`
	Description string `json:"description"`
	Severity    string `json:"severity"`
}

// GeneratedTest is an ephemeral test created by the LLM for this PR.
type GeneratedTest struct {
	Category string             `json:"category"`
	Name     string             `json:"name"`
	TestMD   string             `json:"test_md"`
	Result   *runner.TestResult // filled after execution
}

// coverageResponse is the JSON structure returned by the coverage LLM call.
type coverageResponse struct {
	CoverageGaps   []CoverageGap `json:"coverage_gaps"`
	GeneratedTests []struct {
		Category string `json:"category"`
		Name     string `json:"name"`
		TestMD   string `json:"test_md"`
	} `json:"generated_tests"`
}

// failureResponse is the JSON structure returned by the failure analysis LLM call.
type failureResponse struct {
	Explanations []FailureExplanation `json:"explanations"`
}

// Analyze performs full LLM analysis: coverage gaps, test generation, and failure analysis.
func Analyze(ctx context.Context, apiKey, model, diff string,
	existingTests map[string]string, results []runner.TestResult,
	executor runner.Executor) (*Analysis, error) {

	client, err := adk.NewClient(apiKey, model)
	if err != nil {
		return nil, fmt.Errorf("create LLM client for analysis: %w", err)
	}

	analysis := &Analysis{}

	// Step 1: Coverage analysis + test generation (always runs in PR mode)
	if diff != "" {
		fmt.Println("\n🔍 Analyzing PR coverage...")
		coveragePrompt := BuildCoveragePrompt(diff, existingTests)
		coverageJSON, err := client.GenerateRaw(ctx, coveragePrompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: coverage analysis failed: %v\n", err)
		} else {
			var resp coverageResponse
			if err := json.Unmarshal([]byte(coverageJSON), &resp); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to parse coverage response: %v\n", err)
			} else {
				analysis.CoverageGaps = resp.CoverageGaps

				for _, t := range resp.GeneratedTests {
					analysis.GeneratedTests = append(analysis.GeneratedTests, GeneratedTest{
						Category: t.Category,
						Name:     t.Name,
						TestMD:   t.TestMD,
					})
				}
			}
		}

		// Run generated tests if any
		if len(analysis.GeneratedTests) > 0 && executor != nil {
			fmt.Printf("🧪 Running %d generated tests...\n", len(analysis.GeneratedTests))
			for i := range analysis.GeneratedTests {
				gt := &analysis.GeneratedTests[i]
				result := runner.RunTest(ctx, executor, gt.Category, gt.Name, gt.TestMD)
				gt.Result = &result
			}
		}
	}

	// Step 2: Failure analysis (only if there are failures)
	var failures []FailureInput
	for _, r := range results {
		if !r.Passed && !r.Skipped {
			failures = append(failures, FailureInput{
				Category: r.Category,
				Name:     r.Name,
				TestMD:   r.TestMDContent,
				Error:    r.Error,
				Output:   r.Output,
			})
		}
	}
	// Also check generated test failures
	for _, gt := range analysis.GeneratedTests {
		if gt.Result != nil && !gt.Result.Passed && !gt.Result.Skipped {
			failures = append(failures, FailureInput{
				Category: gt.Category,
				Name:     gt.Name,
				TestMD:   gt.TestMD,
				Error:    gt.Result.Error,
				Output:   gt.Result.Output,
			})
		}
	}

	if len(failures) > 0 {
		fmt.Printf("💡 Analyzing %d failure(s)...\n", len(failures))
		failurePrompt := BuildFailurePrompt(failures)
		failureJSON, err := client.GenerateRaw(ctx, failurePrompt)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: failure analysis failed: %v\n", err)
		} else {
			var resp failureResponse
			if err := json.Unmarshal([]byte(failureJSON), &resp); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to parse failure response: %v\n", err)
			} else {
				analysis.FailureExplanations = resp.Explanations
			}
		}
	}

	return analysis, nil
}

// CollectExistingTests reads all test.md files from .revv/ and returns them as a map.
func CollectExistingTests(revvDir string) (map[string]string, error) {
	tests := make(map[string]string)

	testPaths, err := runner.DiscoverTests(revvDir)
	if err != nil {
		return nil, err
	}

	for _, tp := range testPaths {
		rel, err := filepath.Rel(revvDir, tp)
		if err != nil {
			continue
		}
		content, err := os.ReadFile(tp)
		if err != nil {
			continue
		}
		tests[rel] = string(content)
	}

	return tests, nil
}

// PrintAnalysis prints the analysis results to stdout.
func PrintAnalysis(a *Analysis) {
	if len(a.CoverageGaps) > 0 {
		fmt.Println("\n🔍 Coverage Gaps:")
		for _, g := range a.CoverageGaps {
			icon := "🟡"
			if strings.EqualFold(g.Severity, "high") {
				icon = "🔴"
			}
			fmt.Printf("  %s %s — %s\n", icon, g.File, g.Description)
		}
	}

	if len(a.GeneratedTests) > 0 {
		fmt.Println("\n🧪 Generated Tests:")
		for _, gt := range a.GeneratedTests {
			if gt.Result == nil {
				fmt.Printf("  ⏭️  %s/%s (not executed)\n", gt.Category, gt.Name)
			} else if gt.Result.Passed {
				fmt.Printf("  ✓ %s/%s  PASS  (%.1fs)\n", gt.Category, gt.Name, gt.Result.Duration.Seconds())
			} else {
				fmt.Printf("  ✗ %s/%s  FAIL  %s\n", gt.Category, gt.Name, gt.Result.Error)
			}
		}
	}

	if len(a.FailureExplanations) > 0 {
		fmt.Println("\n💡 Failure Analysis:")
		for _, e := range a.FailureExplanations {
			fmt.Printf("  %s/%s\n", e.Category, e.Name)
			fmt.Printf("    Why: %s\n", e.Explanation)
			fmt.Printf("    Fix: %s\n\n", e.Suggestion)
		}
	}
}
