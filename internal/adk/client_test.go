package adk

import (
	"strings"
	"testing"
)

func TestConstructPrompt(t *testing.T) {
	repoCtx := map[string]string{
		"README.md": "Test project content",
	}
	prompt := ConstructPrompt(repoCtx, nil)
	if !strings.Contains(prompt, "Test project content") {
		t.Errorf("expected prompt to contain README content, got: %s", prompt)
	}
}

func TestNewClient(t *testing.T) {
	_, err := NewClient("", "model")
	if err == nil {
		t.Error("expected error for empty API key")
	}

	c, err := NewClient("key", "model")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c.apiKey != "key" || c.model != "model" {
		t.Errorf("unexpected client state: %+v", c)
	}
}
