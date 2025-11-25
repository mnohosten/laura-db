# GCP Terraform Module for LauraDB

terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

# Local variables
locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_labels = merge(
    {
      project     = var.project_name
      environment = var.environment
      managed_by  = "terraform"
      application = "laura-db"
    },
    var.labels
  )

  # User data script
  user_data = templatefile("${path.module}/../common/user-data.sh", {
    project_name     = var.project_name
    environment      = var.environment
    laura_db_version = var.laura_db_version
    laura_db_port    = var.laura_db_port
    data_dir         = var.data_dir
    log_level        = var.log_level
  })

  zones = length(var.zones) > 0 ? var.zones : [
    "${var.region}-a",
    "${var.region}-b",
    "${var.region}-c"
  ]
}

# VPC Network
resource "google_compute_network" "main" {
  count = var.create_network ? 1 : 0

  name                    = "${local.name_prefix}-network"
  auto_create_subnetworks = false
  project                 = var.project_id

  lifecycle {
    prevent_destroy = false
  }
}

# Subnet
resource "google_compute_subnetwork" "main" {
  count = var.create_network ? 1 : 0

  name          = "${local.name_prefix}-subnet"
  ip_cidr_range = var.network_cidr
  region        = var.region
  network       = google_compute_network.main[0].id
  project       = var.project_id

  private_ip_google_access = true
}

# Firewall Rules
resource "google_compute_firewall" "laura_db" {
  name    = "${local.name_prefix}-allow-laura-db"
  network = var.create_network ? google_compute_network.main[0].name : var.network_name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = [tostring(var.laura_db_port)]
  }

  source_ranges = var.allowed_cidr_blocks
  target_tags   = ["${local.name_prefix}-instance"]
}

resource "google_compute_firewall" "ssh" {
  name    = "${local.name_prefix}-allow-ssh"
  network = var.create_network ? google_compute_network.main[0].name : var.network_name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = ["22"]
  }

  source_ranges = var.allowed_cidr_blocks
  target_tags   = ["${local.name_prefix}-instance"]
}

resource "google_compute_firewall" "health_check" {
  count = var.enable_load_balancer ? 1 : 0

  name    = "${local.name_prefix}-allow-health-check"
  network = var.create_network ? google_compute_network.main[0].name : var.network_name
  project = var.project_id

  allow {
    protocol = "tcp"
    ports    = [tostring(var.laura_db_port)]
  }

  source_ranges = ["35.191.0.0/16", "130.211.0.0/22"] # GCP health check ranges
  target_tags   = ["${local.name_prefix}-instance"]
}

# Service Account
resource "google_service_account" "laura_db" {
  account_id   = "${local.name_prefix}-sa"
  display_name = "LauraDB Service Account"
  project      = var.project_id
}

# IAM - Cloud Storage (for backups)
resource "google_project_iam_member" "storage_admin" {
  count = var.enable_backups ? 1 : 0

  project = var.project_id
  role    = "roles/storage.objectAdmin"
  member  = "serviceAccount:${google_service_account.laura_db.email}"
}

# IAM - Cloud Monitoring (for metrics)
resource "google_project_iam_member" "monitoring_writer" {
  count = var.enable_monitoring ? 1 : 0

  project = var.project_id
  role    = "roles/monitoring.metricWriter"
  member  = "serviceAccount:${google_service_account.laura_db.email}"
}

# IAM - Cloud Logging
resource "google_project_iam_member" "logging_writer" {
  count = var.enable_monitoring ? 1 : 0

  project = var.project_id
  role    = "roles/logging.logWriter"
  member  = "serviceAccount:${google_service_account.laura_db.email}"
}

# Instance Template
resource "google_compute_instance_template" "laura_db" {
  name_prefix  = "${local.name_prefix}-template-"
  machine_type = var.machine_type
  project      = var.project_id
  region       = var.region

  tags = ["${local.name_prefix}-instance"]

  disk {
    source_image = var.image_id != "" ? var.image_id : "projects/ubuntu-os-cloud/global/images/family/ubuntu-2204-lts"
    auto_delete  = true
    boot         = true
    disk_size_gb = var.disk_size_gb
    disk_type    = var.disk_type

    disk_encryption_key {
      kms_key_self_link = var.kms_key_id
    }
  }

  network_interface {
    network    = var.create_network ? google_compute_network.main[0].name : var.network_name
    subnetwork = var.create_network ? google_compute_subnetwork.main[0].name : var.subnetwork_name

    dynamic "access_config" {
      for_each = var.assign_external_ip ? [1] : []
      content {
        network_tier = "PREMIUM"
      }
    }
  }

  service_account {
    email  = google_service_account.laura_db.email
    scopes = ["cloud-platform"]
  }

  metadata = {
    user-data                 = local.user_data
    enable-oslogin           = "true"
    ssh-keys                 = var.ssh_public_key != "" ? "ubuntu:${var.ssh_public_key}" : null
    google-logging-enabled   = var.enable_monitoring ? "true" : "false"
    google-monitoring-enabled = var.enable_monitoring ? "true" : "false"
  }

  labels = local.common_labels

  lifecycle {
    create_before_destroy = true
  }
}

