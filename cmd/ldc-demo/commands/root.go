package commands

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/mitchellh/go-homedir"
	"github.com/spf13/cobra"
)

var (
	stateDir   string
	tofuBinary string
)

var rootCmd = &cobra.Command{
	Use:   "ldc-demo",
	Short: "Provision and manage k3s clusters for Losant device controller demos",
	Long: `ldc-demo wraps OpenTofu to quickly stand up k3s Kubernetes clusters on cloud
providers pre-configured with the losant-device controller.

Environment variables required for create commands:
  LDC_LOSANT_API_TOKEN       Losant API token
  LDC_LOSANT_APPLICATION_ID  Losant application ID
  AWS_ACCESS_KEY_ID          AWS credentials (or use ~/.aws/credentials)
  AWS_SECRET_ACCESS_KEY`,
	PersistentPreRunE: validateEnvironment,
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	home, _ := homedir.Dir()
	defaultStateDir := filepath.Join(home, ".ldc-demo")

	rootCmd.PersistentFlags().StringVar(&stateDir, "state-dir", defaultStateDir, "directory for ldc-demo state and workspaces")
	rootCmd.PersistentFlags().StringVar(&tofuBinary, "tofu-binary", "", "path to tofu binary (default: resolved from PATH)")

	rootCmd.AddCommand(createCmd, listCmd, removeCmd, getKubeconfigCmd, scaleCmd)
}

func validateEnvironment(cmd *cobra.Command, args []string) error {
	if tofuBinary == "" {
		path, err := exec.LookPath("tofu")
		if err != nil {
			// Fall back to terraform if tofu is not found
			path, err = exec.LookPath("terraform")
			if err != nil {
				return fmt.Errorf("neither 'tofu' nor 'terraform' found in PATH; install OpenTofu or use --tofu-binary")
			}
			fmt.Fprintf(os.Stderr, "warning: 'tofu' not found, falling back to 'terraform'\n")
		}
		tofuBinary = path
	}
	return nil
}

func statePath() string {
	return filepath.Join(stateDir, "state.json")
}

func workspaceDir(uid string) string {
	return filepath.Join(stateDir, "workspaces", uid)
}

func kubeconfigDir() string {
	return filepath.Join(stateDir, "kubeconfigs")
}

func repoRoot() string {
	// Walk up from the binary location to find the repo root containing tofu/modules.
	// In development, use the working directory. In installed use, use the directory
	// of the binary.
	exe, err := os.Executable()
	if err != nil {
		return "."
	}
	// Check common relative positions
	candidates := []string{
		filepath.Dir(exe),
		filepath.Join(filepath.Dir(exe), ".."),
		".",
	}
	for _, candidate := range candidates {
		if _, err := os.Stat(filepath.Join(candidate, "tofu", "modules")); err == nil {
			abs, _ := filepath.Abs(candidate)
			return abs
		}
	}
	return "."
}

func moduleDir(moduleName string) string {
	return filepath.Join(repoRoot(), "tofu", "modules", moduleName)
}
