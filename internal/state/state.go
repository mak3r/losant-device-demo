package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/mitchellh/go-homedir"
)

const schemaVersion = 1

type ClusterState struct {
	UID           string    `json:"uid"`
	Name          string    `json:"name"`
	CloudProvider string    `json:"cloud_provider"`
	NodeCount     int       `json:"node_count"`
	WorkerCount   int       `json:"worker_count"`
	Size          string    `json:"size"`
	Region        string    `json:"region"`
	CreatedAt     time.Time `json:"created_at"`
	TofuWorkspace string    `json:"tofu_workspace"`
	Module        string    `json:"module"` // "aws-k3s-single" or "aws-k3s-ha"
	K3sToken      string    `json:"k3s_token,omitempty"`
}

type Registry struct {
	Version  int            `json:"version"`
	Clusters []ClusterState `json:"clusters"`
}

func DefaultStatePath() (string, error) {
	home, err := homedir.Dir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, ".ldc-demo", "state.json"), nil
}

func Load(path string) (*Registry, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return &Registry{Version: schemaVersion}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("read state file: %w", err)
	}
	var r Registry
	if err := json.Unmarshal(data, &r); err != nil {
		return nil, fmt.Errorf("parse state file: %w", err)
	}
	if r.Version != schemaVersion {
		return nil, fmt.Errorf("unsupported state file version %d (expected %d)", r.Version, schemaVersion)
	}
	return &r, nil
}

func (r *Registry) Save(path string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
		return fmt.Errorf("create state directory: %w", err)
	}
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal state: %w", err)
	}
	return os.WriteFile(path, data, 0600)
}

func (r *Registry) Add(c ClusterState) (ClusterState, error) {
	for _, existing := range r.Clusters {
		if existing.Name == c.Name && existing.CloudProvider == c.CloudProvider {
			return ClusterState{}, fmt.Errorf("cluster %q on %s already exists (uid: %s)", c.Name, c.CloudProvider, existing.UID)
		}
	}
	c.UID = uuid.New().String()
	c.TofuWorkspace = c.UID
	if c.CreatedAt.IsZero() {
		c.CreatedAt = time.Now().UTC()
	}
	r.Clusters = append(r.Clusters, c)
	return c, nil
}

func (r *Registry) Remove(uid string) error {
	for i, c := range r.Clusters {
		if c.UID == uid {
			r.Clusters = append(r.Clusters[:i], r.Clusters[i+1:]...)
			return nil
		}
	}
	return fmt.Errorf("cluster with uid %q not found", uid)
}

func (r *Registry) Find(name, provider string) (*ClusterState, error) {
	for i := range r.Clusters {
		if r.Clusters[i].Name == name && r.Clusters[i].CloudProvider == provider {
			return &r.Clusters[i], nil
		}
	}
	return nil, fmt.Errorf("no cluster named %q on %s", name, provider)
}

func (r *Registry) FindByName(name string) (*ClusterState, error) {
	var matches []*ClusterState
	for i := range r.Clusters {
		if r.Clusters[i].Name == name {
			matches = append(matches, &r.Clusters[i])
		}
	}
	switch len(matches) {
	case 0:
		return nil, fmt.Errorf("no cluster named %q", name)
	case 1:
		return matches[0], nil
	default:
		return nil, fmt.Errorf("multiple clusters named %q — specify cloud provider", name)
	}
}

func (r *Registry) All() []ClusterState {
	return r.Clusters
}
