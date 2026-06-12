package adk

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

type Client struct {
	apiKey string
	model  string
}

func NewClient(apiKey, model string) (*Client, error) {
	if apiKey == "" {
		return nil, fmt.Errorf("GEMINI_API_KEY environment variable is not set")
	}
	return &Client{
		apiKey: apiKey,
		model:  model,
	}, nil
}

type GeminiSchema struct {
	Type                 string                  `json:"type"`
	Properties           map[string]GeminiSchema `json:"properties,omitempty"`
	Required             []string                `json:"required,omitempty"`
	Items                *GeminiSchema           `json:"items,omitempty"`
	Enum                 []string                `json:"enum,omitempty"`
	Description          string                  `json:"description,omitempty"`
}

type GenerationConfig struct {
	ResponseMimeType string        `json:"responseMimeType,omitempty"`
	ResponseSchema   *GeminiSchema `json:"responseSchema,omitempty"`
}

type Part struct {
	Text string `json:"text"`
}

type Content struct {
	Parts []Part `json:"parts"`
}

type RequestBody struct {
	Contents         []Content         `json:"contents"`
	GenerationConfig *GenerationConfig `json:"generationConfig,omitempty"`
}

type ResponseBody struct {
	Candidates []struct {
		Content struct {
			Parts []struct {
				Text string `json:"text"`
			} `json:"parts"`
		} `json:"content"`
	} `json:"candidates"`
}

func (c *Client) Generate(ctx context.Context, prompt string) (string, error) {
	url := fmt.Sprintf("https://generativelanguage.googleapis.com/v1beta/models/%s:generateContent?key=%s", c.model, c.apiKey)

	actionField := GeminiSchema{
		Type: "STRING",
		Enum: []string{"add", "update", "keep", "delete"},
		Description: "Action to perform: add (new item), update (modify existing), keep (no change), delete (remove)",
	}

	schema := &GeminiSchema{
		Type: "OBJECT",
		Properties: map[string]GeminiSchema{
			"dockerfile": {Type: "STRING"},
			"helpers": {
				Type: "ARRAY",
				Items: &GeminiSchema{
					Type: "OBJECT",
					Properties: map[string]GeminiSchema{
						"filename": {Type: "STRING"},
						"content":  {Type: "STRING"},
						"action":   actionField,
					},
					Required: []string{"filename", "content", "action"},
				},
			},
			"tests": {
				Type: "ARRAY",
				Items: &GeminiSchema{
					Type: "OBJECT",
					Properties: map[string]GeminiSchema{
						"category": {Type: "STRING"},
						"name":     {Type: "STRING"},
						"test_md":  {Type: "STRING"},
						"action":   actionField,
						"helpers": {
							Type: "ARRAY",
							Items: &GeminiSchema{
								Type: "OBJECT",
								Properties: map[string]GeminiSchema{
									"filename": {Type: "STRING"},
									"content":  {Type: "STRING"},
									"action":   actionField,
								},
								Required: []string{"filename", "content", "action"},
							},
						},
					},
					Required: []string{"category", "name", "test_md", "action"},
				},
			},
		},
		Required: []string{"dockerfile", "helpers", "tests"},
	}

	reqBody := RequestBody{
		Contents: []Content{
			{
				Parts: []Part{
					{Text: prompt},
				},
			},
		},
		GenerationConfig: &GenerationConfig{
			ResponseMimeType: "application/json",
			ResponseSchema:   schema,
		},
	}

	jsonData, err := json.Marshal(reqBody)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("HTTP request failed: %w", err)
	}
	defer resp.Body.Close()

	bodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("failed to read response body: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("API returned non-200 status code: %d, body: %s", resp.StatusCode, string(bodyBytes))
	}

	var geminiResp ResponseBody
	if err := json.Unmarshal(bodyBytes, &geminiResp); err != nil {
		return "", fmt.Errorf("failed to unmarshal response: %w", err)
	}

	if len(geminiResp.Candidates) == 0 || len(geminiResp.Candidates[0].Content.Parts) == 0 {
		return "", fmt.Errorf("empty response candidate from Gemini")
	}

	return geminiResp.Candidates[0].Content.Parts[0].Text, nil
}

