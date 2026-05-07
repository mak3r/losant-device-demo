package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

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

var listConfigsCmd = &cobra.Command{
	Use:   "configs",
	Short: "List available LosantSync CR configs",
	RunE:  runListConfigs,
}

func init() {
	listCmd.AddCommand(listDeployedCmd)
	listCmd.AddCommand(listConfigsCmd)
}

func runListConfigs(cmd *cobra.Command, args []string) error {
	configsDir := filepath.Join(repoRoot(), "configs", "losantsync")
	entries, err := os.ReadDir(configsDir)
	if err != nil {
		return fmt.Errorf("read configs directory %s: %w", configsDir, err)
	}

	fmt.Printf("%-25s %s\n", "NAME", "SYNC_INTERVAL")
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".yaml") {
			continue
		}
		name := strings.TrimSuffix(entry.Name(), ".yaml")
		interval := extractSyncInterval(filepath.Join(configsDir, entry.Name()))
		fmt.Printf("%-25s %s\n", name, interval)
	}
	return nil
}

func extractSyncInterval(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return "(unreadable)"
	}
	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "syncInterval:") {
			val := strings.TrimSpace(strings.TrimPrefix(trimmed, "syncInterval:"))
			return strings.Trim(val, `"'`)
		}
	}
	return "(unknown)"
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
