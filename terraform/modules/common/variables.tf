# Common variables used across all cloud providers

variable "project_name" {
  description = "Name of the project"
  type        = string

  validation {
    condition     = length(var.project_name) > 0 && length(var.project_name) <= 32
    error_message = "Project name must be between 1 and 32 characters."
  }
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
  default     = "production"

  validation {
    condition     = contains(["dev", "development", "staging", "production", "prod"], var.environment)
    error_message = "Environment must be one of: dev, development, staging, production, prod."
  }
}

variable "instance_count" {
  description = "Number of instances to create"
  type        = number
  default     = 1

  validation {
    condition     = var.instance_count >= 1 && var.instance_count <= 100
    error_message = "Instance count must be between 1 and 100."
  }
}

variable "enable_monitoring" {
  description = "Enable cloud provider monitoring"
  type        = bool
  default     = true
}

variable "enable_backups" {
  description = "Enable automated backups"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 30

  validation {
    condition     = var.backup_retention_days >= 1 && var.backup_retention_days <= 365
    error_message = "Backup retention must be between 1 and 365 days."
  }
}

variable "laura_db_version" {
  description = "LauraDB version to deploy"
  type        = string
  default     = "latest"
}

variable "laura_db_port" {
  description = "Port for LauraDB HTTP server"
  type        = number
  default     = 8080

  validation {
    condition     = var.laura_db_port >= 1024 && var.laura_db_port <= 65535
    error_message = "Port must be between 1024 and 65535."
  }
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access LauraDB"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "ssh_public_key" {
  description = "SSH public key for instance access"
  type        = string
  default     = ""
}

variable "enable_load_balancer" {
  description = "Enable load balancer for multiple instances"
  type        = bool
  default     = false
}

variable "enable_auto_scaling" {
  description = "Enable auto-scaling"
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

variable "alert_email" {
  description = "Email address for alerts"
  type        = string
  default     = ""
}

variable "data_dir" {
  description = "Directory for LauraDB data storage"
  type        = string
  default     = "/var/lib/laura-db"
}

variable "log_level" {
  description = "Log level for LauraDB (debug, info, warning, error)"
  type        = string
  default     = "info"

  validation {
    condition     = contains(["debug", "info", "warning", "error"], var.log_level)
    error_message = "Log level must be one of: debug, info, warning, error."
  }
}
