package tofu

import "context"

func (r *Runner) WorkspaceNew(ctx context.Context) error {
	return r.run(ctx, "workspace", "new", r.Workspace)
}

func (r *Runner) WorkspaceSelect(ctx context.Context) error {
	return r.run(ctx, "workspace", "select", r.Workspace)
}

func (r *Runner) WorkspaceDelete(ctx context.Context) error {
	return r.run(ctx, "workspace", "select", "default")
}
