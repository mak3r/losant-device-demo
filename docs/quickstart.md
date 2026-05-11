# ldc-demo Quickstart

Stand up a k3s cluster with the losant-device controller running in under 10 minutes.

## Prerequisites

| Tool | Minimum Version | Install |
|---|---|---|
| Go | 1.22 | https://go.dev/dl/ |
| OpenTofu | 1.6 | https://opentofu.org/docs/intro/install/ |
| AWS CLI (AWS only) | 2.x | https://aws.amazon.com/cli/ |
| gcloud CLI (GCP only) | latest | https://cloud.google.com/sdk/docs/install |
| kubectl | any | https://kubernetes.io/docs/tasks/tools/ |

You will also need:
- A cloud account with permission to create compute instances and networking resources (AWS or GCP)
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

Edit `.env` with your values (the file is gitignored — it will never be committed), then load them:

```bash
source .env
```

### AWS credentials

```bash
export AWS_ACCESS_KEY_ID=AKIA...
export AWS_SECRET_ACCESS_KEY=...
export AWS_DEFAULT_REGION=us-east-1
export LDC_LOSANT_API_TOKEN=...
export LDC_LOSANT_APPLICATION_ID=...
```

### GCP credentials

Create a service account key in the GCP console and download it as a JSON file, then set:

```bash
export GOOGLE_APPLICATION_CREDENTIALS=/path/to/service-account-key.json
export GCLOUD_PROJECT=your-gcp-project-id
export GCLOUD_ZONE=us-central1-a
export LDC_LOSANT_API_TOKEN=your_losant_api_token_here
export LDC_LOSANT_APPLICATION_ID=your_application_id_here
```

**Getting your Losant credentials:**
- API token: Losant dashboard → Settings → API Tokens → Create Token
- Application ID: visible in your application URL — `app.losant.com/applications/<id>`

## 3. Create a single-node cluster

### AWS

```bash
ldc-demo create my-demo aws --size small
```

This provisions a `t3.small` EC2 instance running SUSE Linux Micro with:
- k3s installed and running
- losant-device Helm chart auto-deployed from `/var/lib/rancher/k3s/server/manifests/`
- losant-provisioning-credentials secret created in `losant-system` namespace

**Expected time:** 3–5 minutes (EC2 launch + cloud-init + k3s startup)

### GCP

```bash
ldc-demo create my-demo gcp --size small --region us-central1-a
```

The `--region` flag accepts a GCP zone (e.g., `us-central1-a`). This provisions an `e2-medium` Compute Engine instance running Ubuntu 24.04 LTS with k3s and the losant-device operator.

**Expected time:** 4–6 minutes (instance start + cloud-init + k3s startup; Ubuntu 24.04 cloud-init may take longer than SUSE)

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

### AWS

```bash
ldc-demo create ha my-ha-demo aws --size medium
```

This provisions 3 `t3.medium` instances with embedded etcd HA. The first server bootstraps the cluster; the agent nodes poll the server's k3s API (`/readyz`) every 10 seconds and join automatically once it is ready — no fixed sleep.

**Expected time:** 6–10 minutes

### GCP

```bash
ldc-demo create ha my-ha-demo gcp --size medium --region us-central1-a
```

This provisions 3 `e2-standard-2` Compute Engine instances. The same agent-polling readiness model applies.

**Expected time:** 8–12 minutes

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

**SSH connection refused on `get-kubeconfig` (AWS):** The instance may still be running cloud-init. Wait 2–3 minutes and retry.

**SSH connection refused on `get-kubeconfig` (GCP):** Ubuntu 24.04 cloud-init may take 3–5 minutes. GCP startup scripts run before cloud-init completes — wait longer than you would for AWS and retry.

**HA nodes not joining:** SSH into the server and check `journalctl -u k3s`. Agent nodes poll for server readiness every 10 seconds; if they fail to join after several minutes, the server may have failed to start — check the server node's k3s service logs.

**Losant controller not running:** Check `kubectl get events -n losant-system`. If the Helm chart pull failed, verify the Helm repo URL is reachable from the cluster.

**Credentials not found:** Ensure you ran `source .env` before `ldc-demo create`. The tool reads `LDC_LOSANT_API_TOKEN` and `LDC_LOSANT_APPLICATION_ID` from the environment at create time.

**`gcloud: command not found` (GCP):** Install the gcloud CLI from https://cloud.google.com/sdk/docs/install, then run `gcloud auth application-default login` before using ldc-demo with GCP.

**`fail network` does not block traffic (GCP):** This command is only effective when using the dedicated VPC created by the `gcp-k3s-*` tofu modules. If you provisioned the instance in GCP's `default` VPC, the egress firewall rules do not apply and traffic will continue to flow. See the known limitation in [`docs/security.md`](security.md#known-limitation-fail-network-egress-behavior-on-gcp).

## Security notes

> **Demo use only.** The defaults below are intentionally permissive for ease of use. Read [`docs/security.md`](security.md) before running real workloads.

- **Losant API token in instance user-data** — on AWS, the token is visible to any IAM principal with `ec2:DescribeInstanceAttribute`; on GCP, to any principal with `compute.instances.get`. Accepted risk for demos.
- **Open firewall rules** — SSH (22) and k3s API (6443) are open to `0.0.0.0/0` by default. Use `--allowed-cidr` to restrict access to your IP or corporate range.
- Cloud credentials are never written to disk by ldc-demo; they are read from environment variables or standard credential files.
- Cluster state at `~/.ldc-demo/state.json` (mode `0600`) contains cluster metadata but no credentials.
- **GCP default service account** — demo clusters run under the default Compute Engine SA, which has `Editor` access by default. Short-lived demo use is acceptable; use a scoped SA for anything longer-lived.

For the full credential flow, accepted risks, and mitigation recommendations see [`docs/security.md`](security.md).
