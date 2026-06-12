package github

import (
	"testing"
)

func TestParseRemoteURL(t *testing.T) {
	tests := []struct {
		name      string
		url       string
		wantOwner string
		wantRepo  string
		wantErr   bool
	}{
		{
			name:      "HTTPS with .git",
			url:       "https://github.com/vssinghh/revv.git",
			wantOwner: "vssinghh",
			wantRepo:  "revv",
		},
		{
			name:      "HTTPS without .git",
			url:       "https://github.com/vssinghh/revv",
			wantOwner: "vssinghh",
			wantRepo:  "revv",
		},
		{
			name:      "SSH with .git",
			url:       "git@github.com:vssinghh/revv.git",
			wantOwner: "vssinghh",
			wantRepo:  "revv",
		},
		{
			name:      "SSH without .git",
			url:       "git@github.com:vssinghh/revv",
			wantOwner: "vssinghh",
			wantRepo:  "revv",
		},
		{
			name:      "HTTPS with trailing whitespace",
			url:       "https://github.com/owner/repo.git\n",
			wantOwner: "owner",
			wantRepo:  "repo",
		},
		{
			name:    "invalid URL",
			url:     "not-a-url",
			wantErr: true,
		},
		{
			name:    "empty string",
			url:     "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			owner, repo, err := ParseRemoteURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Errorf("expected error, got owner=%q repo=%q", owner, repo)
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if owner != tt.wantOwner {
				t.Errorf("owner = %q, want %q", owner, tt.wantOwner)
			}
			if repo != tt.wantRepo {
				t.Errorf("repo = %q, want %q", repo, tt.wantRepo)
			}
		})
	}
}
