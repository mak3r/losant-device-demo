package provider

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// fakeAWSScript installs a fake `aws` binary at the front of PATH.
// The script matches on $2 (the AWS subcommand, e.g. "describe-instances").
func fakeAWSScript(t *testing.T, responseBySubcmd map[string]string) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "aws")

	cases := ""
	for subcmd, out := range responseBySubcmd {
		cases += fmt.Sprintf("        %s) echo %q ;;\n", subcmd, out)
	}
	src := fmt.Sprintf("#!/bin/sh\ncase \"$2\" in\n%s        *) echo \"unsupported: $2\" >&2; exit 1 ;;\nesac\n", cases)
	if err := os.WriteFile(script, []byte(src), 0700); err != nil { //nolint:gosec // G306: fake test script must be executable
		t.Fatalf("write fake aws: %v", err)
	}
	old := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+old)
	t.Cleanup(func() { os.Setenv("PATH", old) })
}

// fakeGcloudScript installs a fake `gcloud` binary at the front of PATH.
// The script matches on $3 (the action, e.g. "list", "stop", "update").
func fakeGcloudScript(t *testing.T, responseByAction map[string]string) {
	t.Helper()
	dir := t.TempDir()
	script := filepath.Join(dir, "gcloud")

	cases := ""
	for action, out := range responseByAction {
		cases += fmt.Sprintf("        %s) echo %q ;;\n", action, out)
	}
	src := fmt.Sprintf("#!/bin/sh\ncase \"$3\" in\n%s        *) echo \"unsupported: $3\" >&2; exit 1 ;;\nesac\n", cases)
	if err := os.WriteFile(script, []byte(src), 0700); err != nil { //nolint:gosec // G306: fake test script must be executable
		t.Fatalf("write fake gcloud: %v", err)
	}
	old := os.Getenv("PATH")
	os.Setenv("PATH", dir+":"+old)
	t.Cleanup(func() { os.Setenv("PATH", old) })
}
