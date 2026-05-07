package kubeconfig

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"golang.org/x/crypto/ssh"
)

// sshDial establishes SSH connections. Tests override this to redirect to an
// in-process server without needing a real sshd.
var sshDial = func(network, addr string, cfg *ssh.ClientConfig) (*ssh.Client, error) {
	return ssh.Dial(network, addr, cfg)
}

// rewriteServer replaces the loopback address in a kubeconfig with the public
// server IP so the kubeconfig works from outside the cluster.
func rewriteServer(content, ip string) string {
	return strings.ReplaceAll(content, "127.0.0.1", ip)
}

// Fetch retrieves the kubeconfig from a remote k3s server via SSH and writes
// it to destDir/<clusterName>.yaml. Returns the path written.
func Fetch(serverIP, sshUser, sshPrivateKeyPath, clusterName, destDir string) (string, error) {
	signer, err := loadSigner(sshPrivateKeyPath)
	if err != nil {
		return "", err
	}

	cfg := &ssh.ClientConfig{
		User:            sshUser,
		Auth:            []ssh.AuthMethod{ssh.PublicKeys(signer)},
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), //nolint:gosec // demo tool — no host key verification
	}

	client, err := sshDial("tcp", serverIP+":22", cfg)
	if err != nil {
		return "", fmt.Errorf("ssh dial %s: %w", serverIP, err)
	}
	defer client.Close()

	session, err := client.NewSession()
	if err != nil {
		return "", fmt.Errorf("ssh new session: %w", err)
	}
	defer session.Close()

	raw, err := session.Output("sudo cat /etc/rancher/k3s/k3s.yaml")
	if err != nil {
		return "", fmt.Errorf("read k3s kubeconfig: %w", err)
	}

	content := rewriteServer(string(raw), serverIP)

	if err := os.MkdirAll(destDir, 0700); err != nil {
		return "", fmt.Errorf("create kubeconfig dir: %w", err)
	}

	outPath := filepath.Join(destDir, clusterName+".yaml")
	if err := os.WriteFile(outPath, []byte(content), 0600); err != nil {
		return "", fmt.Errorf("write kubeconfig: %w", err)
	}

	return outPath, nil
}

func loadSigner(keyPath string) (ssh.Signer, error) {
	data, err := os.ReadFile(keyPath)
	if err != nil {
		return nil, fmt.Errorf("read SSH key %s: %w", keyPath, err)
	}
	signer, err := ssh.ParsePrivateKey(data)
	if err != nil {
		return nil, fmt.Errorf("parse SSH key: %w", err)
	}
	return signer, nil
}