func ConstructPrompt(repoContext map[string]string, existingConfig map[string]string) string {
	var sb strings.Builder

	// 1. Specific persona
	sb.WriteString(`You are a senior QA engineer and CI/CD specialist. Your job is to configure automated and manual PR review tests for a repository using a tool called "revv".
You have deep expertise in writing tests that are:
- Executable: every command must be a real shell command, not pseudocode
- Specific: assertions check concrete output, not vague descriptions
- Actionable: a reviewer can run them and immediately know pass/fail

`)

	// 2. Auto-generated project summary
	sb.WriteString("## Project Summary\n")
	if gomod, ok := repoContext["go.mod"]; ok {
		sb.WriteString(fmt.Sprintf("This is a **Go** project.\n```\n%s\n```\n\n", gomod))
	} else if pkg, ok := repoContext["package.json"]; ok {
		sb.WriteString(fmt.Sprintf("This is a **Node.js** project.\n```json\n%s\n```\n\n", pkg))
	} else if pyproj, ok := repoContext["pyproject.toml"]; ok {
		sb.WriteString(fmt.Sprintf("This is a **Python** project.\n```toml\n%s\n```\n\n", pyproj))
	} else if cargo, ok := repoContext["Cargo.toml"]; ok {
		sb.WriteString(fmt.Sprintf("This is a **Rust** project.\n```toml\n%s\n```\n\n", cargo))
	}

	// 3. File tree (always present, cheap context)
	if tree, ok := repoContext["__FILE_TREE__"]; ok {
		sb.WriteString("## Repository File Tree\n```\n")
		sb.WriteString(tree)
		sb.WriteString("\n```\n\n")
	}

	// 4. Full repo context
	sb.WriteString("## Repository Source Files\n")
	// Separate existing test files for coverage awareness
	var testFiles []string
	for name, content := range repoContext {
		if name == "__FILE_TREE__" || name == "go.mod" || name == "package.json" || name == "pyproject.toml" || name == "Cargo.toml" {
			continue // already shown above
		}
		if isTestFile(name) {
			testFiles = append(testFiles, name)
		}
		sb.WriteString(fmt.Sprintf("### %s\n```\n", name))
		sb.WriteString(content)
		sb.WriteString("\n```\n\n")
	}

	// 5. Coverage awareness
	if len(testFiles) > 0 {
		sb.WriteString("## Existing Test Coverage\n")
		sb.WriteString("The repository already has these test files. Focus your generated tests on **gaps** — areas NOT covered by existing tests. Do NOT duplicate what these tests already verify:\n")
		for _, tf := range testFiles {
			sb.WriteString(fmt.Sprintf("- `%s`\n", tf))
		}
		sb.WriteString("\n")
	}

	// 6. Existing .revv/ config (incremental mode)
	if len(existingConfig) > 0 {
		sb.WriteString("## Existing .revv/ Configuration (Incremental Update)\n")
		sb.WriteString("This repository already has a `.revv/` configuration. Review it against the current repo state.\n")
		sb.WriteString("For each item, set the `action` field:\n")
		sb.WriteString("- `keep`: item is correct and unchanged\n")
		sb.WriteString("- `update`: item exists but content needs changes (provide the new content)\n")
		sb.WriteString("- `add`: new item that doesn't exist yet\n")
		sb.WriteString("- `delete`: item is obsolete or redundant (content can be empty)\n\n")
		for path, content := range existingConfig {
			sb.WriteString(fmt.Sprintf("### .revv/%s\n```\n", path))
			sb.WriteString(content)
			sb.WriteString("\n```\n\n")
		}
	} else {
		sb.WriteString("## Mode: Fresh Initialization\n")
		sb.WriteString("No existing `.revv/` configuration found. Set the `action` field to `add` for all items.\n\n")
	}

	// 7. Priority definitions
	sb.WriteString(`## Priority Levels
- **blocking**: PR MUST NOT merge if this test fails. Use for: build compilation, critical functionality, security checks, data integrity.
- **warning**: PR CAN merge but the reviewer should be aware. Use for: style/lint issues, non-critical edge cases, nice-to-have checks.

`)

	// 8. Few-shot example
	sb.WriteString(`## Example of a GOOD test.md

Here is an example of a high-quality test. Follow this style:

` + "```" + `markdown
## Description
Verify that the CLI binary compiles without errors and the --help flag produces valid output listing all subcommands.

## Priority
blocking

## Commands
` + "```" + `bash
# Build the binary
make build

# Verify binary exists and is executable
test -x ./bin/revv || (echo "FAIL: binary not found or not executable" && exit 1)

# Verify help output contains expected subcommands
./bin/revv --help 2>&1 | grep -q "init" || (echo "FAIL: 'init' subcommand not in help output" && exit 1)

echo "PASS: CLI binary builds and help output is correct"
` + "```" + `

## Expected Output
Exit code 0. Final line: "PASS: CLI binary builds and help output is correct"
` + "```" + `

Key qualities:
- Commands are copy-paste runnable shell commands
- Each command has an inline assertion (grep -q, test -x) with a clear FAIL message
- Expected Output specifies the exact exit code and output to check
- No vague language like "should work" or "verify manually"

`)

	// 9. Helper guidance
	sb.WriteString(`## Helper Scripts
Helpers are shared shell scripts or utilities that multiple tests can reference. Good helpers:
- Set up common preconditions (ensure binary is built, check environment)
- Provide reusable assertion functions (assert_file_exists, assert_output_contains)
- Handle cleanup (remove temp files, reset state)
- Are small and focused (one purpose per helper)

`)

	// 10. Generation instructions
	sb.WriteString(`## Your Task
Generate:
1. **Dockerfile**: Sets up build and test dependencies for this project. Use the correct base image, install all required tools, and copy source code. Optimize for layer caching (copy dependency files first, then source).
2. **Global helpers**: Array of {filename, content, action} objects. Reusable scripts shared across test categories.
3. **Tests**: Array of {category, name, test_md, action, helpers} objects. Each test_md MUST contain these sections: ## Description, ## Priority, ## Commands, ## Expected Output.

Rules:
- A "manual" category MUST always be generated (for tests requiring human judgment: UI, UX, visual checks)
- Test names must be lowercase_snake_case
- Categories should be meaningful (unit, integration, build, lint, security, manual — not "test1", "test2")
- Generate at least 5 tests across at least 3 categories
- Commands in test_md must be real, executable shell commands with inline assertions
- Do NOT generate tests that duplicate existing test coverage (see "Existing Test Coverage" above)
`)

	return sb.String()
}

// isTestFile returns true if the filename looks like a test file.
func isTestFile(name string) bool {
	lower := strings.ToLower(name)
	return strings.Contains(lower, "_test.go") ||
		strings.Contains(lower, ".test.") ||
		strings.Contains(lower, "test_") ||
		strings.HasSuffix(lower, "_test.py") ||
		strings.Contains(lower, "/tests/") ||
		strings.Contains(lower, "/test/") ||
		strings.Contains(lower, "spec.")
}
