package e2e

import (
	"bytes"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

var revvBinaryPath string

func TestMain(m *testing.M) {
	// Create a temp directory for compiling the binary
	tmpDir, err := os.MkdirTemp("", "revv-binary-*")
	if err != nil {
		log.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	revvBinaryPath = filepath.Join(tmpDir, "revv")

	// Compile the revv binary from ../../cmd/revv
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

	// Inject standard env vars, overriding with whatever is passed
	customEnv := append(os.Environ(), "REVV_MOCK_LLM=true")
	customEnv = append(customEnv, env...)
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

func setupMockRepo(t *testing.T, files map[string]string, initGit bool) string {
	tmpDir, err := os.MkdirTemp("", "revv-test-repo-*")
	if err != nil {
		t.Fatalf("failed to create temp repo: %v", err)
	}

	for filename, content := range files {
		fullPath := filepath.Join(tmpDir, filename)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("failed to create parent dir: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("failed to write file: %v", err)
		}
	}

	if initGit {
		// Initialize git
		cmdInit := exec.Command("git", "init")
		cmdInit.Dir = tmpDir
		if output, err := cmdInit.CombinedOutput(); err != nil {
			t.Fatalf("git init failed: %v, output: %s", err, string(output))
		}

		// Configure local user
		cmdName := exec.Command("git", "config", "user.name", "Test User")
		cmdName.Dir = tmpDir
		_ = cmdName.Run()

		cmdEmail := exec.Command("git", "config", "user.email", "test@example.com")
		cmdEmail.Dir = tmpDir
		_ = cmdEmail.Run()

		// Add and commit files
		if len(files) > 0 {
			cmdAdd := exec.Command("git", "add", ".")
			cmdAdd.Dir = tmpDir
			_ = cmdAdd.Run()

			cmdCommit := exec.Command("git", "commit", "-m", "Initial commit")
			cmdCommit.Dir = tmpDir
			_ = cmdCommit.Run()
		}
	}

	t.Cleanup(func() {
		os.RemoveAll(tmpDir)
	})

	return tmpDir
}

func assertBranchExists(t *testing.T, dir, branch string) {
	cmd := exec.Command("git", "rev-parse", "--verify", branch)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Errorf("expected branch %s to exist, got error: %v", branch, err)
	}
}

func assertFileInLastCommit(t *testing.T, dir, file string) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("failed to run git diff-tree: %v\nOutput: %s", err, string(output))
		return
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	found := false
	for _, line := range lines {
		if line == file {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected file %s to be in the last commit, but got: %v", file, lines)
	}
}

func assertOnlyRevvFilesCommitted(t *testing.T, dir string) {
	cmd := exec.Command("git", "diff-tree", "--no-commit-id", "--name-only", "-r", "HEAD")
	cmd.Dir = dir
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Errorf("failed to run git diff-tree: %v\nOutput: %s", err, string(output))
		return
	}
	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		if line == "" {
			continue
		}
		if !strings.HasPrefix(line, ".revv/") {
			t.Errorf("expected only .revv/ files to be committed, but found: %s", line)
		}
	}
}

// ==========================================
// Tier 1: Feature Coverage (25 tests)
// ==========================================

