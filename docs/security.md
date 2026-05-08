# Security Notes

This document covers how credentials flow through `ldc-demo`, which risks are accepted for a demo tool, and what users should do before running real workloads.

## Credentials Used

| Credential | Source | How It Flows |
|---|---|---|
| `AWS_ACCESS_KEY_ID` / `AWS_SECRET_ACCESS_KEY` | Environment variable or `~/.aws/credentials` | Read by OpenTofu at `ldc-demo create` time; never written to disk by the CLI |
| `LDC_LOSANT_API_TOKEN` | Environment variable | Passed to the Losant API at cluster creation; not stored in cluster state |
| `LDC_LOSANT_APPLICATION_ID` | Environment variable | Embedded in the Helm values for the losant-device operator |
| SSH private key | `LDC_SSH_PRIVATE_KEY` (default `~/.ssh/id_rsa`) | Used transiently to fetch the k3s kubeconfig over SSH; never copied or logged |
| kubeconfig | Written by `ldc-demo get-kubeconfig` | Saved to the path the user specifies; contains cluster admin credentials |

**None of these values are written to OpenTofu state** in plaintext beyond what the AWS provider requires (instance metadata, resource IDs). The Losant token and SSH key are not stored in `.tfstate`.

## Gitignored Patterns

The following are excluded from version control:

- `*.tfvars`, `*.tfvars.json` — variable files that may contain secrets (templates with `.template` suffix are allowed)
- `.env`, `*.env` — local credential files
- `*.pem`, `*.key` — private keys and certificates
- `*credentials*.json` — credential JSON files (e.g. AWS, service accounts)
- `.terraform/`, `*.tfstate*` — OpenTofu working directory and state
- `kubeconfig`, `*.kubeconfig`, `kube_config_*` — fetched cluster admin credentials

If `ldc-demo get-kubeconfig` saves a kubeconfig, ensure it lands outside the repository root, or that the filename matches one of the patterns above.

## Losant API Token: Two Delivery Modes

`ldc-demo` supports two ways to deliver the `losant_api_token` to EC2 instances, selectable via the `use_secrets_manager` OpenTofu variable (default: `false`).

### Mode 1 — Direct (default, `use_secrets_manager = false`)

The token is embedded in EC2 user-data via `cloud-init.yaml.tpl` and written into the `losant-provisioning-credentials` Kubernetes Secret at boot.

**Risk:** User-data is readable by any IAM principal with `ec2:DescribeInstanceAttribute` on the instance. Anyone with that permission can retrieve the Losant API token in plaintext.

**Why this is acceptable for MVP demos:** The token is scoped to a single Losant application and is used only to provision the losant-device operator. A demo cluster is short-lived (under 4 hours). Token exposure does not grant Kubernetes cluster access or access to other AWS resources.

**Mitigations in place:**
- The secret manifest written to disk by cloud-init has `permissions: "0600"`.
- `runcmd` does not use `set -x` or any shell debug flag that would echo the token to `/var/log/cloud-init-output.log`.
- The token is not surfaced in any OpenTofu output.

### Mode 2 — AWS Secrets Manager (`use_secrets_manager = true`)

When this mode is enabled, the token never appears in EC2 user-data. The flow is:

1. **At `ldc-demo create`:** OpenTofu stores the token in AWS Secrets Manager at path `ldc-demo/<cluster_name>/losant-api-token` and creates an IAM role `ldc-demo-<cluster_name>` with a `secretsmanager:GetSecretValue` policy scoped to that specific ARN. The role is attached to the EC2 instance as an instance profile.

2. **At boot (cloud-init):** A script at `/usr/local/sbin/write-losant-credentials` runs before k3s starts:
   - Discovers the AWS region via IMDSv2 (no region hard-coded in user-data)
   - Installs the AWS CLI if not present
   - Calls `aws secretsmanager get-secret-value --secret-id <ARN>` — authenticated automatically via the instance profile (no credentials in user-data)
   - Writes the `losant-provisioning-credentials` Kubernetes Secret manifest to `/var/lib/rancher/k3s/server/manifests/` with `permissions: "0600"`

