# Multi-Cloud Variables

# Common variables
variable "project_name" {
  description = "Base project name (will be suffixed with cloud provider)"
  type        = string
  default     = "laura-db"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "ssh_public_key" {
  description = "SSH public key for all clouds"
  type        = string
  default     = ""
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access LauraDB across all clouds"
  type        = list(string)
  default     = ["0.0.0.0/0"]  # Change for production!
}

# Features (applied to all clouds)
variable "enable_load_balancer" {
  description = "Enable load balancer in all clouds"
  type        = bool
  default     = true
}

variable "enable_backups" {
  description = "Enable backups in all clouds"
  type        = bool
  default     = true
}

variable "enable_monitoring" {
  description = "Enable monitoring in all clouds"
  type        = bool
  default     = true
}

# LauraDB configuration (consistent across all clouds)
variable "laura_db_version" {
  description = "LauraDB version"
  type        = string
  default     = "latest"
}

variable "laura_db_port" {
  description = "LauraDB HTTP port"
  type        = number
  default     = 8080
}

variable "common_tags" {
  description = "Common tags/labels for all resources"
  type        = map(string)
  default = {
    Project   = "LauraDB"
    ManagedBy = "Terraform"
    MultiCloud = "true"
  }
}

# AWS-specific variables
variable "aws_region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "aws_instance_type" {
  description = "AWS EC2 instance type"
  type        = string
  default     = "t3.medium"
}

variable "aws_instance_count" {
  description = "Number of AWS instances"
  type        = number
  default     = 2
}

variable "aws_volume_type" {
  description = "AWS EBS volume type"
  type        = string
  default     = "gp3"
}

variable "aws_volume_size" {
  description = "AWS EBS volume size (GB)"
  type        = number
  default     = 100
}

variable "aws_vpc_cidr" {
  description = "AWS VPC CIDR block"
  type        = string
  default     = "10.1.0.0/16"
}

# GCP-specific variables
variable "gcp_project_id" {
  description = "GCP project ID"
  type        = string
}

variable "gcp_region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "gcp_machine_type" {
  description = "GCP machine type"
  type        = string
  default     = "e2-medium"
}

variable "gcp_instance_count" {
  description = "Number of GCP instances"
  type        = number
  default     = 2
}

variable "gcp_disk_type" {
  description = "GCP persistent disk type"
  type        = string
  default     = "pd-balanced"
}

variable "gcp_disk_size_gb" {
  description = "GCP disk size (GB)"
  type        = number
  default     = 100
}

variable "gcp_network_cidr" {
  description = "GCP network CIDR block"
  type        = string
  default     = "10.2.0.0/16"
}

# Azure-specific variables
variable "azure_location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "azure_vm_size" {
  description = "Azure VM size"
  type        = string
  default     = "Standard_D2s_v3"
}

variable "azure_instance_count" {
  description = "Number of Azure VMs"
  type        = number
  default     = 2
}

variable "azure_disk_type" {
  description = "Azure managed disk type"
  type        = string
  default     = "Premium_LRS"
}

variable "azure_disk_size_gb" {
  description = "Azure disk size (GB)"
  type        = number
  default     = 100
}

variable "azure_vnet_address_space" {
  description = "Azure VNet address space"
  type        = list(string)
  default     = ["10.3.0.0/16"]
}