func TestTier1FeatureCoverage(t *testing.T) {
	t.Run("e2e_f1_help_global", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil, "--help")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "revv") || !strings.Contains(res.Stdout, "init") {
			t.Errorf("help output missing command info: %s", res.Stdout)
		}
	})

	t.Run("e2e_f1_help_init", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil, "init", "--help")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "--model") {
			t.Errorf("init help output missing --model: %s", res.Stdout)
		}
	})

	t.Run("e2e_f1_invalid_command", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil, "invalid")
		if res.ExitCode == 0 {
			t.Errorf("expected exit code to be non-zero for invalid command")
		}
	})

	t.Run("e2e_f1_no_args", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil)
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "revv") {
			t.Errorf("expected usage output, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f1_help_shorthand", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil, "-h")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "Usage:") {
			t.Errorf("expected usage output, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f2_no_api_key", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY="}, "init")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", res.ExitCode)
		}
		expectedErr := "GEMINI_API_KEY environment variable is not set"
		if !strings.Contains(res.Stderr, expectedErr) && !strings.Contains(res.Stdout, expectedErr) {
			t.Errorf("expected error message %q, got stdout: %q, stderr: %q", expectedErr, res.Stdout, res.Stderr)
		}
	})

	t.Run("e2e_f2_valid_key_empty_repo", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if _, err := os.Stat(filepath.Join(dir, ".revv")); err != nil {
			t.Errorf("expected .revv directory to exist: %v", err)
		}
	})

	t.Run("e2e_f2_default_model", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "Invoking Gemini (gemini-3.5-flash)") {
			t.Errorf("expected default model gemini-3.5-flash to be used, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f2_custom_model", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose", "--model", "gemini-2.5-pro")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "Invoking Gemini (gemini-2.5-pro)") {
			t.Errorf("expected custom model gemini-2.5-pro to be used, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f2_invalid_model_flag", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--model", "")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stderr, "model name cannot be empty") && !strings.Contains(res.Stdout, "model name cannot be empty") {
			t.Errorf("expected invalid model error, got: %s, stderr: %s", res.Stdout, res.Stderr)
		}
	})

	t.Run("e2e_f3_contributing_md", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{"CONTRIBUTING.md": "contrib guidelines"}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "Collected context file: CONTRIBUTING.md") {
			t.Errorf("expected CONTRIBUTING.md context file to be logged, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f3_readme_md", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{"README.md": "readme details"}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "Collected context file: README.md") {
			t.Errorf("expected README.md context file to be logged, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f3_makefile", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{"Makefile": "all:\n\tgo build"}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "Collected context file: Makefile") {
			t.Errorf("expected Makefile context file to be logged, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f3_dockerfile", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{"Dockerfile": "FROM alpine"}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "Collected context file: Dockerfile") {
			t.Errorf("expected Dockerfile context file to be logged, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f3_mixed_files", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			"README.md": "readme details",
			"Makefile":  "all:\n\tgo build",
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "Collected context file: README.md") || !strings.Contains(res.Stdout, "Collected context file: Makefile") {
			t.Errorf("expected context collection of README.md and Makefile to be logged, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f4_dockerfile_created", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		df := filepath.Join(dir, ".revv", "Dockerfile")
		if _, err := os.Stat(df); err != nil {
			t.Errorf("expected %s to exist: %v", df, err)
		}
	})

	t.Run("e2e_f4_visual_tests_created", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		visualDir := filepath.Join(dir, ".revv", "visual")
		if info, err := os.Stat(visualDir); err != nil || !info.IsDir() {
			t.Errorf("expected visual tests dir %s to exist as directory: %v", visualDir, err)
		}
	})

	t.Run("e2e_f4_unit_tests_created", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		unitDir := filepath.Join(dir, ".revv", "unit")
		if info, err := os.Stat(unitDir); err != nil || !info.IsDir() {
			t.Errorf("expected unit tests dir %s to exist as directory: %v", unitDir, err)
		}
	})

	t.Run("e2e_f4_test_md_format", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		testMD := filepath.Join(dir, ".revv", "unit", "unit_test", "test.md")
		content, err := os.ReadFile(testMD)
		if err != nil {
			t.Fatalf("failed to read test.md: %v", err)
		}
		sContent := string(content)
		headings := []string{"Description", "Priority", "Commands", "Expected Output"}
		for _, h := range headings {
			if !strings.Contains(sContent, h) {
				t.Errorf("expected heading %q in test markdown, got: %s", h, sContent)
			}
		}
	})

	t.Run("e2e_f4_helpers_created", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		helperFile := filepath.Join(dir, ".revv", "helpers", "check.sh")
		if _, err := os.Stat(helperFile); err != nil {
			t.Errorf("expected helper file %s to exist: %v", helperFile, err)
		}
	})

	t.Run("e2e_f5_branch_exists", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		assertBranchExists(t, dir, "revv/init")
	})

	t.Run("e2e_f5_commit_created", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		// check if HEAD has a commit
		cmdLog := exec.Command("git", "log", "-n", "1", "--oneline")
		cmdLog.Dir = dir
		output, err := cmdLog.CombinedOutput()
		if err != nil {
			t.Errorf("git log failed: %v, output: %s", err, string(output))
		}
	})

	t.Run("e2e_f5_commit_message", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		cmdLog := exec.Command("git", "log", "-n", "1", "--format=%s")
		cmdLog.Dir = dir
		output, err := cmdLog.CombinedOutput()
		if err != nil {
			t.Fatalf("git log failed: %v", err)
		}
		msg := strings.TrimSpace(string(output))
		if !strings.Contains(msg, "Initialize revv configuration") {
			t.Errorf("expected commit message containing 'Initialize revv configuration', got %q", msg)
		}
	})

	t.Run("e2e_f5_files_staged", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		assertOnlyRevvFilesCommitted(t, dir)
	})

	t.Run("e2e_f5_push_instructions", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "git push") {
			t.Errorf("expected push instructions to be printed, got: %s", res.Stdout)
		}
	})
}

