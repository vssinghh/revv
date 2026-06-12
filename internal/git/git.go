package git

import (
	"fmt"
	"os/exec"
	"strings"
)

// PrepareBranchAndCommit switches to (or creates) the branch 'revv/init',
// stages the specified files, and commits them.
func PrepareBranchAndCommit(dir string, files []string) error {
	if len(files) == 0 {
		return nil
	}

	// Checkout/create the branch
	cmdCheckout := exec.Command("git", "checkout", "-B", "revv/init")
	cmdCheckout.Dir = dir
	if output, err := cmdCheckout.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to checkout branch revv/init: %w (output: %s)", err, string(output))
	}

	// Stage the files
	addArgs := append([]string{"add"}, files...)
	cmdAdd := exec.Command("git", addArgs...)
	cmdAdd.Dir = dir
	if output, err := cmdAdd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to stage files: %w (output: %s)", err, string(output))
	}

	// Commit changes
	// We check if there's anything staged to commit to avoid failing on empty commit
	cmdStatus := exec.Command("git", "diff", "--cached", "--quiet")
	cmdStatus.Dir = dir
	// If exit code is not 0, there are changes staged
	if err := cmdStatus.Run(); err != nil {
		cmdCommit := exec.Command("git", "commit", "-m", "Initialize revv configuration")
		cmdCommit.Dir = dir
		if output, err := cmdCommit.CombinedOutput(); err != nil {
			return fmt.Errorf("failed to commit changes: %w (output: %s)", err, string(output))
		}
	}

	return nil
}

// GetCurrentBranch returns the current branch name.
func GetCurrentBranch(dir string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--abbrev-ref", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get current branch: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}

// FetchAndCheckout fetches a remote branch and checks it out.
func FetchAndCheckout(dir string, branch string) error {
	// Fetch the branch from origin
	cmdFetch := exec.Command("git", "fetch", "origin", branch)
	cmdFetch.Dir = dir
	if output, err := cmdFetch.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to fetch branch %q: %w (output: %s)", branch, err, string(output))
	}

	// Checkout the branch
	cmdCheckout := exec.Command("git", "checkout", branch)
	cmdCheckout.Dir = dir
	if output, err := cmdCheckout.CombinedOutput(); err != nil {
		// Branch might not exist locally yet — try creating from remote
		cmdCheckout2 := exec.Command("git", "checkout", "-b", branch, "origin/"+branch)
		cmdCheckout2.Dir = dir
		if output2, err2 := cmdCheckout2.CombinedOutput(); err2 != nil {
			return fmt.Errorf("failed to checkout branch %q: %w (output: %s / %s)", branch, err, string(output), string(output2))
		}
	}

	return nil
}

// RestoreBranch switches back to a previously saved branch.
func RestoreBranch(dir string, branch string) error {
	cmd := exec.Command("git", "checkout", branch)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("failed to restore branch %q: %w (output: %s)", branch, err, string(output))
	}
	return nil
}
