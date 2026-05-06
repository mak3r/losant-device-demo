output "server_public_ip" {
  description = "Public IP of the first (init) server — use this as the API endpoint."
  value       = aws_eip.server_init.public_ip
}

output "server_ips" {
  description = "Public IPs of all server instances."
  value = concat(
    [aws_eip.server_init.public_ip],
    aws_instance.server_join[*].private_ip
  )
}

output "cluster_name" {
  description = "Cluster name."
  value       = var.cluster_name
}

output "ssh_user" {
  description = "SSH username for the server instances."
  value       = "ec2-user"
}

output "kubeconfig_remote_path" {
  description = "Path to kubeconfig on the server."
  value       = "/etc/rancher/k3s/k3s.yaml"
}
