# ldc-demo Quickstart

Stand up a k3s cluster with the losant-device controller running in under 10 minutes.

## Prerequisites

| Tool | Minimum Version | Install |
|---|---|---|
| Go | 1.22 | https://go.dev/dl/ |
| OpenTofu | 1.6 | https://opentofu.org/docs/intro/install/ |
| AWS CLI | 2.x | https://aws.amazon.com/cli/ |
| kubectl | any | https://kubernetes.io/docs/tasks/tools/ |

You will also need:
- An AWS account with permission to create EC2 instances, security groups, key pairs, and Elastic IPs
- An SSH key pair at `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub` (or specify a custom path)
- A Losant account with an API token and Application ID

## 1. Install ldc-demo

```bash
git clone https://github.com/mak3r/losant-device-demo
cd losant-device-demo
make install
ldc-demo help
```

## 2. Configure credentials

```bash
cp .env.template .env
```

Edit `.env` with your values (the file is gitignored — it will never be committed):

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_DEFAULT_REGION=us-east-1
export LDC_LOSANT_API_TOKEN=...
export LDC_LOSANT_APPLICATION_ID=...
```

Then load them:

```bash
source .env
```

**Getting your Losant credentials:**
- API token: Losant dashboard → Settings → API Tokens → Create Token
- Application ID: visible in your application URL — `app.losant.com/applications/<id>`

## 3. Create a single-node cluster

```bash
ldc-demo create my-demo aws --size small
```

This provisions a `t3.small` EC2 instance running SUSE Linux Micro with:
- k3s installed and running
- losant-device Helm chart auto-deployed from `/var/lib/rancher/k3s/server/manifests/`
- losant-provisioning-credentials secret created in `losant-system` namespace

**Expected time:** 3–5 minutes (EC2 launch + cloud-init + k3s startup)

## 4. Verify the cluster

```bash
# List all deployed clusters
ldc-demo list deployed

# Fetch the kubeconfig
ldc-demo get-kubeconfig my-demo

# Check nodes
kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/my-demo.yaml get nodes

# Check the losant-device controller
kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/my-demo.yaml get pods -n losant-system
```

## 5. Create a 3-node HA cluster

```bash
ldc-demo create ha my-ha-demo aws --size medium
```

This provisions 3 `t3.medium` instances with embedded etcd HA. The first server bootstraps the cluster; the agent nodes poll the server's k3s API (`/readyz`) every 10 seconds and join automatically once it is ready — no fixed sleep.

**Expected time:** 6–10 minutes

## 6. Clean up

Remove a specific cluster:
```bash
# (not yet implemented in MVP — use remove all)
```

Remove everything:
```bash
ldc-demo remove all --confirm
```

This destroys all EC2 instances, security groups, Elastic IPs, and key pairs created by ldc-demo.

## Running E2E Tests

> **Maintainers / release validation only.** E2E tests provision real AWS infrastructure and incur real costs. Do not run them on every commit — they are gated by a build tag and excluded from normal CI.

### Additional prerequisites

All items from [Prerequisites](#prerequisites) above, plus:

- Go toolchain (already required to build ldc-demo)
- Real AWS credentials with permission to create EC2 instances, security groups, key pairs, and Elastic IPs
- Losant API token and Application ID (`LDC_LOSANT_API_TOKEN`, `LDC_LOSANT_APPLICATION_ID`)
- SSH key pair at `~/.ssh/id_rsa` / `~/.ssh/id_rsa.pub` (or set `LDC_SSH_PRIVATE_KEY` / `LDC_SSH_PUBLIC_KEY`)

### Run the suite

```bash
go test -tags e2e ./test/e2e/... -v -timeout 20m
```

The suite covers the full happy-path lifecycle: `create` → `list` → `get-kubeconfig` → node/pod health check → `remove all` → empty-list verify.

### Optional overrides

| Variable | Default | Purpose |
|---|---|---|
| `LDC_DEMO_BIN` | *(built from source)* | Path to a pre-built `ldc-demo` binary — skips the `go build` step in `TestMain` |
| `E2E_AWS_REGION` | `us-east-1` | AWS region for provisioned resources |

### Full acceptance checklist

For the complete human-readable checklist used during release validation (error cases, state integrity, security baseline), see [`docs/acceptance-criteria.md`](acceptance-criteria.md).

## Troubleshooting

**`tofu` not found:** Install OpenTofu from https://opentofu.org or use `--tofu-binary /path/to/terraform` to fall back to Terraform.

**SSH connection refused on `get-kubeconfig`:** The instance may still be running cloud-init. Wait 2–3 minutes and retry.

**HA nodes not joining:** SSH into the server and check `journalctl -u k3s`. Agent nodes poll for server readiness every 10 seconds; if they fail to join after several minutes, the server may have failed to start — check the server node's k3s service logs.

**Losant controller not running:** Check `kubectl get events -n losant-system`. If the Helm chart pull failed, verify the Helm repo URL is reachable from the cluster.

**Credentials not found:** Ensure you ran `source .env` before `ldc-demo create`. The tool reads `LDC_LOSANT_API_TOKEN` and `LDC_LOSANT_APPLICATION_ID` from the environment at create time.

## Security notes

> **Demo use only.** The defaults below are intentionally permissive for ease of use. Read [`docs/security.md`](security.md) before running real workloads.

- **Losant API token in EC2 user-data** — the token is visible to any AWS user with `ec2:DescribeInstanceAttribute` permission. Accepted risk for demos; use AWS Secrets Manager for production.
- **Open security groups** — SSH (22) and k3s API (6443) are open to `0.0.0.0/0` by default. Use `--allowed-cidr` to restrict access to your IP or corporate range.
- AWS credentials are never written to disk by ldc-demo; they are read from environment variables or `~/.aws/credentials`.
- Cluster state at `~/.ldc-demo/state.json` (mode `0600`) contains cluster metadata but no credentials.

For the full credential flow, accepted risks, and mitigation recommendations see [`docs/security.md`](security.md).
