output "server_public_ip" {
  description = "Public IP address of the k3s server node."
  value       = aws_eip.server.public_ip
}

output "cluster_name" {
  description = "Cluster name."
  value       = var.cluster_name
}

output "ssh_user" {
  description = "SSH username for the server instance."
  value       = "ec2-user"
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
