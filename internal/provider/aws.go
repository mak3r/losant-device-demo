package provider

import (
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/mak3r/ldc-demo/internal/state"
)

type AWSProvider struct{}

func (a *AWSProvider) ModuleName(ha bool) string {
	if ha {
		return "aws-k3s-ha"
	}
	return "aws-k3s-single"
}

func (a *AWSProvider) SSHUser() string {
	return "ec2-user"
}

func (a *AWSProvider) VarFileVars(cluster state.ClusterState) map[string]string {
	return map[string]string{"aws_region": cluster.Region}
}

func (a *AWSProvider) FindInstance(ctx context.Context, cluster *state.ClusterState) (string, error) {
	//nolint:gosec // G204: cluster.Name comes from state registry, not raw user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "describe-instances",
		"--filters",
		"Name=tag:ldc-demo-cluster,Values="+cluster.Name,
		"Name=instance-state-name,Values=running",
		"--query", "Reservations[0].Instances[0].InstanceId",
		"--output", "text").Output()
	if err != nil {
		return "", fmt.Errorf("describe instances: %w", err)
	}
	id := strings.TrimSpace(string(out))
	if id == "" || id == "None" {
		return "", fmt.Errorf("no running instance found for cluster %q (expected tag ldc-demo-cluster=%s)", cluster.Name, cluster.Name)
	}
	return id, nil
}

func (a *AWSProvider) FindStoppedInstance(ctx context.Context, cluster *state.ClusterState) (string, error) {
	//nolint:gosec // G204: cluster.Name comes from state registry, not raw user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "describe-instances",
		"--filters",
		"Name=tag:ldc-demo-cluster,Values="+cluster.Name,
		"Name=instance-state-name,Values=stopped",
		"--query", "Reservations[0].Instances[0].InstanceId",
		"--output", "text").Output()
	if err != nil {
		return "", fmt.Errorf("describe instances: %w", err)
	}
	id := strings.TrimSpace(string(out))
	if id == "" || id == "None" {
		return "", fmt.Errorf("no stopped instance found for cluster %q", cluster.Name)
	}
	return id, nil
}

func (a *AWSProvider) StopInstance(ctx context.Context, instanceRef string, cluster *state.ClusterState) error {
	//nolint:gosec // G204: instanceRef comes from AWS API response, not user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "stop-instances",
		"--instance-ids", instanceRef).CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop instance: %w\n%s", err, out)
	}
	return nil
}

func (a *AWSProvider) StartInstance(ctx context.Context, instanceRef string, cluster *state.ClusterState) error {
	//nolint:gosec // G204: instanceRef comes from AWS API response, not user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "start-instances",
		"--instance-ids", instanceRef).CombinedOutput()
	if err != nil {
		return fmt.Errorf("start instance: %w\n%s", err, out)
	}
	return nil
}

func (a *AWSProvider) FindNetworkBarrier(ctx context.Context, cluster *state.ClusterState) (string, error) {
	//nolint:gosec // G204: cluster.Name comes from state registry, not raw user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "describe-instances",
		"--filters",
		"Name=tag:ldc-demo-cluster,Values="+cluster.Name,
		"Name=instance-state-name,Values=running",
		"--query", "Reservations[0].Instances[0].SecurityGroups[0].GroupId",
		"--output", "text").Output()
	if err != nil {
		return "", fmt.Errorf("describe instances for SG: %w", err)
	}
	sgID := strings.TrimSpace(string(out))
	if sgID == "" || sgID == "None" {
		return "", fmt.Errorf("no security group found for cluster %q", cluster.Name)
	}
	return sgID, nil
}

func (a *AWSProvider) BlockOutbound(ctx context.Context, barrierRef string, cluster *state.ClusterState) error {
	//nolint:gosec // G204: barrierRef comes from AWS API response, not user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "revoke-security-group-egress",
		"--group-id", barrierRef,
		"--protocol", "-1",
		"--cidr", "0.0.0.0/0").CombinedOutput()
	if err != nil {
		return fmt.Errorf("revoke egress rules: %w\n%s", err, out)
	}
	return nil
}

func (a *AWSProvider) RestoreOutbound(ctx context.Context, barrierRef string, cluster *state.ClusterState) error {
	//nolint:gosec // G204: barrierRef comes from AWS API response, not user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "authorize-security-group-egress",
		"--group-id", barrierRef,
		"--protocol", "-1",
		"--cidr", "0.0.0.0/0").CombinedOutput()
	if err != nil {
		return fmt.Errorf("authorize egress rules: %w\n%s", err, out)
	}
	return nil
}
