# Outputs for GCP example deployment

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
  description = "Cloud Storage backup bucket"
  value       = module.laura_db.backup_bucket_name
}

output "monitoring_console" {
  description = "Cloud Monitoring console"
  value       = module.laura_db.monitoring_console
}

output "logs_explorer" {
  description = "Cloud Logging explorer"
  value       = module.laura_db.logs_explorer
}

output "deployment_summary" {
  description = "Deployment summary"
  value       = module.laura_db.deployment_summary
}

output "gcloud_ssh_command" {
  description = "gcloud command to SSH to first instance"
  value       = length(module.laura_db.instance_names) > 0 ? "gcloud compute ssh ${module.laura_db.instance_names[0]} --project=${var.project_id}" : "No instances available"
}

output "health_check_command" {
  description = "Command to check LauraDB health"
  value       = module.laura_db.load_balancer_endpoint != null ? "curl ${module.laura_db.load_balancer_endpoint}/_health" : (length(module.laura_db.public_ips) > 0 ? "curl http://${module.laura_db.public_ips[0]}:${var.laura_db_port}/_health" : "No endpoints available")
}
