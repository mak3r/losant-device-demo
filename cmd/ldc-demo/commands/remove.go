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

var removeCmd = &cobra.Command{
	Use:   "remove",
	Short: "Remove clusters",
}

var removeAllCmd = &cobra.Command{
	Use:   "all",
	Short: "Destroy all managed clusters and their cloud resources",
	RunE:  runRemoveAll,
}

func init() {
	removeAllCmd.Flags().BoolVar(&removeConfirm, "confirm", false, "confirm destructive removal without interactive prompt")
	removeCmd.AddCommand(removeAllCmd)
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
			reg.Remove(cluster.UID)
			if err := reg.Save(statePath()); err != nil {
				fmt.Fprintf(os.Stderr, "warning: failed to save state after removing %s: %v\n", cluster.Name, err)
			}
			fmt.Printf("Cluster %q removed.\n", cluster.Name)
		}
		cleanup()
	}

	return lastErr
}

func promptYesNo(prompt string) bool {
	fmt.Printf("%s [yes/N]: ", prompt)
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	return strings.TrimSpace(strings.ToLower(scanner.Text())) == "yes"
}
