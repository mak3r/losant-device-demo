package commands

import (
	"strings"
	"testing"
)

func TestApplyConfigNotFound(t *testing.T) {
	withTestState(t, nil)
	err := runApplyConfig(dummyCmd(), []string{"nonexistent-config", "any-cluster"})
	if err == nil {
		t.Fatal("expected error for missing config file, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}
