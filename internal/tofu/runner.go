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
	return r.runWithEnv(ctx, nil, args...)
}

// runWithEnv runs tofu with extra TF_VAR_* env vars for secret injection.
// Secrets in extraEnv are passed as environment variables instead of -var= flags
// to prevent exposure in /proc/<pid>/cmdline and ps output.
func (r *Runner) runWithEnv(ctx context.Context, extraEnv map[string]string, args ...string) error {
	if err := os.MkdirAll(r.WorkDir, 0700); err != nil {
		return fmt.Errorf("create workspace dir: %w", err)
	}
	//nolint:gosec // G204: Binary is a known tofu path set at construction; args are program-controlled
	cmd := exec.CommandContext(ctx, r.Binary, args...)
	cmd.Dir = r.WorkDir
	env := os.Environ()
	for k, v := range extraEnv {
		env = append(env, "TF_VAR_"+k+"="+v)
	}
	cmd.Env = env
	cmd.Stdout = r.stdout()
	cmd.Stderr = r.stderr()
	return cmd.Run()
}

func (r *Runner) Init(ctx context.Context) error {
	absModule, err := filepath.Abs(r.ModuleDir)
	if err != nil {
		return fmt.Errorf("resolve module path: %w", err)
	}
	return r.run(ctx, "init", "-input=false", "-from-module="+absModule)
}

// Apply runs tofu apply. extraVars are passed as -var= flags (safe for non-secret
// values). extraEnv keys are injected as TF_VAR_<key>=<value> env vars to keep
// secrets out of the process table.
func (r *Runner) Apply(ctx context.Context, varFile string, extraVars map[string]string, extraEnv ...map[string]string) error {
	args := []string{"apply", "-auto-approve", "-input=false"}
	if varFile != "" {
		args = append(args, "-var-file="+varFile)
	}
	for k, v := range extraVars {
		args = append(args, fmt.Sprintf("-var=%s=%s", k, v))
	}
	var secretEnv map[string]string
	if len(extraEnv) > 0 {
		secretEnv = extraEnv[0]
	}
	return r.runWithEnv(ctx, secretEnv, args...)
}

// Destroy runs tofu destroy. extraVars are passed as -var= flags (safe for non-secret
// values). extraEnv keys are injected as TF_VAR_<key>=<value> env vars to keep
// secrets out of the process table.
func (r *Runner) Destroy(ctx context.Context, varFile string, extraVars map[string]string, extraEnv ...map[string]string) error {
	args := []string{"destroy", "-auto-approve", "-input=false"}
	if varFile != "" {
		args = append(args, "-var-file="+varFile)
	}
	for k, v := range extraVars {
		args = append(args, fmt.Sprintf("-var=%s=%s", k, v))
	}
	var secretEnv map[string]string
	if len(extraEnv) > 0 {
		secretEnv = extraEnv[0]
	}
	return r.runWithEnv(ctx, secretEnv, args...)
}

func (r *Runner) Output(ctx context.Context, key string) (string, error) {
	//nolint:gosec // G204: Binary is a known tofu path set at construction; key is a program-controlled constant
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
