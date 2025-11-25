# Azure Terraform Module for LauraDB

terraform {
  required_version = ">= 1.0"

  required_providers {
    azurerm = {
      source  = "hashicorp/azurerm"
      version = "~> 3.0"
    }
  }
}

# Local variables
locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = merge(
    {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "Terraform"
      Application = "LauraDB"
    },
    var.tags
  )

  # User data script (cloud-init)
  user_data = templatefile("${path.module}/../common/user-data.sh", {
    project_name     = var.project_name
    environment      = var.environment
    laura_db_version = var.laura_db_version
    laura_db_port    = var.laura_db_port
    data_dir         = var.data_dir
    log_level        = var.log_level
  })

  zones = length(var.zones) > 0 ? var.zones : ["1", "2", "3"]
}

# Resource Group
resource "azurerm_resource_group" "main" {
  name     = "${local.name_prefix}-rg"
  location = var.location

  tags = local.common_tags
}

# Virtual Network
resource "azurerm_virtual_network" "main" {
  count = var.create_vnet ? 1 : 0

  name                = "${local.name_prefix}-vnet"
  address_space       = var.vnet_address_space
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  tags = local.common_tags
}

# Subnet
resource "azurerm_subnet" "main" {
  count = var.create_vnet ? 1 : 0

  name                 = "${local.name_prefix}-subnet"
  resource_group_name  = azurerm_resource_group.main.name
  virtual_network_name = azurerm_virtual_network.main[0].name
  address_prefixes     = [cidrsubnet(var.vnet_address_space[0], 8, 0)]
}

# Network Security Group
resource "azurerm_network_security_group" "main" {
  name                = "${local.name_prefix}-nsg"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  # Allow LauraDB port
  security_rule {
    name                       = "AllowLauraDB"
    priority                   = 100
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = tostring(var.laura_db_port)
    source_address_prefixes    = var.allowed_ip_ranges
    destination_address_prefix = "*"
  }

  # Allow SSH
  security_rule {
    name                       = "AllowSSH"
    priority                   = 110
    direction                  = "Inbound"
    access                     = "Allow"
    protocol                   = "Tcp"
    source_port_range          = "*"
    destination_port_range     = "22"
    source_address_prefixes    = var.allowed_ip_ranges
    destination_address_prefix = "*"
  }

  tags = local.common_tags
}

# User Assigned Managed Identity
resource "azurerm_user_assigned_identity" "main" {
  name                = "${local.name_prefix}-identity"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  tags = local.common_tags
}

# Storage Account for backups
resource "azurerm_storage_account" "backups" {
  count = var.enable_backups ? 1 : 0

  name                     = lower(replace("${substr(local.name_prefix, 0, min(length(local.name_prefix), 18))}bkp${substr(md5(azurerm_resource_group.main.id), 0, 5)}", "-", ""))
  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  account_tier             = "Standard"
  account_replication_type = var.storage_replication_type
  min_tls_version          = "TLS1_2"

  blob_properties {
    versioning_enabled = true

    delete_retention_policy {
      days = var.backup_retention_days
    }

    container_delete_retention_policy {
      days = 7
    }
  }

  tags = local.common_tags
}

# Storage Container for backups
resource "azurerm_storage_container" "backups" {
  count = var.enable_backups ? 1 : 0

  name                  = "laura-db-backups"
  storage_account_name  = azurerm_storage_account.backups[0].name
  container_access_type = "private"
}

# Role Assignment - Storage Blob Data Contributor
resource "azurerm_role_assignment" "storage_contributor" {
  count = var.enable_backups ? 1 : 0

  scope                = azurerm_storage_account.backups[0].id
  role_definition_name = "Storage Blob Data Contributor"
  principal_id         = azurerm_user_assigned_identity.main.principal_id
}

# Public IP addresses for VMs
resource "azurerm_public_ip" "main" {
  count = var.enable_auto_scaling ? 0 : (var.assign_public_ip ? var.instance_count : 0)

  name                = "${local.name_prefix}-pip-${count.index}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"
  zones               = var.enable_availability_zones ? local.zones : []

  tags = local.common_tags
}

# Network Interfaces
resource "azurerm_network_interface" "main" {
  count = var.enable_auto_scaling ? 0 : var.instance_count

  name                = "${local.name_prefix}-nic-${count.index}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name

  ip_configuration {
    name                          = "internal"
    subnet_id                     = var.create_vnet ? azurerm_subnet.main[0].id : var.subnet_id
    private_ip_address_allocation = "Dynamic"
    public_ip_address_id          = var.assign_public_ip ? azurerm_public_ip.main[count.index].id : null
  }

  tags = local.common_tags
}

