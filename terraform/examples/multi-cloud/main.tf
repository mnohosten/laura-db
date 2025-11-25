# Multi-Cloud LauraDB Deployment
# Deploy LauraDB simultaneously across AWS, GCP, and Azure

terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    google = {
      source  = "hashicorp/google"
      version = "~> 5.0"
    }
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

# Provider configurations
provider "aws" {
  region = var.aws_region
}

provider "google" {
  project = var.gcp_project_id
  region  = var.gcp_region
}

provider "azurerm" {
  features {}
}

# AWS Deployment
module "laura_db_aws" {
  source = "../../modules/aws"

  project_name = "${var.project_name}-aws"
  environment  = var.environment
  region       = var.aws_region

  # Instance configuration
  instance_type  = var.aws_instance_type
  instance_count = var.aws_instance_count

  # Storage
  volume_type = var.aws_volume_type
  volume_size = var.aws_volume_size

  # Networking
  vpc_cidr            = var.aws_vpc_cidr
  allowed_cidr_blocks = var.allowed_cidr_blocks
  ssh_public_key      = var.ssh_public_key

  # Features
  enable_load_balancer = var.enable_load_balancer
  enable_backups       = var.enable_backups
  enable_monitoring    = var.enable_monitoring

  # LauraDB
  laura_db_version = var.laura_db_version
  laura_db_port    = var.laura_db_port

  tags = merge(
    var.common_tags,
    {
      Cloud = "AWS"
    }
  )
}

# GCP Deployment
module "laura_db_gcp" {
  source = "../../modules/gcp"

  project_id   = var.gcp_project_id
  project_name = "${var.project_name}-gcp"
  environment  = var.environment
  region       = var.gcp_region

  # Instance configuration
  machine_type   = var.gcp_machine_type
  instance_count = var.gcp_instance_count

  # Storage
  disk_type    = var.gcp_disk_type
  disk_size_gb = var.gcp_disk_size_gb

  # Networking
  network_cidr        = var.gcp_network_cidr
  allowed_cidr_blocks = var.allowed_cidr_blocks
  assign_external_ip  = true
  ssh_public_key      = var.ssh_public_key

  # Features
  enable_load_balancer = var.enable_load_balancer
  enable_backups       = var.enable_backups
  enable_monitoring    = var.enable_monitoring

  # LauraDB
  laura_db_version = var.laura_db_version
  laura_db_port    = var.laura_db_port

  labels = merge(
    var.common_tags,
    {
      cloud = "gcp"
    }
  )
}

# Azure Deployment
module "laura_db_azure" {
  source = "../../modules/azure"

  project_name = "${var.project_name}-azure"
  environment  = var.environment
  location     = var.azure_location

  # VM configuration
  vm_size        = var.azure_vm_size
  instance_count = var.azure_instance_count

  # Storage
  disk_type    = var.azure_disk_type
  disk_size_gb = var.azure_disk_size_gb

  # Networking
  vnet_address_space = var.azure_vnet_address_space
  allowed_ip_ranges  = var.allowed_cidr_blocks
  assign_public_ip   = true
  ssh_public_key     = var.ssh_public_key

  # Features
  enable_load_balancer = var.enable_load_balancer
  enable_backups       = var.enable_backups
  enable_monitoring    = var.enable_monitoring

  # LauraDB
  laura_db_version = var.laura_db_version
  laura_db_port    = var.laura_db_port

  tags = merge(
    var.common_tags,
    {
      Cloud = "Azure"
    }
  )
}
