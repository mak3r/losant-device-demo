package commands

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/mak3r/ldc-demo/internal/state"
)

func TestCreateCmdHasAllowedCIDRFlag(t *testing.T) {
	f := createCmd.Flags().Lookup("allowed-cidr")
	if f == nil {
		t.Fatal("--allowed-cidr flag not registered on createCmd")
	}
	if f.DefValue != "" {
		t.Errorf("--allowed-cidr default: got %q, want %q", f.DefValue, "")
	}
}

func TestCreateInvalidSize(t *testing.T) {
	old := createSize
	createSize = "xlarge"
	t.Cleanup(func() { createSize = old })

	err := runCreate(dummyCmd(), []string{"mycluster", "aws"})
	if err == nil {
		t.Fatal("expected error for invalid --size, got nil")
	}
	if !strings.Contains(err.Error(), "invalid --size") {
		t.Errorf("expected 'invalid --size' error, got: %v", err)
	}
}

func TestCreateUnsupportedProvider(t *testing.T) {
	old := createSize
	createSize = "small"
	t.Cleanup(func() { createSize = old })

	err := runCreate(dummyCmd(), []string{"mycluster", "azure"})
	if err == nil {
		t.Fatal("expected error for unsupported provider, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported cloud provider") {
		t.Errorf("expected 'unsupported cloud provider' error, got: %v", err)
	}
}

func TestRunCreateAcceptsGCPProvider(t *testing.T) {
	oldSize := createSize
	oldKey := createSSHKey
	createSize = "small"
	t.Cleanup(func() {
		createSize = oldSize
		createSSHKey = oldKey
	})

	// Create a temp SSH public key file.
	keyFile := filepath.Join(t.TempDir(), "id_rsa.pub")
	if err := os.WriteFile(keyFile, []byte("ssh-rsa AAAAB3 test"), 0600); err != nil {
		t.Fatalf("write temp key: %v", err)
	}
	createSSHKey = keyFile

	t.Setenv("GCLOUD_PROJECT", "test-proj")
	t.Setenv("LDC_LOSANT_API_TOKEN", "tok")
	t.Setenv("LDC_LOSANT_APPLICATION_ID", "appid")

	withTestState(t, nil)

	err := runCreate(dummyCmd(), []string{"mycluster", "gcp"})
	// GCP is now valid; error should be at the var file or tofu stage, not "unsupported".
	if err == nil {
		t.Fatal("expected error (no template / tofu not available), got nil")
	}
	if strings.Contains(err.Error(), "unsupported cloud provider") {
		t.Errorf("GCP should be accepted; got unexpected 'unsupported cloud provider' error: %v", err)
	}
}

// setupTemplateDir creates a temp directory tree that satisfies repoRoot():
// - tofu/modules/ (presence triggers the root detection)
// - tofu/resources/<provider>-small.tfvars.template
// It changes the working directory to that temp dir and registers cleanup.
func setupTemplateDir(t *testing.T, provider, content string) {
	t.Helper()
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "tofu", "modules"), 0755); err != nil {
		t.Fatalf("mkdir tofu/modules: %v", err)
	}
	resourcesDir := filepath.Join(dir, "tofu", "resources")
	if err := os.MkdirAll(resourcesDir, 0755); err != nil {
		t.Fatalf("mkdir tofu/resources: %v", err)
	}
	tmplPath := filepath.Join(resourcesDir, provider+"-small.tfvars.template")
	if err := os.WriteFile(tmplPath, []byte(content), 0600); err != nil {
		t.Fatalf("write template: %v", err)
	}
	orig, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("chdir: %v", err)
	}
	t.Cleanup(func() { os.Chdir(orig) }) //nolint:errcheck
}

func TestCreateAWSBothCredentialsMissing(t *testing.T) {
	oldSize := createSize
	createSize = "small"
	t.Cleanup(func() { createSize = oldSize })

	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	err := runCreate(dummyCmd(), []string{"mycluster", "aws"})
	if err == nil {
		t.Fatal("expected error for missing AWS credentials, got nil")
	}
	if !strings.Contains(err.Error(), "AWS_ACCESS_KEY_ID") {
		t.Errorf("expected error to mention AWS_ACCESS_KEY_ID, got: %v", err)
	}
	if !strings.Contains(err.Error(), "AWS_SECRET_ACCESS_KEY") {
		t.Errorf("expected error to mention AWS_SECRET_ACCESS_KEY, got: %v", err)
	}
}

