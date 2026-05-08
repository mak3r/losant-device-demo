locals {
  prefix = "ldc-demo-${var.cluster_name}"
  common_tags = {
    "ldc-demo-cluster" = var.cluster_name
    "managed-by"       = "ldc-demo"
  }
  k3s_token         = var.k3s_token != "" ? var.k3s_token : random_id.k3s_token[0].hex
  losant_secret_arn = one(aws_secretsmanager_secret.losant_api_token[*].arn)
}

resource "random_id" "k3s_token" {
  count       = var.k3s_token == "" ? 1 : 0
  byte_length = 32
}

# ── Secrets Manager (optional) ────────────────────────────────────────────────

resource "aws_secretsmanager_secret" "losant_api_token" {
  count = var.use_secrets_manager ? 1 : 0
  name  = "ldc-demo/${var.cluster_name}/losant-api-token"
  tags  = local.common_tags
}

resource "aws_secretsmanager_secret_version" "losant_api_token" {
  count         = var.use_secrets_manager ? 1 : 0
  secret_id     = aws_secretsmanager_secret.losant_api_token[0].id
  secret_string = var.losant_api_token
}

resource "aws_iam_role" "ldc_demo" {
  count = var.use_secrets_manager ? 1 : 0
  name  = local.prefix
  tags  = local.common_tags

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect    = "Allow"
      Action    = "sts:AssumeRole"
      Principal = { Service = "ec2.amazonaws.com" }
    }]
  })
}

resource "aws_iam_role_policy" "losant_secret" {
  count = var.use_secrets_manager ? 1 : 0
  name  = "${local.prefix}-losant-secret"
  role  = aws_iam_role.ldc_demo[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Effect   = "Allow"
      Action   = "secretsmanager:GetSecretValue"
      Resource = aws_secretsmanager_secret.losant_api_token[0].arn
    }]
  })
}

resource "aws_iam_instance_profile" "ldc_demo" {
  count = var.use_secrets_manager ? 1 : 0
  name  = local.prefix
  role  = aws_iam_role.ldc_demo[0].name
  tags  = local.common_tags
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
  description = "ldc-demo HA cluster ${var.cluster_name}"
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

  # k3s embedded etcd peer communication
  ingress {
    description = "etcd peer"
    from_port   = 2379
    to_port     = 2380
    protocol    = "tcp"
    self        = true
  }

  # k3s flannel VXLAN
  ingress {
    description = "flannel VXLAN"
    from_port   = 8472
    to_port     = 8472
    protocol    = "udp"
    self        = true
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

# First server — bootstraps the embedded etcd cluster
resource "aws_instance" "server_init" {
  ami                    = data.aws_ami.suse_micro.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.ldc_demo.key_name
  vpc_security_group_ids = [aws_security_group.ldc_demo.id]
  iam_instance_profile   = one(aws_iam_instance_profile.ldc_demo[*].name)

  root_block_device {
    volume_size = var.volume_size_gb
    volume_type = "gp3"
  }

  user_data = local.losant_secret_arn != null ? templatefile("${path.module}/cloud-init-server-sm.yaml.tpl", {
    k3s_channel           = var.k3s_channel
    k3s_token             = local.k3s_token
    losant_secret_arn     = local.losant_secret_arn
    losant_application_id = var.losant_application_id
    }) : templatefile("${path.module}/cloud-init-server.yaml.tpl", {
    k3s_channel           = var.k3s_channel
    k3s_token             = local.k3s_token
    losant_api_token      = var.losant_api_token
    losant_application_id = var.losant_application_id
  })

  tags = merge(local.common_tags, {
    Name = "${local.prefix}-server-0"
  })
}

resource "aws_eip" "server_init" {
  instance = aws_instance.server_init.id
  domain   = "vpc"
  tags     = local.common_tags
}

# Servers 1 and 2 — join the cluster bootstrapped by server_init
resource "aws_instance" "server_join" {
  count                  = 2
  ami                    = data.aws_ami.suse_micro.id
  instance_type          = var.instance_type
  key_name               = aws_key_pair.ldc_demo.key_name
  vpc_security_group_ids = [aws_security_group.ldc_demo.id]

  root_block_device {
    volume_size = var.volume_size_gb
    volume_type = "gp3"
  }

  # Depend on the first server so AWS starts it first, giving it more time
  # to complete cloud-init before the joining nodes wake up.
  depends_on = [aws_eip.server_init]

  user_data = templatefile("${path.module}/cloud-init-agent.yaml.tpl", {
    k3s_channel = var.k3s_channel
    k3s_token   = local.k3s_token
    server0_ip  = aws_eip.server_init.public_ip
  })

  tags = merge(local.common_tags, {
    Name = "${local.prefix}-server-${count.index + 1}"
  })
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

  depends_on = [aws_eip.server_init]

  user_data = templatefile("${path.module}/cloud-init-worker.yaml.tpl", {
    k3s_channel = var.k3s_channel
    k3s_token   = local.k3s_token
    server_ip   = aws_eip.server_init.public_ip
  })

  tags = merge(local.common_tags, {
    Name = "${local.prefix}-worker-${count.index}"
  })
}
