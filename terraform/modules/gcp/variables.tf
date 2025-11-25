# GCP-specific variables for LauraDB deployment

# Project configuration
variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "project_name" {
  description = "Name of the project"
  type        = string
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
  default     = "production"
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "zones" {
  description = "List of zones (defaults to region-a, region-b, region-c if empty)"
  type        = list(string)
  default     = []
}

# Instance configuration
variable "machine_type" {
  description = "GCE machine type"
  type        = string
  default     = "e2-medium"
}

variable "instance_count" {
  description = "Number of instances"
  type        = number
  default     = 1
}

variable "image_id" {
  description = "Image ID (leave empty to use latest Ubuntu 22.04)"
  type        = string
  default     = ""
}

variable "ssh_public_key" {
  description = "SSH public key for instance access"
  type        = string
  default     = ""
}

# Storage configuration
variable "disk_type" {
  description = "Persistent disk type (pd-standard, pd-balanced, pd-ssd, pd-extreme)"
  type        = string
  default     = "pd-balanced"

  validation {
    condition     = contains(["pd-standard", "pd-balanced", "pd-ssd", "pd-extreme"], var.disk_type)
    error_message = "Disk type must be one of: pd-standard, pd-balanced, pd-ssd, pd-extreme."
  }
}

variable "disk_size_gb" {
  description = "Persistent disk size in GB"
  type        = number
  default     = 100
}

variable "kms_key_id" {
  description = "KMS key ID for disk encryption (optional)"
  type        = string
  default     = null
}

# Network configuration
variable "create_network" {
  description = "Create a new VPC network"
  type        = bool
  default     = true
}

variable "network_name" {
  description = "Existing network name (if create_network is false)"
  type        = string
  default     = "default"
}

variable "subnetwork_name" {
  description = "Existing subnetwork name (if create_network is false)"
  type        = string
  default     = "default"
}

variable "network_cidr" {
  description = "CIDR block for VPC network"
  type        = string
  default     = "10.0.0.0/16"
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access LauraDB"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "assign_external_ip" {
  description = "Assign external IP addresses to instances"
  type        = bool
  default     = true
}

# Load balancer configuration
variable "enable_load_balancer" {
  description = "Enable Cloud Load Balancer"
  type        = bool
  default     = false
}

# Auto-scaling configuration
variable "enable_auto_scaling" {
  description = "Enable Managed Instance Group with autoscaling"
  type        = bool
  default     = false
}

variable "min_instances" {
  description = "Minimum number of instances for auto-scaling"
  type        = number
  default     = 1
}

variable "max_instances" {
  description = "Maximum number of instances for auto-scaling"
  type        = number
  default     = 10
}

# Backup configuration
variable "enable_backups" {
  description = "Enable Cloud Storage backups"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 30
}

# Monitoring configuration
variable "enable_monitoring" {
  description = "Enable Cloud Monitoring"
  type        = bool
  default     = true
}

variable "alert_email" {
  description = "Email address for alerts"
  type        = string
  default     = ""
}

# LauraDB configuration
variable "laura_db_version" {
  description = "LauraDB version to deploy"
  type        = string
  default     = "latest"
}

variable "laura_db_port" {
  description = "Port for LauraDB HTTP server"
  type        = number
  default     = 8080
}

variable "data_dir" {
  description = "Directory for LauraDB data storage"
  type        = string
  default     = "/var/lib/laura-db"
}

variable "log_level" {
  description = "Log level for LauraDB"
  type        = string
  default     = "info"

  validation {
    condition     = contains(["debug", "info", "warning", "error"], var.log_level)
    error_message = "Log level must be one of: debug, info, warning, error."
  }
}

# Labels
variable "labels" {
  description = "Additional labels for resources"
  type        = map(string)
  default     = {}
}