func TestCreateAWSAccessKeyIDMissing(t *testing.T) {
	oldSize := createSize
	createSize = "small"
	t.Cleanup(func() { createSize = oldSize })

	t.Setenv("AWS_ACCESS_KEY_ID", "")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "some-secret")

	err := runCreate(dummyCmd(), []string{"mycluster", "aws"})
	if err == nil {
		t.Fatal("expected error for missing AWS_ACCESS_KEY_ID, got nil")
	}
	if !strings.Contains(err.Error(), "AWS_ACCESS_KEY_ID") {
		t.Errorf("expected error to mention AWS_ACCESS_KEY_ID, got: %v", err)
	}
}

func TestCreateAWSSecretKeyMissing(t *testing.T) {
	oldSize := createSize
	createSize = "small"
	t.Cleanup(func() { createSize = oldSize })

	t.Setenv("AWS_ACCESS_KEY_ID", "some-key-id")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "")

	err := runCreate(dummyCmd(), []string{"mycluster", "aws"})
	if err == nil {
		t.Fatal("expected error for missing AWS_SECRET_ACCESS_KEY, got nil")
	}
	if !strings.Contains(err.Error(), "AWS_SECRET_ACCESS_KEY") {
		t.Errorf("expected error to mention AWS_SECRET_ACCESS_KEY, got: %v", err)
	}
}

func TestCreateGCPProjectMissing(t *testing.T) {
	oldSize := createSize
	createSize = "small"
	t.Cleanup(func() { createSize = oldSize })

	t.Setenv("GCLOUD_PROJECT", "")

	err := runCreate(dummyCmd(), []string{"mycluster", "gcp"})
	if err == nil {
		t.Fatal("expected error for missing GCLOUD_PROJECT, got nil")
	}
	if !strings.Contains(err.Error(), "GCLOUD_PROJECT") {
		t.Errorf("expected error to mention GCLOUD_PROJECT, got: %v", err)
	}
	if strings.Contains(err.Error(), "unsupported cloud provider") {
		t.Errorf("GCP should be a valid provider; got unexpected error: %v", err)
	}
}

func TestWriteTempVarFileGCP(t *testing.T) {
	setupTemplateDir(t, "gcp", "machine_type = \"e2-standard-2\"\n")

	keyFile := filepath.Join(t.TempDir(), "id_rsa.pub")
	if err := os.WriteFile(keyFile, []byte("ssh-rsa AAAAB3 test"), 0600); err != nil {
		t.Fatalf("write temp key: %v", err)
	}

	cluster := state.ClusterState{
		Name:          "demo",
		CloudProvider: "gcp",
		Size:          "small",
		ProviderConfig: map[string]string{
			"gcp_project": "my-proj",
			"gcp_zone":    "us-central1-a",
		},
	}

	path, cleanup, err := writeTempVarFile(cluster, keyFile)
	if err != nil {
		t.Fatalf("writeTempVarFile: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read var file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "gcp_project") {
		t.Errorf("var file missing gcp_project; got:\n%s", content)
	}
	if !strings.Contains(content, "gcp_zone") {
		t.Errorf("var file missing gcp_zone; got:\n%s", content)
	}
	if strings.Contains(content, "aws_region") {
		t.Errorf("var file should not contain aws_region; got:\n%s", content)
	}
}

func TestWriteTempVarFileAWS(t *testing.T) {
	setupTemplateDir(t, "aws", "instance_type = \"t3.small\"\n")

	keyFile := filepath.Join(t.TempDir(), "id_rsa.pub")
	if err := os.WriteFile(keyFile, []byte("ssh-rsa AAAAB3 test"), 0600); err != nil {
		t.Fatalf("write temp key: %v", err)
	}

	cluster := state.ClusterState{
		Name:          "demo",
		CloudProvider: "aws",
		Size:          "small",
		Region:        "us-west-2",
	}

	path, cleanup, err := writeTempVarFile(cluster, keyFile)
	if err != nil {
		t.Fatalf("writeTempVarFile: %v", err)
	}
	defer cleanup()

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read var file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "aws_region") {
		t.Errorf("var file missing aws_region; got:\n%s", content)
	}
}