3. **At `ldc-demo remove`:** `tofu destroy` deletes the Secrets Manager secret, the IAM role, and the instance profile along with all other cluster resources.

**Trade-off:** This mode adds IAM role, instance profile, and Secrets Manager resources to the OpenTofu plan. It is recommended for non-ephemeral or shared demo environments where the AWS account is accessible to multiple people. For a short-lived personal demo, Mode 1 is sufficient.

## Accepted Risks: Open Security Group Rules

Both the `aws-k3s-single` and `aws-k3s-ha` OpenTofu modules expose two ports to `0.0.0.0/0` by default:

| Port | Purpose | Risk |
|---|---|---|
| 22/TCP | SSH — used by `ldc-demo get-kubeconfig` to fetch the kubeconfig | Any IP can attempt SSH; mitigated by key-based auth only |
| 6443/TCP | k3s API server — used by `kubectl` and the losant-device operator | Any IP can reach the Kubernetes API; mitigated by kubeconfig bearer tokens |

**Why this is acceptable for a demo tool:** The goal is zero-friction cluster bring-up. Requiring users to know their egress IP before cluster creation adds a step that breaks the "10 minutes to running cluster" promise.

**Why this is not acceptable for real workloads:** An exposed API server is a direct attack surface. Compromising the kubeconfig gives full cluster-admin access. A brute-forced or leaked SSH key gives root on every node.

### Recommended Mitigations for Real Workloads

1. **Restrict by CIDR.** Pass `--allowed-cidr <your-ip>/32` to `ldc-demo create` to limit SSH and k3s API access to a specific address. Without this flag, both ports are open to `0.0.0.0/0`.

2. **Rotate the kubeconfig.** After provisioning, regenerate the k3s server token and replace the admin kubeconfig if the cluster will run for more than a few hours.

3. **Use an SSH bastion or VPN.** Place the nodes in a private subnet and only expose 6443 through an internal load balancer.

4. **Enable AWS Security Hub or GuardDuty.** For any cluster touching real data, enable threat detection on the AWS account.

## OpenTofu Module Security Review

Formal review completed on both `aws-k3s-single` and `aws-k3s-ha` modules.

### Variables

| Variable | Module | `sensitive = true` |
|---|---|---|
| `losant_api_token` | both | ✓ |
| `k3s_token` | aws-k3s-ha only | ✓ |

No sensitive variables are missing the `sensitive` attribute.

### Outputs

No sensitive values are exposed in `outputs.tf` for either module. Outputs are limited to public IPs, cluster name, SSH username, and the remote kubeconfig path.

### Hardcoded Credentials

None found. All secret values flow through Terraform variables injected at `tofu apply` time.

### cloud-init File Permissions

Both the `losant-device.yaml` Helm manifest and the `losant-provisioning-credentials.yaml` secret manifest are written with `permissions: "0600"`. No token-bearing file is world-readable.

### runcmd Logging

No `set -x` or equivalent shell debug flag is present in any `runcmd` block. Token values passed as environment variables to `curl | sh` pipelines are not echoed to `/var/log/cloud-init-output.log`.

### Security Group `allowed_cidr` Variable

**Decision: approved.** Adding a `var.allowed_cidr` variable (default `"0.0.0.0/0"`) to both modules is the right pattern. It preserves the zero-friction default while letting users restrict access at `ldc-demo create` time without editing HCL. Implementation is a `persona/gitops-manager` task (see handoff issue).

## Recommendations Summary

| Scenario | Action |
|---|---|
| Local dev / conference demo (< 4 hours) | Default settings (`use_secrets_manager = false`) are fine; tear down when done |
| Shared or multi-day demo | Set `use_secrets_manager = true`; restrict SG rules with `--allowed-cidr`; rotate kubeconfig daily |
| Production or regulated data | Do not use this tool as-is; harden per items above |
