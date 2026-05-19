package commands

import (
	"bytes"
	"strings"
	"testing"
)

func TestSilenceUsageOnError(t *testing.T) {
	// Bypass the tofu-binary lookup so PersistentPreRunE passes.
	oldBinary := tofuBinary
	tofuBinary = "fake-tofu-binary"
	t.Cleanup(func() { tofuBinary = oldBinary })

	// Ensure AWS credentials are unset so runCreate returns an error early.
	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	var buf bytes.Buffer
	rootCmd.SetErr(&buf)
	rootCmd.SetOut(&buf)
	t.Cleanup(func() {
		rootCmd.SetErr(nil)
		rootCmd.SetOut(nil)
	})

	rootCmd.SetArgs([]string{"create", "mycluster", "aws"})
	t.Cleanup(func() { rootCmd.SetArgs(nil) })

	_ = rootCmd.Execute()

	if strings.Contains(buf.String(), "Usage:") {
		t.Errorf("SilenceUsage should suppress usage text on error, but output contains 'Usage:':\n%s", buf.String())
	}
}

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
