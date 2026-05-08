package provider

import (
	"context"

	"github.com/mak3r/ldc-demo/internal/state"
)

type Provider interface {
	ModuleName(ha bool) string
	SSHUser() string
	VarFileVars(cluster state.ClusterState) map[string]string
	FindInstance(ctx context.Context, cluster *state.ClusterState) (string, error)
	FindStoppedInstance(ctx context.Context, cluster *state.ClusterState) (string, error)
	StopInstance(ctx context.Context, instanceRef string, cluster *state.ClusterState) error
	StartInstance(ctx context.Context, instanceRef string, cluster *state.ClusterState) error
	FindNetworkBarrier(ctx context.Context, cluster *state.ClusterState) (string, error)
	BlockOutbound(ctx context.Context, barrierRef string, cluster *state.ClusterState) error
	RestoreOutbound(ctx context.Context, barrierRef string, cluster *state.ClusterState) error
}
