package git

import (
	"os"
	"os/exec"
	"testing"
)

func TestGetCurrentBranch(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "revv_git_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo
	cmdInit := exec.Command("git", "init")
	cmdInit.Dir = tempDir
	if err := cmdInit.Run(); err != nil {
		t.Fatalf("failed to initialize git repo: %v", err)
	}

	// Configure git user
	cmdUser := exec.Command("git", "config", "user.name", "Test User")
	cmdUser.Dir = tempDir
	_ = cmdUser.Run()
	cmdEmail := exec.Command("git", "config", "user.email", "test@example.com")
	cmdEmail.Dir = tempDir
	_ = cmdEmail.Run()

	// Create initial commit so branch exists
	if err := os.WriteFile(tempDir+"/test.txt", []byte("test"), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
	cmdAdd := exec.Command("git", "add", ".")
	cmdAdd.Dir = tempDir
	_ = cmdAdd.Run()
	cmdCommit := exec.Command("git", "commit", "-m", "init")
	cmdCommit.Dir = tempDir
	_ = cmdCommit.Run()

	branch, err := GetCurrentBranch(tempDir)
	if err != nil {
		t.Fatalf("GetCurrentBranch failed: %v", err)
	}

	// Default branch is usually "main" or "master"
	if branch != "main" && branch != "master" {
		t.Errorf("expected 'main' or 'master', got %q", branch)
	}
}
