package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/kubeconfig"
	"github.com/mak3r/ldc-demo/internal/state"
	"github.com/mak3r/ldc-demo/internal/tofu"
)

var fixCmd = &cobra.Command{
	Use:   "fix",
	Short: "Restore a cluster from a simulated failure",
}

var fixNodeCmd = &cobra.Command{
	Use:   "node <cluster-name>",
	Short: "Restart any stopped EC2 instances in the cluster",
	Args:  cobra.ExactArgs(1),
	RunE:  runFixNode,
}

var fixNetworkCmd = &cobra.Command{
	Use:   "network <cluster-name>",
	Short: "Restore outbound traffic for the cluster (re-add allow-all egress rule)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFixNetwork,
}

var fixPodCmd = &cobra.Command{
	Use:   "pod <cluster-name>",
	Short: "Remove the crashlooping pod from the cluster",
	Args:  cobra.ExactArgs(1),
	RunE:  runFixPod,
}

var fixSSHKey string

func init() {
	fixPodCmd.Flags().StringVar(&fixSSHKey, "ssh-key", "", "SSH private key for kubeconfig fetch (default: LDC_SSH_PRIVATE_KEY or ~/.ssh/id_rsa)")
	fixCmd.AddCommand(fixNodeCmd, fixNetworkCmd, fixPodCmd)
	rootCmd.AddCommand(fixCmd)
}

func runFixNode(cmd *cobra.Command, args []string) error {
	clusterName := args[0]
	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	if _, err := reg.FindByName(clusterName); err != nil {
		return err
	}

	instanceID, err := findStoppedClusterInstance(cmd.Context(), clusterName)
	if err != nil {
		return err
	}

	fmt.Printf("Starting instance %s in cluster %q ...\n", instanceID, clusterName)
	//nolint:gosec // G204: instanceID comes from AWS API, not raw user input
	out, err := exec.CommandContext(cmd.Context(), "aws", "ec2", "start-instances",
		"--instance-ids", instanceID).CombinedOutput()
	if err != nil {
		return fmt.Errorf("start instance: %w\n%s", err, out)
	}
	fmt.Printf("Instance %s started. The node will rejoin the cluster shortly.\n", instanceID)
	return nil
}

func runFixNetwork(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("Restoring outbound traffic for cluster %q (SG: %s) ...\n", clusterName, sgID)
	//nolint:gosec // G204: sgID comes from AWS API, not raw user input
	out, err := exec.CommandContext(cmd.Context(), "aws", "ec2", "authorize-security-group-egress",
		"--group-id", sgID,
		"--protocol", "-1",
		"--cidr", "0.0.0.0/0").CombinedOutput()
	if err != nil {
		return fmt.Errorf("authorize egress rules: %w\n%s", err, out)
	}
	fmt.Printf("Outbound traffic restored for cluster %q.\n", clusterName)
	return nil
}

func runFixPod(cmd *cobra.Command, args []string) error {
	clusterName := args[0]

	kcPath, err := ensureKubeconfig(cmd.Context(), clusterName, fixSSHKey)
	if err != nil {
		return err
	}

	fmt.Printf("Removing crashlooping pod from cluster %q ...\n", clusterName)
	//nolint:gosec // G204: kcPath is program-controlled
	kubectlCmd := exec.CommandContext(cmd.Context(), "kubectl", "delete", "deployment",
		"ldc-demo-crashloop", "--namespace", "default", "--kubeconfig", kcPath, "--ignore-not-found")
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("kubectl delete: %w", err)
	}
	fmt.Printf("Crashlooping pod removed from cluster %q.\n", clusterName)
	return nil
}

// findStoppedClusterInstance returns a stopped instance ID tagged with ldc-demo-cluster=<name>.
func findStoppedClusterInstance(ctx context.Context, clusterName string) (string, error) {
	//nolint:gosec // G204: clusterName comes from state registry
	out, err := exec.CommandContext(ctx, "aws", "ec2", "describe-instances",
		"--filters",
		"Name=tag:ldc-demo-cluster,Values="+clusterName,
		"Name=instance-state-name,Values=stopped",
		"--query", "Reservations[0].Instances[0].InstanceId",
		"--output", "text").Output()
	if err != nil {
		return "", fmt.Errorf("describe instances: %w", err)
	}
	id := strings.TrimSpace(string(out))
	if id == "" || id == "None" {
		return "", fmt.Errorf("no stopped instance found for cluster %q", clusterName)
	}
	return id, nil
}

// ensureKubeconfig fetches (or reuses a cached) kubeconfig for the named cluster.
func ensureKubeconfig(ctx context.Context, clusterName, sshKeyOverride string) (string, error) {
	reg, err := state.Load(statePath())
	if err != nil {
		return "", fmt.Errorf("load state: %w", err)
	}
	cluster, err := reg.FindByName(clusterName)
	if err != nil {
		return "", err
	}

	sshPrivKey := sshKeyOverride
	if sshPrivKey == "" {
		sshPrivKey = envOrDefault("LDC_SSH_PRIVATE_KEY", filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"))
	}
	if _, err := os.Stat(sshPrivKey); err != nil {
		return "", fmt.Errorf("SSH private key not found at %s", sshPrivKey)
	}

	runner := &tofu.Runner{
		Binary:    tofuBinary,
		ModuleDir: moduleDir(cluster.Module),
		WorkDir:   workspaceDir(cluster.UID),
		Workspace: cluster.UID,
	}
	serverIP, err := runner.Output(ctx, "server_public_ip")
	if err != nil {
		return "", fmt.Errorf("get server IP: %w", err)
	}

	return kubeconfig.Fetch(serverIP, "ec2-user", sshPrivKey, cluster.Name, kubeconfigDir())
}
