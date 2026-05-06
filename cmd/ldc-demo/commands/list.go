package commands

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/state"
)

var listCmd = &cobra.Command{
	Use:   "list",
	Short: "List resources",
}

var listDeployedCmd = &cobra.Command{
	Use:   "deployed",
	Short: "List all deployed clusters",
	RunE:  runListDeployed,
}

func init() {
	listCmd.AddCommand(listDeployedCmd)
}

func runListDeployed(cmd *cobra.Command, args []string) error {
	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	clusters := reg.All()
	if len(clusters) == 0 {
		fmt.Println("No clusters deployed.")
		return nil
	}
	printClusters(os.Stdout, clusters)
	return nil
}
