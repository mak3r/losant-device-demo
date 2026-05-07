variable "cluster_name" {
  type        = string
  description = "Unique name for this cluster. Used in resource names and tags."
}

variable "aws_region" {
  type        = string
  description = "AWS region to deploy into."
  default     = "us-east-1"
}

variable "instance_type" {
  type        = string
  description = "EC2 instance type."
  default     = "t3.medium"
}

variable "volume_size_gb" {
  type        = number
  description = "Root EBS volume size in GB."
  default     = 40
}

variable "ssh_public_key_path" {
  type        = string
  description = "Path to the SSH public key to install on the instance."
}

variable "k3s_channel" {
  type        = string
  description = "k3s release channel (stable, latest, or a version like v1.29)."
  default     = "stable"
}

variable "k3s_token" {
  type        = string
  description = "Shared secret for k3s cluster formation. Generated automatically if empty."
  sensitive   = true
  default     = ""
}

variable "losant_api_token" {
  type        = string
  description = "Losant API token for the device controller. Injected via -var at apply time."
  sensitive   = true
}

variable "losant_application_id" {
  type        = string
  description = "Losant Application ID."
}

variable "worker_count" {
  type        = number
  description = "Number of worker (agent) nodes to add. Pass a higher value to scale out."
  default     = 0
}

variable "allowed_cidr" {
  type        = string
  description = "CIDR to allow inbound SSH and k3s API access. Defaults to 0.0.0.0/0 for demo use."
  default     = "0.0.0.0/0"
}
