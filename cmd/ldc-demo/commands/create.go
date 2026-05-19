package commands

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	cloudprovider "github.com/mak3r/ldc-demo/internal/provider"
	"github.com/mak3r/ldc-demo/internal/state"
	"github.com/mak3r/ldc-demo/internal/tofu"
)

var (
	createSize        string
	createK3sChannel  string
	createSSHKey      string
	createRegion      string
	createAllowedCIDR string
)

var createCmd = &cobra.Command{
	Use:   "create [ha] <name> <cloud-provider>",
	Short: "Create a k3s cluster (prefix with 'ha' for 3-node HA)",
	Long: `Create a single-node or HA k3s cluster on the specified cloud provider.

Examples:
  ldc-demo create my-demo aws
  ldc-demo create ha my-ha-demo aws --size medium`,
	Args:    cobra.RangeArgs(2, 3),
	RunE:    runCreate,
}

func init() {
	createCmd.Flags().StringVar(&createSize, "size", "small", "instance size: small, medium, large")
	createCmd.Flags().StringVar(&createK3sChannel, "k3s-channel", "stable", "k3s release channel")
	createCmd.Flags().StringVar(&createSSHKey, "ssh-key", "", "path to SSH public key (default: ~/.ssh/id_rsa.pub)")
	createCmd.Flags().StringVar(&createRegion, "region", "us-east-1", "cloud provider region (for GCP, pass a zone, e.g. us-central1-a)")
	createCmd.Flags().StringVar(&createAllowedCIDR, "allowed-cidr", "", "restrict SSH and k3s API access to this CIDR (default: 0.0.0.0/0)")
	rootCmd.AddCommand(createCmd)
}

func runCreate(cmd *cobra.Command, args []string) error {
	ha := false
	var name, provider string

	if args[0] == "ha" {
		if len(args) != 3 {
			return fmt.Errorf("usage: ldc-demo create ha <name> <cloud-provider>")
		}
		ha = true
		name = args[1]
		provider = args[2]
	} else {
		if len(args) != 2 {
			return fmt.Errorf("usage: ldc-demo create <name> <cloud-provider>")
		}
		name = args[0]
		provider = args[1]
	}

	if !isValidSize(createSize) {
		return fmt.Errorf("invalid --size %q: must be small, medium, or large", createSize)
	}
	prov, err := cloudprovider.ForName(provider)
	if err != nil {
		return err
	}
	if provider == "gcp" {
		if err := checkRequiredEnv("GCLOUD_PROJECT"); err != nil {
			return err
		}
	}
	if provider == "aws" {
		if err := checkRequiredEnv("AWS_ACCESS_KEY_ID", "AWS_SECRET_ACCESS_KEY"); err != nil {
			return err
		}
	}

	if err := checkRequiredEnv("LDC_LOSANT_API_TOKEN", "LDC_LOSANT_APPLICATION_ID"); err != nil {
		return err
	}

	sshPublicKey := createSSHKey
	if sshPublicKey == "" {
		sshPublicKey = envOrDefault("LDC_SSH_PUBLIC_KEY", filepath.Join(os.Getenv("HOME"), ".ssh", "id_rsa.pub"))
	}
	if _, err := os.Stat(sshPublicKey); err != nil {
		return fmt.Errorf("SSH public key not found at %s (set --ssh-key or LDC_SSH_PUBLIC_KEY)", sshPublicKey)
	}

	moduleName := prov.ModuleName(ha)
	nodeCount := 1
	if ha {
		nodeCount = 3
	}

	reg, err := state.Load(statePath())
	if err != nil {
		return fmt.Errorf("load state: %w", err)
	}

	cluster, err := reg.Add(state.ClusterState{
		Name:           name,
		CloudProvider:  provider,
		NodeCount:      nodeCount,
		Size:           createSize,
		Region:         createRegion,
		Module:         moduleName,
		ProviderConfig: gcpProviderConfig(provider, createRegion),
	})
	if err != nil {
		return err
	}

	varFile, cleanup, err := writeTempVarFile(cluster, sshPublicKey)
	if err != nil {
		return fmt.Errorf("prepare var file: %w", err)
	}
	defer cleanup()

	runner := &tofu.Runner{
		Binary:    tofuBinary,
		ModuleDir: moduleDir(moduleName),
		WorkDir:   workspaceDir(cluster.UID),
		Workspace: cluster.UID,
	}

	ctx := context.Background()
	fmt.Printf("Initializing OpenTofu for cluster %q ...\n", name)
	if err := runner.Init(ctx); err != nil {
		return fmt.Errorf("tofu init: %w", err)
	}

	fmt.Printf("Provisioning %s cluster %q on %s (size: %s) ...\n", clusterType(ha), name, provider, createSize)
	extraVars := map[string]string{
		"k3s_channel": createK3sChannel,
	}
	if createAllowedCIDR != "" {
		extraVars["allowed_cidr"] = createAllowedCIDR
	}
	secretEnv := map[string]string{
		"losant_api_token":      os.Getenv("LDC_LOSANT_API_TOKEN"),
		"losant_application_id": os.Getenv("LDC_LOSANT_APPLICATION_ID"),
	}
	if err := runner.Apply(ctx, varFile, extraVars, secretEnv); err != nil {
		return fmt.Errorf("tofu apply: %w", err)
	}

	// Read and store the k3s join token so scale can pass it to worker nodes.
	token, err := runner.Output(ctx, "k3s_token")
	if err != nil {
		return fmt.Errorf("read k3s_token output: %w", err)
	}
	if cs, err := reg.FindByName(name); err == nil {
		cs.K3sToken = token
	}

	// Persist state only after successful apply.
	if err := reg.Save(statePath()); err != nil {
		return fmt.Errorf("save state: %w", err)
	}

	fmt.Println()
	printClusters(os.Stdout, []state.ClusterState{cluster})
	fmt.Printf("\nRun 'ldc-demo get-kubeconfig %s' to fetch the kubeconfig.\n", name)
	return nil
}

