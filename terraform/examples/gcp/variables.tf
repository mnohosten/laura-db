# Variables for GCP example deployment

variable "project_id" {
  description = "GCP project ID"
  type        = string
}

variable "project_name" {
  description = "Name of the project"
  type        = string
  default     = "laura-db"
}

variable "environment" {
  description = "Environment name"
  type        = string
  default     = "production"
}

variable "region" {
  description = "GCP region"
  type        = string
  default     = "us-central1"
}

variable "zones" {
  description = "List of zones"
  type        = list(string)
  default     = []  # Will use region-a, region-b, region-c
}

variable "machine_type" {
  description = "GCE machine type"
  type        = string
  default     = "e2-medium"
}

variable "instance_count" {
  description = "Number of instances"
  type        = number
  default     = 2
}

variable "disk_type" {
  description = "Persistent disk type"
  type        = string
  default     = "pd-balanced"
}

variable "disk_size_gb" {
  description = "Persistent disk size in GB"
  type        = number
  default     = 100
}

variable "kms_key_id" {
  description = "KMS key ID for encryption"
  type        = string
  default     = null
}

variable "network_cidr" {
  description = "VPC network CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "allowed_cidr_blocks" {
  description = "Allowed CIDR blocks"
  type        = list(string)
  default     = ["0.0.0.0/0"]  # Change this for production!
}

variable "assign_external_ip" {
  description = "Assign external IP addresses"
  type        = bool
  default     = true
}

variable "ssh_public_key" {
  description = "SSH public key"
  type        = string
  default     = ""  # Provide your SSH public key
}

variable "enable_load_balancer" {
  description = "Enable load balancer"
  type        = bool
  default     = true
}

variable "enable_auto_scaling" {
  description = "Enable auto-scaling"
  type        = bool
  default     = false
}

variable "min_instances" {
  description = "Minimum instances for auto-scaling"
  type        = number
  default     = 1
}

variable "max_instances" {
  description = "Maximum instances for auto-scaling"
  type        = number
  default     = 10
}

variable "enable_backups" {
  description = "Enable backups"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Backup retention days"
  type        = number
  default     = 30
}

variable "enable_monitoring" {
  description = "Enable monitoring"
  type        = bool
  default     = true
}

variable "alert_email" {
  description = "Alert email address"
  type        = string
  default     = ""
}

variable "laura_db_version" {
  description = "LauraDB version"
  type        = string
  default     = "latest"
}

variable "laura_db_port" {
  description = "LauraDB port"
  type        = number
  default     = 8080
}

variable "data_dir" {
  description = "Data directory"
  type        = string
  default     = "/var/lib/laura-db"
}

variable "log_level" {
  description = "Log level"
  type        = string
  default     = "info"
}

variable "labels" {
  description = "Additional labels"
  type        = map(string)
  default     = {}
}
