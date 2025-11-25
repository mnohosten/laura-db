# Variables for Azure example deployment

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

variable "location" {
  description = "Azure region"
  type        = string
  default     = "eastus"
}

variable "vm_size" {
  description = "Azure VM size"
  type        = string
  default     = "Standard_D2s_v3"
}

variable "instance_count" {
  description = "Number of instances"
  type        = number
  default     = 2
}

variable "disk_type" {
  description = "Managed disk type"
  type        = string
  default     = "Premium_LRS"
}

variable "disk_size_gb" {
  description = "Disk size in GB"
  type        = number
  default     = 100
}

variable "vnet_address_space" {
  description = "VNet address space"
  type        = list(string)
  default     = ["10.0.0.0/16"]
}

variable "allowed_ip_ranges" {
  description = "Allowed IP ranges"
  type        = list(string)
  default     = ["0.0.0.0/0"]  # Change this for production!
}

variable "assign_public_ip" {
  description = "Assign public IP addresses"
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

variable "enable_availability_zones" {
  description = "Deploy across availability zones"
  type        = bool
  default     = true
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

variable "storage_replication_type" {
  description = "Storage replication type"
  type        = string
  default     = "GRS"
}

variable "enable_monitoring" {
  description = "Enable monitoring"
  type        = bool
  default     = true
}

variable "log_retention_days" {
  description = "Log retention days"
  type        = number
  default     = 30
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

variable "tags" {
  description = "Additional tags"
  type        = map(string)
  default     = {}
}
