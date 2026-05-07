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

## Accepted Risk: EC2 User-Data Token Exposure

The `losant_api_token` (and, in the HA module, the `k3s_token`) are injected into EC2 user-data via the `cloud-init.yaml.tpl` templates. User-data is readable by any IAM principal with `ec2:DescribeInstanceAttribute` on the instance.

**Risk:** Anyone with describe access to the EC2 instance can retrieve the Losant API token in plaintext from the user-data field.

**Why this is acceptable for MVP:** The Losant API token is scoped to a single application and is used only to provision the losant-device operator. A demo cluster is short-lived (under 4 hours). Exposing this token does not grant access to the Kubernetes cluster or to other AWS resources.

**Mitigation path (future):** Store `LDC_LOSANT_API_TOKEN` in AWS Secrets Manager at create time and have cloud-init fetch it via `aws secretsmanager get-secret-value` using the instance's IAM role. This requires an IAM role and instance profile in the OpenTofu modules. Tracked for `persona/gitops-manager` as an optional enhancement — **not required for MVP**.

**Mitigation in place now:**
- The secret manifest written to disk by cloud-init has `permissions: "0600"`.
- `runcmd` does not use `set -x` or any shell debug flag that would echo the token to `/var/log/cloud-init-output.log`.
- The token is not surfaced in any OpenTofu output.

## Accepted Risks: Open Security Group Rules

Both the `aws-k3s-single` and `aws-k3s-ha` OpenTofu modules expose two ports to `0.0.0.0/0` by default:

| Port | Purpose | Risk |
|---|---|---|
| 22/TCP | SSH — used by `ldc-demo get-kubeconfig` to fetch the kubeconfig | Any IP can attempt SSH; mitigated by key-based auth only |
| 6443/TCP | k3s API server — used by `kubectl` and the losant-device operator | Any IP can reach the Kubernetes API; mitigated by kubeconfig bearer tokens |

**Why this is acceptable for a demo tool:** The goal is zero-friction cluster bring-up. Requiring users to know their egress IP before cluster creation adds a step that breaks the "10 minutes to running cluster" promise.

**Why this is not acceptable for real workloads:** An exposed API server is a direct attack surface. Compromising the kubeconfig gives full cluster-admin access. A brute-forced or leaked SSH key gives root on every node.

### Recommended Mitigations for Real Workloads

1. **Restrict by CIDR.** A planned `--allowed-cidr` flag (tracked in a separate issue for `persona/gitops-manager`) will let users pass their IP or CIDR at `ldc-demo create` time. Until that flag exists, users can set the variable directly in `tofu/modules/*/main.tf`.

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
| Local dev / conference demo (< 4 hours) | Default settings are fine; tear down when done |
| Workshop / multi-day demo | Restrict SG rules to your CIDR; rotate kubeconfig daily |
| Production or regulated data | Do not use this tool as-is; harden per items above |
