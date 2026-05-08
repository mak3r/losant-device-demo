package commands

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/state"
	"github.com/mak3r/ldc-demo/internal/tofu"
)

var scaleCmd = &cobra.Command{
	Use:   "scale <name>",
	Short: "Add a worker node to an existing cluster",
	Long: `Add a worker node to an existing cluster identified by <name>.

The cluster must have been created with ldc-demo create. After scaling,
outputs: uid, name, cloud-provider, node count.`,
	Args: cobra.ExactArgs(1),
	RunE: runScale,
}

func init() {
	rootCmd.AddCommand(scaleCmd)
}

func runScale(cmd *cobra.Command, args []string) error {
	name := args[0]

	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	cluster, err := reg.FindByName(name)
	if err != nil {
		return err
	}

	if cluster.K3sToken == "" {
		return fmt.Errorf("cluster %q has no k3s join token in state — it may have been created before scale support was added; recreate the cluster to enable scaling", name)
	}

	sshPublicKey := envOrDefault("LDC_SSH_PUBLIC_KEY", os.Getenv("HOME")+"/.ssh/id_rsa.pub")

	varFile, cleanup, err := writeTempVarFile(*cluster, sshPublicKey)
	if err != nil {
		return fmt.Errorf("prepare var file: %w", err)
	}
	defer cleanup()

	cluster.WorkerCount++

	runner := &tofu.Runner{
		Binary:    tofuBinary,
		ModuleDir: moduleDir(cluster.Module),
		WorkDir:   workspaceDir(cluster.UID),
		Workspace: cluster.UID,
	}

	ctx := context.Background()
	fmt.Printf("Adding worker node to cluster %q (total workers: %d) ...\n", name, cluster.WorkerCount)

	extraVars := map[string]string{
		"losant_api_token":      os.Getenv("LDC_LOSANT_API_TOKEN"),
		"losant_application_id": os.Getenv("LDC_LOSANT_APPLICATION_ID"),
		"worker_count":          fmt.Sprintf("%d", cluster.WorkerCount),
		"k3s_token":             cluster.K3sToken,
	}
	if err := runner.Apply(ctx, varFile, extraVars); err != nil {
		cluster.WorkerCount--
		return fmt.Errorf("tofu apply: %w", err)
	}

	cluster.NodeCount = baseNodeCount(cluster.Module) + cluster.WorkerCount

	if err := reg.Save(statePath()); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	fmt.Println()
	printClusters(os.Stdout, []state.ClusterState{*cluster})
	return nil
}

func baseNodeCount(module string) int {
	if strings.HasSuffix(module, "-ha") {
		return 3
	}
	return 1
}
