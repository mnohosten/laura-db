# GCP Module Outputs

output "instance_ids" {
  description = "IDs of GCE instances"
  value       = var.enable_auto_scaling ? [] : google_compute_instance.laura_db[*].id
}

output "instance_names" {
  description = "Names of GCE instances"
  value       = var.enable_auto_scaling ? [] : google_compute_instance.laura_db[*].name
}

output "public_ips" {
  description = "Public IP addresses of instances"
  value = var.assign_external_ip ? (
    var.enable_auto_scaling ? [] : [
      for instance in google_compute_instance.laura_db :
      try(instance.network_interface[0].access_config[0].nat_ip, "")
    ]
  ) : []
}

output "private_ips" {
  description = "Private IP addresses of instances"
  value       = var.enable_auto_scaling ? [] : google_compute_instance.laura_db[*].network_interface[0].network_ip
}

output "load_balancer_ip" {
  description = "Load balancer IP address"
  value       = var.enable_load_balancer ? google_compute_global_address.laura_db[0].address : null
}

output "load_balancer_endpoint" {
  description = "Full endpoint URL of the load balancer"
  value       = var.enable_load_balancer ? "http://${google_compute_global_address.laura_db[0].address}:${var.laura_db_port}" : null
}

output "backup_bucket_name" {
  description = "Name of the Cloud Storage backup bucket"
  value       = var.enable_backups ? google_storage_bucket.backups[0].name : null
}

output "backup_bucket_url" {
  description = "URL of the Cloud Storage backup bucket"
  value       = var.enable_backups ? google_storage_bucket.backups[0].url : null
}

output "service_account_email" {
  description = "Email of the service account"
  value       = google_service_account.laura_db.email
}

output "network_name" {
  description = "Name of the VPC network"
  value       = var.create_network ? google_compute_network.main[0].name : var.network_name
}

output "network_id" {
  description = "ID of the VPC network"
  value       = var.create_network ? google_compute_network.main[0].id : null
}

output "subnetwork_name" {
  description = "Name of the subnetwork"
  value       = var.create_network ? google_compute_subnetwork.main[0].name : var.subnetwork_name
}

output "instance_group_manager_id" {
  description = "ID of the managed instance group"
  value       = var.enable_auto_scaling ? google_compute_region_instance_group_manager.laura_db[0].id : null
}

output "connection_info" {
  description = "Connection information for LauraDB"
  value = {
    endpoints = var.enable_load_balancer ? [
      "http://${google_compute_global_address.laura_db[0].address}:${var.laura_db_port}"
    ] : [
      for ip in (var.assign_external_ip ? [
        for instance in google_compute_instance.laura_db :
        try(instance.network_interface[0].access_config[0].nat_ip, "")
      ] : []) : "http://${ip}:${var.laura_db_port}"
    ]
    port          = var.laura_db_port
    health_check  = "/_health"
    admin_console = "/admin"
  }
}

output "monitoring_console" {
  description = "Cloud Monitoring console URL"
  value       = var.enable_monitoring ? "https://console.cloud.google.com/monitoring?project=${var.project_id}" : null
}

output "logs_explorer" {
  description = "Cloud Logging explorer URL"
  value       = var.enable_monitoring ? "https://console.cloud.google.com/logs/query?project=${var.project_id}" : null
}

output "deployment_summary" {
  description = "Summary of the deployment"
  value = {
    project_id         = var.project_id
    project_name       = var.project_name
    environment        = var.environment
    region             = var.region
    instance_count     = var.enable_auto_scaling ? "auto-scaling (${var.min_instances}-${var.max_instances})" : var.instance_count
    machine_type       = var.machine_type
    laura_db_version   = var.laura_db_version
    backups_enabled    = var.enable_backups
    monitoring_enabled = var.enable_monitoring
    load_balancer_enabled = var.enable_load_balancer
  }
}
