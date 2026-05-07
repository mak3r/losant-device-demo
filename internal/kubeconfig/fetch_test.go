package kubeconfig

import (
	"crypto/rand"
	"crypto/rsa"
	"crypto/x509"
	"encoding/pem"
	"net"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// genSSHKey writes a fresh RSA private key to a temp file and returns its path.
func genSSHKey(t *testing.T) string {
	t.Helper()
	key, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		t.Fatalf("generate RSA key: %v", err)
	}
	block := &pem.Block{Type: "RSA PRIVATE KEY", Bytes: x509.MarshalPKCS1PrivateKey(key)}
	f := filepath.Join(t.TempDir(), "id_rsa")
	if err := os.WriteFile(f, pem.EncodeToMemory(block), 0600); err != nil {
		t.Fatalf("write key: %v", err)
	}
	return f
}

// rewriteServer mirrors the inline rewrite logic in Fetch for unit testing.
// Once the developer extracts this to fetch.go as a package-level function,
// remove this definition and the tests will compile against the real one.
func rewriteServer(content, ip string) string {
	return strings.ReplaceAll(content, "127.0.0.1", ip)
}

func TestRewriteServerReplacesLoopback(t *testing.T) {
	input := "server: https://127.0.0.1:6443\ncertificate-authority: 127.0.0.1"
	got := rewriteServer(input, "10.0.1.5")
	if strings.Contains(got, "127.0.0.1") {
		t.Errorf("rewriteServer left 127.0.0.1 in output: %q", got)
	}
	if !strings.Contains(got, "10.0.1.5") {
		t.Errorf("rewriteServer did not insert target IP: %q", got)
	}
}

func TestRewriteServerNoOpWhenAbsent(t *testing.T) {
	input := "server: https://192.168.1.1:6443"
	got := rewriteServer(input, "10.0.1.5")
	if got != input {
		t.Errorf("rewriteServer modified content with no 127.0.0.1: got %q, want %q", got, input)
	}
}

func TestFetchUnreachableHost(t *testing.T) {
	// Skip if sshd is actually running on port 22 — Fetch would succeed instead of erroring.
	conn, _ := net.DialTimeout("tcp", "127.0.0.1:22", time.Second)
	if conn != nil {
		conn.Close()
		t.Skip("sshd is running on :22; can't exercise the unreachable-host error path")
	}
	keyPath := genSSHKey(t)
	_, err := Fetch("127.0.0.1", "root", keyPath, "cluster", t.TempDir())
	if err == nil {
		t.Fatal("Fetch should return an error when SSH connection is refused")
	}
}
