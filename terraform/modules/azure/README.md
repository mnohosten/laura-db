# LauraDB Azure Terraform Module

Terraform module for deploying LauraDB on Microsoft Azure.

## Features

- **Virtual Machines**: Configurable VM sizes and counts
- **Virtual Machine Scale Sets**: Auto-scaling with health checks
- **Virtual Network**: Optional VNet creation or use existing
- **Load Balancing**: Azure Load Balancer for high availability
- **Blob Storage Backups**: Automated backups with lifecycle policies
- **Azure Monitor**: Comprehensive monitoring, logging, and alerting
- **Managed Identity**: Secure access to Azure services
- **Availability Zones**: Multi-zone deployment for resilience

## Usage

### Basic Deployment

```hcl
module "laura_db" {
  source = "./modules/azure"

  project_name = "laura-db"
  environment  = "production"
  location     = "eastus"

  vm_size        = "Standard_D2s_v3"
  instance_count = 2

  enable_backups    = true
  enable_monitoring = true
}
```

### Production Deployment with HA

```hcl
module "laura_db" {
  source = "./modules/azure"

  project_name = "laura-db"
  environment  = "production"
  location     = "eastus"

  # High availability
  vm_size                 = "Standard_D4s_v3"
  instance_count          = 3
  enable_availability_zones = true

  # Load balancer
  enable_load_balancer = true

  # Networking
  vnet_address_space = ["10.0.0.0/16"]
  allowed_ip_ranges  = ["10.0.0.0/8"]

  # Storage
  disk_type    = "Premium_LRS"
  disk_size_gb = 200

  # Backups
  enable_backups           = true
  backup_retention_days    = 90
  storage_replication_type = "GRS"

  # Monitoring
  enable_monitoring  = true
  log_retention_days = 90
  alert_email        = "ops@example.com"

  # Tags
  tags = {
    Team       = "Platform"
    CostCenter = "Engineering"
    Compliance = "HIPAA"
  }
}
```

### Auto-Scaling Deployment

```hcl
module "laura_db" {
  source = "./modules/azure"

  project_name = "laura-db"
  environment  = "production"
  location     = "eastus"

  # Auto-scaling configuration
  enable_auto_scaling = true
  vm_size             = "Standard_D2s_v3"
  min_instances       = 2
  max_instances       = 10

  # Load balancer required for auto-scaling
  enable_load_balancer = true

  enable_backups    = true
  enable_monitoring = true
}
```

## Requirements

| Name | Version |
|------|---------|
| terraform | >= 1.0 |
| azurerm | ~> 3.0 |

## Providers

| Name | Version |
|------|---------|
| azurerm | ~> 3.0 |

## Inputs

### Required

| Name | Description | Type |
|------|-------------|------|
| project_name | Name of the project | string |

### Optional

| Name | Description | Type | Default |
|------|-------------|------|---------|
| environment | Environment name | string | `"production"` |
| location | Azure region | string | `"eastus"` |
| vm_size | Azure VM size | string | `"Standard_D2s_v3"` |
| instance_count | Number of instances | number | `1` |
| disk_type | Managed disk type | string | `"Premium_LRS"` |
| disk_size_gb | Disk size (GB) | number | `100` |
| create_vnet | Create new VNet | bool | `true` |
| vnet_address_space | VNet address space | list(string) | `["10.0.0.0/16"]` |
| enable_load_balancer | Enable Load Balancer | bool | `false` |
| enable_auto_scaling | Enable auto-scaling | bool | `false` |
| enable_backups | Enable Blob Storage backups | bool | `true` |
| enable_monitoring | Enable Azure Monitor | bool | `true` |
| laura_db_port | LauraDB HTTP port | number | `8080` |

See [variables.tf](./variables.tf) for complete list.

## Outputs

| Name | Description |
|------|-------------|
| resource_group_name | Resource group name |
| vm_ids | Virtual machine IDs |
| vm_names | Virtual machine names |
| public_ips | Public IP addresses |
| private_ips | Private IP addresses |
| load_balancer_ip | Load balancer IP |
| load_balancer_endpoint | Full LB endpoint URL |
| storage_account_name | Storage account name |
| managed_identity_id | Managed identity ID |
| vnet_id | Virtual network ID |
| connection_info | Connection details |
| azure_portal_links | Azure Portal links |
| deployment_summary | Deployment summary |

