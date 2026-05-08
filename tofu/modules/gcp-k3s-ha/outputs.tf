output "server_public_ip" {
  description = "Public IP of the first (init) server — use this as the API endpoint."
  value       = google_compute_address.server.address
}

output "server_ips" {
  description = "Public IPs of all server instances."
  value = concat(
    [google_compute_address.server.address],
    google_compute_instance.server_join[*].network_interface[0].network_ip
  )
}

output "cluster_name" {
  description = "Cluster name."
  value       = var.cluster_name
}

output "ssh_user" {
  description = "SSH username for the server instances."
  value       = "ubuntu"
}

output "kubeconfig_remote_path" {
  description = "Path to kubeconfig on the server."
  value       = "/etc/rancher/k3s/k3s.yaml"
}

output "k3s_token" {
  description = "k3s cluster join token. Read this after create and store in state so workers can be added later."
  value       = local.k3s_token
  sensitive   = true
}
