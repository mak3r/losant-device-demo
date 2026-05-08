package commands

import (
	"fmt"
	"os"
	"os/exec"
	"strings"

	"github.com/spf13/cobra"

	cloudprovider "github.com/mak3r/ldc-demo/internal/provider"
	"github.com/mak3r/ldc-demo/internal/state"
)

var failCmd = &cobra.Command{
	Use:   "fail",
	Short: "Simulate failure scenarios in a deployed cluster",
}

var failNodeCmd = &cobra.Command{
	Use:   "node <cluster-name>",
	Short: "Stop one compute instance in the cluster (simulates node failure)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFailNode,
}

var failNetworkCmd = &cobra.Command{
	Use:   "network <cluster-name>",
	Short: "Block outbound traffic from cluster instances (simulates network failure)",
	Args:  cobra.ExactArgs(1),
	RunE:  runFailNetwork,
}

var failPodCmd = &cobra.Command{
	Use:   "pod <cluster-name>",
	Short: "Deploy a crashlooping pod to the cluster",
	Args:  cobra.ExactArgs(1),
	RunE:  runFailPod,
}

var failSSHKey string

func init() {
	failPodCmd.Flags().StringVar(&failSSHKey, "ssh-key", "", "SSH private key for kubeconfig fetch (default: LDC_SSH_PRIVATE_KEY or ~/.ssh/id_rsa)")
	failCmd.AddCommand(failNodeCmd, failNetworkCmd, failPodCmd)
	rootCmd.AddCommand(failCmd)
}

func runFailNode(cmd *cobra.Command, args []string) error {
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

	instanceRef, err := prov.FindInstance(cmd.Context(), cluster)
	if err != nil {
		return err
	}

	fmt.Printf("Stopping instance %s in cluster %q ...\n", instanceRef, clusterName)
	if err := prov.StopInstance(cmd.Context(), instanceRef, cluster); err != nil {
		return fmt.Errorf("stop instance: %w", err)
	}
	fmt.Printf("Instance %s stopped. Use 'ldc-demo fix node %s' to restore.\n", instanceRef, clusterName)
	return nil
}

func runFailNetwork(cmd *cobra.Command, args []string) error {
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

	fmt.Printf("Blocking outbound traffic for cluster %q (barrier: %s) ...\n", clusterName, barrierRef)
	if err := prov.BlockOutbound(cmd.Context(), barrierRef, cluster); err != nil {
		return fmt.Errorf("block outbound: %w", err)
	}
	fmt.Printf("Outbound traffic blocked for cluster %q. Use 'ldc-demo fix network %s' to restore.\n", clusterName, clusterName)
	return nil
}

func runFailPod(cmd *cobra.Command, args []string) error {
	clusterName := args[0]

	kcPath, err := ensureKubeconfig(cmd.Context(), clusterName, failSSHKey)
	if err != nil {
		return err
	}

	crashManifest := `apiVersion: apps/v1
kind: Deployment
metadata:
  name: ldc-demo-crashloop
  namespace: default
  labels:
    app: ldc-demo-crashloop
    managed-by: ldc-demo
spec:
  replicas: 1
  selector:
    matchLabels:
      app: ldc-demo-crashloop
  template:
    metadata:
      labels:
        app: ldc-demo-crashloop
    spec:
      containers:
      - name: crasher
        image: busybox:latest
        command: ["sh", "-c", "exit 1"]`

	fmt.Printf("Deploying crashlooping pod to cluster %q ...\n", clusterName)
	//nolint:gosec // G204: kcPath is a program-controlled path, stdin provides manifest
	kubectlCmd := exec.CommandContext(cmd.Context(), "kubectl", "apply", "-f", "-", "--kubeconfig", kcPath)
	kubectlCmd.Stdin = strings.NewReader(crashManifest)
	kubectlCmd.Stdout = os.Stdout
	kubectlCmd.Stderr = os.Stderr
	if err := kubectlCmd.Run(); err != nil {
		return fmt.Errorf("kubectl apply: %w", err)
	}
	fmt.Printf("Crashlooping pod deployed to cluster %q. Use 'ldc-demo fix pod %s' to remove.\n", clusterName, clusterName)
	return nil
}
