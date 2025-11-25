# Variables for AWS example deployment

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
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.medium"
}

variable "instance_count" {
  description = "Number of instances"
  type        = number
  default     = 2
}

variable "volume_type" {
  description = "EBS volume type"
  type        = string
  default     = "gp3"
}

variable "volume_size" {
  description = "EBS volume size in GB"
  type        = number
  default     = 100
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  type        = string
  default     = "10.0.0.0/16"
}

variable "allowed_cidr_blocks" {
  description = "Allowed CIDR blocks"
  type        = list(string)
  default     = ["0.0.0.0/0"]  # Change this for production!
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

variable "enable_elastic_ips" {
  description = "Enable Elastic IPs"
  type        = bool
  default     = false
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