# Compute Instances (if not using MIG)
resource "google_compute_instance" "laura_db" {
  count = var.enable_auto_scaling ? 0 : var.instance_count

  name         = "${local.name_prefix}-instance-${count.index}"
  machine_type = var.machine_type
  zone         = local.zones[count.index % length(local.zones)]
  project      = var.project_id

  tags = ["${local.name_prefix}-instance"]

  boot_disk {
    initialize_params {
      image = var.image_id != "" ? var.image_id : "projects/ubuntu-os-cloud/global/images/family/ubuntu-2204-lts"
      size  = var.disk_size_gb
      type  = var.disk_type
    }

    kms_key_self_link = var.kms_key_id
  }

  network_interface {
    network    = var.create_network ? google_compute_network.main[0].name : var.network_name
    subnetwork = var.create_network ? google_compute_subnetwork.main[0].name : var.subnetwork_name

    dynamic "access_config" {
      for_each = var.assign_external_ip ? [1] : []
      content {
        network_tier = "PREMIUM"
      }
    }
  }

  service_account {
    email  = google_service_account.laura_db.email
    scopes = ["cloud-platform"]
  }

  metadata = {
    user-data                 = local.user_data
    enable-oslogin           = "true"
    ssh-keys                 = var.ssh_public_key != "" ? "ubuntu:${var.ssh_public_key}" : null
    google-logging-enabled   = var.enable_monitoring ? "true" : "false"
    google-monitoring-enabled = var.enable_monitoring ? "true" : "false"
  }

  labels = local.common_labels

  lifecycle {
    create_before_destroy = true
  }

  allow_stopping_for_update = true
}

# Managed Instance Group (for auto-scaling)
resource "google_compute_region_instance_group_manager" "laura_db" {
  count = var.enable_auto_scaling ? 1 : 0

  name               = "${local.name_prefix}-mig"
  base_instance_name = "${local.name_prefix}-instance"
  region             = var.region
  project            = var.project_id

  version {
    instance_template = google_compute_instance_template.laura_db.id
  }

  target_size = var.instance_count

  named_port {
    name = "http"
    port = var.laura_db_port
  }

  auto_healing_policies {
    health_check      = google_compute_health_check.laura_db[0].id
    initial_delay_sec = 300
  }

  update_policy {
    type                  = "PROACTIVE"
    minimal_action        = "REPLACE"
    max_surge_fixed       = 3
    max_unavailable_fixed = 0
  }

  lifecycle {
    create_before_destroy = true
  }
}

# Autoscaler
resource "google_compute_region_autoscaler" "laura_db" {
  count = var.enable_auto_scaling ? 1 : 0

  name    = "${local.name_prefix}-autoscaler"
  region  = var.region
  target  = google_compute_region_instance_group_manager.laura_db[0].id
  project = var.project_id

  autoscaling_policy {
    max_replicas    = var.max_instances
    min_replicas    = var.min_instances
    cooldown_period = 60

    cpu_utilization {
      target = 0.7
    }
  }
}

# Health Check
resource "google_compute_health_check" "laura_db" {
  count = var.enable_auto_scaling || var.enable_load_balancer ? 1 : 0

  name    = "${local.name_prefix}-health-check"
  project = var.project_id

  timeout_sec         = 5
  check_interval_sec  = 10
  healthy_threshold   = 2
  unhealthy_threshold = 3

  http_health_check {
    port         = var.laura_db_port
    request_path = "/_health"
  }
}

# Load Balancer Components
resource "google_compute_global_address" "laura_db" {
  count = var.enable_load_balancer ? 1 : 0

  name    = "${local.name_prefix}-lb-ip"
  project = var.project_id
}

