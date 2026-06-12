package git

import (
	"fmt"
	"os/exec"
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
