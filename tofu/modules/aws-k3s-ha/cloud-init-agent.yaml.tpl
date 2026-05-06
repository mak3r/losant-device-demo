#cloud-config

runcmd:
  # Wait for the init server to be ready before joining.
  # The fixed sleep is a best-effort approach for a demo tool.
  # See docs/architecture.md for the known limitation and future improvement.
  - sleep 90
  - |
    curl -sfL https://get.k3s.io | \
      K3S_TOKEN="${k3s_token}" \
      K3S_URL="https://${server0_ip}:6443" \
      INSTALL_K3S_CHANNEL="${k3s_channel}" \
      sh -s - server
