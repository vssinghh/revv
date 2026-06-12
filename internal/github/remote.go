package github

import (
	"fmt"
	"os/exec"
	"regexp"
	"strings"
)

// httpsPattern matches https://github.com/owner/repo.git or https://github.com/owner/repo
var httpsPattern = regexp.MustCompile(`https?://[^/]+/([^/]+)/([^/.\s]+?)(?:\.git)?$`)

// sshPattern matches git@github.com:owner/repo.git or git@github.com:owner/repo
var sshPattern = regexp.MustCompile(`git@[^:]+:([^/]+)/([^/.\s]+?)(?:\.git)?$`)

// ParseRemoteURL extracts owner and repo from a git remote URL.
// Supports both HTTPS and SSH formats.
func ParseRemoteURL(url string) (owner, repo string, err error) {
	url = strings.TrimSpace(url)

	if matches := httpsPattern.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	if matches := sshPattern.FindStringSubmatch(url); len(matches) == 3 {
		return matches[1], matches[2], nil
	}

	return "", "", fmt.Errorf("cannot parse GitHub owner/repo from remote URL: %s", url)
}

// GetRemoteURL returns the URL of the 'origin' remote for the repo at dir.
func GetRemoteURL(dir string) (string, error) {
	cmd := exec.Command("git", "remote", "get-url", "origin")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get git remote URL: %w", err)
	}
	return strings.TrimSpace(string(output)), nil
}