# Network Interface - NSG Association
resource "azurerm_network_interface_security_group_association" "main" {
  count = var.enable_auto_scaling ? 0 : var.instance_count

  network_interface_id      = azurerm_network_interface.main[count.index].id
  network_security_group_id = azurerm_network_security_group.main.id
}

# Linux Virtual Machines
resource "azurerm_linux_virtual_machine" "main" {
  count = var.enable_auto_scaling ? 0 : var.instance_count

  name                = "${local.name_prefix}-vm-${count.index}"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  size                = var.vm_size
  zone                = var.enable_availability_zones ? local.zones[count.index % length(local.zones)] : null

  admin_username                  = "ubuntu"
  disable_password_authentication = true

  admin_ssh_key {
    username   = "ubuntu"
    public_key = var.ssh_public_key != "" ? var.ssh_public_key : file("~/.ssh/id_rsa.pub")
  }

  network_interface_ids = [
    azurerm_network_interface.main[count.index].id
  ]

  os_disk {
    name                 = "${local.name_prefix}-osdisk-${count.index}"
    caching              = "ReadWrite"
    storage_account_type = var.disk_type
    disk_size_gb         = var.disk_size_gb
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.main.id]
  }

  custom_data = base64encode(local.user_data)

  boot_diagnostics {
    storage_account_uri = var.enable_monitoring ? azurerm_storage_account.diagnostics[0].primary_blob_endpoint : null
  }

  tags = local.common_tags

  lifecycle {
    ignore_changes = [custom_data]
  }
}

# Load Balancer (if enabled)
resource "azurerm_public_ip" "lb" {
  count = var.enable_load_balancer ? 1 : 0

  name                = "${local.name_prefix}-lb-pip"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  allocation_method   = "Static"
  sku                 = "Standard"

  tags = local.common_tags
}

resource "azurerm_lb" "main" {
  count = var.enable_load_balancer ? 1 : 0

  name                = "${local.name_prefix}-lb"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "Standard"

  frontend_ip_configuration {
    name                 = "PublicIPAddress"
    public_ip_address_id = azurerm_public_ip.lb[0].id
  }

  tags = local.common_tags
}

resource "azurerm_lb_backend_address_pool" "main" {
  count = var.enable_load_balancer ? 1 : 0

  loadbalancer_id = azurerm_lb.main[0].id
  name            = "${local.name_prefix}-backend-pool"
}

resource "azurerm_lb_probe" "main" {
  count = var.enable_load_balancer ? 1 : 0

  loadbalancer_id = azurerm_lb.main[0].id
  name            = "laura-db-health-probe"
  protocol        = "Http"
  port            = var.laura_db_port
  request_path    = "/_health"
}

resource "azurerm_lb_rule" "main" {
  count = var.enable_load_balancer ? 1 : 0

  loadbalancer_id                = azurerm_lb.main[0].id
  name                           = "LauraDB"
  protocol                       = "Tcp"
  frontend_port                  = var.laura_db_port
  backend_port                   = var.laura_db_port
  frontend_ip_configuration_name = "PublicIPAddress"
  backend_address_pool_ids       = [azurerm_lb_backend_address_pool.main[0].id]
  probe_id                       = azurerm_lb_probe.main[0].id
  enable_tcp_reset               = true
  idle_timeout_in_minutes        = 15
}

resource "azurerm_network_interface_backend_address_pool_association" "main" {
  count = var.enable_load_balancer && !var.enable_auto_scaling ? var.instance_count : 0

  network_interface_id    = azurerm_network_interface.main[count.index].id
  ip_configuration_name   = "internal"
  backend_address_pool_id = azurerm_lb_backend_address_pool.main[0].id
}

# Virtual Machine Scale Set (for auto-scaling)
resource "azurerm_linux_virtual_machine_scale_set" "main" {
  count = var.enable_auto_scaling ? 1 : 0

  name                = "${local.name_prefix}-vmss"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = var.vm_size
  instances           = var.instance_count
  zones               = var.enable_availability_zones ? local.zones : []

  admin_username                  = "ubuntu"
  disable_password_authentication = true

  admin_ssh_key {
    username   = "ubuntu"
    public_key = var.ssh_public_key != "" ? var.ssh_public_key : file("~/.ssh/id_rsa.pub")
  }

  source_image_reference {
    publisher = "Canonical"
    offer     = "0001-com-ubuntu-server-jammy"
    sku       = "22_04-lts-gen2"
    version   = "latest"
  }

  os_disk {
    caching              = "ReadWrite"
    storage_account_type = var.disk_type
    disk_size_gb         = var.disk_size_gb
  }

  network_interface {
    name    = "primary"
    primary = true

    ip_configuration {
      name      = "internal"
      primary   = true
      subnet_id = var.create_vnet ? azurerm_subnet.main[0].id : var.subnet_id

      dynamic "public_ip_address" {
        for_each = var.assign_public_ip ? [1] : []
        content {
          name = "public-ip"
        }
      }

      load_balancer_backend_address_pool_ids = var.enable_load_balancer ? [azurerm_lb_backend_address_pool.main[0].id] : []
    }

    network_security_group_id = azurerm_network_security_group.main.id
  }

  identity {
    type         = "UserAssigned"
    identity_ids = [azurerm_user_assigned_identity.main.id]
  }

  custom_data = base64encode(local.user_data)

  upgrade_mode = "Automatic"

  health_probe_id = var.enable_load_balancer ? azurerm_lb_probe.main[0].id : null

  automatic_instance_repair {
    enabled      = true
    grace_period = "PT30M"
  }

  boot_diagnostics {
    storage_account_uri = var.enable_monitoring ? azurerm_storage_account.diagnostics[0].primary_blob_endpoint : null
  }

  tags = local.common_tags

  lifecycle {
    ignore_changes = [custom_data, instances]
  }
}

