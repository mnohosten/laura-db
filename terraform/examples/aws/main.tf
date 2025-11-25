# Example AWS deployment of LauraDB

terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

provider "aws" {
  region = var.region
}

# Deploy LauraDB on AWS
module "laura_db" {
  source = "../../modules/aws"

  # Project configuration
  project_name = var.project_name
  environment  = var.environment
  region       = var.region

  # Instance configuration
  instance_type  = var.instance_type
  instance_count = var.instance_count

  # Storage
  volume_type = var.volume_type
  volume_size = var.volume_size

  # Networking
  vpc_cidr            = var.vpc_cidr
  allowed_cidr_blocks = var.allowed_cidr_blocks
  ssh_public_key      = var.ssh_public_key

  # High availability (optional)
  enable_load_balancer = var.enable_load_balancer
  enable_elastic_ips   = var.enable_elastic_ips

  # Backups
  enable_backups        = var.enable_backups
  backup_retention_days = var.backup_retention_days

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
