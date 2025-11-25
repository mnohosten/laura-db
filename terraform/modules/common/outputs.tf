# Common outputs structure
# Each cloud module should provide these outputs for consistency

output "common_outputs" {
  description = "Common output structure for all cloud providers"
  value = {
    # Computed instance IDs should be provided by each cloud module
    instance_ids_description = "List of instance IDs created"

    # Public IP addresses
    public_ips_description = "List of public IP addresses"

    # Private IP addresses
    private_ips_description = "List of private IP addresses"

    # Load balancer endpoint (if enabled)
    load_balancer_endpoint_description = "Load balancer endpoint (DNS or IP)"

    # Storage bucket/container name
    backup_storage_description = "Backup storage bucket/container name"

    # Monitoring dashboard URL
    monitoring_dashboard_description = "URL to monitoring dashboard"

    # Connection information
    connection_info_description = "Connection string and endpoints for LauraDB"
  }
}

# Helper locals for generating names and tags
locals {
  # Name prefix for resources
  name_prefix = "${var.project_name}-${var.environment}"

  # Common tags/labels that all resources should have
  common_tags = {
    Project     = var.project_name
    Environment = var.environment
    ManagedBy   = "Terraform"
    Application = "LauraDB"
  }

  # Resource naming convention
  # Format: {project}-{environment}-{resource_type}-{suffix}
  resource_name_format = "${local.name_prefix}-%s"
}
