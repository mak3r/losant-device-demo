//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	testCluster = "e2e-test"
	testTimeout = 15 * time.Minute
)

var ldcBin string

func TestMain(m *testing.M) {
	required := []string{
		"LDC_LOSANT_API_TOKEN",
		"LDC_LOSANT_APPLICATION_ID",
		"AWS_ACCESS_KEY_ID",
		"AWS_SECRET_ACCESS_KEY",
	}
	var missing []string
	for _, k := range required {
		if os.Getenv(k) == "" {
			missing = append(missing, k)
		}
	}
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "e2e: missing required env vars: %s\nSee docs/acceptance-criteria.md for setup.\n",
			strings.Join(missing, ", "))
		os.Exit(1)
	}

	bin, err := buildOrLocateBinary()
	if err != nil {
		fmt.Fprintf(os.Stderr, "e2e: %v\n", err)
		os.Exit(1)
	}
	ldcBin = bin
	os.Exit(m.Run())
}

func buildOrLocateBinary() (string, error) {
	if bin := os.Getenv("LDC_DEMO_BIN"); bin != "" {
		return bin, nil
	}
	root, err := findModuleRoot()
	if err != nil {
		return "", fmt.Errorf("locate module root: %w", err)
	}
	tmp, err := os.MkdirTemp("", "ldc-demo-e2e-*")
	if err != nil {
		return "", err
	}
	bin := filepath.Join(tmp, "ldc-demo")
	cmd := exec.Command("go", "build", "-o", bin, "./cmd/ldc-demo")
	cmd.Dir = root
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("go build failed: %w\n%s", err, stderr.String())
	}
	return bin, nil
}

func findModuleRoot() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", err
	}
	dir := cwd
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("go.mod not found starting from %s", cwd)
		}
		dir = parent
	}
}

func TestLifecycle(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), testTimeout)
	defer cancel()

	stateDir := t.TempDir()
	region := envOrDefault("E2E_AWS_REGION", "us-east-1")

	// Ensure removal even on test failure.
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cleanupCancel()
		runLDC(cleanupCtx, stateDir, "remove", "all", "--confirm")
	})

	// 1. Create
	t.Log("Step 1: create cluster")
	mustRunLDC(t, ctx, stateDir, "create", testCluster, "aws", "--size", "small", "--region", region)

	// 2. Verify EC2 instance running
	t.Log("Step 2: verify EC2 instance running in AWS")
	instanceID := awsRunningInstance(t, ctx, testCluster, region)
	t.Logf("EC2 instance: %s", instanceID)

	// 3. List deployed — cluster must appear
	t.Log("Step 3: list deployed")
	listOut := mustRunLDC(t, ctx, stateDir, "list", "deployed")
	if !strings.Contains(listOut, testCluster) {
		t.Fatalf("cluster %q not found in list output:\n%s", testCluster, listOut)
	}

	// 4. Fetch kubeconfig
	t.Log("Step 4: get-kubeconfig")
	mustRunLDC(t, ctx, stateDir, "get-kubeconfig", testCluster)
	kubeconfigPath := filepath.Join(stateDir, "kubeconfigs", testCluster+".yaml")
	if _, err := os.Stat(kubeconfigPath); err != nil {
		t.Fatalf("kubeconfig not written to %s: %v", kubeconfigPath, err)
	}

	// 5. Verify 1 node Ready
	t.Log("Step 5: kubectl get nodes")
	nodesOut := mustRunCmd(t, ctx, "kubectl", "--kubeconfig", kubeconfigPath, "get", "nodes")
	t.Logf("nodes:\n%s", nodesOut)
	if !strings.Contains(nodesOut, "Ready") {
		t.Fatalf("no Ready node found:\n%s", nodesOut)
	}

	// 6. Verify losant-device controller Running
	t.Log("Step 6: kubectl get pods -n losant-system")
	podsOut := mustRunCmd(t, ctx, "kubectl", "--kubeconfig", kubeconfigPath,
		"get", "pods", "-n", "losant-system")
	t.Logf("pods:\n%s", podsOut)
	if !strings.Contains(podsOut, "Running") {
		t.Fatalf("no Running pod in losant-system:\n%s", podsOut)
	}

	// 7. Remove all
	t.Log("Step 7: remove all")
	mustRunLDC(t, ctx, stateDir, "remove", "all", "--confirm")

	// 8. Verify EC2 terminated (poll up to 3 minutes for AWS propagation)
	t.Log("Step 8: verify EC2 instance terminated")
	deadline := time.Now().Add(3 * time.Minute)
	var stillRunning string
	for time.Now().Before(deadline) {
		stillRunning = awsRunningInstanceNoFail(ctx, testCluster, region)
		if stillRunning == "" {
			break
		}
		time.Sleep(15 * time.Second)
	}
	if stillRunning != "" {
		t.Fatalf("EC2 instance still running after removal: %s", stillRunning)
	}

	// 9. List deployed — must be empty
	t.Log("Step 9: list deployed (expect empty)")
	listOut = mustRunLDC(t, ctx, stateDir, "list", "deployed")
	if strings.Contains(listOut, testCluster) {
		t.Fatalf("cluster %q still appears in list after removal:\n%s", testCluster, listOut)
	}
}

func mustRunLDC(t *testing.T, ctx context.Context, stateDir string, args ...string) string {
	t.Helper()
	allArgs := append([]string{"--state-dir", stateDir}, args...)
	out, err := runCmd(ctx, ldcBin, allArgs...)
	if err != nil {
		t.Fatalf("ldc-demo %s: %v\n%s", strings.Join(args, " "), err, out)
	}
	return out
}

func runLDC(ctx context.Context, stateDir string, args ...string) {
	allArgs := append([]string{"--state-dir", stateDir}, args...)
	runCmd(ctx, ldcBin, allArgs...) //nolint:errcheck
}

func mustRunCmd(t *testing.T, ctx context.Context, name string, args ...string) string {
	t.Helper()
	out, err := runCmd(ctx, name, args...)
	if err != nil {
		t.Fatalf("%s %s: %v\n%s", name, strings.Join(args, " "), err, out)
	}
	return out
}

func runCmd(ctx context.Context, name string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	return buf.String(), cmd.Run()
}

func awsRunningInstance(t *testing.T, ctx context.Context, cluster, region string) string {
	t.Helper()
	out := mustRunCmd(t, ctx,
		"aws", "ec2", "describe-instances",
		"--filters",
		"Name=tag:Name,Values="+cluster,
		"Name=instance-state-name,Values=running",
		"--query", "Reservations[].Instances[].InstanceId",
		"--output", "text",
		"--region", region,
	)
	id := strings.TrimSpace(out)
	if id == "" {
		t.Fatalf("no running EC2 instance with tag Name=%s in region %s", cluster, region)
	}
	return id
}

func awsRunningInstanceNoFail(ctx context.Context, cluster, region string) string {
	out, err := runCmd(ctx,
		"aws", "ec2", "describe-instances",
		"--filters",
		"Name=tag:Name,Values="+cluster,
		"Name=instance-state-name,Values=running",
		"--query", "Reservations[].Instances[].InstanceId",
		"--output", "text",
		"--region", region,
	)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(out)
}

func envOrDefault(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}
