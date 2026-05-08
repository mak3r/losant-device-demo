package provider

import (
	"context"
	"testing"

	"github.com/mak3r/ldc-demo/internal/state"
)

func TestGCPModuleName(t *testing.T) {
	g := &GCPProvider{}
	if got := g.ModuleName(false); got != "gcp-k3s-single" {
		t.Errorf("ModuleName(false) = %q, want %q", got, "gcp-k3s-single")
	}
	if got := g.ModuleName(true); got != "gcp-k3s-ha" {
		t.Errorf("ModuleName(true) = %q, want %q", got, "gcp-k3s-ha")
	}
}

func TestGCPSSHUser(t *testing.T) {
	g := &GCPProvider{}
	if got := g.SSHUser(); got != "ubuntu" {
		t.Errorf("SSHUser() = %q, want %q", got, "ubuntu")
	}
}

func TestGCPVarFileVars(t *testing.T) {
	g := &GCPProvider{}
	cluster := state.ClusterState{
		ProviderConfig: map[string]string{
			"gcp_project": "my-proj",
			"gcp_zone":    "us-central1-a",
		},
	}
	vars := g.VarFileVars(cluster)
	if vars["gcp_project"] != "my-proj" {
		t.Errorf("gcp_project = %q, want %q", vars["gcp_project"], "my-proj")
	}
	if vars["gcp_zone"] != "us-central1-a" {
		t.Errorf("gcp_zone = %q, want %q", vars["gcp_zone"], "us-central1-a")
	}
}

func TestGCPFindInstance(t *testing.T) {
	fakeGcloudScript(t, map[string]string{
		"list": "demo-node-0",
	})
	g := &GCPProvider{}
	cluster := &state.ClusterState{
		Name:           "demo",
		ProviderConfig: map[string]string{"gcp_zone": "us-central1-a"},
	}
	name, err := g.FindInstance(context.Background(), cluster)
	if err != nil {
		t.Fatalf("FindInstance: %v", err)
	}
	if name != "demo-node-0" {
		t.Errorf("instance name = %q, want %q", name, "demo-node-0")
	}
}

func TestGCPFindInstanceNoneFound(t *testing.T) {
	fakeGcloudScript(t, map[string]string{
		"list": "",
	})
	g := &GCPProvider{}
	cluster := &state.ClusterState{
		Name:           "demo",
		ProviderConfig: map[string]string{"gcp_zone": "us-central1-a"},
	}
	_, err := g.FindInstance(context.Background(), cluster)
	if err == nil {
		t.Fatal("expected error when no instance found, got nil")
	}
}

func TestGCPStopInstance(t *testing.T) {
	fakeGcloudScript(t, map[string]string{
		"stop": "",
	})
	g := &GCPProvider{}
	cluster := &state.ClusterState{
		Name:           "demo",
		ProviderConfig: map[string]string{"gcp_zone": "us-central1-a"},
	}
	if err := g.StopInstance(context.Background(), "demo-node-0", cluster); err != nil {
		t.Fatalf("StopInstance: %v", err)
	}
}

func TestGCPFindNetworkBarrier(t *testing.T) {
	g := &GCPProvider{}
	cluster := &state.ClusterState{Name: "demo"}
	barrier, err := g.FindNetworkBarrier(context.Background(), cluster)
	if err != nil {
		t.Fatalf("FindNetworkBarrier: %v", err)
	}
	want := "ldc-demo-demo-allow-egress"
	if barrier != want {
		t.Errorf("barrier = %q, want %q", barrier, want)
	}
}

func TestGCPBlockOutbound(t *testing.T) {
	fakeGcloudScript(t, map[string]string{
		"update": "",
	})
	g := &GCPProvider{}
	cluster := &state.ClusterState{Name: "demo"}
	if err := g.BlockOutbound(context.Background(), "ldc-demo-demo-allow-egress", cluster); err != nil {
		t.Fatalf("BlockOutbound: %v", err)
	}
}

func TestGCPRestoreOutbound(t *testing.T) {
	fakeGcloudScript(t, map[string]string{
		"update": "",
	})
	g := &GCPProvider{}
	cluster := &state.ClusterState{Name: "demo"}
	if err := g.RestoreOutbound(context.Background(), "ldc-demo-demo-allow-egress", cluster); err != nil {
		t.Fatalf("RestoreOutbound: %v", err)
	}
}
