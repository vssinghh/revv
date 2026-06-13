package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// TestResult holds the outcome of a single test execution.
type TestResult struct {
	Category      string
	Name          string
	Priority      string
	Passed        bool
	Skipped       bool
	Output        string
	Duration      time.Duration
	Error         string
	TestMDContent string // raw test.md content (used for failure analysis)
}

// Executor is the interface for running commands inside a sandbox.
type Executor interface {
	Exec(ctx context.Context, cmd []string) (*ExecResult, error)
}

// ExecResult holds the output of a command execution.
type ExecResult struct {
	ExitCode int
	Stdout   string
	Stderr   string
	Duration time.Duration
	TimedOut bool
}

// DiscoverTests walks the .revv/ directory and returns all test.md file paths,
// sorted by category/name.
func DiscoverTests(revvDir string) ([]string, error) {
	var tests []string

	err := filepath.Walk(revvDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil // skip unreadable
		}
		if info.IsDir() {
			return nil
		}
		if info.Name() == "test.md" {
			tests = append(tests, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("discover tests: %w", err)
	}

	sort.Strings(tests)
	return tests, nil
}

// TestInfo extracts category and name from a test.md path relative to the .revv/ directory.
// e.g., ".revv/unit/build_check/test.md" -> category="unit", name="build_check"
func TestInfo(revvDir, testPath string) (category, name string) {
	rel, err := filepath.Rel(revvDir, testPath)
	if err != nil {
		return "unknown", "unknown"
	}
	parts := strings.Split(filepath.ToSlash(rel), "/")
	if len(parts) >= 2 {
		category = parts[0]
		name = parts[1]
	} else if len(parts) == 1 {
		category = "default"
		name = strings.TrimSuffix(parts[0], ".md")
	}
	return
}

// RunAll discovers and executes all tests in the .revv/ directory.
// Tests run in parallel — each gets its own container for full isolation.
func RunAll(ctx context.Context, executor Executor, revvDir string, filter FilterOpts) ([]TestResult, error) {
	testPaths, err := DiscoverTests(revvDir)
	if err != nil {
		return nil, err
	}

	type testJob struct {
		index    int
		category string
		name     string
		content  string
	}

	var jobs []testJob
	for _, tp := range testPaths {
		category, name := TestInfo(revvDir, tp)

		if filter.Category != "" && category != filter.Category {
			continue
		}
		if filter.Test != "" && (category+"/"+name) != filter.Test {
			continue
		}

		content, err := os.ReadFile(tp)
		if err != nil {
			jobs = append(jobs, testJob{
				index:    len(jobs),
				category: category,
				name:     name,
				content:  "",
			})
			continue
		}

		jobs = append(jobs, testJob{
			index:    len(jobs),
			category: category,
			name:     name,
			content:  string(content),
		})
	}

	results := make([]TestResult, len(jobs))
	var wg sync.WaitGroup

	for _, job := range jobs {
		wg.Add(1)
		go func(j testJob) {
			defer wg.Done()
			if j.content == "" {
				results[j.index] = TestResult{
					Category: j.category,
					Name:     j.name,
					Priority: "warning",
					Passed:   false,
					Error:    "failed to read test file",
				}
				return
			}
			results[j.index] = RunTest(ctx, executor, j.category, j.name, j.content)
		}(job)
	}

	wg.Wait()
	return results, nil
}

// RunTest parses and executes a single test.
func RunTest(ctx context.Context, executor Executor, category, name, content string) TestResult {
	parsed, err := ParseTestMD(content)
	if err != nil {
		return TestResult{
			Category: category,
			Name:     name,
			Priority: "warning",
			Passed:   false,
			Error:    fmt.Sprintf("failed to parse test.md: %v", err),
		}
	}

	result := TestResult{
		Category:      category,
		Name:          name,
		Priority:      parsed.Priority,
		TestMDContent: content,
	}

	// Skip tests with no commands
	if parsed.NoCommands {
		result.Skipped = true
		result.Passed = true
		result.Error = "no commands to execute"
		return result
	}

	// Execute via sandbox
	execResult, err := executor.Exec(ctx, []string{"sh", "-c", parsed.Commands})
	if err != nil {
		result.Passed = false
		result.Error = fmt.Sprintf("execution error: %v", err)
		return result
	}

	result.Duration = execResult.Duration
	result.Output = execResult.Stdout
	if execResult.Stderr != "" {
		result.Output += "\n" + execResult.Stderr
	}

	if execResult.TimedOut {
		result.Passed = false
		result.Error = "test timed out"
		return result
	}

	result.Passed = execResult.ExitCode == 0
	if !result.Passed {
		result.Error = fmt.Sprintf("exit code %d", execResult.ExitCode)
	}

	return result
}

// FilterOpts controls which tests to run.
type FilterOpts struct {
	Category string // only run tests in this category
	Test     string // only run this specific test (category/name)
}

// Summary generates a human-readable summary of test results.
func Summary(results []TestResult) string {
	var sb strings.Builder
	var passed, failed, skipped int
	var blockingPassed, blockingTotal int

	for _, r := range results {
		if r.Skipped {
			skipped++
			continue
		}
		if r.Passed {
			passed++
		} else {
			failed++
		}
		if r.Priority == "blocking" {
			blockingTotal++
			if r.Passed {
				blockingPassed++
			}
		}
	}

	sb.WriteString(fmt.Sprintf("\nResults: %d passed, %d failed, %d skipped\n", passed, failed, skipped))

	if blockingTotal > 0 {
		if blockingPassed == blockingTotal {
			sb.WriteString(fmt.Sprintf("Blocking: %d/%d passed ✓\n", blockingPassed, blockingTotal))
		} else {
			sb.WriteString(fmt.Sprintf("Blocking: %d/%d passed ✗\n", blockingPassed, blockingTotal))
		}
	}

	return sb.String()
}

// HasBlockingFailure returns true if any blocking test failed.
func HasBlockingFailure(results []TestResult) bool {
	for _, r := range results {
		if r.Priority == "blocking" && !r.Passed && !r.Skipped {
			return true
		}
	}
	return false
}
