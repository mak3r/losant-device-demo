# ldc-demo

[![License](https://img.shields.io/badge/License-Apache_2.0-blue.svg)](LICENSE)

A CLI tool and companion OpenTofu infrastructure modules for quickly provisioning and managing k3s Kubernetes clusters on cloud providers to demo the [losant-device](https://github.com/mak3r/losant-device) controller. Go from `git clone` to a running cluster with the losant-device operator installed in under 10 minutes.

## Architecture

```
ldc-demo CLI
    │
    ▼
OpenTofu modules
    │
    ▼
AWS EC2 instance(s)
    │
    ▼
k3s Kubernetes cluster
    │
    ▼
losant-device controller (Helm)
    │
    ▼
Losant IoT platform
```

## Prerequisites

| Tool | Minimum Version | Install |
|---|---|---|
| Go | 1.22 | https://go.dev/dl/ |
| OpenTofu | 1.6 | https://opentofu.org/docs/intro/install/ |
| AWS CLI | 2.x | https://aws.amazon.com/cli/ |
| kubectl | any | https://kubernetes.io/docs/tasks/tools/ |

You will also need:
- An AWS account with permission to create EC2 instances, security groups, key pairs, and Elastic IPs
- An SSH key pair at `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub`
- A [Losant](https://app.losant.com) account with an API token and Application ID

## Quick Install

```bash
git clone https://github.com/mak3r/losant-device-demo
cd losant-device-demo
make install
```

## Quick Start

```bash
# 1. Copy and populate credentials
cp .env.template .env
# Edit .env with your AWS and Losant credentials, then:
source .env

# 2. Create a single-node cluster
ldc-demo create my-demo aws --size small

# 3. Fetch the kubeconfig and verify
ldc-demo get-kubeconfig my-demo
kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/my-demo.yaml get pods -n losant-system
```

See [docs/quickstart.md](docs/quickstart.md) for the full walkthrough, including HA clusters, credential setup, and troubleshooting.

## Available Commands

| Command | Description |
|---|---|
| `ldc-demo create <name> <provider> [--size <s\|m\|l>]` | Provision a single-node k3s cluster |
| `ldc-demo create ha <name> <provider> [--size <s\|m\|l>]` | Provision a 3-node HA k3s cluster |
| `ldc-demo list deployed` | List all deployed clusters and their status |
| `ldc-demo get-kubeconfig <name>` | Fetch kubeconfig for a deployed cluster |
| `ldc-demo remove all --confirm` | Destroy all managed clusters and cloud resources |

Global flags: `--state-dir <path>` (default: `~/.ldc-demo`), `--tofu-binary <path>` (falls back to `terraform` if `tofu` is not found).

## License

Apache 2.0 — see [LICENSE](LICENSE).