func clusterType(ha bool) string {
	if ha {
		return "HA"
	}
	return "single-node"
}

func isValidSize(s string) bool {
	return s == "small" || s == "medium" || s == "large"
}

func checkRequiredEnv(keys ...string) error {
	var missing []string
	for _, k := range keys {
		if os.Getenv(k) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		return fmt.Errorf("required environment variable(s) not set: %s\nSee .env.template for setup instructions.", strings.Join(missing, ", "))
	}
	return nil
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

// writeTempVarFile copies the resource size template into a temporary file and
// adds cluster-specific variables. Returns path and a cleanup func.
func writeTempVarFile(cluster state.ClusterState, sshPublicKey string) (string, func(), error) {
	templatePath := filepath.Join(repoRoot(), "tofu", "resources", cluster.CloudProvider+"-"+cluster.Size+".tfvars.template")
	templateData, err := os.ReadFile(templatePath)
	if err != nil {
		return "", func() {}, fmt.Errorf("read size template %s: %w", templatePath, err)
	}

	prov, _ := cloudprovider.ForName(cluster.CloudProvider) // error impossible: already validated at create time
	extra := buildVarBlock(cluster.Name, sshPublicKey, prov.VarFileVars(cluster))

	content := string(templateData) + extra

	f, err := os.CreateTemp("", "ldc-demo-*.tfvars")
	if err != nil {
		return "", func() {}, fmt.Errorf("create temp var file: %w", err)
	}
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		return "", func() {}, err
	}
	f.Close()

	return f.Name(), func() { os.Remove(f.Name()) }, nil
}

func gcpProviderConfig(providerName, zone string) map[string]string {
	if providerName != "gcp" {
		return nil
	}
	return map[string]string{
		"gcp_project": os.Getenv("GCLOUD_PROJECT"),
		"gcp_zone":    zone,
	}
}

func buildVarBlock(name, sshKey string, provVars map[string]string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "\ncluster_name        = %q\n", name)
	for k, v := range provVars {
		fmt.Fprintf(&b, "%-20s = %q\n", k, v)
	}
	fmt.Fprintf(&b, "ssh_public_key_path  = %q\n", sshKey)
	return b.String()
}

func printClusters(w io.Writer, clusters []state.ClusterState) {
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	fmt.Fprintln(tw, "UID\tNAME\tPROVIDER\tNODES\tSIZE\tCREATED")
	for _, c := range clusters {
		fmt.Fprintf(tw, "%s\t%s\t%s\t%d\t%s\t%s\n",
			c.UID, c.Name, c.CloudProvider, c.NodeCount, c.Size,
			c.CreatedAt.Format("2006-01-02 15:04"))
	}
	tw.Flush()
}
