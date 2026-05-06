package tofu

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
)

// Runner invokes OpenTofu as a subprocess for a single cluster workspace.
type Runner struct {
	// Binary is the path to the tofu (or terraform) binary.
	Binary string
	// ModuleDir is the absolute path to the OpenTofu module source (e.g. tofu/modules/aws-k3s-single).
	ModuleDir string
	// WorkDir is the per-cluster working directory (e.g. ~/.ldc-demo/workspaces/<uid>/).
	WorkDir string
	// Workspace is the OpenTofu workspace name (equals cluster UID).
	Workspace string
	// Stdout and Stderr receive live output. Defaults to os.Stdout / os.Stderr if nil.
	Stdout io.Writer
	Stderr io.Writer
}

func (r *Runner) stdout() io.Writer {
	if r.Stdout != nil {
		return r.Stdout
	}
	return os.Stdout
}

func (r *Runner) stderr() io.Writer {
	if r.Stderr != nil {
		return r.Stderr
	}
	return os.Stderr
}

func (r *Runner) run(ctx context.Context, args ...string) error {
	if err := os.MkdirAll(r.WorkDir, 0700); err != nil {
		return fmt.Errorf("create workspace dir: %w", err)
	}
	cmd := exec.CommandContext(ctx, r.Binary, args...)
	cmd.Dir = r.WorkDir
	cmd.Env = os.Environ()
	cmd.Stdout = r.stdout()
	cmd.Stderr = r.stderr()
	return cmd.Run()
}

func (r *Runner) Init(ctx context.Context) error {
	absModule, err := filepath.Abs(r.ModuleDir)
	if err != nil {
		return fmt.Errorf("resolve module path: %w", err)
	}
	return r.run(ctx, "init", "-input=false", absModule)
}

func (r *Runner) Apply(ctx context.Context, varFile string, extraVars map[string]string) error {
	absModule, err := filepath.Abs(r.ModuleDir)
	if err != nil {
		return fmt.Errorf("resolve module path: %w", err)
	}
	args := []string{"apply", "-auto-approve", "-input=false"}
	if varFile != "" {
		args = append(args, "-var-file="+varFile)
	}
	for k, v := range extraVars {
		args = append(args, fmt.Sprintf("-var=%s=%s", k, v))
	}
	args = append(args, absModule)
	return r.run(ctx, args...)
}

func (r *Runner) Destroy(ctx context.Context, varFile string, extraVars map[string]string) error {
	absModule, err := filepath.Abs(r.ModuleDir)
	if err != nil {
		return fmt.Errorf("resolve module path: %w", err)
	}
	args := []string{"destroy", "-auto-approve", "-input=false"}
	if varFile != "" {
		args = append(args, "-var-file="+varFile)
	}
	for k, v := range extraVars {
		args = append(args, fmt.Sprintf("-var=%s=%s", k, v))
	}
	args = append(args, absModule)
	return r.run(ctx, args...)
}

func (r *Runner) Output(ctx context.Context, key string) (string, error) {
	cmd := exec.CommandContext(ctx, r.Binary, "output", "-raw", key)
	cmd.Dir = r.WorkDir
	cmd.Env = os.Environ()
	cmd.Stderr = r.stderr()
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("tofu output %s: %w", key, err)
	}
	return string(out), nil
}
