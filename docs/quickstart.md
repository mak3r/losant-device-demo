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

This provisions 3 `t3.medium` instances with embedded etcd HA. The first server bootstraps the cluster; the other two join after a 90-second delay.

**Expected time:** 6–10 minutes

**Note:** The fixed join delay is a known MVP limitation. If nodes fail to join, see [Troubleshooting](#troubleshooting).

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

## Troubleshooting

**`tofu` not found:** Install OpenTofu from https://opentofu.org or use `--tofu-binary /path/to/terraform` to fall back to Terraform.

**SSH connection refused on `get-kubeconfig`:** The instance may still be running cloud-init. Wait 2–3 minutes and retry.

**HA nodes not joining:** SSH into the server and check `journalctl -u k3s`. The 90-second join delay may need to be increased for slow instance types. This is a known MVP limitation.

**Losant controller not running:** Check `kubectl get events -n losant-system`. If the Helm chart pull failed, verify the Helm repo URL is reachable from the cluster.

**Credentials not found:** Ensure you ran `source .env` before `ldc-demo create`. The tool reads `LDC_LOSANT_API_TOKEN` and `LDC_LOSANT_APPLICATION_ID` from the environment at create time.

## Security notes

- The Losant API token is injected into EC2 user-data via cloud-init. It is not stored in any local file but is visible to AWS users with `ec2:DescribeInstanceAttribute` permission on the instance.
- AWS credentials are never stored by ldc-demo. They are read from environment variables or `~/.aws/credentials`.
- Cluster state is stored at `~/.ldc-demo/state.json` (mode `0600`). This file contains cluster metadata but no credentials.
- Security groups allow SSH (22) and k3s API (6443) from all IPs (`0.0.0.0/0`). For production use, restrict these to known CIDR ranges.
