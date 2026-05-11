# Acceptance Criteria — ldc-demo MVP

Human-readable checklist for validating that each MVP command is working correctly.
Run through this checklist against a real AWS or GCP account before tagging a release.

## Prerequisites — AWS

All AWS items below require these tools in PATH:

- `ldc-demo` (built from this repo)
- `aws` CLI v2 (configured with an IAM user that can create EC2, security groups, key pairs, Elastic IPs)
- `kubectl` (any recent version)
- `tofu` or `terraform`

And these environment variables set:

```
AWS_ACCESS_KEY_ID
AWS_SECRET_ACCESS_KEY
AWS_DEFAULT_REGION       (default: us-east-1)
LDC_LOSANT_API_TOKEN
LDC_LOSANT_APPLICATION_ID
```

SSH key pair present at `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub` (or set `LDC_SSH_PRIVATE_KEY` / `LDC_SSH_PUBLIC_KEY`).

## Prerequisites — GCP

All GCP items below require these tools in PATH:

- `ldc-demo` (built from this repo)
- `gcloud` CLI (authenticated; default project set)
- `kubectl` (any recent version)
- `tofu` or `terraform`

And these environment variables / files set:

```
GOOGLE_APPLICATION_CREDENTIALS   (path to a service account JSON key file)
LDC_LOSANT_API_TOKEN
LDC_LOSANT_APPLICATION_ID
```

The service account referenced by `GOOGLE_APPLICATION_CREDENTIALS` must have the following IAM roles on the target project:

- `roles/compute.admin` (Compute Engine Admin)
- `roles/compute.networkAdmin` (Compute Network Admin)

SSH key pair present at `~/.ssh/id_rsa` and `~/.ssh/id_rsa.pub` (or set `LDC_SSH_PRIVATE_KEY` / `LDC_SSH_PUBLIC_KEY`).

> **Known limitation — AC-GCP-04 on default VPC:** GCP default VPCs may not support per-rule enable/disable via the `disabled` flag on implied firewall rules. If `ldc-demo fail network` returns an error on a default VPC, verify you are using a custom VPC with explicit egress rules. See `docs/security.md` for details.

---

## Command: `ldc-demo create`

**Happy path — single-node cluster**

- [ ] `ldc-demo create e2e-test aws --size small` exits 0
- [ ] Progress output includes "Initializing OpenTofu" and "Provisioning single-node cluster"
- [ ] AWS console shows an EC2 instance tagged `Name=e2e-test` in state `running`
- [ ] `~/.ldc-demo/state.json` contains an entry with `"name": "e2e-test"`
- [ ] Final output table shows: UID, name, provider=aws, nodes=1, size=small

**Happy path — HA cluster**

- [ ] `ldc-demo create ha e2e-ha aws --size medium` exits 0
- [ ] AWS console shows 3 EC2 instances tagged with the cluster name
- [ ] `~/.ldc-demo/state.json` contains an entry with `"node_count": 3`

**Error cases**

- [ ] Running create with a duplicate name on the same provider returns a non-zero exit code and prints a meaningful error message
- [ ] Running create without `LDC_LOSANT_API_TOKEN` set prints "required environment variable(s) not set" and exits non-zero
- [ ] Running create with `--size jumbo` prints "invalid --size" and exits non-zero
- [ ] Running create with `--cloud-provider gcp` prints "only 'aws' is supported" and exits non-zero

---

## Command: `ldc-demo list deployed`

- [ ] After creating `e2e-test`, `ldc-demo list deployed` exits 0 and includes `e2e-test` in the output table
- [ ] Output table columns include: UID, NAME, PROVIDER, NODES, SIZE, CREATED
- [ ] After `remove all`, `ldc-demo list deployed` prints "No clusters deployed."

---

## Command: `ldc-demo get-kubeconfig`

- [ ] `ldc-demo get-kubeconfig e2e-test` exits 0
- [ ] Output confirms kubeconfig path: `~/.ldc-demo/kubeconfigs/e2e-test.yaml`
- [ ] Kubeconfig file exists and is valid YAML
- [ ] `kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/e2e-test.yaml get nodes` shows exactly 1 node in `Ready` state
- [ ] `kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/e2e-test.yaml get pods -n losant-system` shows the losant-device controller pod in `Running` state
- [ ] Running `get-kubeconfig` for a name that does not exist in state exits non-zero with a clear error

---

## Command: `ldc-demo remove all`

- [ ] `ldc-demo remove all` (without `--confirm`) prompts for confirmation before proceeding
- [ ] Entering anything other than "yes" at the prompt prints "Aborted." and exits 0 without destroying anything
- [ ] `ldc-demo remove all --confirm` exits 0
- [ ] All EC2 instances, security groups, Elastic IPs, and key pairs created by ldc-demo are terminated/deleted in AWS (verify in AWS console or via `aws ec2 describe-instances`)
- [ ] `~/.ldc-demo/state.json` no longer contains removed cluster entries
- [ ] `ldc-demo list deployed` prints "No clusters deployed." after removal
- [ ] Running `remove all` when no clusters are deployed prints "No clusters deployed — nothing to remove." and exits 0

---

## Full Lifecycle (Happy Path)

This sequence must complete end-to-end without errors:

