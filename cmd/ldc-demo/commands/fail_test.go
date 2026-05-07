package commands

import (
	"context"
	"testing"

	"github.com/mak3r/ldc-demo/internal/state"
)

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
