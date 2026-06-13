package cli

import (
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	if cmd.Use != "revv" {
		t.Errorf("expected command use 'revv', got %q", cmd.Use)
	}

	execCmd, _, err := cmd.Find([]string{"exec"})
	if err != nil || execCmd == nil {
		t.Errorf("exec command not found in root command: %v", err)
	}

	// init should NOT exist anymore
	_, _, err = cmd.Find([]string{"init"})
	if err == nil {
		t.Errorf("init command should not exist anymore")
	}
}
