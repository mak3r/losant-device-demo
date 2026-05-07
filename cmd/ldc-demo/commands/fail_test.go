package commands

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"

	"github.com/mak3r/ldc-demo/internal/state"
)

// dummyCmd returns a cobra.Command with a background context, suitable for
// passing to RunE functions that call cmd.Context().
// If remove_test.go is merged into this branch, remove this definition.
func dummyCmd() *cobra.Command {
	cmd := &cobra.Command{}
	cmd.SetContext(context.Background())
	return cmd
}

// withTestState creates a temp state registry and overrides stateDir.
// If remove_test.go is merged into this branch, remove this definition.
func withTestState(t *testing.T, clusters []state.ClusterState) {
	t.Helper()
	dir := t.TempDir()
	reg, _ := state.Load(filepath.Join(dir, "nope.json"))
	for _, c := range clusters {
		if _, err := reg.Add(c); err != nil {
			t.Fatalf("setup: add cluster %q: %v", c.Name, err)
		}
	}
	if err := reg.Save(filepath.Join(dir, "state.json")); err != nil {
		t.Fatalf("setup: save state: %v", err)
	}
	old := stateDir
	stateDir = dir
	t.Cleanup(func() { stateDir = old })
}

// fakeAWSScript installs a fake `aws` binary at the front of PATH.
// The script outputs responseBySubcmd[subcommand] and exits 0.
// Subcommands not in the map cause exit 1.
func fakeAWSScript(t *testing.T, responseBySubcmd map[string]string) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "aws")

	cases := ""
	for subcmd, out := range responseBySubcmd {
		cases += fmt.Sprintf("        %s) echo %q ;;\n", subcmd, out)
	}
	src := fmt.Sprintf("#!/bin/sh\ncase \"$2\" in\n%s        *) echo \"unsupported: $2\" >&2; exit 1 ;;\nesac\n", cases)
	if err := os.WriteFile(script, []byte(src), 0700); err != nil {
		t.Fatalf("write fake aws: %v", err)
	}
	old := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+old)
	t.Cleanup(func() { os.Setenv("PATH", old) })
}

// --- fail node ---------------------------------------------------------

func TestFailNodeClusterNotFound(t *testing.T) {
	withTestState(t, nil)
	if err := runFailNode(dummyCmd(), []string{"missing"}); err == nil {
		t.Fatal("expected error for unknown cluster")
	}
}

// --- fail network -------------------------------------------------------

func TestFailNetworkClusterNotFound(t *testing.T) {
	withTestState(t, nil)
	if err := runFailNetwork(dummyCmd(), []string{"missing"}); err == nil {
		t.Fatal("expected error for unknown cluster")
	}
}

// --- fix node -----------------------------------------------------------

func TestFixNodeClusterNotFound(t *testing.T) {
	withTestState(t, nil)
	if err := runFixNode(dummyCmd(), []string{"missing"}); err == nil {
		t.Fatal("expected error for unknown cluster")
	}
}

// --- fix network --------------------------------------------------------

func TestFixNetworkClusterNotFound(t *testing.T) {
	withTestState(t, nil)
	if err := runFixNetwork(dummyCmd(), []string{"missing"}); err == nil {
		t.Fatal("expected error for unknown cluster")
	}
}

// --- findClusterInstance ------------------------------------------------

func TestFindClusterInstanceReturnsID(t *testing.T) {
	fakeAWSScript(t, map[string]string{"describe-instances": "i-0abc12345"})
	id, err := findClusterInstance(context.Background(), "demo")
	if err != nil {
		t.Fatalf("findClusterInstance: %v", err)
	}
	if id != "i-0abc12345" {
		t.Errorf("got instance ID %q, want %q", id, "i-0abc12345")
	}
}

func TestFindClusterInstanceNoneFound(t *testing.T) {
	fakeAWSScript(t, map[string]string{"describe-instances": "None"})
	if _, err := findClusterInstance(context.Background(), "demo"); err == nil {
		t.Fatal("expected error when describe-instances returns None")
	}
}

// --- findClusterSecurityGroup -------------------------------------------

func TestFindClusterSecurityGroupReturnsID(t *testing.T) {
	fakeAWSScript(t, map[string]string{"describe-instances": "sg-0abc12345"})
	id, err := findClusterSecurityGroup(context.Background(), "demo")
	if err != nil {
		t.Fatalf("findClusterSecurityGroup: %v", err)
	}
	if id != "sg-0abc12345" {
		t.Errorf("got SG ID %q, want %q", id, "sg-0abc12345")
	}
}

// --- findStoppedClusterInstance -----------------------------------------

func TestFindStoppedClusterInstanceReturnsID(t *testing.T) {
	fakeAWSScript(t, map[string]string{"describe-instances": "i-stopped1234"})
	id, err := findStoppedClusterInstance(context.Background(), "demo")
	if err != nil {
		t.Fatalf("findStoppedClusterInstance: %v", err)
	}
	if id != "i-stopped1234" {
		t.Errorf("got instance ID %q, want %q", id, "i-stopped1234")
	}
}

// --- runFailNode end-to-end (fake aws) ----------------------------------

func TestRunFailNodeStopsInstance(t *testing.T) {
	withTestState(t, []state.ClusterState{{Name: "demo", CloudProvider: "aws"}})
	fakeAWSScript(t, map[string]string{
		"describe-instances": "i-0abc12345",
		"stop-instances":     "StoppingInstances",
	})
	if err := runFailNode(dummyCmd(), []string{"demo"}); err != nil {
		t.Fatalf("runFailNode: %v", err)
	}
}

// --- runFixNode end-to-end (fake aws) -----------------------------------

func TestRunFixNodeStartsInstance(t *testing.T) {
	withTestState(t, []state.ClusterState{{Name: "demo", CloudProvider: "aws"}})
	fakeAWSScript(t, map[string]string{
		"describe-instances": "i-stopped1234",
		"start-instances":    "StartingInstances",
	})
	if err := runFixNode(dummyCmd(), []string{"demo"}); err != nil {
		t.Fatalf("runFixNode: %v", err)
	}
}
