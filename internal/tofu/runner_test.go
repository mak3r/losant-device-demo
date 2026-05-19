package tofu

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeTofu writes a shell script that echoes its args to stdout.
// For the "output" subcommand it prints "fake-value" so Output() can be tested.
func fakeTofu(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "tofu")
	src := "#!/bin/sh\nif [ \"$1\" = \"output\" ]; then\n  printf 'fake-value'\nelse\n  echo \"$@\"\nfi\n"
	if err := os.WriteFile(script, []byte(src), 0700); err != nil { //nolint:gosec // G306: fake test script must be executable
		t.Fatalf("write fake tofu: %v", err)
	}
	return script
}

// newRunner returns a Runner backed by the fake binary and two temp dirs.
// The returned buffer captures stdout (and stderr) from every run() call.
func newRunner(t *testing.T) (*Runner, *bytes.Buffer) {
	t.Helper()
	var buf bytes.Buffer
	r := &Runner{
		Binary:    fakeTofu(t),
		ModuleDir: t.TempDir(),
		WorkDir:   filepath.Join(t.TempDir(), "workspace"),
		Workspace: "test-ws",
		Stdout:    &buf,
		Stderr:    &buf,
	}
	return r, &buf
}

func TestRunnerCreatesWorkDir(t *testing.T) {
	r, _ := newRunner(t)
	if _, err := os.Stat(r.WorkDir); !os.IsNotExist(err) {
		t.Skip("WorkDir already exists")
	}
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	if _, err := os.Stat(r.WorkDir); err != nil {
		t.Errorf("WorkDir not created: %v", err)
	}
}

func TestRunnerInit(t *testing.T) {
	r, buf := newRunner(t)
	if err := r.Init(context.Background()); err != nil {
		t.Fatalf("Init: %v", err)
	}
	absModule, _ := filepath.Abs(r.ModuleDir)
	out := buf.String()
	for _, want := range []string{"init", "-input=false", "-from-module=" + absModule} {
		if !strings.Contains(out, want) {
			t.Errorf("Init args missing %q; got: %q", want, out)
		}
	}
	if strings.Contains(out, " "+absModule) {
		t.Errorf("Init must not pass bare module path as positional arg; got: %q", out)
	}
}

func TestRunnerApply(t *testing.T) {
	r, buf := newRunner(t)
	if err := r.Apply(context.Background(), "cluster.tfvars", map[string]string{"region": "us-east-1"}); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	absModule, _ := filepath.Abs(r.ModuleDir)
	out := buf.String()
	for _, want := range []string{"apply", "-auto-approve", "-input=false", "-var-file=cluster.tfvars", "-var=region=us-east-1"} {
		if !strings.Contains(out, want) {
			t.Errorf("Apply args missing %q; got: %q", want, out)
		}
	}
	if strings.Contains(out, absModule) {
		t.Errorf("Apply must not pass module path as arg; got: %q", out)
	}
}

func TestRunnerApplyNoVarFile(t *testing.T) {
	r, buf := newRunner(t)
	if err := r.Apply(context.Background(), "", nil); err != nil {
		t.Fatalf("Apply (no var-file): %v", err)
	}
	if strings.Contains(buf.String(), "-var-file") {
		t.Errorf("Apply should not emit -var-file when varFile is empty; got: %q", buf.String())
	}
}

func TestRunnerDestroy(t *testing.T) {
	r, buf := newRunner(t)
	if err := r.Destroy(context.Background(), "cluster.tfvars", map[string]string{"k": "v"}); err != nil {
		t.Fatalf("Destroy: %v", err)
	}
	absModule, _ := filepath.Abs(r.ModuleDir)
	out := buf.String()
	for _, want := range []string{"destroy", "-auto-approve", "-input=false", "-var-file=cluster.tfvars", "-var=k=v"} {
		if !strings.Contains(out, want) {
			t.Errorf("Destroy args missing %q; got: %q", want, out)
		}
	}
	if strings.Contains(out, absModule) {
		t.Errorf("Destroy must not pass module path as arg; got: %q", out)
	}
}

func TestRunnerOutput(t *testing.T) {
	r, _ := newRunner(t)
	if err := os.MkdirAll(r.WorkDir, 0700); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	val, err := r.Output(context.Background(), "server_ip")
	if err != nil {
		t.Fatalf("Output: %v", err)
	}
	if val != "fake-value" {
		t.Errorf("Output returned %q, want %q", val, "fake-value")
	}
}

func TestRunnerWorkspaceNew(t *testing.T) {
	r, buf := newRunner(t)
	if err := r.WorkspaceNew(context.Background()); err != nil {
		t.Fatalf("WorkspaceNew: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"workspace", "new", "test-ws"} {
		if !strings.Contains(out, want) {
			t.Errorf("WorkspaceNew args missing %q; got: %q", want, out)
		}
	}
}

func TestRunnerWorkspaceSelect(t *testing.T) {
	r, buf := newRunner(t)
	if err := r.WorkspaceSelect(context.Background()); err != nil {
		t.Fatalf("WorkspaceSelect: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"workspace", "select", "test-ws"} {
		if !strings.Contains(out, want) {
			t.Errorf("WorkspaceSelect args missing %q; got: %q", want, out)
		}
	}
}