# Autoscale Settings
resource "azurerm_monitor_autoscale_setting" "main" {
  count = var.enable_auto_scaling ? 1 : 0

  name                = "${local.name_prefix}-autoscale"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  target_resource_id  = azurerm_linux_virtual_machine_scale_set.main[0].id

  profile {
    name = "AutoScale"

    capacity {
      default = var.instance_count
      minimum = var.min_instances
      maximum = var.max_instances
    }

    rule {
      metric_trigger {
        metric_name        = "Percentage CPU"
        metric_resource_id = azurerm_linux_virtual_machine_scale_set.main[0].id
        time_grain         = "PT1M"
        statistic          = "Average"
        time_window        = "PT5M"
        time_aggregation   = "Average"
        operator           = "GreaterThan"
        threshold          = 75
      }

      scale_action {
        direction = "Increase"
        type      = "ChangeCount"
        value     = "1"
        cooldown  = "PT5M"
      }
    }

    rule {
      metric_trigger {
        metric_name        = "Percentage CPU"
        metric_resource_id = azurerm_linux_virtual_machine_scale_set.main[0].id
        time_grain         = "PT1M"
        statistic          = "Average"
        time_window        = "PT5M"
        time_aggregation   = "Average"
        operator           = "LessThan"
        threshold          = 25
      }

      scale_action {
        direction = "Decrease"
        type      = "ChangeCount"
        value     = "1"
        cooldown  = "PT5M"
      }
    }
  }

  tags = local.common_tags
}

# Log Analytics Workspace (for monitoring)
resource "azurerm_log_analytics_workspace" "main" {
  count = var.enable_monitoring ? 1 : 0

  name                = "${local.name_prefix}-logs"
  location            = azurerm_resource_group.main.location
  resource_group_name = azurerm_resource_group.main.name
  sku                 = "PerGB2018"
  retention_in_days   = var.log_retention_days

  tags = local.common_tags
}

# Storage Account for diagnostics
resource "azurerm_storage_account" "diagnostics" {
  count = var.enable_monitoring ? 1 : 0

  name                     = lower(replace("${substr(local.name_prefix, 0, min(length(local.name_prefix), 18))}diag${substr(md5(azurerm_resource_group.main.id), 0, 5)}", "-", ""))
  resource_group_name      = azurerm_resource_group.main.name
  location                 = azurerm_resource_group.main.location
  account_tier             = "Standard"
  account_replication_type = "LRS"
  min_tls_version          = "TLS1_2"

  tags = local.common_tags
}

# Action Group for alerts
resource "azurerm_monitor_action_group" "main" {
  count = var.enable_monitoring && var.alert_email != "" ? 1 : 0

  name                = "${local.name_prefix}-alerts"
  resource_group_name = azurerm_resource_group.main.name
  short_name          = "lauraalert"

  email_receiver {
    name          = "sendtoadmin"
    email_address = var.alert_email
  }

  tags = local.common_tags
}

# Metric Alert - High CPU
resource "azurerm_monitor_metric_alert" "cpu" {
  count = var.enable_monitoring && var.alert_email != "" ? 1 : 0

  name                = "${local.name_prefix}-high-cpu"
  resource_group_name = azurerm_resource_group.main.name
  scopes              = var.enable_auto_scaling ? [azurerm_linux_virtual_machine_scale_set.main[0].id] : [for vm in azurerm_linux_virtual_machine.main : vm.id]
  description         = "Alert when CPU usage exceeds 80%"
  severity            = 2
  frequency           = "PT1M"
  window_size         = "PT5M"

  criteria {
    metric_namespace = "Microsoft.Compute/virtualMachines"
    metric_name      = "Percentage CPU"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 80
  }

  action {
    action_group_id = azurerm_monitor_action_group.main[0].id
  }

  tags = local.common_tags
}

# Data source for current client config
data "azurerm_client_config" "current" {}
