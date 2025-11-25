# Outputs for AWS example deployment

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

output "backup_bucket" {
  description = "S3 backup bucket"
  value       = module.laura_db.backup_bucket_name
}

output "monitoring_dashboard" {
  description = "CloudWatch monitoring dashboard"
  value       = module.laura_db.monitoring_dashboard
}

output "deployment_summary" {
  description = "Deployment summary"
  value       = module.laura_db.deployment_summary
}

output "ssh_command" {
  description = "SSH command to connect to first instance"
  value       = length(module.laura_db.public_ips) > 0 ? "ssh ubuntu@${module.laura_db.public_ips[0]}" : "No public IPs available"
}

output "health_check_command" {
  description = "Command to check LauraDB health"
  value       = module.laura_db.load_balancer_endpoint != null ? "curl ${module.laura_db.load_balancer_endpoint}/_health" : (length(module.laura_db.public_ips) > 0 ? "curl http://${module.laura_db.public_ips[0]}:${var.laura_db_port}/_health" : "No endpoints available")
}
