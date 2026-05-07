package commands

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/state"
)

// dummyCmd returns a minimal cobra.Command suitable for passing to RunE functions.
func dummyCmd() *cobra.Command { return &cobra.Command{} }

// withTestState writes a state registry to a temp dir and overrides stateDir.
func withTestState(t *testing.T, clusters []state.ClusterState) {
	t.Helper()
	dir := t.TempDir()
	reg, _ := state.Load(filepath.Join(dir, "nope.json")) // missing file → empty registry
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

// withRemoveFlags temporarily sets the package-level flag variables.
func withRemoveFlags(t *testing.T, confirm bool, provider string) {
	t.Helper()
	oldC, oldP := removeConfirm, removeProvider
	removeConfirm = confirm
	removeProvider = provider
	t.Cleanup(func() {
		removeConfirm = oldC
		removeProvider = oldP
	})
}

func TestRemoveNameNotFound(t *testing.T) {
	withTestState(t, nil) // empty registry
	withRemoveFlags(t, true, "")

	err := runRemoveName(dummyCmd(), []string{"nonexistent"})
	if err == nil {
		t.Fatal("expected error for cluster not in registry, got nil")
	}
}

func TestRemoveNameAmbiguous(t *testing.T) {
	withTestState(t, []state.ClusterState{
		{Name: "demo", CloudProvider: "aws"},
		{Name: "demo", CloudProvider: "gcp"},
	})
	withRemoveFlags(t, true, "") // no provider → FindByName → ambiguous error

	err := runRemoveName(dummyCmd(), []string{"demo"})
	if err == nil {
		t.Fatal("expected ambiguous-name error, got nil")
	}
}

func TestRemoveNameDispatchesWithProvider(t *testing.T) {
	withTestState(t, []state.ClusterState{
		{Name: "demo", CloudProvider: "aws"},
		{Name: "demo", CloudProvider: "gcp"},
	})
	withRemoveFlags(t, true, "gcp") // provider set → Find(name, provider) path

	// Find succeeds; writeTempVarFile fails because no template exists in the
	// test environment. The error should come from "prepare var file", NOT from
	// a not-found or ambiguous dispatch.
	err := runRemoveName(dummyCmd(), []string{"demo"})
	if err == nil {
		t.Fatal("expected error after dispatch (from writeTempVarFile), got nil")
	}
	if !strings.Contains(err.Error(), "prepare var file") {
		t.Errorf("expected 'prepare var file' error after successful dispatch, got: %v", err)
	}
}