// ==========================================
// Tier 2: Boundary & Corner Cases (25 tests)
// ==========================================

func TestTier2BoundaryCorner(t *testing.T) {
	t.Run("e2e_f1_extra_args", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil, "init", "extra_arg")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1 for extra args, got %d", res.ExitCode)
		}
	})

	t.Run("e2e_f1_unknown_flag", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil, "init", "--unknown")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1 for unknown flag, got %d", res.ExitCode)
		}
	})

	t.Run("e2e_f1_duplicate_help", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil, "init", "--help", "-h")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "--model") {
			t.Errorf("expected help output, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f1_help_override", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY="}, "init", "--help")
		if res.ExitCode != 0 {
			t.Errorf("expected help flag to bypass API key check and exit 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_f1_empty_init_arg", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, nil, "init", "")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", res.ExitCode)
		}
	})

	t.Run("e2e_f2_empty_api_key", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY="}, "init")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", res.ExitCode)
		}
	})

	t.Run("e2e_f2_spaces_api_key", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=   "}, "init")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1 for spaces API key, got %d", res.ExitCode)
		}
	})

	t.Run("e2e_f2_large_model_name", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		largeModel := strings.Repeat("a", 10000)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--model", largeModel)
		if res.ExitCode != 0 {
			t.Errorf("expected large model name to run gracefully (exit code 0), got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_f2_special_chars_model", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--model", "gemini-@#$%-pro")
		if res.ExitCode != 0 {
			t.Errorf("expected special character model to succeed, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_f2_env_model_override", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose", "--model", "gemini-override-model")
		if res.ExitCode != 0 {
			t.Errorf("expected success: %d", res.ExitCode)
		}
		if !strings.Contains(res.Stdout, "Invoking Gemini (gemini-override-model)") {
			t.Errorf("expected model flag override in logs, got: %s", res.Stdout)
		}
	})

	t.Run("e2e_f3_empty_files", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			"README.md":       "",
			"CONTRIBUTING.md": "",
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--verbose")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		if !strings.Contains(res.Stdout, "Collected context file: README.md") || !strings.Contains(res.Stdout, "Collected context file: CONTRIBUTING.md") {
			t.Errorf("expected empty context files to be read and logged: %s", res.Stdout)
		}
	})

	t.Run("e2e_f3_massive_files", func(t *testing.T) {
		largeContent := strings.Repeat("A", 1024*1024) // 1MB
		dir := setupMockRepo(t, map[string]string{
			"README.md": largeContent,
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected exit code 0 for massive files, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_f3_no_readable_files", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected success with no files: %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_f3_unreadable_files", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			"README.md": "readme info",
		}, true)
		// Set write-only permissions on README.md (unreadable)
		if err := os.Chmod(filepath.Join(dir, "README.md"), 0200); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		// Restore permissions so cleanup works
		_ = os.Chmod(filepath.Join(dir, "README.md"), 0644)

		// Unreadable files are gracefully skipped, so init should still succeed
		if res.ExitCode != 0 {
			t.Errorf("expected success when context file is unreadable (graceful skip), got exit code %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_f3_special_chars_in_files", func(t *testing.T) {
		nonUTF8 := string([]byte{0xff, 0xfe, 0xfd, 0x00, 0x01})
		dir := setupMockRepo(t, map[string]string{
			"README.md": nonUTF8,
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected success with special characters in files, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_f4_existing_revv_dir", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			".revv/Dockerfile":       "OLD DOCKERFILE CONTENT",
			".revv/helpers/check.sh": "OLD HELPER",
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		// Verify overwritten
		dfContent, err := os.ReadFile(filepath.Join(dir, ".revv", "Dockerfile"))
		if err != nil {
			t.Fatalf("failed to read Dockerfile: %v", err)
		}
		if strings.Contains(string(dfContent), "OLD DOCKERFILE CONTENT") {
			t.Errorf("expected Dockerfile to be overwritten, got: %s", string(dfContent))
		}
	})

	t.Run("e2e_f4_read_only_revv_dir", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		revvDir := filepath.Join(dir, ".revv")
		if err := os.MkdirAll(revvDir, 0755); err != nil {
			t.Fatalf("failed to create .revv: %v", err)
		}
		// Write a file to make sure it exists
		if err := os.WriteFile(filepath.Join(revvDir, "dummy"), []byte("dummy"), 0644); err != nil {
			t.Fatalf("failed to write dummy: %v", err)
		}
		// Make it read-only
		if err := os.Chmod(revvDir, 0400); err != nil {
			t.Fatalf("failed to chmod: %v", err)
		}
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		// Restore permissions so cleanup works
		_ = os.Chmod(revvDir, 0755)

		if res.ExitCode == 0 {
			t.Errorf("expected failure when .revv directory is read-only")
		}
	})

	t.Run("e2e_f4_empty_config_output", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key", "REVV_MOCK_EMPTY=true"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		files, err := os.ReadDir(filepath.Join(dir, ".revv"))
		if err != nil {
			t.Fatalf("failed to read .revv: %v", err)
		}
		for _, f := range files {
			if f.IsDir() && f.Name() != "visual" && f.Name() != "helpers" {
				t.Errorf("expected no other test category directories, but found: %s", f.Name())
			}
		}
	})

	t.Run("e2e_f4_special_chars_test_names", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key", "REVV_MOCK_SPECIAL_CHARS=true"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		specialTest := filepath.Join(dir, ".revv", "special@category", "special#test", "test.md")
		if _, err := os.Stat(specialTest); err != nil {
			t.Errorf("expected special char test file to exist: %v", err)
		}
	})

	t.Run("e2e_f4_large_dockerfile", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key", "REVV_MOCK_LARGE_DOCKERFILE=true"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		df := filepath.Join(dir, ".revv", "Dockerfile")
		info, err := os.Stat(df)
		if err != nil {
			t.Fatalf("failed to stat Dockerfile: %v", err)
		}
		if info.Size() < 1024*1024 {
			t.Errorf("expected Dockerfile size to be at least 1MB, got %d bytes", info.Size())
		}
	})

	t.Run("e2e_f5_no_git_repo", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1 when outside git repo, got %d", res.ExitCode)
		}
	})

	t.Run("e2e_f5_dirty_workdir", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{"README.md": "initial"}, true)
		// Make it dirty
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified"), 0644); err != nil {
			t.Fatalf("failed to modify: %v", err)
		}
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		// Verify branch created and only .revv committed
		assertBranchExists(t, dir, "revv/init")
		assertOnlyRevvFilesCommitted(t, dir)

		// README.md should still be modified (unstaged/uncommitted on revv/init)
		cmdDiff := exec.Command("git", "diff", "README.md")
		cmdDiff.Dir = dir
		output, err := cmdDiff.CombinedOutput()
		if err != nil {
			t.Fatalf("git diff failed: %v", err)
		}
		if !strings.Contains(string(output), "+modified") {
			t.Errorf("expected README.md modifications to remain uncommitted, diff: %s", string(output))
		}
	})

	t.Run("e2e_f5_existing_init_branch", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		// Create branch revv/init beforehand
		cmdBranch := exec.Command("git", "checkout", "-b", "revv/init")
		cmdBranch.Dir = dir
		if err := cmdBranch.Run(); err != nil {
			t.Fatalf("failed to create branch: %v", err)
		}
		// Checkout main branch back
		cmdCheckout := exec.Command("git", "checkout", "-")
		cmdCheckout.Dir = dir
		_ = cmdCheckout.Run()

		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		assertBranchExists(t, dir, "revv/init")
	})

	t.Run("e2e_f5_untracked_files_outside_revv", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		// Create untracked file
		untracked := filepath.Join(dir, "untracked.txt")
		if err := os.WriteFile(untracked, []byte("untracked content"), 0644); err != nil {
			t.Fatalf("failed to write untracked: %v", err)
		}
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		assertOnlyRevvFilesCommitted(t, dir)

		// Verify untracked.txt is still untracked
		cmdStatus := exec.Command("git", "status", "--porcelain", "untracked.txt")
		cmdStatus.Dir = dir
		output, err := cmdStatus.CombinedOutput()
		if err != nil {
			t.Fatalf("git status failed: %v", err)
		}
		if !strings.HasPrefix(string(output), "??") {
			t.Errorf("expected untracked.txt to remain untracked, status: %q", string(output))
		}
	})

	t.Run("e2e_f5_detached_head", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{"README.md": "initial"}, true)
		// Detach HEAD
		cmdDetach := exec.Command("git", "checkout", "HEAD~0")
		cmdDetach.Dir = dir
		if err := cmdDetach.Run(); err != nil {
			t.Fatalf("failed to detach HEAD: %v", err)
		}
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		assertBranchExists(t, dir, "revv/init")
	})
}

