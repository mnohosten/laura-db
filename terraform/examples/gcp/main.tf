# Example GCP deployment of LauraDB

terraform {
  required_version = ">= 1.0"

  required_providers {
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
  }
}

provider "google" {
  project = var.project_id
  region  = var.region
}

# Deploy LauraDB on GCP
module "laura_db" {
  source = "../../modules/gcp"

  # Project configuration
  project_id   = var.project_id
  project_name = var.project_name
  environment  = var.environment
  region       = var.region
  zones        = var.zones

  # Instance configuration
  machine_type   = var.machine_type
  instance_count = var.instance_count

  # Storage
  disk_type    = var.disk_type
  disk_size_gb = var.disk_size_gb
  kms_key_id   = var.kms_key_id

  # Networking
  network_cidr        = var.network_cidr
  allowed_cidr_blocks = var.allowed_cidr_blocks
  assign_external_ip  = var.assign_external_ip
  ssh_public_key      = var.ssh_public_key

  # High availability (optional)
  enable_load_balancer = var.enable_load_balancer
  enable_auto_scaling  = var.enable_auto_scaling
  min_instances        = var.min_instances
  max_instances        = var.max_instances

  # Backups
  enable_backups        = var.enable_backups
  backup_retention_days = var.backup_retention_days

  # Monitoring
  enable_monitoring = var.enable_monitoring
  alert_email       = var.alert_email

  # LauraDB configuration
  laura_db_version = var.laura_db_version
  laura_db_port    = var.laura_db_port
  data_dir         = var.data_dir
  log_level        = var.log_level

  # Labels
  labels = var.labels
}
