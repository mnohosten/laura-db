# AWS-specific variables for LauraDB deployment

# Import common variables
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
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

variable "availability_zones" {
  description = "List of availability zones"
  type        = list(string)
  default     = ["us-east-1a", "us-east-1b"]
}

# Instance configuration
variable "instance_type" {
  description = "EC2 instance type"
  type        = string
  default     = "t3.medium"
}

variable "instance_count" {
  description = "Number of EC2 instances"
  type        = number
  default     = 1
}

variable "ami_id" {
  description = "AMI ID (leave empty to use latest Ubuntu 22.04)"
  type        = string
  default     = ""
}

variable "ssh_public_key" {
  description = "SSH public key for instance access"
  type        = string
  default     = ""
}

# Storage configuration
variable "volume_type" {
  description = "EBS volume type (gp3, gp2, io1, io2)"
  type        = string
  default     = "gp3"

  validation {
    condition     = contains(["gp3", "gp2", "io1", "io2"], var.volume_type)
    error_message = "Volume type must be one of: gp3, gp2, io1, io2."
  }
}

variable "volume_size" {
  description = "EBS volume size in GB"
  type        = number
  default     = 100
}

# Network configuration
variable "create_vpc" {
  description = "Create a new VPC"
  type        = bool
  default     = true
}

variable "vpc_id" {
  description = "Existing VPC ID (if create_vpc is false)"
  type        = string
  default     = ""
}

variable "vpc_cidr" {
  description = "CIDR block for VPC"
  type        = string
  default     = "10.0.0.0/16"
}

variable "subnet_ids" {
  description = "Existing subnet IDs (if create_vpc is false)"
  type        = list(string)
  default     = []
}

variable "allowed_cidr_blocks" {
  description = "CIDR blocks allowed to access LauraDB"
  type        = list(string)
  default     = ["0.0.0.0/0"]
}

variable "enable_elastic_ips" {
  description = "Assign Elastic IPs to instances"
  type        = bool
  default     = false
}

# Load balancer configuration
variable "enable_load_balancer" {
  description = "Enable Application Load Balancer"
  type        = bool
  default     = false
}

# Auto-scaling configuration
variable "enable_auto_scaling" {
  description = "Enable Auto Scaling Group"
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
  description = "Enable S3 backups"
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
  description = "Enable CloudWatch monitoring"
  type        = bool
  default     = true
}

variable "log_retention_days" {
  description = "CloudWatch log retention in days"
  type        = number
  default     = 30
}

variable "alert_email" {
  description = "Email address for CloudWatch alerts"
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
