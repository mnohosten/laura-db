# Multi-Cloud Outputs

# AWS Outputs
output "aws_endpoints" {
  description = "AWS LauraDB endpoints"
  value       = module.laura_db_aws.connection_info
}

output "aws_load_balancer" {
  description = "AWS load balancer endpoint"
  value       = module.laura_db_aws.load_balancer_endpoint
}

output "aws_public_ips" {
  description = "AWS instance public IPs"
  value       = module.laura_db_aws.public_ips
}

output "aws_backup_bucket" {
  description = "AWS S3 backup bucket"
  value       = module.laura_db_aws.backup_bucket_name
}

output "aws_monitoring_dashboard" {
  description = "AWS CloudWatch dashboard"
  value       = module.laura_db_aws.monitoring_dashboard
}

# GCP Outputs
output "gcp_endpoints" {
  description = "GCP LauraDB endpoints"
  value       = module.laura_db_gcp.connection_info
}

output "gcp_load_balancer" {
  description = "GCP load balancer endpoint"
  value       = module.laura_db_gcp.load_balancer_endpoint
}

output "gcp_public_ips" {
  description = "GCP instance public IPs"
  value       = module.laura_db_gcp.public_ips
}

output "gcp_backup_bucket" {
  description = "GCP Cloud Storage backup bucket"
  value       = module.laura_db_gcp.backup_bucket_name
}

output "gcp_monitoring_console" {
  description = "GCP Cloud Monitoring console"
  value       = module.laura_db_gcp.monitoring_console
}

# Azure Outputs
output "azure_endpoints" {
  description = "Azure LauraDB endpoints"
  value       = module.laura_db_azure.connection_info
}

output "azure_load_balancer" {
  description = "Azure load balancer endpoint"
  value       = module.laura_db_azure.load_balancer_endpoint
}

output "azure_public_ips" {
  description = "Azure VM public IPs"
  value       = module.laura_db_azure.public_ips
}

output "azure_storage_account" {
  description = "Azure storage account for backups"
  value       = module.laura_db_azure.storage_account_name
}

output "azure_portal_links" {
  description = "Azure Portal links"
  value       = module.laura_db_azure.azure_portal_links
}

# Combined outputs
output "all_endpoints" {
  description = "All LauraDB endpoints across clouds"
  value = {
    aws = module.laura_db_aws.load_balancer_endpoint != null ? module.laura_db_aws.load_balancer_endpoint : (
      length(module.laura_db_aws.public_ips) > 0 ? "http://${module.laura_db_aws.public_ips[0]}:${var.laura_db_port}" : "No AWS endpoint"
    )
    gcp = module.laura_db_gcp.load_balancer_endpoint != null ? module.laura_db_gcp.load_balancer_endpoint : (
      length(module.laura_db_gcp.public_ips) > 0 ? "http://${module.laura_db_gcp.public_ips[0]}:${var.laura_db_port}" : "No GCP endpoint"
    )
    azure = module.laura_db_azure.load_balancer_endpoint != null ? module.laura_db_azure.load_balancer_endpoint : (
      length(module.laura_db_azure.public_ips) > 0 ? "http://${module.laura_db_azure.public_ips[0]}:${var.laura_db_port}" : "No Azure endpoint"
    )
  }
}

output "deployment_summary" {
  description = "Summary of multi-cloud deployment"
  value = {
    aws = module.laura_db_aws.deployment_summary
    gcp = module.laura_db_gcp.deployment_summary
    azure = module.laura_db_azure.deployment_summary
  }
}

output "health_check_commands" {
  description = "Commands to check health across all clouds"
  value = {
    aws = module.laura_db_aws.load_balancer_endpoint != null ?
      "curl ${module.laura_db_aws.load_balancer_endpoint}/_health" :
      (length(module.laura_db_aws.public_ips) > 0 ? "curl http://${module.laura_db_aws.public_ips[0]}:${var.laura_db_port}/_health" : "N/A")

    gcp = module.laura_db_gcp.load_balancer_endpoint != null ?
      "curl ${module.laura_db_gcp.load_balancer_endpoint}/_health" :
      (length(module.laura_db_gcp.public_ips) > 0 ? "curl http://${module.laura_db_gcp.public_ips[0]}:${var.laura_db_port}/_health" : "N/A")

    azure = module.laura_db_azure.load_balancer_endpoint != null ?
      "curl ${module.laura_db_azure.load_balancer_endpoint}/_health" :
      (length(module.laura_db_azure.public_ips) > 0 ? "curl http://${module.laura_db_azure.public_ips[0]}:${var.laura_db_port}/_health" : "N/A")
  }
}

output "backup_locations" {
  description = "Backup storage locations across all clouds"
  value = {
    aws_s3           = module.laura_db_aws.backup_bucket_name
    gcp_gcs          = module.laura_db_gcp.backup_bucket_name
    azure_blob       = module.laura_db_azure.storage_account_name
  }
}

output "monitoring_dashboards" {
  description = "Monitoring dashboard links for all clouds"
  value = {
    aws_cloudwatch    = module.laura_db_aws.monitoring_dashboard
    gcp_monitoring    = module.laura_db_gcp.monitoring_console
    azure_monitor     = lookup(module.laura_db_azure.azure_portal_links, "monitoring", null)
  }
}
