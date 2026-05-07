package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/state"
)

var failCmd = &cobra.Command{
	Use:   "fail",
	Short: "Simulate failure scenarios in a deployed cluster",
}

var failNodeCmd = &cobra.Command{
	Use:   "node <cluster-name>",
	Short: "Stop one EC2 instance in the cluster (simulates node failure)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFailNode,
}

var failNetworkCmd = &cobra.Command{
	Use:   "network <cluster-name>",
	Short: "Block outbound traffic from cluster instances (simulates network failure)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFailNetwork,
}

var failPodCmd = &cobra.Command{
	Use:   "pod <cluster-name>",
	Short: "Deploy a crashlooping pod to the cluster",
	Args:  cobra.ExactArgs(1),
	RunE:  runFailPod,
}

var failSSHKey string

func init() {
	failPodCmd.Flags().StringVar(&failSSHKey, "ssh-key", "", "SSH private key for kubeconfig fetch (default: LDC_SSH_PRIVATE_KEY or ~/.ssh/id_rsa)")
	failCmd.AddCommand(failNodeCmd, failNetworkCmd, failPodCmd)
}

func runFailNode(cmd *cobra.Command, args []string) error {
	clusterName := args[0]
	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	if _, err := reg.FindByName(clusterName); err != nil {
		return err
	}

	instanceID, err := findClusterInstance(cmd.Context(), clusterName)
	if err != nil {
		return err
	}

	fmt.Printf("Stopping instance %s in cluster %q ...\n", instanceID, clusterName)
	//nolint:gosec // G204: args constructed from validated state
	out, err := exec.CommandContext(cmd.Context(), "aws", "ec2", "stop-instances",
		"--instance-ids", instanceID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("stop instance: %w\n%s", err, out)
	}
	fmt.Printf("Instance %s stopped. Use 'ldc-demo fix node %s' to restore.\n", instanceID, clusterName)
	return nil
}

func runFailNetwork(cmd *cobra.Command, args []string) error {
	clusterName := args[0]
	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	if _, err := reg.FindByName(clusterName); err != nil {
		return err
	}

	sgID, err := findClusterSecurityGroup(cmd.Context(), clusterName)
	if err != nil {
		return err
	}

	// Remove all egress rules — with no allow rules, all outbound traffic is blocked.
	fmt.Printf("Blocking outbound traffic for cluster %q (SG: %s) ...\n", clusterName, sgID)
	//nolint:gosec // G204: sgID comes from AWS API response, not user input
	out, err := exec.CommandContext(cmd.Context(), "aws", "ec2", "revoke-security-group-egress",
		"--group-id", sgID,
		"--protocol", "-1",
		"--cidr", "0.0.0.0/0").CombinedOutput()
	if err != nil {
		return fmt.Errorf("revoke egress rules: %w\n%s", err, out)
	}
	fmt.Printf("Outbound traffic blocked for cluster %q. Use 'ldc-demo fix network %s' to restore.\n", clusterName, clusterName)
	return nil
}

func runFailPod(cmd *cobra.Command, args []string) error {
	clusterName := args[0]

	kcPath, err := ensureKubeconfig(cmd.Context(), clusterName, failSSHKey)
	if err != nil {
		return err
	}

	crashManifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ldc-demo-crashloop
  namespace: default
  labels:
    app: ldc-demo-crashloop
    managed-by: ldc-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ldc-demo-crashloop
  template:
    metadata:
      labels:
        app: ldc-demo-crashloop
    spec:
      containers:
      - name: crasher
        image: busybox:latest
        command: ["sh", "-c", "exit 1"]`

	fmt.Printf("Deploying crashlooping pod to cluster %q ...\n", clusterName)
	//nolint:gosec // G204: kcPath is a program-controlled path, stdin provides manifest
	kubectlCmd := exec.CommandContext(cmd.Context(), "kubectl", "apply", "-f", "-", "--kubeconfig", kcPath)
	kubectlCmd.Stdin = strings.NewReader(crashManifest)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply: %w", err)
	}
	fmt.Printf("Crashlooping pod deployed to cluster %q. Use 'ldc-demo fix pod %s' to remove.\n", clusterName, clusterName)
	return nil
}

// findClusterInstance returns the instance ID of one running EC2 instance tagged with ldc-demo-cluster=<name>.
func findClusterInstance(ctx context.Context, clusterName string) (string, error) {
	//nolint:gosec // G204: clusterName comes from state registry, not raw user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "describe-instances",
		"--filters",
		"Name=tag:ldc-demo-cluster,Values="+clusterName,
		"Name=instance-state-name,Values=running",
		"--query", "Reservations[0].Instances[0].InstanceId",
		"--output", "text").Output()
	if err != nil {
		return "", fmt.Errorf("describe instances: %w", err)
	}
	id := strings.TrimSpace(string(out))
	if id == "" || id == "None" {
		return "", fmt.Errorf("no running instance found for cluster %q (expected tag ldc-demo-cluster=%s)", clusterName, clusterName)
	}
	return id, nil
}

// findClusterSecurityGroup returns the security group ID associated with cluster instances.
func findClusterSecurityGroup(ctx context.Context, clusterName string) (string, error) {
	//nolint:gosec // G204: clusterName comes from state registry, not raw user input
	out, err := exec.CommandContext(ctx, "aws", "ec2", "describe-instances",
		"--filters",
		"Name=tag:ldc-demo-cluster,Values="+clusterName,
		"Name=instance-state-name,Values=running",
		"--query", "Reservations[0].Instances[0].SecurityGroups[0].GroupId",
		"--output", "text").Output()
	if err != nil {
		return "", fmt.Errorf("describe instances for SG: %w", err)
	}
	sgID := strings.TrimSpace(string(out))
	if sgID == "" || sgID == "None" {
		return "", fmt.Errorf("no security group found for cluster %q", clusterName)
	}
	return sgID, nil
}
