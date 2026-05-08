package commands

import (
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

// --- runFailNode end-to-end (AWS) ---------------------------------------

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

// --- runFixNode end-to-end (AWS) ----------------------------------------

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

// --- GCP variants -------------------------------------------------------

var gcpCluster = state.ClusterState{
	Name:          "demo",
	CloudProvider: "gcp",
	ProviderConfig: map[string]string{
		"gcp_zone": "us-central1-a",
	},
}

func TestRunFailNodeGCP(t *testing.T) {
	withTestState(t, []state.ClusterState{gcpCluster})
	fakeGcloudScript(t, map[string]string{
		"list": "demo-node-0",
		"stop": "",
	})
	if err := runFailNode(dummyCmd(), []string{"demo"}); err != nil {
		t.Fatalf("runFailNode GCP: %v", err)
	}
}

func TestRunFixNodeGCP(t *testing.T) {
	withTestState(t, []state.ClusterState{gcpCluster})
	fakeGcloudScript(t, map[string]string{
		"list":  "demo-node-0",
		"start": "",
	})
	if err := runFixNode(dummyCmd(), []string{"demo"}); err != nil {
		t.Fatalf("runFixNode GCP: %v", err)
	}
}

func TestRunFailNetworkGCP(t *testing.T) {
	withTestState(t, []state.ClusterState{gcpCluster})
	fakeGcloudScript(t, map[string]string{
		"update": "",
	})
	if err := runFailNetwork(dummyCmd(), []string{"demo"}); err != nil {
		t.Fatalf("runFailNetwork GCP: %v", err)
	}
}

func TestRunFixNetworkGCP(t *testing.T) {
	withTestState(t, []state.ClusterState{gcpCluster})
	fakeGcloudScript(t, map[string]string{
		"update": "",
	})
	if err := runFixNetwork(dummyCmd(), []string{"demo"}); err != nil {
		t.Fatalf("runFixNetwork GCP: %v", err)
	}
}
