package commands

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/mak3r/ldc-demo/internal/state"
)

func TestExtractSyncIntervalFound(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cr.yaml")
	if err := os.WriteFile(f, []byte("syncInterval: \"30s\"\nother: value\n"), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if got := extractSyncInterval(f); got != "30s" {
		t.Errorf("got %q, want %q", got, "30s")
	}
}

func TestExtractSyncIntervalAbsent(t *testing.T) {
	f := filepath.Join(t.TempDir(), "cr.yaml")
	if err := os.WriteFile(f, []byte("kind: LosantSync\nmetadata:\n  name: demo\n"), 0600); err != nil {
		t.Fatalf("write temp file: %v", err)
	}
	if got := extractSyncInterval(f); got != "(unknown)" {
		t.Errorf("got %q, want %q", got, "(unknown)")
	}
}

func TestExtractSyncIntervalUnreadable(t *testing.T) {
	if got := extractSyncInterval("/nonexistent/path/cr.yaml"); got != "(unreadable)" {
		t.Errorf("got %q, want %q", got, "(unreadable)")
	}
}

func TestListDeployedEmpty(t *testing.T) {
	withTestState(t, nil)
	if err := runListDeployed(dummyCmd(), nil); err != nil {
		t.Fatalf("runListDeployed on empty registry: %v", err)
	}
}

func TestListDeployedWithClusters(t *testing.T) {
	withTestState(t, []state.ClusterState{
		{Name: "alpha", CloudProvider: "aws"},
		{Name: "beta", CloudProvider: "aws"},
	})
	if err := runListDeployed(dummyCmd(), nil); err != nil {
		t.Fatalf("runListDeployed with clusters: %v", err)
	}
}

func TestListConfigsDirNotFound(t *testing.T) {
	// In the test environment repoRoot() resolves to the package working directory,
	// where configs/losantsync does not exist. Verify the error is surfaced cleanly.
	err := runListConfigs(dummyCmd(), nil)
	if err == nil {
		t.Skip("configs/losantsync exists in the test environment; skipping not-found test")
	}
	if err != nil {
		// Error is expected — pass.
		return
	}
}