// ==========================================
// Tier 3: Cross-Feature Combinations (5 tests)
// ==========================================

func TestTier3CrossFeature(t *testing.T) {
	t.Run("e2e_t3_key_and_model_combinations", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=another-key"}, "init", "--model", "gemini-2.5-pro")
		if res.ExitCode != 0 {
			t.Errorf("expected success with custom key/model combo, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_t3_dirty_repo_and_custom_model", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{"README.md": "initial"}, true)
		if err := os.WriteFile(filepath.Join(dir, "README.md"), []byte("modified"), 0644); err != nil {
			t.Fatalf("failed to modify: %v", err)
		}
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init", "--model", "gemini-2.5-pro")
		if res.ExitCode != 0 {
			t.Errorf("expected success: %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		assertOnlyRevvFilesCommitted(t, dir)
	})

	t.Run("e2e_t3_no_git_and_no_key", func(t *testing.T) {
		dir := setupMockRepo(t, nil, false)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY="}, "init")
		// Missing API key error must take precedence over git repo checks
		if res.ExitCode != 1 {
			t.Errorf("expected exit code 1, got %d", res.ExitCode)
		}
		if !strings.Contains(res.Stderr, "GEMINI_API_KEY environment variable is not set") && !strings.Contains(res.Stdout, "GEMINI_API_KEY environment variable is not set") {
			t.Errorf("expected API key missing error to take precedence, got stdout: %q, stderr: %q", res.Stdout, res.Stderr)
		}
	})

	t.Run("e2e_t3_existing_branch_and_existing_dir", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			".revv/Dockerfile": "OLD CONTENT",
		}, true)
		// Checkout branch revv/init beforehand
		cmdBranch := exec.Command("git", "checkout", "-b", "revv/init")
		cmdBranch.Dir = dir
		_ = cmdBranch.Run()

		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected success with existing branch and dir, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})

	t.Run("e2e_t3_empty_context_and_valid_git", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			"README.md":       "",
			"CONTRIBUTING.md": "",
			"Makefile":        "",
			"Dockerfile":      "",
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected success with empty context files, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
	})
}

