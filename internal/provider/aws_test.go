package provider

import (
	"context"
	"testing"

	"github.com/mak3r/ldc-demo/internal/state"
)

func TestAWSModuleName(t *testing.T) {
	a := &AWSProvider{}
	if got := a.ModuleName(false); got != "aws-k3s-single" {
		t.Errorf("ModuleName(false) = %q, want %q", got, "aws-k3s-single")
	}
	if got := a.ModuleName(true); got != "aws-k3s-ha" {
		t.Errorf("ModuleName(true) = %q, want %q", got, "aws-k3s-ha")
	}
}

func TestAWSSSHUser(t *testing.T) {
	a := &AWSProvider{}
	if got := a.SSHUser(); got != "ec2-user" {
		t.Errorf("SSHUser() = %q, want %q", got, "ec2-user")
	}
}

func TestAWSVarFileVars(t *testing.T) {
	a := &AWSProvider{}
	cluster := state.ClusterState{Region: "us-west-2"}
	vars := a.VarFileVars(cluster)
	if vars["aws_region"] != "us-west-2" {
		t.Errorf("aws_region = %q, want %q", vars["aws_region"], "us-west-2")
	}
}

func TestAWSFindInstance(t *testing.T) {
	fakeAWSScript(t, map[string]string{
		"describe-instances": "i-0abc123",
	})
	a := &AWSProvider{}
	cluster := &state.ClusterState{Name: "demo"}
	id, err := a.FindInstance(context.Background(), cluster)
	if err != nil {
		t.Fatalf("FindInstance: %v", err)
	}
	if id != "i-0abc123" {
		t.Errorf("instance ID = %q, want %q", id, "i-0abc123")
	}
}

func TestAWSFindInstanceNoneFound(t *testing.T) {
	fakeAWSScript(t, map[string]string{
		"describe-instances": "None",
	})
	a := &AWSProvider{}
	cluster := &state.ClusterState{Name: "demo"}
	_, err := a.FindInstance(context.Background(), cluster)
	if err == nil {
		t.Fatal("expected error when no instance found, got nil")
	}
}

func TestAWSStopInstance(t *testing.T) {
	fakeAWSScript(t, map[string]string{
		"stop-instances": "",
	})
	a := &AWSProvider{}
	cluster := &state.ClusterState{Name: "demo"}
	if err := a.StopInstance(context.Background(), "i-0abc123", cluster); err != nil {
		t.Fatalf("StopInstance: %v", err)
	}
}

func TestAWSFindNetworkBarrier(t *testing.T) {
	fakeAWSScript(t, map[string]string{
		"describe-instances": "sg-0xyz789",
	})
	a := &AWSProvider{}
	cluster := &state.ClusterState{Name: "demo"}
	sgID, err := a.FindNetworkBarrier(context.Background(), cluster)
	if err != nil {
		t.Fatalf("FindNetworkBarrier: %v", err)
	}
	if sgID != "sg-0xyz789" {
		t.Errorf("sgID = %q, want %q", sgID, "sg-0xyz789")
	}
}

func TestAWSBlockOutbound(t *testing.T) {
	fakeAWSScript(t, map[string]string{
		"revoke-security-group-egress": "",
	})
	a := &AWSProvider{}
	cluster := &state.ClusterState{Name: "demo"}
	if err := a.BlockOutbound(context.Background(), "sg-0xyz789", cluster); err != nil {
		t.Fatalf("BlockOutbound: %v", err)
	}
}
