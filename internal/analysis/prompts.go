package analysis

import (
	"fmt"
	"strings"
)

// BuildCoveragePrompt constructs a prompt for coverage analysis and test generation.
func BuildCoveragePrompt(diff string, existingTests map[string]string) string {
	var sb strings.Builder

	sb.WriteString(`You are a senior QA engineer reviewing a Pull Request. Your job is to:
1. Identify what the PR changes that is NOT adequately covered by the existing test suite.
2. For each coverage gap, generate a test.md file that would verify the change.

## PR Diff
`)
	sb.WriteString("```diff\n")
	sb.WriteString(diff)
	sb.WriteString("\n```\n\n")

	sb.WriteString("## Existing Tests\n\n")
	if len(existingTests) == 0 {
		sb.WriteString("No existing tests.\n\n")
	} else {
		for path, content := range existingTests {
			sb.WriteString(fmt.Sprintf("### %s\n```markdown\n%s\n```\n\n", path, content))
		}
	}

	sb.WriteString(`## Instructions

Analyze the diff carefully. For each meaningful change that is NOT covered by existing tests, generate a new test.

Output a JSON object with this exact structure:
{
  "coverage_gaps": [
    {
      "file": "path/to/changed/file.go",
      "description": "What changed and why it needs a test",
      "severity": "high" or "low"
    }
  ],
  "generated_tests": [
    {
      "category": "generated",
      "name": "descriptive_test_name",
      "test_md": "## Description\n...\n## Priority\nblocking\n## Commands\n` + "```" + `bash\n...\n` + "```" + `\n## Expected Output\n..."
    }
  ]
}

Rules:
- Only flag REAL coverage gaps — don't flag cosmetic changes (comments, whitespace)
- Generated test commands must be real, executable bash commands
- Tests run inside a Docker container with the project built at /workspace
- The binary is at ./bin/revv (already compiled via Makefile)
- Do NOT use mock modes or fake backends
- If no coverage gaps exist, return empty arrays
- Keep test commands simple and self-contained
- Generated tests should have category "generated"
`)

	return sb.String()
}

// BuildFailurePrompt constructs a prompt for analyzing a test failure.
func BuildFailurePrompt(failures []FailureInput) string {
	var sb strings.Builder

	sb.WriteString(`You are a senior QA engineer analyzing test failures from a CI run. For each failed test, explain:
1. WHY it failed (root cause)
2. How to FIX it (actionable suggestion)

## Failed Tests

`)

	for i, f := range failures {
		sb.WriteString(fmt.Sprintf("### Failure %d: %s/%s\n\n", i+1, f.Category, f.Name))
		sb.WriteString("**Test definition:**\n```markdown\n")
		sb.WriteString(f.TestMD)
		sb.WriteString("\n```\n\n")
		sb.WriteString("**Error:** " + f.Error + "\n\n")
		if f.Output != "" {
			sb.WriteString("**Output:**\n```\n")
			output := f.Output
			if len(output) > 3000 {
				output = output[:3000] + "\n... (truncated)"
			}
			sb.WriteString(output)
			sb.WriteString("\n```\n\n")
		}
	}

	sb.WriteString(`## Instructions

Output a JSON object with this exact structure:
{
  "explanations": [
    {
      "category": "category_name",
      "name": "test_name",
      "explanation": "Clear explanation of why this test failed",
      "suggestion": "Specific, actionable suggestion to fix the issue"
    }
  ]
}

Rules:
- Be specific — reference exact lines, commands, or patterns from the test/output
- Distinguish between: test bug (test is wrong), code bug (product has a defect), or environment issue
- Keep explanations concise (2-3 sentences max)
- Keep suggestions actionable (what exact change to make)
`)

	return sb.String()
}

// FailureInput holds the data needed to analyze a single test failure.
type FailureInput struct {
	Category string
	Name     string
	TestMD   string
	Error    string
	Output   string
}
