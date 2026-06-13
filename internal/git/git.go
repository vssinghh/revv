package git

import (
	"fmt"
	"os/exec"
	"strings"
)

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

// GetDiff returns the diff of uncommitted changes.
func GetDiff(dir string) (string, error) {
	cmd := exec.Command("git", "diff", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff: %w", err)
	}
	return string(output), nil
}
