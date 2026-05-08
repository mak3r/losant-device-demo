# ldc-demo Architecture

## Component Map

```
┌─────────────────────────────────────────────────────────────┐
│  User workstation                                           │
│                                                             │
│   ldc-demo CLI (Go/cobra)                                   │
│       │                                                     │
│       ▼                                                     │
│   OpenTofu modules  (tofu/**/)                              │
│       │  tofu apply / destroy                               │
└───────┼─────────────────────────────────────────────────────┘
        │  AWS API calls
        ▼
┌─────────────────────────────────────────────────────────────┐
│  AWS (per cluster)                                          │
│   EC2 instance(s)   Security Group   Elastic IP   Key Pair  │
│        │                                                    │
│        │  cloud-init user-data (SSH + k3s install script)   │
│        ▼                                                    │
│   k3s Kubernetes cluster                                    │
│        │                                                    │
│        │  auto-manifest in /var/lib/rancher/k3s/server/     │
│        │                   manifests/                       │
│        ▼                                                    │
│   losant-device Helm chart  (losant-system namespace)       │
│        │                                                    │
│        │  HTTPS outbound                                    │
└────────┼────────────────────────────────────────────────────┘
         ▼
    Losant IoT platform (app.losant.com)
```

### Components

| Component | Description |
|---|---|
| `ldc-demo` CLI | Go + cobra binary; orchestrates state, tofu, and kubeconfig operations |
| OpenTofu modules | `tofu/modules/aws-k3s-single/` and `tofu/modules/aws-k3s-ha/`; declare AWS resources |
| EC2 instance(s) | SUSE Linux Micro; bootstrapped via cloud-init; runs k3s |
| Security Group | Opens port 22 (SSH) and 6443 (k3s API) |
| Elastic IP | Static public IP per cluster; used for kubeconfig server address |
| Key Pair | AWS-managed SSH key pair derived from the user's public key |
| k3s | Lightweight Kubernetes distribution; server mode on all nodes |
| losant-device Helm chart | Kubernetes controller that registers devices with Losant |
| Losant platform | External IoT cloud; receives device registrations from the controller |

## Data Flow — `ldc-demo create`

1. **CLI validates** environment variables (`LDC_LOSANT_API_TOKEN`, `LDC_LOSANT_APPLICATION_ID`) and SSH public key path.
2. **State check** — CLI loads `~/.ldc-demo/state.json` and returns an error if a cluster with the same name already exists.
3. **Tofu workspace** — CLI calls `tofu workspace new <uid>` in the appropriate module directory, then `tofu apply` with all cluster parameters passed as `-var` flags. The Losant API token is passed here and embedded into the EC2 user-data script.
4. **EC2 launch** — tofu creates the key pair, security group, Elastic IP, and EC2 instance. User-data runs on first boot:
   - Installs k3s via the official install script
   - Writes the Helm chart manifest (including the Losant credentials secret) to the k3s auto-deploy path
   - k3s starts and immediately applies the manifest
5. **State write** — CLI adds the cluster record to `state.json` (mode `0600`) and saves it.
6. **Output** — CLI prints the cluster name, provider, node count, and Elastic IP.

### HA variant

For `ldc-demo create ha`, tofu provisions 3 EC2 instances. The first bootstraps the cluster with `--cluster-init`; the other two join using the first node's IP after a 90-second fixed delay. See [Known Limitations](#known-limitations).

## State Management

State is stored at `~/.ldc-demo/state.json` (created with mode `0600`).

### Schema (version 1)

```json
{
  "version": 1,
  "clusters": [
    {
      "uid": "<uuid>",
      "name": "my-demo",
      "cloud_provider": "aws",
      "node_count": 1,
      "size": "small",
      "region": "us-east-1",
      "created_at": "2026-01-01T00:00:00Z",
      "tofu_workspace": "<uuid>",
      "module": "aws-k3s-single"
    }
  ]
}
```

**What is tracked:** cluster identity, provider, size, region, creation time, and the tofu workspace name (used to destroy resources later).

**What is not tracked:** Losant credentials, AWS credentials, SSH private keys, kubeconfig content, or any other secret. Credentials live in environment variables only.

**Concurrency limitation:** The state file is read and written without locking. Running two `ldc-demo create` commands simultaneously can corrupt the registry. See [Known Limitations](#known-limitations).

## Credential Flow

```
User's shell environment
  LDC_LOSANT_API_TOKEN   ─────┐
  LDC_LOSANT_APPLICATION_ID   │
  AWS_ACCESS_KEY_ID           │
  AWS_SECRET_ACCESS_KEY       │
                              ▼
                       ldc-demo CLI
                              │
                    ┌─────────┼──────────────┐
                    │                        │
                    ▼                        ▼
               tofu -var flags          (not forwarded)
               (Losant token +           AWS creds go
               app ID)                  directly to
                    │                  AWS SDK/CLI
                    ▼
           EC2 user-data script
           (Losant token + app ID
            embedded in cloud-init)
                    │
                    ▼
           k3s Secret: losant-provisioning-credentials
           (namespace: losant-system)
                    │
                    ▼
           losant-device controller reads Secret
           and authenticates with Losant platform
```

AWS credentials are consumed by the AWS SDK in the tofu process and are never written to disk by ldc-demo. Losant credentials travel through the tofu variable chain into EC2 user-data and are visible to any AWS user with `ec2:DescribeInstanceAttribute` permission on the instance — see `docs/security.md` for details and mitigations.

## Known Limitations

| Limitation | Detail | Tracking |
|---|---|---|
| HA join timing | Agent nodes poll the server's `/readyz` endpoint every 10 seconds before joining. If the server takes longer than expected, agents retry until timeout. No fixed sleep delay. | — |
| Token in user-data | Losant API token is embedded in EC2 user-data and visible to AWS users with sufficient IAM permissions. | Issue #3 |
| Local tofu state | OpenTofu state is stored locally in `~/.ldc-demo/`. No S3 backend, no state locking, no multi-user support. | — |
| State file concurrency | `state.json` is read/written without a file lock. Concurrent CLI invocations can corrupt the registry. | — |
| Single provider | Only AWS is supported. `--cloud-provider` is validated and rejects any non-`aws` value. | — |

## Future Work

- **S3 backend** for tofu state — enables multi-user and multi-machine use
- **AWS Secrets Manager** for Losant credentials — eliminates user-data token exposure
- **RKE2 support** as an alternative to k3s for production-grade clusters
- **Additional cloud providers** — GCP and Azure module variants
- **State file locking** — prevent concurrent mutation of `state.json`
