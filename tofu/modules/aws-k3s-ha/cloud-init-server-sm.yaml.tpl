#cloud-config

write_files:
  # Drop the losant-device Helm chart into k3s auto-deploy manifests.
  - path: /var/lib/rancher/k3s/server/manifests/losant-device.yaml
    permissions: "0600"
    content: |
      apiVersion: helm.cattle.io/v1
      kind: HelmChart
      metadata:
        name: losant-device
        namespace: kube-system
      spec:
        chart: losant-device
        repo: https://mak3r.github.io/losant-device
        targetNamespace: losant-system
        createNamespace: true
        valuesContent: |-
          provisioning:
            existingSecret: losant-provisioning-credentials

  # Script to fetch the Losant API token from Secrets Manager and write the
  # k8s provisioning credentials manifest. Runs in runcmd before k3s installs
  # so the manifest is present when k3s first scans its auto-deploy directory.
  - path: /usr/local/sbin/write-losant-credentials
    permissions: "0700"
    owner: root:root
    content: |
      #!/bin/bash
      set -euo pipefail

      SECRET_ARN="${losant_secret_arn}"
      APP_ID="${losant_application_id}"

      # Discover region via IMDSv2 (avoids hard-coding the region in user-data).
      IMDS_TOKEN=$(curl -sf -X PUT "http://169.254.169.254/latest/api/token" \
        -H "X-aws-ec2-metadata-token-ttl-seconds: 60")
      REGION=$(curl -sf -H "X-aws-ec2-metadata-token: $IMDS_TOKEN" \
        "http://169.254.169.254/latest/meta-data/placement/region")

      # Install the AWS CLI binary bundle (not present on SLE Micro base image).
      if ! command -v aws &>/dev/null; then
        curl -fsSL https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip \
          -o /tmp/awscliv2.zip
        cd /tmp && unzip -q awscliv2.zip
        /tmp/aws/install --install-dir /usr/local/aws-cli --bin-dir /usr/local/bin
        rm -rf /tmp/aws /tmp/awscliv2.zip
      fi

      # Fetch the token — instance profile credentials are used automatically.
      TOKEN=$(AWS_DEFAULT_REGION="$REGION" aws secretsmanager get-secret-value \
        --secret-id "$SECRET_ARN" \
        --query SecretString \
        --output text)

      # Write the k8s provisioning credentials manifest.
      mkdir -p /var/lib/rancher/k3s/server/manifests
      python3 -c "
import sys, os
token, app_id = sys.argv[1], sys.argv[2]
path = '/var/lib/rancher/k3s/server/manifests/losant-provisioning-credentials.yaml'
content = (
    'apiVersion: v1\n'
    'kind: Secret\n'
    'metadata:\n'
    '  name: losant-provisioning-credentials\n'
    '  namespace: losant-system\n'
    'stringData:\n'
    '  apiToken: ' + token + '\n'
    '  applicationId: ' + app_id + '\n'
)
open(path, 'w').write(content)
os.chmod(path, 0o600)
" "$TOKEN" "$APP_ID"

runcmd:
  - /usr/local/sbin/write-losant-credentials
  - |
    curl -sfL https://get.k3s.io | \
      K3S_TOKEN="${k3s_token}" \
      INSTALL_K3S_CHANNEL="${k3s_channel}" \
      sh -s - server --cluster-init
  - until kubectl get nodes 2>/dev/null | grep -q Ready; do sleep 5; done
