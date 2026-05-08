package state

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAddAndFind(t *testing.T) {
	r := &Registry{Version: schemaVersion}

	c := ClusterState{Name: "demo", CloudProvider: "aws", NodeCount: 1, Size: "small", Module: "aws-k3s-single"}
	added, err := r.Add(c)
	if err != nil {
		t.Fatalf("Add: %v", err)
	}
	if added.UID == "" {
		t.Fatal("UID not assigned")
	}
	if added.TofuWorkspace != added.UID {
		t.Errorf("TofuWorkspace %q != UID %q", added.TofuWorkspace, added.UID)
	}

	found, err := r.Find("demo", "aws")
	if err != nil {
		t.Fatalf("Find: %v", err)
	}
	if found.UID != added.UID {
		t.Errorf("Find returned UID %q, want %q", found.UID, added.UID)
	}
}

func TestAddDuplicateReturnsError(t *testing.T) {
	r := &Registry{Version: schemaVersion}
	c := ClusterState{Name: "demo", CloudProvider: "aws"}
	if _, err := r.Add(c); err != nil {
		t.Fatalf("first Add: %v", err)
	}
	if _, err := r.Add(c); err == nil {
		t.Fatal("expected error for duplicate, got nil")
	}
}

func TestRemove(t *testing.T) {
	r := &Registry{Version: schemaVersion}
	added, _ := r.Add(ClusterState{Name: "x", CloudProvider: "aws"})
	if err := r.Remove(added.UID); err != nil {
		t.Fatalf("Remove: %v", err)
	}
	if len(r.Clusters) != 0 {
		t.Errorf("expected 0 clusters after Remove, got %d", len(r.Clusters))
	}
}

func TestSaveAndLoad(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")

	r := &Registry{Version: schemaVersion}
	if _, err := r.Add(ClusterState{Name: "demo", CloudProvider: "aws", NodeCount: 1}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	r2, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if len(r2.Clusters) != 1 {
		t.Errorf("expected 1 cluster after Load, got %d", len(r2.Clusters))
	}
	if r2.Clusters[0].Name != "demo" {
		t.Errorf("unexpected cluster name %q", r2.Clusters[0].Name)
	}
}

func TestLoadMissingFile(t *testing.T) {
	r, err := Load(filepath.Join(t.TempDir(), "nonexistent.json"))
	if err != nil {
		t.Fatalf("Load of missing file should return empty registry, got: %v", err)
	}
	if len(r.Clusters) != 0 {
		t.Errorf("expected empty registry, got %d clusters", len(r.Clusters))
	}
}

func TestSavePermissions(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "state.json")
	r := &Registry{Version: schemaVersion}
	if err := r.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0600 {
		t.Errorf("file permissions %o, want 0600", info.Mode().Perm())
	}
}

func TestFindByName(t *testing.T) {
	r := &Registry{Version: schemaVersion}
	if _, err := r.Add(ClusterState{Name: "demo", CloudProvider: "aws"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	c, err := r.FindByName("demo")
	if err != nil {
		t.Fatalf("FindByName: %v", err)
	}
	if c.Name != "demo" {
		t.Errorf("unexpected name %q", c.Name)
	}
}

func TestFindByNameAmbiguous(t *testing.T) {
	r := &Registry{Version: schemaVersion}
	if _, err := r.Add(ClusterState{Name: "demo", CloudProvider: "aws"}); err != nil {
		t.Fatalf("Add: %v", err)
	}
	if _, err := r.Add(ClusterState{Name: "demo", CloudProvider: "gcp"}); err != nil {
		t.Fatalf("Add: %v", err)
	}

	if _, err := r.FindByName("demo"); err == nil {
		t.Fatal("expected error for ambiguous name, got nil")
	}
}
