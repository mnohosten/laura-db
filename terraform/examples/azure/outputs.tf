# Outputs for Azure example deployment

output "laura_db_endpoints" {
  description = "LauraDB connection endpoints"
  value       = module.laura_db.connection_info
}

output "public_ips" {
  description = "Public IP addresses"
  value       = module.laura_db.public_ips
}

output "load_balancer_endpoint" {
  description = "Load balancer endpoint"
  value       = module.laura_db.load_balancer_endpoint
}

output "storage_account" {
  description = "Storage account name"
  value       = module.laura_db.storage_account_name
}

output "resource_group_name" {
  description = "Resource group name"
  value       = module.laura_db.resource_group_name
}

output "azure_portal_links" {
  description = "Azure Portal links"
  value       = module.laura_db.azure_portal_links
}

output "deployment_summary" {
  description = "Deployment summary"
  value       = module.laura_db.deployment_summary
}

output "ssh_command" {
  description = "SSH command to connect to first VM"
  value       = length(module.laura_db.public_ips) > 0 ? "ssh ubuntu@${module.laura_db.public_ips[0]}" : "No public IPs available"
}

output "health_check_command" {
  description = "Command to check LauraDB health"
  value       = module.laura_db.load_balancer_endpoint != null ? "curl ${module.laura_db.load_balancer_endpoint}/_health" : (length(module.laura_db.public_ips) > 0 ? "curl http://${module.laura_db.public_ips[0]}:${var.laura_db_port}/_health" : "No endpoints available")
}
