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

var applyCmd = &cobra.Command{
	Use:   "apply",
	Short: "Apply resources to a cluster",
}

var applyConfigCmd = &cobra.Command{
	Use:   "config <config-name> <cluster-name>",
	Short: "Apply a pre-defined LosantSync CR config to a deployed cluster",
	Args:  cobra.ExactArgs(2),
	RunE:  runApplyConfig,
}

var applySSHKey string

func init() {
	applyConfigCmd.Flags().StringVar(&applySSHKey, "ssh-key", "", "path to SSH private key (default: ~/.ssh/id_rsa or LDC_SSH_PRIVATE_KEY)")
	applyCmd.AddCommand(applyConfigCmd)
	rootCmd.AddCommand(applyCmd)
}

func runApplyConfig(cmd *cobra.Command, args []string) error {
	configName := args[0]
	clusterName := args[1]

	configPath := filepath.Join(repoRoot(), "configs", "losantsync", configName+".yaml")
	if _, err := os.Stat(configPath); err != nil {
		return fmt.Errorf("config %q not found at %s", configName, configPath)
	}

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

	sshPrivKey := applySSHKey
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

	kcPath, err := kubeconfig.Fetch(serverIP, prov.SSHUser(), sshPrivKey, cluster.Name, kubeconfigDir())
	if err != nil {
		return fmt.Errorf("fetch kubeconfig: %w", err)
	}

	fmt.Printf("Applying config %q to cluster %q ...\n", configName, clusterName)

	//nolint:gosec // G204: configPath and kcPath are constructed from validated, program-controlled inputs
	kubectlCmd := exec.CommandContext(ctx, "kubectl", "apply", "-f", configPath, "--kubeconfig", kcPath)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply: %w", err)
	}

	fmt.Printf("Config %q applied successfully to cluster %q.\n", configName, clusterName)
	return nil
}
