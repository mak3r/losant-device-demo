package provider

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mak3r/ldc-demo/internal/state"
)

type GCPProvider struct{}

func (g *GCPProvider) ModuleName(ha bool) string {
	if ha {
		return "gcp-k3s-ha"
	}
	return "gcp-k3s-single"
}

func (g *GCPProvider) SSHUser() string {
	return "ubuntu"
}

func (g *GCPProvider) VarFileVars(cluster state.ClusterState) map[string]string {
	return map[string]string{
		"gcp_project": cluster.ProviderConfig["gcp_project"],
		"gcp_zone":    cluster.ProviderConfig["gcp_zone"],
	}
}

func (g *GCPProvider) FindInstance(ctx context.Context, cluster *state.ClusterState) (string, error) {
	zone := cluster.ProviderConfig["gcp_zone"]
	//nolint:gosec // G204: cluster.Name comes from state registry, not raw user input
	out, err := exec.CommandContext(ctx, "gcloud", "compute", "instances", "list",
		"--filter", "labels.ldc-demo-cluster="+cluster.Name+" AND status=RUNNING",
		"--zones", zone,
		"--format", "value(name)").Output()
	if err != nil {
		return "", fmt.Errorf("gcloud list instances: %w", err)
	}
	name := strings.TrimSpace(string(out))
	if name == "" {
		return "", fmt.Errorf("no running instance found for cluster %q in zone %q", cluster.Name, zone)
	}
	return name, nil
}

func (g *GCPProvider) StopInstance(ctx context.Context, instanceRef string, cluster *state.ClusterState) error {
	zone := cluster.ProviderConfig["gcp_zone"]
	//nolint:gosec // G204: instanceRef from gcloud API, zone from state
	out, err := exec.CommandContext(ctx, "gcloud", "compute", "instances", "stop",
		instanceRef, "--zone", zone).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gcloud stop instance: %w\n%s", err, out)
	}
	return nil
}

func (g *GCPProvider) StartInstance(ctx context.Context, instanceRef string, cluster *state.ClusterState) error {
	zone := cluster.ProviderConfig["gcp_zone"]
	//nolint:gosec // G204: instanceRef from gcloud API, zone from state
	out, err := exec.CommandContext(ctx, "gcloud", "compute", "instances", "start",
		instanceRef, "--zone", zone).CombinedOutput()
	if err != nil {
		return fmt.Errorf("gcloud start instance: %w\n%s", err, out)
	}
	return nil
}

func (g *GCPProvider) FindNetworkBarrier(_ context.Context, cluster *state.ClusterState) (string, error) {
	return "ldc-demo-" + cluster.Name + "-allow-egress", nil
}

func (g *GCPProvider) BlockOutbound(ctx context.Context, barrierRef string, cluster *state.ClusterState) error {
	//nolint:gosec // G204: barrierRef is deterministic name derived from cluster state
	out, err := exec.CommandContext(ctx, "gcloud", "compute", "firewall-rules", "update",
		barrierRef, "--disabled").CombinedOutput()
	if err != nil {
		return fmt.Errorf("gcloud disable firewall rule: %w\n%s", err, out)
	}
	return nil
}

func (g *GCPProvider) RestoreOutbound(ctx context.Context, barrierRef string, cluster *state.ClusterState) error {
	//nolint:gosec // G204: barrierRef is deterministic name derived from cluster state
	out, err := exec.CommandContext(ctx, "gcloud", "compute", "firewall-rules", "update",
		barrierRef, "--no-disabled").CombinedOutput()
	if err != nil {
		return fmt.Errorf("gcloud enable firewall rule: %w\n%s", err, out)
	}
	return nil
}
