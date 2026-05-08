package commands

import (
	"strings"
	"testing"

	"github.com/mak3r/ldc-demo/internal/state"
)

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
