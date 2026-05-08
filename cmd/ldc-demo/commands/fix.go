package commands

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/spf13/cobra"

	cloudprovider "github.com/mak3r/ldc-demo/internal/provider"
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
	Short: "Restart any stopped compute instances in the cluster",
	Args:  cobra.ExactArgs(1),
	RunE:  runFixNode,
}

var fixNetworkCmd = &cobra.Command{
	Use:   "network <cluster-name>",
	Short: "Restore outbound traffic for the cluster",
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
	cluster, err := reg.FindByName(clusterName)
	if err != nil {
		return err
	}

	prov, err := cloudprovider.ForName(cluster.CloudProvider)
	if err != nil {
		return err
	}

	instanceRef, err := prov.FindStoppedInstance(cmd.Context(), cluster)
	if err != nil {
		return err
	}

	fmt.Printf("Starting instance %s in cluster %q ...\n", instanceRef, clusterName)
	if err := prov.StartInstance(cmd.Context(), instanceRef, cluster); err != nil {
		return fmt.Errorf("start instance: %w", err)
	}
	fmt.Printf("Instance %s started. The node will rejoin the cluster shortly.\n", instanceRef)
	return nil
}

func runFixNetwork(cmd *cobra.Command, args []string) error {
	clusterName := args[0]
	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	cluster, err := reg.FindByName(clusterName)
	if err != nil {
		return err
	}

	prov, err := cloudprovider.ForName(cluster.CloudProvider)
	if err != nil {
		return err
	}

	barrierRef, err := prov.FindNetworkBarrier(cmd.Context(), cluster)
	if err != nil {
		return err
	}

	fmt.Printf("Restoring outbound traffic for cluster %q (barrier: %s) ...\n", clusterName, barrierRef)
	if err := prov.RestoreOutbound(cmd.Context(), barrierRef, cluster); err != nil {
		return fmt.Errorf("restore outbound: %w", err)
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

	prov, err := cloudprovider.ForName(cluster.CloudProvider)
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

	return kubeconfig.Fetch(serverIP, prov.SSHUser(), sshPrivKey, cluster.Name, kubeconfigDir())
}
