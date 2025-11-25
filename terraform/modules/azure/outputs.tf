# Azure Module Outputs

output "resource_group_name" {
  description = "Name of the resource group"
  value       = azurerm_resource_group.main.name
}

output "resource_group_id" {
  description = "ID of the resource group"
  value       = azurerm_resource_group.main.id
}

output "vm_ids" {
  description = "IDs of virtual machines"
  value       = var.enable_auto_scaling ? [] : azurerm_linux_virtual_machine.main[*].id
}

output "vm_names" {
  description = "Names of virtual machines"
  value       = var.enable_auto_scaling ? [] : azurerm_linux_virtual_machine.main[*].name
}

output "public_ips" {
  description = "Public IP addresses"
  value       = var.assign_public_ip && !var.enable_auto_scaling ? azurerm_public_ip.main[*].ip_address : []
}

output "private_ips" {
  description = "Private IP addresses"
  value       = var.enable_auto_scaling ? [] : azurerm_network_interface.main[*].private_ip_address
}

output "load_balancer_ip" {
  description = "Load balancer public IP address"
  value       = var.enable_load_balancer ? azurerm_public_ip.lb[0].ip_address : null
}

output "load_balancer_endpoint" {
  description = "Full endpoint URL of the load balancer"
  value       = var.enable_load_balancer ? "http://${azurerm_public_ip.lb[0].ip_address}:${var.laura_db_port}" : null
}

output "storage_account_name" {
  description = "Name of the backup storage account"
  value       = var.enable_backups ? azurerm_storage_account.backups[0].name : null
}

output "storage_container_name" {
  description = "Name of the backup storage container"
  value       = var.enable_backups ? azurerm_storage_container.backups[0].name : null
}

output "managed_identity_id" {
  description = "ID of the managed identity"
  value       = azurerm_user_assigned_identity.main.id
}

output "managed_identity_client_id" {
  description = "Client ID of the managed identity"
  value       = azurerm_user_assigned_identity.main.client_id
}

output "managed_identity_principal_id" {
  description = "Principal ID of the managed identity"
  value       = azurerm_user_assigned_identity.main.principal_id
}

output "network_security_group_id" {
  description = "ID of the network security group"
  value       = azurerm_network_security_group.main.id
}

output "vnet_id" {
  description = "ID of the virtual network"
  value       = var.create_vnet ? azurerm_virtual_network.main[0].id : null
}

output "vnet_name" {
  description = "Name of the virtual network"
  value       = var.create_vnet ? azurerm_virtual_network.main[0].name : null
}

output "subnet_id" {
  description = "ID of the subnet"
  value       = var.create_vnet ? azurerm_subnet.main[0].id : var.subnet_id
}

output "vmss_id" {
  description = "ID of the virtual machine scale set"
  value       = var.enable_auto_scaling ? azurerm_linux_virtual_machine_scale_set.main[0].id : null
}

output "log_analytics_workspace_id" {
  description = "ID of the Log Analytics workspace"
  value       = var.enable_monitoring ? azurerm_log_analytics_workspace.main[0].id : null
}

output "log_analytics_workspace_name" {
  description = "Name of the Log Analytics workspace"
  value       = var.enable_monitoring ? azurerm_log_analytics_workspace.main[0].name : null
}

output "connection_info" {
  description = "Connection information for LauraDB"
  value = {
    endpoints = var.enable_load_balancer ? [
      "http://${azurerm_public_ip.lb[0].ip_address}:${var.laura_db_port}"
    ] : (var.assign_public_ip && !var.enable_auto_scaling ? [
      for ip in azurerm_public_ip.main[*].ip_address :
      "http://${ip}:${var.laura_db_port}"
    ] : [])
    port          = var.laura_db_port
    health_check  = "/_health"
    admin_console = "/admin"
  }
}

output "azure_portal_links" {
  description = "Azure Portal links"
  value = {
    resource_group = "https://portal.azure.com/#@/resource${azurerm_resource_group.main.id}"
    monitoring     = var.enable_monitoring ? "https://portal.azure.com/#@/resource${azurerm_log_analytics_workspace.main[0].id}" : null
    storage        = var.enable_backups ? "https://portal.azure.com/#@/resource${azurerm_storage_account.backups[0].id}" : null
  }
}

output "deployment_summary" {
  description = "Summary of the deployment"
  value = {
    project_name       = var.project_name
    environment        = var.environment
    location           = var.location
    instance_count     = var.enable_auto_scaling ? "auto-scaling (${var.min_instances}-${var.max_instances})" : var.instance_count
    vm_size            = var.vm_size
    laura_db_version   = var.laura_db_version
    backups_enabled    = var.enable_backups
    monitoring_enabled = var.enable_monitoring
    load_balancer_enabled = var.enable_load_balancer
    availability_zones = var.enable_availability_zones
  }
}
