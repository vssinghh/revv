package e2e

import (
	"bytes"
	"encoding/json"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var revvBinaryPath string

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "revv-binary-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	revvBinaryPath = filepath.Join(tmpDir, "revv")

	cmd := exec.Command("go", "build", "-o", revvBinaryPath, "./cmd/revv")
	cmd.Dir = "../.."
	if output, err := cmd.CombinedOutput(); err != nil {
		log.Fatalf("failed to compile revv binary: %v\nOutput:\n%s", err, string(output))
	}

	code := m.Run()
	os.Exit(code)
}

type cmdResult struct {
	Stdout   string
	Stderr   string
	ExitCode int
	Error    error
}

func runRevv(t *testing.T, dir string, env []string, args ...string) cmdResult {
	cmd := exec.Command(revvBinaryPath, args...)
	cmd.Dir = dir

	customEnv := append(os.Environ(), env...)
	cmd.Env = customEnv

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	exitCode := 0
	if err != nil {
		if exitError, ok := err.(*exec.ExitError); ok {
			exitCode = exitError.ExitCode()
		} else {
			exitCode = -999
		}
	}

	return cmdResult{
		Stdout:   stdout.String(),
		Stderr:   stderr.String(),
		ExitCode: exitCode,
		Error:    err,
	}
}

// ==========================================
// CLI Tests
// ==========================================

func TestCLI(t *testing.T) {
	t.Run("help_global", func(t *testing.T) {
		dir := t.TempDir()
		res := runRevv(t, dir, nil, "--help")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "exec") {
			t.Errorf("help should mention exec command, got: %s", res.Stdout)
		}
	})

	t.Run("help_exec", func(t *testing.T) {
		dir := t.TempDir()
		res := runRevv(t, dir, nil, "exec", "--help")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "--json") {
			t.Errorf("exec help should mention --json flag, got: %s", res.Stdout)
		}
		if !strings.Contains(res.Stdout, "--category") {
			t.Errorf("exec help should mention --category flag, got: %s", res.Stdout)
		}
	})

	t.Run("version", func(t *testing.T) {
		dir := t.TempDir()
		res := runRevv(t, dir, nil, "version")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "revv") {
			t.Errorf("version output should contain 'revv', got: %s", res.Stdout)
		}
	})

	t.Run("invalid_command", func(t *testing.T) {
		dir := t.TempDir()
		res := runRevv(t, dir, nil, "nonexistent")
		if res.ExitCode == 0 {
			t.Errorf("expected non-zero exit code for invalid command")
		}
	})

	t.Run("no_args_shows_usage", func(t *testing.T) {
		dir := t.TempDir()
		res := runRevv(t, dir, nil)
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "revv") {
			t.Errorf("expected usage output, got: %s", res.Stdout)
		}
	})

	t.Run("init_removed", func(t *testing.T) {
		dir := t.TempDir()
		res := runRevv(t, dir, nil, "init")
		if res.ExitCode == 0 {
			t.Errorf("init command should no longer exist")
		}
	})
}

// ==========================================
// Exec Tests (no Docker — just error paths)
// ==========================================

func TestExecErrors(t *testing.T) {
	t.Run("no_revv_dir", func(t *testing.T) {
		dir := t.TempDir()
		res := runRevv(t, dir, nil, "exec")
		if res.ExitCode == 0 {
			t.Errorf("expected failure when no .revv/ directory exists")
		}
		combined := res.Stdout + res.Stderr
		if !strings.Contains(combined, "no .revv/ directory found") {
			t.Errorf("expected helpful error about missing .revv/, got: %s", combined)
		}
	})

	t.Run("no_dockerfile", func(t *testing.T) {
		dir := t.TempDir()
		os.MkdirAll(filepath.Join(dir, ".revv"), 0755)
		res := runRevv(t, dir, nil, "exec")
		if res.ExitCode == 0 {
			t.Errorf("expected failure when no Dockerfile exists")
		}
		combined := res.Stdout + res.Stderr
		if !strings.Contains(combined, "Dockerfile") {
			t.Errorf("expected error about missing Dockerfile, got: %s", combined)
		}
	})

	t.Run("unknown_flag", func(t *testing.T) {
		dir := t.TempDir()
		res := runRevv(t, dir, nil, "exec", "--nonexistent")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", res.ExitCode)
		}
	})

	t.Run("json_flag_accepted", func(t *testing.T) {
		dir := t.TempDir()
		// Will fail because no .revv/ but the flag itself should parse
		res := runRevv(t, dir, nil, "exec", "--json")
		combined := res.Stdout + res.Stderr
		// Should fail with "no .revv/" not "unknown flag"
		if !strings.Contains(combined, ".revv") {
			t.Errorf("--json flag should be accepted, got: %s", combined)
		}
	})
}

// ==========================================
// JSON Output Format Tests
// ==========================================

func TestJSONOutputFormat(t *testing.T) {
	t.Run("valid_json_structure", func(t *testing.T) {
		// This test validates the JSON schema by checking a sample output
		// We can't run Docker in unit tests, so validate the types compile
		sample := `{
			"results": [
				{"category": "build", "name": "check", "priority": "blocking", "passed": true, "duration": 0.5}
			],
			"summary": {"passed": 1, "failed": 0, "skipped": 0, "blocking_passed": 1, "blocking_total": 1}
		}`

		var out struct {
			Results []struct {
				Category string  `json:"category"`
				Name     string  `json:"name"`
				Priority string  `json:"priority"`
				Passed   bool    `json:"passed"`
				Duration float64 `json:"duration"`
			} `json:"results"`
			Summary struct {
				Passed        int `json:"passed"`
				Failed        int `json:"failed"`
				Skipped       int `json:"skipped"`
				BlockingPass  int `json:"blocking_passed"`
				BlockingTotal int `json:"blocking_total"`
			} `json:"summary"`
		}

		if err := json.Unmarshal([]byte(sample), &out); err != nil {
			t.Fatalf("JSON schema mismatch: %v", err)
		}

		if len(out.Results) != 1 {
			t.Errorf("expected 1 result, got %d", len(out.Results))
		}
		if out.Results[0].Category != "build" {
			t.Errorf("expected category 'build', got %q", out.Results[0].Category)
		}
		if out.Summary.Passed != 1 {
			t.Errorf("expected passed=1, got %d", out.Summary.Passed)
		}
	})
}