// ==========================================
// Tier 4: Real-World Application Scenarios (5 tests)
// ==========================================

func TestTier4RealWorld(t *testing.T) {
	t.Run("e2e_t4_standard_go_repo", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			"README.md":       "# My Go Project\nA Go application.",
			"CONTRIBUTING.md": "Please send PRs.",
			"Makefile":        "build:\n\tgo build -o app cmd/main.go",
			"Dockerfile":      "FROM golang:1.26\nCOPY . /app\n",
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected success on standard Go repo, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		assertBranchExists(t, dir, "revv/init")
	})

	t.Run("e2e_t4_node_repo", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			"package.json": `{"name": "app", "version": "1.0.0"}`,
			"README.md":    "# Node App",
			"Dockerfile":   "FROM node:18\n",
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected success on Node.js repo, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		assertBranchExists(t, dir, "revv/init")
	})

	t.Run("e2e_t4_python_repo", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			"requirements.txt": "requests==2.28.1\n",
			"CONTRIBUTING.md":  "Guidelines...",
			"Makefile":         "test:\n\tpytest tests/",
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected success on Python repo, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		assertBranchExists(t, dir, "revv/init")
	})

	t.Run("e2e_t4_cplusplus_repo", func(t *testing.T) {
		dir := setupMockRepo(t, map[string]string{
			"CMakeLists.txt": "project(cpp_app)\n",
			"README.md":      "# C++ App",
		}, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Errorf("expected success on C++ repo, got %d, stderr: %s", res.ExitCode, res.Stderr)
		}
		assertBranchExists(t, dir, "revv/init")
	})

	t.Run("e2e_t4_multiple_categories", func(t *testing.T) {
		dir := setupMockRepo(t, nil, true)
		res := runRevv(t, dir, []string{"GEMINI_API_KEY=test-key"}, "init")
		if res.ExitCode != 0 {
			t.Fatalf("expected success: %d", res.ExitCode)
		}
		// Standard rich configuration generates: unit, integration, lint, visual, build
		categories := []string{"unit", "integration", "lint", "visual", "build"}
		for _, cat := range categories {
			catPath := filepath.Join(dir, ".revv", cat)
			if info, err := os.Stat(catPath); err != nil || !info.IsDir() {
				t.Errorf("expected category directory %s to be created: %v", catPath, err)
			}
		}
	})
}
