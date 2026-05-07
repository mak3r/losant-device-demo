package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/state"
)

// dummyCmd returns a cobra.Command with a background context, suitable for
// passing to RunE functions that call cmd.Context().
func dummyCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

// withTestState creates a temp state registry and overrides stateDir.
func withTestState(t *testing.T, clusters []state.ClusterState) {
	t.Helper()
	dir := t.TempDir()
	reg, _ := state.Load(filepath.Join(dir, "nope.json"))
	for _, c := range clusters {
		if _, err := reg.Add(c); err != nil {
			t.Fatalf("setup: add cluster %q: %v", c.Name, err)
		}
	}
	if err := reg.Save(filepath.Join(dir, "state.json")); err != nil {
		t.Fatalf("setup: save state: %v", err)
	}
	old := stateDir
	stateDir = dir
	t.Cleanup(func() { stateDir = old })
}

// fakeAWSScript installs a fake `aws` binary at the front of PATH.
// The script outputs responseBySubcmd[subcommand] and exits 0.
// Subcommands not in the map cause exit 1.
func fakeAWSScript(t *testing.T, responseBySubcmd map[string]string) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "aws")

	cases := ""
	for subcmd, out := range responseBySubcmd {
		cases += fmt.Sprintf("        %s) echo %q ;;\n", subcmd, out)
	}
	src := fmt.Sprintf("#!/bin/sh\ncase \"$2\" in\n%s        *) echo \"unsupported: $2\" >&2; exit 1 ;;\nesac\n", cases)
	if err := os.WriteFile(script, []byte(src), 0700); err != nil { //nolint:gosec // G306: fake test script must be executable
		t.Fatalf("write fake aws: %v", err)
	}
	old := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+old)
	t.Cleanup(func() { os.Setenv("PATH", old) })
}
