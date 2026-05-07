package commands

import (
	"testing"
)

func TestRootCmdRegistration(t *testing.T) {
	registered := make(map[string]bool)
	for _, cmd := range rootCmd.Commands() {
		registered[cmd.Name()] = true
	}

	// All commands that self-register via init() in their own files.
	// If any is silently dropped by a refactor of root.go, this test fails.
	want := []string{"create", "list", "remove", "get-kubeconfig", "apply", "fail", "fix", "scale"}
	for _, name := range want {
		if !registered[name] {
			t.Errorf("command %q not registered on rootCmd; registered: %v", name, registered)
		}
	}
}
