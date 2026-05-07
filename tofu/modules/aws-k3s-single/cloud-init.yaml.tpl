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
  - curl -sfL https://get.k3s.io | K3S_TOKEN="${k3s_token}" INSTALL_K3S_CHANNEL=${k3s_channel} sh -
  - until kubectl get nodes 2>/dev/null | grep -q Ready; do sleep 5; done
