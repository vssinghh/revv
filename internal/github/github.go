package github

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
)

const (
	apiBaseURL     = "https://api.github.com"
	commentMarker  = "<!-- revv-comment-marker -->"
)

// PR holds information about a GitHub pull request.
type PR struct {
	Number  int    `json:"number"`
	Title   string `json:"title"`
	Branch  string `json:"head_ref"`   // head branch name
	BaseSHA string `json:"base_sha"`
	HeadSHA string `json:"head_sha"`
}

// Client interacts with the GitHub API for PR operations.
type Client struct {
	token   string
	owner   string
	repo    string
	httpCli *http.Client
}

// New creates a GitHub client with explicit owner and repo.
func New(token, owner, repo string) *Client {
	return &Client{
		token:   token,
		owner:   owner,
		repo:    repo,
		httpCli: &http.Client{},
	}
}

// NewFromRemote creates a GitHub client by parsing the git remote of the repo at dir.
func NewFromRemote(token, dir string) (*Client, error) {
	remoteURL, err := GetRemoteURL(dir)
	if err != nil {
		return nil, err
	}
	owner, repo, err := ParseRemoteURL(remoteURL)
	if err != nil {
		return nil, err
	}
	return New(token, owner, repo), nil
}

// GetPR fetches pull request details by number.
func (c *Client) GetPR(ctx context.Context, number int) (*PR, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/pulls/%d", apiBaseURL, c.owner, c.repo, number)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("get PR #%d: %w", number, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("get PR #%d: HTTP %d: %s", number, resp.StatusCode, string(body))
	}

	// GitHub API returns nested objects — extract what we need
	var raw struct {
		Number int    `json:"number"`
		Title  string `json:"title"`
		Head   struct {
			Ref string `json:"ref"`
			SHA string `json:"sha"`
		} `json:"head"`
		Base struct {
			SHA string `json:"sha"`
		} `json:"base"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		return nil, fmt.Errorf("parse PR response: %w", err)
	}

	return &PR{
		Number:  raw.Number,
		Title:   raw.Title,
		Branch:  raw.Head.Ref,
		HeadSHA: raw.Head.SHA,
		BaseSHA: raw.Base.SHA,
	}, nil
}

// PostComment posts a new comment on a PR.
func (c *Client) PostComment(ctx context.Context, prNumber int, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments", apiBaseURL, c.owner, c.repo, prNumber)

	payload, _ := json.Marshal(map[string]string{"body": body})

	resp, err := c.doRequest(ctx, "POST", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("post comment on PR #%d: %w", prNumber, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("post comment on PR #%d: HTTP %d: %s", prNumber, resp.StatusCode, string(respBody))
	}

	return nil
}

// UpsertComment posts or updates the revv comment on a PR.
// If a previous revv comment exists (identified by the marker), it updates it.
// Otherwise, it creates a new comment.
func (c *Client) UpsertComment(ctx context.Context, prNumber int, body string) error {
	// Find existing revv comment
	commentID, err := c.findRevvComment(ctx, prNumber)
	if err != nil {
		// If we can't list comments, fall back to posting a new one
		return c.PostComment(ctx, prNumber, body)
	}

	if commentID != 0 {
		// Update existing comment
		return c.updateComment(ctx, commentID, body)
	}

	// No existing comment — post new
	return c.PostComment(ctx, prNumber, body)
}

// findRevvComment searches for an existing revv comment on a PR.
// Returns the comment ID if found, 0 if not.
func (c *Client) findRevvComment(ctx context.Context, prNumber int) (int64, error) {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/%d/comments?per_page=100", apiBaseURL, c.owner, c.repo, prNumber)

	resp, err := c.doRequest(ctx, "GET", url, nil)
	if err != nil {
		return 0, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return 0, fmt.Errorf("list comments: HTTP %d", resp.StatusCode)
	}

	var comments []struct {
		ID   int64  `json:"id"`
		Body string `json:"body"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&comments); err != nil {
		return 0, err
	}

	for _, comment := range comments {
		if strings.Contains(comment.Body, commentMarker) {
			return comment.ID, nil
		}
	}

	return 0, nil
}

// updateComment updates an existing comment by ID.
func (c *Client) updateComment(ctx context.Context, commentID int64, body string) error {
	url := fmt.Sprintf("%s/repos/%s/%s/issues/comments/%d", apiBaseURL, c.owner, c.repo, commentID)

	payload, _ := json.Marshal(map[string]string{"body": body})

	resp, err := c.doRequest(ctx, "PATCH", url, bytes.NewReader(payload))
	if err != nil {
		return fmt.Errorf("update comment %d: %w", commentID, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("update comment %d: HTTP %d: %s", commentID, resp.StatusCode, string(respBody))
	}

	return nil
}

// doRequest performs an authenticated HTTP request.
func (c *Client) doRequest(ctx context.Context, method, url string, body io.Reader) (*http.Response, error) {
	req, err := http.NewRequestWithContext(ctx, method, url, body)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.httpCli.Do(req)
}
