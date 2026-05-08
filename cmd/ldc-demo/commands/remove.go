package commands

import (
	"bufio"
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/state"
	"github.com/mak3r/ldc-demo/internal/tofu"
)

var removeConfirm bool
var removeProvider string

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove clusters",
}

var removeAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Destroy all managed clusters and their cloud resources",
	RunE:  runRemoveAll,
}

var removeNameCmd = &cobra.Command{
	Use:   "name <cluster-name>",
	Short: "Destroy a single named cluster and remove it from state",
	Args:  cobra.ExactArgs(1),
	RunE:  runRemoveName,
}

func init() {
	removeAllCmd.Flags().BoolVar(&removeConfirm, "confirm", false, "confirm destructive removal without interactive prompt")
	removeNameCmd.Flags().BoolVar(&removeConfirm, "confirm", false, "confirm destructive removal without interactive prompt")
	removeNameCmd.Flags().StringVar(&removeProvider, "provider", "", "cloud provider to disambiguate clusters with the same name")
	removeCmd.AddCommand(removeAllCmd)
	removeCmd.AddCommand(removeNameCmd)
	rootCmd.AddCommand(removeCmd)
}

func runRemoveAll(cmd *cobra.Command, args []string) error {
	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}
	clusters := reg.All()
	if len(clusters) == 0 {
		fmt.Println("No clusters deployed — nothing to remove.")
		return nil
	}

	fmt.Printf("WARNING: This will permanently destroy %d cluster(s):\n\n", len(clusters))
	printClusters(os.Stdout, clusters)
	fmt.Println()

	if !removeConfirm {
		if !promptYesNo("Type 'yes' to confirm removal of all clusters") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	ctx := context.Background()
	var lastErr error

	for _, cluster := range clusters {
		fmt.Printf("\nDestroying cluster %q (uid: %s) ...\n", cluster.Name, cluster.UID)

		varFile, cleanup, err := writeTempVarFile(cluster, envOrDefault("LDC_SSH_PUBLIC_KEY", os.Getenv("HOME")+"/.ssh/id_rsa.pub"))
		if err != nil {
			fmt.Fprintf(os.Stderr, "warning: cannot prepare var file for %s: %v — skipping\n", cluster.Name, err)
			lastErr = err
			continue
		}

		runner := &tofu.Runner{
			Binary:    tofuBinary,
			ModuleDir: moduleDir(cluster.Module),
			WorkDir:   workspaceDir(cluster.UID),
			Workspace: cluster.UID,
		}

		extraVars := map[string]string{
			"losant_api_token":      os.Getenv("LDC_LOSANT_API_TOKEN"),
			"losant_application_id": os.Getenv("LDC_LOSANT_APPLICATION_ID"),
		}

		if err := runner.Destroy(ctx, varFile, extraVars); err != nil {
			fmt.Fprintf(os.Stderr, "error destroying %s: %v\n", cluster.Name, err)
			lastErr = err
		} else {
			if err := reg.Remove(cluster.UID); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to remove %s from state: %v\n", cluster.Name, err)
				lastErr = err
			}
			if err := reg.Save(statePath()); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to save state after removing %s: %v\n", cluster.Name, err)
			}
			fmt.Printf("Cluster %q removed.\n", cluster.Name)
		}
		cleanup()
	}

	return lastErr
}

func runRemoveName(cmd *cobra.Command, args []string) error {
	name := args[0]
	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	var cluster *state.ClusterState
	if removeProvider != "" {
		cluster, err = reg.Find(name, removeProvider)
	} else {
		cluster, err = reg.FindByName(name)
	}
	if err != nil {
		return err
	}

	fmt.Printf("WARNING: This will permanently destroy cluster %q:\n\n", cluster.Name)
	printClusters(os.Stdout, []state.ClusterState{*cluster})
	fmt.Println()

	if !removeConfirm {
		if !promptYesNo("Type 'yes' to confirm removal") {
			fmt.Println("Aborted.")
			return nil
		}
	}

	ctx := context.Background()

	varFile, cleanup, err := writeTempVarFile(*cluster, envOrDefault("LDC_SSH_PUBLIC_KEY", os.Getenv("HOME")+"/.ssh/id_rsa.pub"))
	if err != nil {
		return fmt.Errorf("prepare var file: %w", err)
	}
	defer cleanup()

	runner := &tofu.Runner{
		Binary:    tofuBinary,
		ModuleDir: moduleDir(cluster.Module),
		WorkDir:   workspaceDir(cluster.UID),
		Workspace: cluster.UID,
	}

	extraVars := map[string]string{
		"losant_api_token":      os.Getenv("LDC_LOSANT_API_TOKEN"),
		"losant_application_id": os.Getenv("LDC_LOSANT_APPLICATION_ID"),
	}

	if err := runner.Destroy(ctx, varFile, extraVars); err != nil {
		return fmt.Errorf("destroy cluster %s: %w", cluster.Name, err)
	}

	if err := reg.Remove(cluster.UID); err != nil {
		return fmt.Errorf("remove from state: %w", err)
	}
	if err := reg.Save(statePath()); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	fmt.Printf("Cluster %q removed.\n", cluster.Name)
	return nil
}

func promptYesNo(prompt string) bool {
	fmt.Printf("%s [yes/N]: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(strings.ToLower(scanner.Text())) == "yes"
}