```
ldc-demo create e2e-test aws --size small
ldc-demo list deployed               # e2e-test appears
ldc-demo get-kubeconfig e2e-test
kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/e2e-test.yaml get nodes   # 1 node Ready
kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/e2e-test.yaml get pods -n losant-system  # controller Running
ldc-demo remove all --confirm
ldc-demo list deployed               # empty
```

Automated coverage: `go test -tags e2e ./test/e2e/... -v -timeout 20m`

---

## State File Integrity

- [ ] `~/.ldc-demo/state.json` has file permissions `0600`
- [ ] `~/.ldc-demo/state.json` is valid JSON after every operation
- [ ] State directory `~/.ldc-demo/` has permissions `0700`

---

## Security Baseline

- [ ] `~/.ldc-demo/state.json` does not contain any AWS credentials or Losant API tokens
- [ ] Running `ldc-demo create` without AWS credentials in env/`~/.aws/credentials` exits non-zero within 60 seconds (no hanging)
- [ ] `.env` (if created from `.env.template`) is not committed to git (`.gitignore` covers it)

---

## GCP Acceptance Criteria

### AC-GCP-01: Single-node cluster create

- [ ] Given `GOOGLE_APPLICATION_CREDENTIALS` points to a valid SA key with Compute Engine Admin and Compute Network Admin roles
- [ ] When `ldc-demo create gcp-smoke-test gcp --size small --region us-central1-a` is run, it exits 0
- [ ] OpenTofu provisions an `e2-medium` Compute Engine instance in `us-central1-a` (verify in GCP Console or `gcloud compute instances list`)
- [ ] `ldc-demo list deployed` shows a cluster named `gcp-smoke-test` with `PROVIDER=gcp`
- [ ] `ldc-demo get-kubeconfig gcp-smoke-test` exits 0 (SSH user `ubuntu`; retries up to 5 minutes for cloud-init to complete)
- [ ] Kubeconfig is saved to `~/.ldc-demo/kubeconfigs/gcp-smoke-test.yaml` and is valid YAML
- [ ] `kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/gcp-smoke-test.yaml get nodes` shows exactly 1 node in `Ready` state

### AC-GCP-02: HA cluster create

- [ ] `ldc-demo create ha gcp-ha-test gcp --size medium --region us-central1-a` exits 0
- [ ] GCP Console shows 3 Compute Engine instances tagged with the cluster name
- [ ] `kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/gcp-ha-test.yaml get nodes` shows 3 nodes in `Ready` state

### AC-GCP-03: Fail node / fix node

- [ ] Given a running single-node GCP cluster (`gcp-smoke-test`)
- [ ] `ldc-demo fail node gcp-smoke-test` exits 0 and the instance reaches `TERMINATED` state (`gcloud compute instances describe ...`)
- [ ] `ldc-demo fix node gcp-smoke-test` exits 0 and the instance returns to `RUNNING` state

### AC-GCP-04: Fail network / fix network

> **Note:** Requires a custom VPC with explicit egress firewall rules. Behavior on GCP default VPCs may differ — see the known limitation note in the GCP Prerequisites section.

- [ ] Given a running GCP cluster (`gcp-smoke-test`)
- [ ] `ldc-demo fail network gcp-smoke-test` exits 0 and firewall rule `ldc-demo-gcp-smoke-test-allow-egress` is disabled (`gcloud compute firewall-rules describe ...`)
- [ ] `ldc-demo fix network gcp-smoke-test` exits 0 and the firewall rule is re-enabled

### AC-GCP-05: Remove all

- [ ] `ldc-demo remove all --confirm` exits 0 for a running GCP cluster
- [ ] All Compute Engine instances, static IPs, and firewall rules created by ldc-demo are destroyed (verify via `gcloud compute instances list` and `gcloud compute firewall-rules list`)
- [ ] `~/.ldc-demo/state.json` no longer contains the removed cluster entries
- [ ] `ldc-demo list deployed` prints "No clusters deployed." after removal

### AC-GCP-06: Mixed-provider list

- [ ] Given one AWS cluster and one GCP cluster both present in state
- [ ] `ldc-demo list deployed` exits 0 and shows both clusters in the output table
- [ ] The `PROVIDER` column shows `aws` for the AWS cluster and `gcp` for the GCP cluster

### AC-GCP-07: Provider disambiguation

- [ ] Given clusters named `demo` on both `aws` and `gcp`
- [ ] `ldc-demo remove name demo` (without `--provider`) exits non-zero and prints an error that mentions "multiple clusters named"
- [ ] `ldc-demo remove name demo --provider gcp` exits 0 and destroys only the GCP cluster
- [ ] `ldc-demo list deployed` still shows the AWS `demo` cluster after the above command

---

## GCP Full Lifecycle (Happy Path)

This sequence must complete end-to-end without errors:

```
ldc-demo create gcp-smoke-test gcp --size small --region us-central1-a
ldc-demo list deployed                  # gcp-smoke-test appears with provider=gcp
ldc-demo get-kubeconfig gcp-smoke-test
kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/gcp-smoke-test.yaml get nodes   # 1 node Ready
kubectl --kubeconfig ~/.ldc-demo/kubeconfigs/gcp-smoke-test.yaml get pods -n losant-system  # controller Running
ldc-demo remove all --confirm
ldc-demo list deployed                  # empty
```
