locals {
  prefix = "ldc-demo-${var.cluster_name}"
  common_tags = {
    "ldc-demo-cluster" = var.cluster_name
    "managed-by"       = "ldc-demo"
  }
  k3s_token = var.k3s_token != "" ? var.k3s_token : random_id.k3s_token[0].hex
}

resource "random_id" "k3s_token" {
  count       = var.k3s_token == "" ? 1 : 0
  byte_length = 32
}

data "aws_ami" "suse_micro" {
  most_recent = true
  owners      = ["013907871322"] # SUSE official AWS account

  filter {
    name   = "name"
    values = ["suse-sles-micro-*-x86_64*"]
  }
  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
  filter {
    name   = "architecture"
    values = ["x86_64"]
  }
}

resource "aws_key_pair" "ldc_demo" {
  key_name   = local.prefix
  public_key = file(var.ssh_public_key_path)
  tags       = local.common_tags
}

resource "aws_security_group" "ldc_demo" {
  name        = local.prefix
  description = "ldc-demo cluster ${var.cluster_name}"
  tags        = local.common_tags

  ingress {
    description = "SSH"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }

  ingress {
    description = "k3s API server"
    from_port   = 6443
    to_port     = 6443
    protocol    = "tcp"
    cidr_blocks = [var.allowed_cidr]
  }

  ingress {
    description = "Intra-cluster traffic (kubelet, flannel, etc.)"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    self        = true
  }

  egress {
    description = "Allow all outbound"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

resource "aws_instance" "server" {
  ami                    = data.aws_ami.suse_micro.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.ldc_demo.key_name
  vpc_security_group_ids = [aws_security_group.ldc_demo.id]

  root_block_device {
    volume_size = var.volume_size_gb
    volume_type = "gp3"
  }

  user_data = templatefile("${path.module}/cloud-init.yaml.tpl", {
    k3s_channel           = var.k3s_channel
    k3s_token             = local.k3s_token
    losant_api_token      = var.losant_api_token
    losant_application_id = var.losant_application_id
  })

  tags = merge(local.common_tags, {
    Name = local.prefix
  })
}

resource "aws_eip" "server" {
  instance = aws_instance.server.id
  domain   = "vpc"
  tags     = local.common_tags
}

resource "aws_instance" "worker" {
  count                  = var.worker_count
  ami                    = data.aws_ami.suse_micro.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.ldc_demo.key_name
  vpc_security_group_ids = [aws_security_group.ldc_demo.id]

  root_block_device {
    volume_size = var.volume_size_gb
    volume_type = "gp3"
  }

  depends_on = [aws_eip.server]

  user_data = templatefile("${path.module}/cloud-init-worker.yaml.tpl", {
    k3s_channel = var.k3s_channel
    k3s_token   = local.k3s_token
    server_ip   = aws_eip.server.public_ip
  })

  tags = merge(local.common_tags, {
    Name = "${local.prefix}-worker-${count.index}"
  })
}
