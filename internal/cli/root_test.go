package cli

import (
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "revv" {
		t.Errorf("expected command use 'revv', got %q", cmd.Use)
	}

	initCmd, _, err := cmd.Find([]string{"init"})
	if err != nil || initCmd == nil {
		t.Errorf("init command not found in root command: %v", err)
	}
}