See [outputs.tf](./outputs.tf) for complete list.

## Examples

### 1. Single VM (Development)

```hcl
module "laura_db_dev" {
  source = "./modules/azure"

  project_name = "laura-db-dev"
  environment  = "development"
  location     = "eastus"
  vm_size      = "Standard_B2s"

  enable_backups    = false
  enable_monitoring = true
}
```

### 2. Multi-VM with Load Balancer

```hcl
module "laura_db_prod" {
  source = "./modules/azure"

  project_name   = "laura-db-prod"
  environment    = "production"
  location       = "eastus"
  instance_count = 3
  vm_size        = "Standard_D4s_v3"

  enable_load_balancer = true
  enable_backups       = true
  enable_monitoring    = true
}
```

### 3. Using Existing VNet

```hcl
module "laura_db" {
  source = "./modules/azure"

  project_name = "laura-db"
  location     = "eastus"

  # Use existing VNet
  create_vnet = false
  subnet_id   = "/subscriptions/.../subnets/my-subnet"

  instance_count = 2
}
```

### 4. Private VMs (No Public IPs)

```hcl
module "laura_db" {
  source = "./modules/azure"

  project_name = "laura-db"
  location     = "eastus"

  # No public IPs
  assign_public_ip = false

  # Restrict access
  allowed_ip_ranges = ["10.0.0.0/8"]

  # SSH key
  ssh_public_key = file("~/.ssh/id_rsa.pub")

  instance_count = 2
}
```

## Post-Deployment

### Connect to LauraDB

```bash
# Get endpoint from outputs
terraform output connection_info

# Using load balancer
curl http://$(terraform output -raw load_balancer_ip):8080/_health

# Direct VM connection
curl http://$(terraform output -json public_ips | jq -r '.[0]'):8080/_health
```

### SSH Access

```bash
# Get public IP
VM_IP=$(terraform output -json public_ips | jq -r '.[0]')

# SSH into VM
ssh ubuntu@$VM_IP

# Check LauraDB status
sudo systemctl status laura-db
```

### View Logs

```bash
# Azure CLI logs
az monitor activity-log list --resource-group $(terraform output -raw resource_group_name)

# Or visit Azure Portal
open $(terraform output -json azure_portal_links | jq -r '.monitoring')
```

### Backup Management

```bash
# List backups
az storage blob list \
  --account-name $(terraform output -raw storage_account_name) \
  --container-name laura-db-backups

# Download backup
az storage blob download \
  --account-name $(terraform output -raw storage_account_name) \
  --container-name laura-db-backups \
  --name backup.tar.gz \
  --file backup.tar.gz

# Upload backup
az storage blob upload \
  --account-name $(terraform output -raw storage_account_name) \
  --container-name laura-db-backups \
  --name backup.tar.gz \
  --file backup.tar.gz
```

## Monitoring

### Azure Monitor Metrics

The module automatically publishes metrics to Azure Monitor:

- CPU Utilization
- Memory Usage
- Disk I/O
- Network Traffic
- LauraDB-specific metrics

### Alerts

Configure additional metric alerts:

```hcl
resource "azurerm_monitor_metric_alert" "memory" {
  name                = "${module.laura_db.deployment_summary.project_name}-high-memory"
  resource_group_name = module.laura_db.resource_group_name
  scopes              = module.laura_db.vm_ids
  description         = "Alert when memory usage exceeds 80%"
  severity            = 2
  frequency           = "PT1M"
  window_size         = "PT5M"

  criteria {
    metric_namespace = "Microsoft.Compute/virtualMachines"
    metric_name      = "Available Memory Bytes"
    aggregation      = "Average"
    operator         = "LessThan"
    threshold        = 858993459  # 20% of 4GB
  }

  action {
    action_group_id = azurerm_monitor_action_group.main.id
  }
}
```

## Backup and Recovery

### Manual Backup

```bash
# SSH to VM
ssh ubuntu@<vm-ip>

# Run backup
sudo /usr/local/bin/laura-db-backup

# Backups are automatically uploaded to Blob Storage
```

### Restore from Backup

