# Acceptance Criteria — ldc-demo MVP

Human-readable checklist for validating that each MVP command is working correctly.
Run through this checklist against a real AWS account before tagging a release.

## Prerequisites

All items below require these tools in PATH:

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
