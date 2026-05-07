package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/kubeconfig"
	"github.com/mak3r/ldc-demo/internal/state"
	"github.com/mak3r/ldc-demo/internal/tofu"
)

var getKubeconfigSSHKey string

var getKubeconfigCmd = &cobra.Command{
	Use:   "get-kubeconfig <name>",
	Short: "Fetch kubeconfig for a deployed cluster",
	Long: `Fetches the kubeconfig from the cluster's server node via SSH and writes it to
~/.ldc-demo/kubeconfigs/<name>.yaml.

Use with kubectl:
  kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/<name>.yaml get nodes`,
	Args: cobra.ExactArgs(1),
	RunE: runGetKubeconfig,
}

func init() {
	getKubeconfigCmd.Flags().StringVar(&getKubeconfigSSHKey, "ssh-key", "", "path to SSH private key (default: ~/.ssh/id_rsa or LDC_SSH_PRIVATE_KEY)")
	rootCmd.AddCommand(getKubeconfigCmd)
}

func runGetKubeconfig(cmd *cobra.Command, args []string) error {
	name := args[0]

	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	cluster, err := reg.FindByName(name)
	if err != nil {
		return err
	}

	sshPrivKey := getKubeconfigSSHKey
	if sshPrivKey == "" {
		sshPrivKey = envOrDefault("LDC_SSH_PRIVATE_KEY", filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa"))
	}
	if _, err := os.Stat(sshPrivKey); err != nil {
		return fmt.Errorf("SSH private key not found at %s (set --ssh-key or LDC_SSH_PRIVATE_KEY)", sshPrivKey)
	}

	runner := &tofu.Runner{
		Binary:    tofuBinary,
		ModuleDir: moduleDir(cluster.Module),
		WorkDir:   workspaceDir(cluster.UID),
		Workspace: cluster.UID,
	}

	ctx := context.Background()
	serverIP, err := runner.Output(ctx, "server_public_ip")
	if err != nil {
		return fmt.Errorf("get server IP from tofu output: %w", err)
	}

	fmt.Printf("Fetching kubeconfig from %s (%s) ...\n", cluster.Name, serverIP)

	outPath, err := kubeconfig.Fetch(serverIP, "ec2-user", sshPrivKey, cluster.Name, kubeconfigDir())
	if err != nil {
		return fmt.Errorf("fetch kubeconfig: %w", err)
	}

	fmt.Printf("Kubeconfig written to: %s\n\n", outPath)
	fmt.Printf("Use with kubectl:\n  kubectl --kubeconfig %s get nodes\n", outPath)
	return nil
}