```bash
# Download backup
az storage blob download \
  --account-name <storage-account> \
  --container-name laura-db-backups \
  --name backup.tar.gz \
  --file backup.tar.gz

# Copy to VM
scp backup.tar.gz ubuntu@<vm-ip>:/tmp/

# On VM
sudo systemctl stop laura-db
sudo tar -xzf /tmp/backup.tar.gz -C /var/lib/laura-db
sudo systemctl start laura-db
```

## Scaling

### Vertical Scaling (Resize VM)

```hcl
# Update vm_size in your config
vm_size = "Standard_D8s_v3"  # from Standard_D4s_v3

# Apply changes
terraform apply
```

### Horizontal Scaling (Add VMs)

```hcl
# Increase instance count
instance_count = 5  # from 3

# Enable load balancer if not already
enable_load_balancer = true

# Apply changes
terraform apply
```

## Cost Optimization

### 1. Use Reserved Instances

Purchase 1 or 3-year reservations for predictable workloads (up to 72% savings).

### 2. Use Spot VMs

For fault-tolerant workloads (up to 90% savings):

```hcl
# In VMSS configuration
resource "azurerm_linux_virtual_machine_scale_set" "spot" {
  priority        = "Spot"
  eviction_policy = "Deallocate"
  max_bid_price   = -1  # Pay up to regular price
}
```

### 3. Right-Size VMs

Monitor Azure Monitor metrics and adjust `vm_size` based on actual usage.

### 4. Use Azure Hybrid Benefit

If you have Windows Server or SQL Server licenses:

```hcl
resource "azurerm_linux_virtual_machine" "main" {
  license_type = "None"  # or "Windows_Server" for Windows VMs
}
```

### 5. Optimize Storage

```hcl
# Use Standard SSD instead of Premium SSD for non-critical workloads
disk_type = "StandardSSD_LRS"  # ~50% cheaper than Premium_LRS
```

## Security Best Practices

1. **Restrict IP ranges**: Don't use `0.0.0.0/0` in production
2. **Use private IPs**: Set `assign_public_ip = false` and use Azure Bastion
3. **Enable disk encryption**: Use Azure Disk Encryption with Key Vault
4. **Managed Identity**: Module uses Managed Identity (no credentials in code)
5. **Use Azure Key Vault**: Store sensitive configuration
6. **Enable Azure Security Center**: For threat detection
7. **Implement NSG rules**: Restrict traffic to necessary ports only

## Troubleshooting

### VM Won't Start

```bash
# Check VM status
az vm get-instance-view \
  --name <vm-name> \
  --resource-group <resource-group>

# View boot diagnostics
az vm boot-diagnostics get-boot-log \
  --name <vm-name> \
  --resource-group <resource-group>

# Check cloud-init log
ssh ubuntu@<vm-ip> sudo cat /var/log/cloud-init-output.log
```

### Can't Connect

```bash
# Check NSG rules
az network nsg show \
  --name <nsg-name> \
  --resource-group <resource-group>

# Verify port is open
nc -zv <vm-ip> 8080

# Check service status
ssh ubuntu@<vm-ip> sudo systemctl status laura-db
```

### High Costs

```bash
# Review resources
terraform state list

# Check VM sizes
az vm list \
  --resource-group <resource-group> \
  --query "[].{Name:name, Size:hardwareProfile.vmSize}"

# Review storage usage
az storage account show-usage \
  --name <storage-account>
```

## Migration

### From Manual Setup

```bash
# Import existing resources
terraform import module.laura_db.azurerm_resource_group.main /subscriptions/.../resourceGroups/laura-db-rg
terraform import module.laura_db.azurerm_linux_virtual_machine.main[0] /subscriptions/.../virtualMachines/laura-db-vm-0

# Run plan to see differences
terraform plan
```

## References

- [Azure VM Documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/)
- [Azure Blob Storage Documentation](https://docs.microsoft.com/en-us/azure/storage/blobs/)
- [Azure Monitor Documentation](https://docs.microsoft.com/en-us/azure/azure-monitor/)
- [LauraDB Documentation](../../../README.md)
- [LauraDB Azure Deployment Guide](../../../docs/cloud/azure/)

## Support

For issues or questions:

- [GitHub Issues](https://github.com/mnohosten/laura-db/issues)
- [Azure Deployment Guide](../../../docs/cloud/azure/README.md)

## License

Same as LauraDB project.
