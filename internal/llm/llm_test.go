package llm

import (
	"context"
	"testing"
)

func TestGenerateConfig(t *testing.T) {
	t.Setenv("REVV_MOCK_LLM", "true")
	ctx := context.Background()
	repoCtx := map[string]string{"README.md": "Dummy Content"}

	config, err := GenerateConfig(ctx, "mock-model", repoCtx)
	if err != nil {
		t.Fatalf("GenerateConfig returned error: %v", err)
	}

	if config == nil {
		t.Fatal("expected config to be non-nil")
	}

	if config.Dockerfile == "" {
		t.Error("expected non-empty Dockerfile")
	}
}
