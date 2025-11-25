# Example Azure deployment of LauraDB

terraform {
  required_version = ">= 1.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

provider "azurerm" {
  features {}
}

# Deploy LauraDB on Azure
module "laura_db" {
  source = "../../modules/azure"

  # Project configuration
  project_name = var.project_name
  environment  = var.environment
  location     = var.location

  # VM configuration
  vm_size        = var.vm_size
  instance_count = var.instance_count

  # Storage
  disk_type    = var.disk_type
  disk_size_gb = var.disk_size_gb

  # Networking
  vnet_address_space = var.vnet_address_space
  allowed_ip_ranges  = var.allowed_ip_ranges
  assign_public_ip   = var.assign_public_ip
  ssh_public_key     = var.ssh_public_key

  # High availability (optional)
  enable_load_balancer      = var.enable_load_balancer
  enable_auto_scaling       = var.enable_auto_scaling
  enable_availability_zones = var.enable_availability_zones
  min_instances             = var.min_instances
  max_instances             = var.max_instances

  # Backups
  enable_backups           = var.enable_backups
  backup_retention_days    = var.backup_retention_days
  storage_replication_type = var.storage_replication_type

  # Monitoring
  enable_monitoring  = var.enable_monitoring
  log_retention_days = var.log_retention_days
  alert_email        = var.alert_email

  # LauraDB configuration
  laura_db_version = var.laura_db_version
  laura_db_port    = var.laura_db_port
  data_dir         = var.data_dir
  log_level        = var.log_level

  # Tags
  tags = var.tags
}
