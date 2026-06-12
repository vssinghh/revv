package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestPrepareBranchAndCommit(t *testing.T) {
	// Create a temporary directory
	tempDir, err := os.MkdirTemp("", "revv_git_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	// Initialize git repo in the temp dir
	cmdInit := exec.Command("git", "init")
	cmdInit.Dir = tempDir
	if err := cmdInit.Run(); err != nil {
		t.Fatalf("failed to initialize git repo: %v", err)
	}

	// Configure git user info for local commit to work in temp repository
	cmdUser := exec.Command("git", "config", "user.name", "Test User")
	cmdUser.Dir = tempDir
	_ = cmdUser.Run()
	cmdEmail := exec.Command("git", "config", "user.email", "test@example.com")
	cmdEmail.Dir = tempDir
	_ = cmdEmail.Run()

	// Ensure we have a default initial commit
	testFile := "initial.txt"
	err = os.WriteFile(filepath.Join(tempDir, testFile), []byte("initial content"), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	cmdAdd := exec.Command("git", "add", testFile)
	cmdAdd.Dir = tempDir
	_ = cmdAdd.Run()

	cmdCommit := exec.Command("git", "commit", "-m", "Initial commit")
	cmdCommit.Dir = tempDir
	_ = cmdCommit.Run()

	// Now write a new file to prepare and commit
	fileToCommit := "file1.txt"
	err = os.WriteFile(filepath.Join(tempDir, fileToCommit), []byte("hello world"), 0644)
	if err != nil {
		t.Fatalf("failed to write file to commit: %v", err)
	}

	// Call the function under test
	err = PrepareBranchAndCommit(tempDir, []string{fileToCommit})
	if err != nil {
		t.Fatalf("PrepareBranchAndCommit failed: %v", err)
	}

	// Verify we are on the correct branch
	cmdBranch := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmdBranch.Dir = tempDir
	out, err := cmdBranch.Output()
	if err != nil {
		t.Fatalf("failed to get current branch: %v", err)
	}

	branchName := string(out)
	// Remove newline
	if len(branchName) > 0 && branchName[len(branchName)-1] == '\n' {
		branchName = branchName[:len(branchName)-1]
	}

	if branchName != "revv/init" {
		t.Errorf("expected branch 'revv/init', got %q", branchName)
	}

	// Verify the file was committed
	cmdLog := exec.Command("git", "log", "-n", "1", "--pretty=format:%s")
	cmdLog.Dir = tempDir
	logOut, err := cmdLog.Output()
	if err != nil {
		t.Fatalf("failed to get git log: %v", err)
	}

	if string(logOut) != "Initialize revv configuration" {
		t.Errorf("expected commit message 'Initialize revv configuration', got %q", string(logOut))
	}
}
