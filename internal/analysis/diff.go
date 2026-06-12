package analysis

import (
	"fmt"
	"os/exec"
	"strings"
)

// GetPRDiff returns the unified diff between the current branch and the base branch.
// Truncates to maxBytes to fit LLM context windows.
func GetPRDiff(dir, baseBranch string) (string, error) {
	// Try origin/baseBranch first (works for fetched PRs)
	ref := "origin/" + baseBranch
	cmd := exec.Command("git", "diff", ref+"...HEAD", "--unified=3", "--stat")
	cmd.Dir = dir
	statOut, err := cmd.Output()
	if err != nil {
		// Fallback to local branch
		ref = baseBranch
	}

	// Get the full diff
	cmd = exec.Command("git", "diff", ref+"...HEAD", "--unified=3")
	cmd.Dir = dir
	diffOut, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get diff against %s: %w", ref, err)
	}

	diff := string(diffOut)
	if diff == "" {
		return "", fmt.Errorf("no diff found between %s and HEAD", ref)
	}

	// Truncate if too large for LLM context
	const maxBytes = 50000
	if len(diff) > maxBytes {
		diff = diff[:maxBytes] + "\n... (diff truncated, " + fmt.Sprintf("%d", len(diff)-maxBytes) + " bytes omitted)"
	}

	// Prepend stat summary for quick overview
	result := string(statOut) + "\n" + diff
	return strings.TrimSpace(result), nil
}

// GetChangedFiles returns the list of files changed in the PR.
func GetChangedFiles(dir, baseBranch string) ([]string, error) {
	ref := "origin/" + baseBranch
	cmd := exec.Command("git", "diff", ref+"...HEAD", "--name-only")
	cmd.Dir = dir
	out, err := cmd.Output()
	if err != nil {
		// Fallback
		cmd = exec.Command("git", "diff", baseBranch+"...HEAD", "--name-only")
		cmd.Dir = dir
		out, err = cmd.Output()
		if err != nil {
			return nil, fmt.Errorf("failed to get changed files: %w", err)
		}
	}

	var files []string
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		line = strings.TrimSpace(line)
		if line != "" {
			files = append(files, line)
		}
	}
	return files, nil
}
