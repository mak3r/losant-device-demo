locals {
  prefix = "ldc-demo-${var.cluster_name}"
  common_labels = {
    "ldc-demo-cluster" = var.cluster_name
    "managed-by"       = "ldc-demo"
  }
  k3s_token = var.k3s_token != "" ? var.k3s_token : random_id.k3s_token[0].hex
}

resource "random_id" "k3s_token" {
  count       = var.k3s_token == "" ? 1 : 0
  byte_length = 32
}

data "google_compute_image" "ubuntu" {
  family  = "ubuntu-2404-lts-amd64"
  project = "ubuntu-os-cloud"
}

resource "google_compute_address" "server" {
  name = "${local.prefix}-server"
}

resource "google_compute_firewall" "allow_ingress" {
  name    = "${local.prefix}-allow-ingress"
  network = "default"

  allow {
    protocol = "tcp"
    ports    = ["22", "6443"]
  }

  source_ranges = [var.allowed_cidr]
  target_tags   = ["ldc-demo-${var.cluster_name}"]
}

resource "google_compute_firewall" "allow_intra_cluster" {
  name    = "${local.prefix}-allow-intra"
  network = "default"

  allow {
    protocol = "all"
  }

  source_tags = ["ldc-demo-${var.cluster_name}"]
  target_tags = ["ldc-demo-${var.cluster_name}"]
}

resource "google_compute_firewall" "allow_egress" {
  name      = "${local.prefix}-allow-egress"
  network   = "default"
  direction = "EGRESS"

  allow {
    protocol = "all"
  }

  destination_ranges = ["0.0.0.0/0"]
  target_tags        = ["ldc-demo-${var.cluster_name}"]
}

resource "google_compute_instance" "server" {
  name         = "${local.prefix}-server"
  machine_type = var.machine_type

  boot_disk {
    initialize_params {
      image = data.google_compute_image.ubuntu.self_link
      size  = var.disk_size_gb
    }
  }

  network_interface {
    network = "default"
    access_config {
      nat_ip = google_compute_address.server.address
    }
  }

  tags   = ["ldc-demo-${var.cluster_name}"]
  labels = local.common_labels

  metadata = {
    ssh-keys = "ubuntu:${file(var.ssh_public_key_path)}"
    user-data = templatefile("${path.module}/cloud-init.yaml.tpl", {
      k3s_channel           = var.k3s_channel
      k3s_token             = local.k3s_token
      losant_api_token      = var.losant_api_token
      losant_application_id = var.losant_application_id
    })
  }
}

resource "google_compute_instance" "worker" {
  count        = var.worker_count
  name         = "${local.prefix}-worker-${count.index}"
  machine_type = var.machine_type

  boot_disk {
    initialize_params {
      image = data.google_compute_image.ubuntu.self_link
      size  = var.disk_size_gb
    }
  }

  network_interface {
    network = "default"
    access_config {}
  }

  depends_on = [google_compute_address.server]

  tags   = ["ldc-demo-${var.cluster_name}"]
  labels = local.common_labels

  metadata = {
    ssh-keys = "ubuntu:${file(var.ssh_public_key_path)}"
    user-data = templatefile("${path.module}/cloud-init-worker.yaml.tpl", {
      k3s_channel = var.k3s_channel
      k3s_token   = local.k3s_token
      server_ip   = google_compute_address.server.address
    })
  }
}
