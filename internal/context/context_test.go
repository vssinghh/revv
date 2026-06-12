package context

import (
	"os"
	"path/filepath"
	"testing"
)

func TestReadRepositoryContext(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "revv_test")
	if err != nil {
		t.Fatalf("failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tempDir)

	testFile := "README.md"
	testContent := "Test Content"
	err = os.WriteFile(filepath.Join(tempDir, testFile), []byte(testContent), 0644)
	if err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	ctx, err := ReadRepositoryContext(tempDir)
	if err != nil {
		t.Fatalf("ReadRepositoryContext failed: %v", err)
	}

	if content, exists := ctx[testFile]; !exists || content != testContent {
		t.Errorf("expected README.md to contain %q, got %q (exists: %v)", testContent, content, exists)
	}
}
