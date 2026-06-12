package runner

import (
	"testing"
)

func TestParseTestMD_FullExample(t *testing.T) {
	content := `## Description
Verify that the CLI binary compiles cleanly using the Makefile.

## Priority
blocking

## Commands
` + "```bash" + `
make clean && make build
test -x ./bin/revv || (echo "FAIL: binary not found" && exit 1)
echo "PASS: compilation successful"
` + "```" + `

## Expected Output
Exit code 0. Output ends with "PASS: compilation successful".
`

	pt, err := ParseTestMD(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pt.Description != "Verify that the CLI binary compiles cleanly using the Makefile." {
		t.Errorf("unexpected description: %q", pt.Description)
	}

	if pt.Priority != "blocking" {
		t.Errorf("expected priority 'blocking', got %q", pt.Priority)
	}

	if pt.NoCommands {
		t.Error("expected NoCommands=false for test with commands")
	}

	if pt.Commands == "" {
		t.Error("expected non-empty commands")
	}

	if !contains(pt.Commands, "make clean") {
		t.Errorf("expected commands to contain 'make clean', got: %q", pt.Commands)
	}

	if pt.Expected == "" {
		t.Error("expected non-empty expected output")
	}
}

func TestParseTestMD_NoCommandsTest(t *testing.T) {
	content := `## Description
Manually verify the UI looks correct.

## Priority
warning

## Expected Output
Reviewer confirms visual correctness.
`

	pt, err := ParseTestMD(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !pt.NoCommands {
		t.Error("expected NoCommands=true for test without commands")
	}

	if pt.Priority != "warning" {
		t.Errorf("expected priority 'warning', got %q", pt.Priority)
	}
}

func TestParseTestMD_DefaultPriority(t *testing.T) {
	content := `## Description
A test without explicit priority.

## Commands
echo "hello"
`

	pt, err := ParseTestMD(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pt.Priority != "warning" {
		t.Errorf("expected default priority 'warning', got %q", pt.Priority)
	}
}

func TestParseTestMD_InvalidPriority(t *testing.T) {
	content := `## Description
Test with bad priority.

## Priority
critical
`

	_, err := ParseTestMD(content)
	if err == nil {
		t.Error("expected error for invalid priority")
	}
}

func TestParseTestMD_CommandsWithoutCodeBlock(t *testing.T) {
	content := `## Description
Simple test.

## Priority
blocking

## Commands
go test ./...
echo "PASS"

## Expected Output
PASS
`

	pt, err := ParseTestMD(content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if pt.Commands == "" {
		t.Error("expected non-empty commands even without code block")
	}

	if pt.NoCommands {
		t.Error("expected NoCommands=false")
	}
}

func TestExtractCodeBlock(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "bash block",
			input:    "```bash\necho hello\n```",
			expected: "echo hello",
		},
		{
			name:     "plain block",
			input:    "```\nfoo\nbar\n```",
			expected: "foo\nbar",
		},
		{
			name:     "no block",
			input:    "just text",
			expected: "",
		},
		{
			name:     "empty block",
			input:    "```\n```",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := extractCodeBlock(tt.input)
			if got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, got)
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsSubstring(s, substr))
}

func containsSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