resource "google_compute_backend_service" "laura_db" {
  count = var.enable_load_balancer ? 1 : 0

  name                  = "${local.name_prefix}-backend"
  protocol              = "HTTP"
  port_name             = "http"
  timeout_sec           = 30
  enable_cdn            = false
  health_checks         = [google_compute_health_check.laura_db[0].id]
  load_balancing_scheme = "EXTERNAL"
  project               = var.project_id

  dynamic "backend" {
    for_each = var.enable_auto_scaling ? [1] : []
    content {
      group           = google_compute_region_instance_group_manager.laura_db[0].instance_group
      balancing_mode  = "UTILIZATION"
      capacity_scaler = 1.0
    }
  }

  dynamic "backend" {
    for_each = var.enable_auto_scaling ? [] : google_compute_instance.laura_db[*]
    content {
      group          = google_compute_instance_group.unmanaged[backend.key].id
      balancing_mode = "UTILIZATION"
    }
  }
}

# Unmanaged instance groups (for non-MIG deployments with LB)
resource "google_compute_instance_group" "unmanaged" {
  count = var.enable_load_balancer && !var.enable_auto_scaling ? var.instance_count : 0

  name    = "${local.name_prefix}-ig-${count.index}"
  zone    = google_compute_instance.laura_db[count.index].zone
  project = var.project_id

  instances = [google_compute_instance.laura_db[count.index].id]

  named_port {
    name = "http"
    port = var.laura_db_port
  }
}

resource "google_compute_url_map" "laura_db" {
  count = var.enable_load_balancer ? 1 : 0

  name            = "${local.name_prefix}-url-map"
  default_service = google_compute_backend_service.laura_db[0].id
  project         = var.project_id
}

resource "google_compute_target_http_proxy" "laura_db" {
  count = var.enable_load_balancer ? 1 : 0

  name    = "${local.name_prefix}-http-proxy"
  url_map = google_compute_url_map.laura_db[0].id
  project = var.project_id
}

resource "google_compute_global_forwarding_rule" "laura_db" {
  count = var.enable_load_balancer ? 1 : 0

  name                  = "${local.name_prefix}-forwarding-rule"
  target                = google_compute_target_http_proxy.laura_db[0].id
  port_range            = tostring(var.laura_db_port)
  ip_address            = google_compute_global_address.laura_db[0].address
  load_balancing_scheme = "EXTERNAL"
  project               = var.project_id
}

# Cloud Storage Bucket for backups
resource "google_storage_bucket" "backups" {
  count = var.enable_backups ? 1 : 0

  name          = "${local.name_prefix}-backups-${var.project_id}"
  location      = var.region
  project       = var.project_id
  storage_class = "STANDARD"
  force_destroy = var.environment != "production"

  uniform_bucket_level_access = true

  versioning {
    enabled = true
  }

  lifecycle_rule {
    condition {
      age = var.backup_retention_days
    }
    action {
      type = "Delete"
    }
  }

  lifecycle_rule {
    condition {
      age                = 30
      matches_storage_class = ["STANDARD"]
    }
    action {
      type          = "SetStorageClass"
      storage_class = "NEARLINE"
    }
  }

  encryption {
    default_kms_key_name = var.kms_key_id
  }

  labels = local.common_labels
}

# Cloud Monitoring Alert Policy (CPU)
resource "google_monitoring_alert_policy" "high_cpu" {
  count = var.enable_monitoring && var.alert_email != "" ? 1 : 0

  display_name = "${local.name_prefix} - High CPU Usage"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "CPU usage above 80%"

    condition_threshold {
      filter          = "resource.type = \"gce_instance\" AND resource.labels.instance_id = starts_with(\"${local.name_prefix}\")"
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }

  notification_channels = var.alert_email != "" ? [google_monitoring_notification_channel.email[0].id] : []
}

# Notification Channel
resource "google_monitoring_notification_channel" "email" {
  count = var.enable_monitoring && var.alert_email != "" ? 1 : 0

  display_name = "${local.name_prefix} Email Alerts"
  type         = "email"
  project      = var.project_id

  labels = {
    email_address = var.alert_email
  }
}

# Cloud NAT (if not assigning external IPs)
resource "google_compute_router" "main" {
  count = var.create_network && !var.assign_external_ip ? 1 : 0

  name    = "${local.name_prefix}-router"
  region  = var.region
  network = google_compute_network.main[0].id
  project = var.project_id
}

resource "google_compute_router_nat" "main" {
  count = var.create_network && !var.assign_external_ip ? 1 : 0

  name                               = "${local.name_prefix}-nat"
  router                             = google_compute_router.main[0].name
  region                             = var.region
  nat_ip_allocate_option             = "AUTO_ONLY"
  source_subnetwork_ip_ranges_to_nat = "ALL_SUBNETWORKS_ALL_IP_RANGES"
  project                            = var.project_id
}
