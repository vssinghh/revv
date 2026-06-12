package github

import (
	"fmt"
	"strings"
	"time"

	"github.com/vipinsingh/revv/internal/runner"
)

// FormatComment formats test results as a markdown PR comment.
func FormatComment(results []runner.TestResult, prNumber int, branch string) string {
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

	// Error details (collapsible)
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
				// Truncate very long output
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

	// Footer
	sb.WriteString("---\n")
	if blockingTotal > 0 {
		blockingIcon := "✅"
		if !allBlockingPass {
			blockingIcon = "❌"
		}
		sb.WriteString(fmt.Sprintf("**Blocking: %d/%d passed %s**\n\n", blockingPassed, blockingTotal, blockingIcon))
	}
	sb.WriteString(fmt.Sprintf("*Posted by [revv](https://github.com/vssinghh/revv)*\n"))
	sb.WriteString(commentMarker + "\n")

	return sb.String()
}
