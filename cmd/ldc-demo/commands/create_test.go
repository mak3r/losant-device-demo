package commands

import (
	"strings"
	"testing"
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

	err := runCreate(dummyCmd(), []string{"mycluster", "gcp"})
	if err == nil {
		t.Fatal("expected error for unsupported provider, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported cloud provider") {
		t.Errorf("expected 'unsupported cloud provider' error, got: %v", err)
	}
}
