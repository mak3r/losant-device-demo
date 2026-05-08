package provider

import (
	"strings"
	"testing"
)

func TestForNameAWS(t *testing.T) {
	p, err := ForName("aws")
	if err != nil {
		t.Fatalf("ForName(aws): %v", err)
	}
	if _, ok := p.(*AWSProvider); !ok {
		t.Errorf("expected *AWSProvider, got %T", p)
	}
}

func TestForNameGCP(t *testing.T) {
	p, err := ForName("gcp")
	if err != nil {
		t.Fatalf("ForName(gcp): %v", err)
	}
	if _, ok := p.(*GCPProvider); !ok {
		t.Errorf("expected *GCPProvider, got %T", p)
	}
}

func TestForNameUnknown(t *testing.T) {
	_, err := ForName("azure")
	if err == nil {
		t.Fatal("expected error for unknown provider, got nil")
	}
	if !strings.Contains(err.Error(), "unsupported") {
		t.Errorf("error %q does not contain 'unsupported'", err.Error())
	}
}
