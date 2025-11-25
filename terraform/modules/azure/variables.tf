# Azure-specific variables for LauraDB deployment

# Project configuration
variable "project_name" {
  description = "Name of the project"
  type        = string
}

variable "environment" {
  description = "Environment name (dev, staging, production)"
  type        = string
  default     = "production"
}

variable "location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "zones" {
  description = "List of availability zones (defaults to 1, 2, 3 if empty)"
  type        = list(string)
  default     = []
}

variable "enable_availability_zones" {
  description = "Deploy across availability zones"
  type        = bool
  default     = true
}

# VM configuration
variable "vm_size" {
  description = "Azure VM size"
  type        = string
  default     = "Standard_D2s_v3"
}

variable "instance_count" {
  description = "Number of VM instances"
  type        = number
  default     = 1
}

variable "ssh_public_key" {
  description = "SSH public key for VM access"
  type        = string
  default     = ""
}

# Storage configuration
variable "disk_type" {
  description = "Managed disk type (Standard_LRS, StandardSSD_LRS, Premium_LRS, UltraSSD_LRS)"
  type        = string
  default     = "Premium_LRS"

  validation {
    condition     = contains(["Standard_LRS", "StandardSSD_LRS", "Premium_LRS", "UltraSSD_LRS"], var.disk_type)
    error_message = "Disk type must be one of: Standard_LRS, StandardSSD_LRS, Premium_LRS, UltraSSD_LRS."
  }
}

variable "disk_size_gb" {
  description = "OS disk size in GB"
  type        = number
  default     = 100
}

# Network configuration
variable "create_vnet" {
  description = "Create a new Virtual Network"
  type        = bool
  default     = true
}

variable "vnet_address_space" {
  description = "Address space for Virtual Network"
  type        = list(string)
  default     = ["10.0.0.0/16"]
}

variable "subnet_id" {
  description = "Existing subnet ID (if create_vnet is false)"
  type        = string
  default     = ""
}

variable "allowed_ip_ranges" {
  description = "IP ranges allowed to access LauraDB"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "assign_public_ip" {
  description = "Assign public IP addresses to VMs"
  type        = bool
  default     = true
}

# Load balancer configuration
variable "enable_load_balancer" {
  description = "Enable Azure Load Balancer"
  type        = bool
  default     = false
}

# Auto-scaling configuration
variable "enable_auto_scaling" {
  description = "Enable Virtual Machine Scale Set with autoscaling"
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
  description = "Enable Blob Storage backups"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Number of days to retain backups"
  type        = number
  default     = 30
}

variable "storage_replication_type" {
  description = "Storage account replication type (LRS, GRS, RAGRS, ZRS)"
  type        = string
  default     = "GRS"

  validation {
    condition     = contains(["LRS", "GRS", "RAGRS", "ZRS", "GZRS", "RAGZRS"], var.storage_replication_type)
    error_message = "Storage replication type must be one of: LRS, GRS, RAGRS, ZRS, GZRS, RAGZRS."
  }
}

# Monitoring configuration
variable "enable_monitoring" {
  description = "Enable Azure Monitor"
  type        = bool
  default     = true
}

variable "log_retention_days" {
  description = "Log Analytics workspace retention in days"
  type        = number
  default     = 30
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

# Tags
variable "tags" {
  description = "Additional tags for resources"
  type        = map(string)
  default     = {}
}
