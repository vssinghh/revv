package github

import (
	"fmt"
	"strings"
	"time"

	"github.com/vipinsingh/revv/internal/runner"
)

// AnalysisData holds LLM analysis results for comment formatting.
type AnalysisData struct {
	FailureExplanations []FailureExplanation
	CoverageGaps        []CoverageGap
	GeneratedTests      []GeneratedTestResult
}

// FailureExplanation explains why a test failed.
type FailureExplanation struct {
	Category    string
	Name        string
	Explanation string
	Suggestion  string
}

// CoverageGap identifies an untested change.
type CoverageGap struct {
	File        string
	Description string
	Severity    string
}

// GeneratedTestResult is a generated test with its execution result.
type GeneratedTestResult struct {
	Category string
	Name     string
	Passed   bool
	Error    string
	Duration time.Duration
}

// FormatComment formats test results as a markdown PR comment.
func FormatComment(results []runner.TestResult, prNumber int, branch string) string {
	return FormatCommentWithAnalysis(results, prNumber, branch, nil)
}

// FormatCommentWithAnalysis formats test results plus LLM analysis as a markdown PR comment.
func FormatCommentWithAnalysis(results []runner.TestResult, prNumber int, branch string, analysis *AnalysisData) string {
	var sb strings.Builder
	var passed, failed, skipped int
	var blockingPassed, blockingTotal int
	var totalDuration time.Duration

	for _, r := range results {
		totalDuration += r.Duration
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

	allBlockingPass := blockingPassed == blockingTotal

	// Header
	statusEmoji := "✅"
	statusText := "All tests passed"
	if failed > 0 {
		statusEmoji = "❌"
		statusText = fmt.Sprintf("%d test(s) failed", failed)
	}

	sb.WriteString(fmt.Sprintf("## %s Revv Test Results\n\n", statusEmoji))
	sb.WriteString(fmt.Sprintf("**%s** | %d passed, %d failed, %d skipped | ⏱️ %.1fs | 🕐 %s\n\n",
		statusText, passed, failed, skipped,
		totalDuration.Seconds(),
		time.Now().UTC().Format("Jan 2, 2006 3:04 PM UTC"),
	))

	// Results table
	sb.WriteString("| Status | Test | Priority | Duration |\n")
	sb.WriteString("|--------|------|----------|----------|\n")

	for _, r := range results {
		icon := "✅"
		dur := fmt.Sprintf("%.1fs", r.Duration.Seconds())
		if r.Skipped {
			icon = "⏭️"
			dur = "—"
		} else if !r.Passed {
			icon = "❌"
		}
		sb.WriteString(fmt.Sprintf("| %s | `%s/%s` | %s | %s |\n",
			icon, r.Category, r.Name, r.Priority, dur))
	}

	sb.WriteString("\n")

	// Failure details (collapsible)
	var failures []runner.TestResult
	for _, r := range results {
		if !r.Passed && !r.Skipped {
			failures = append(failures, r)
		}
	}

	if len(failures) > 0 {
		for _, r := range failures {
			sb.WriteString(fmt.Sprintf("<details>\n<summary>❌ %s/%s — %s</summary>\n\n",
				r.Category, r.Name, r.Error))
			if r.Output != "" {
				sb.WriteString("```\n")
				output := r.Output
				if len(output) > 2000 {
					output = output[:2000] + "\n... (truncated)"
				}
				sb.WriteString(output)
				sb.WriteString("\n```\n")
			}
			sb.WriteString("\n</details>\n\n")
		}
	}

	// LLM Analysis sections
	if analysis != nil {
		// Failure explanations
		if len(analysis.FailureExplanations) > 0 {
			sb.WriteString("### 💡 Failure Analysis\n\n")
			for _, e := range analysis.FailureExplanations {
				sb.WriteString(fmt.Sprintf("**%s/%s** — %s\n\n", e.Category, e.Name, e.Explanation))
				sb.WriteString(fmt.Sprintf("> **Suggested fix:** %s\n\n", e.Suggestion))
			}
		}

		// Coverage gaps
		if len(analysis.CoverageGaps) > 0 {
			sb.WriteString("### 🔍 Coverage Gaps\n\n")
			sb.WriteString("| File Changed | Gap | Severity |\n")
			sb.WriteString("|-------------|-----|----------|\n")
			for _, g := range analysis.CoverageGaps {
				sevIcon := "🟡"
				if strings.EqualFold(g.Severity, "high") {
					sevIcon = "🔴"
				}
				sb.WriteString(fmt.Sprintf("| `%s` | %s | %s %s |\n",
					g.File, g.Description, sevIcon, g.Severity))
			}
			sb.WriteString("\n")
		}

		// Generated tests
		if len(analysis.GeneratedTests) > 0 {
			sb.WriteString("### 🧪 Generated Tests\n\n")
			sb.WriteString("| Status | Test | Duration |\n")
			sb.WriteString("|--------|------|----------|\n")
			for _, gt := range analysis.GeneratedTests {
				icon := "✅"
				dur := fmt.Sprintf("%.1fs", gt.Duration.Seconds())
				if !gt.Passed {
					icon = "❌"
				}
				sb.WriteString(fmt.Sprintf("| %s | `%s/%s` | %s |\n",
					icon, gt.Category, gt.Name, dur))
			}
			sb.WriteString("\n")
		}
	}

	// Footer
	sb.WriteString("---\n")
	if blockingTotal > 0 {
		blockingIcon := "✅"
		if !allBlockingPass {
			blockingIcon = "❌"
		}
		sb.WriteString(fmt.Sprintf("**Blocking: %d/%d passed %s**\n\n", blockingPassed, blockingTotal, blockingIcon))
	}
	sb.WriteString("*Posted by [revv](https://github.com/vssinghh/revv)*\n")
	sb.WriteString(commentMarker + "\n")

	return sb.String()
}
