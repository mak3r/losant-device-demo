#cloud-config

write_files:
  # Drop the losant-device Helm chart into k3s auto-deploy manifests.
  # k3s applies everything in this directory at startup.
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

  # Losant provisioning credentials secret — consumed by the controller.
  - path: /var/lib/rancher/k3s/server/manifests/losant-provisioning-credentials.yaml
    permissions: "0600"
    content: |
      apiVersion: v1
      kind: Secret
      metadata:
        name: losant-provisioning-credentials
        namespace: losant-system
      stringData:
        apiToken: "${losant_api_token}"
        applicationId: "${losant_application_id}"

runcmd:
  # Install k3s using the official install script.
  - curl -sfL https://get.k3s.io | INSTALL_K3S_CHANNEL=${k3s_channel} sh -
  # Wait for k3s to be ready before completing cloud-init.
  - until kubectl get nodes 2>/dev/null | grep -q Ready; do sleep 5; done
