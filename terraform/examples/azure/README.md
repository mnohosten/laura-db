# LauraDB Azure Example

Complete example for deploying LauraDB on Microsoft Azure using Terraform.

## Overview

This example demonstrates:
- Multi-VM deployment with Azure Load Balancer
- Blob Storage backups with lifecycle management
- Azure Monitor and Log Analytics integration
- Customizable configuration via variables

## Prerequisites

1. **Azure Subscription** with appropriate permissions
2. **Azure CLI** configured with credentials
3. **Terraform** >= 1.0 installed

## Quick Start

### 1. Configure Azure Authentication

```bash
# Install Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Login to Azure
az login

# Set subscription
az account set --subscription "Your Subscription Name"

# Verify
az account show
```

### 2. Customize Variables

Create `terraform.tfvars`:

```hcl
project_name = "my-laura-db"
environment  = "production"
location     = "eastus"

# VMs
vm_size        = "Standard_D4s_v3"
instance_count = 3

# Networking
allowed_ip_ranges = ["10.0.0.0/8"]  # Restrict access
ssh_public_key    = "ssh-rsa AAAAB3... your-key"

# Features
enable_load_balancer      = true
enable_availability_zones = true
enable_backups            = true
enable_monitoring         = true

# Contact
alert_email = "ops@example.com"

# Tags
tags = {
  Team       = "Platform"
  CostCenter = "Engineering"
}
```

### 3. Deploy

```bash
# Initialize Terraform
terraform init

# Review plan
terraform plan

# Deploy infrastructure
terraform apply

# View outputs
terraform output
```

### 4. Access LauraDB

```bash
# Get endpoint
ENDPOINT=$(terraform output -raw load_balancer_endpoint)

# Check health
curl $ENDPOINT/_health

# Access admin console
open $ENDPOINT/admin
```

## Configuration Examples

### Development (Minimal Cost)

```hcl
# terraform.tfvars
environment  = "development"
location     = "eastus"
vm_size      = "Standard_B2s"
instance_count = 1

enable_load_balancer = false
enable_backups       = false
enable_monitoring    = true
```

**Estimated cost**: ~$30/month

### Production (High Availability)

```hcl
# terraform.tfvars
environment    = "production"
location       = "eastus"
vm_size        = "Standard_D4s_v3"
instance_count = 3

enable_load_balancer      = true
enable_availability_zones = true
enable_backups            = true
backup_retention_days     = 90
storage_replication_type  = "GRS"

enable_monitoring  = true
log_retention_days = 90

disk_type    = "Premium_LRS"
disk_size_gb = 200
```

**Estimated cost**: ~$500/month

### Auto-Scaling (Dynamic Workloads)

```hcl
# terraform.tfvars
environment = "production"
location    = "eastus"
vm_size     = "Standard_D2s_v3"

enable_auto_scaling = true
min_instances       = 2
max_instances       = 10

enable_load_balancer = true
enable_backups       = true
enable_monitoring    = true
```

**Estimated cost**: ~$200-800/month (depending on load)

## Outputs

After deployment, Terraform provides:

| Output | Description |
|--------|-------------|
| `laura_db_endpoints` | Connection information |
| `public_ips` | VM public IPs |
| `load_balancer_endpoint` | Load balancer URL |
| `storage_account` | Storage account name |
| `resource_group_name` | Resource group name |
| `azure_portal_links` | Azure Portal links |
| `ssh_command` | Command to SSH into VM |
| `health_check_command` | Command to check health |

View all outputs:

```bash
terraform output
```

## Post-Deployment

### Verify Deployment

```bash
# Check health
eval $(terraform output -raw health_check_command)

# SSH to VM
eval $(terraform output -raw ssh_command)

# Check service status
sudo systemctl status laura-db
```

### View Logs

```bash
# Azure CLI logs
RG=$(terraform output -raw resource_group_name)
az monitor activity-log list --resource-group $RG

# Or visit Azure Portal
open $(terraform output -json azure_portal_links | jq -r '.monitoring')
```

### Backup Management

```bash
# List backups
STORAGE=$(terraform output -raw storage_account)
az storage blob list \
  --account-name $STORAGE \
  --container-name laura-db-backups \
  --auth-mode login

# Download backup
az storage blob download \
  --account-name $STORAGE \
  --container-name laura-db-backups \
  --name backup-latest.tar.gz \
  --file backup-latest.tar.gz \
  --auth-mode login
```

## Updating Infrastructure

### Scale Up

```hcl
# Update terraform.tfvars
instance_count = 5  # from 3

# Apply changes
terraform apply
```

### Change VM Size

```hcl
# Update terraform.tfvars
vm_size = "Standard_D8s_v3"  # from Standard_D4s_v3

# Apply changes
terraform apply
```

### Enable Auto-Scaling

```hcl
# Update terraform.tfvars
enable_auto_scaling = true
min_instances       = 2
max_instances       = 10

# Remove fixed instance count or set to min_instances
instance_count = 2

# Apply changes
terraform apply
```

## Cleanup

```bash
# Destroy all resources
terraform destroy

# Or destroy specific resources
terraform destroy -target=module.laura_db.azurerm_linux_virtual_machine.main
```

**Warning**: This will delete all data including backups.

## Customization

### Use Existing VNet

```hcl
# In main.tf, add to module:
module "laura_db" {
  # ... other config

  create_vnet = false
  subnet_id   = "/subscriptions/.../subnets/my-subnet"
}
```

### Private VMs (No Public IPs)

```hcl
# In terraform.tfvars
assign_public_ip  = false
allowed_ip_ranges = ["10.0.0.0/8"]  # Internal only
```

### Custom LauraDB Version

```hcl
# In terraform.tfvars
laura_db_version = "v1.2.3"  # or "latest"
```

### Use Azure Key Vault for Secrets

```bash
# Create Key Vault
az keyvault create \
  --name laura-db-kv \
  --resource-group $(terraform output -raw resource_group_name) \
  --location $(terraform output -json deployment_summary | jq -r '.location')

# Store secret
az keyvault secret set \
  --vault-name laura-db-kv \
  --name db-admin-password \
  --value "YourSecurePassword"

# Grant Managed Identity access
MI_ID=$(terraform output -raw managed_identity_principal_id)
az keyvault set-policy \
  --name laura-db-kv \
  --object-id $MI_ID \
  --secret-permissions get list
```

## Troubleshooting

### Deployment Fails

```bash
# Check Terraform logs
export TF_LOG=DEBUG
terraform apply

# Verify Azure authentication
az account show

# Check subscription quotas
az vm list-usage --location eastus
```

### Can't Connect to VMs

```bash
# Check NSG rules
RG=$(terraform output -raw resource_group_name)
az network nsg rule list \
  --resource-group $RG \
  --nsg-name $(terraform output -json deployment_summary | jq -r '.project_name')-production-nsg

# Test connectivity
VM_IP=$(terraform output -json public_ips | jq -r '.[0]')
nc -zv $VM_IP 8080

# Check VM status
az vm get-instance-view \
  --resource-group $RG \
  --name $(terraform output -json vm_names | jq -r '.[0]')
```

### High Costs

```bash
# Review resources
terraform state list

# Check VM sizes
RG=$(terraform output -raw resource_group_name)
az vm list \
  --resource-group $RG \
  --query "[].{Name:name, Size:hardwareProfile.vmSize}" \
  --output table

# Storage usage
STORAGE=$(terraform output -raw storage_account)
az storage account show-usage \
  --name $STORAGE
```

## Cost Optimization

1. **Use Reserved Instances**: Save up to 72% with 1 or 3-year commitments
2. **Use Spot VMs**: Up to 90% savings for fault-tolerant workloads
3. **Right-size VMs**: Monitor usage with Azure Monitor
4. **Use Standard SSD**: Cheaper than Premium SSD for non-critical workloads
5. **Implement lifecycle policies**: Automatic tiering for old backups (included)
6. **Use Azure Hybrid Benefit**: If you have existing licenses
7. **Enable auto-shutdown**: For dev/test environments

## Security Hardening

1. **Restrict IP ranges**: Update `allowed_ip_ranges`
2. **Use private IPs**: Set `assign_public_ip = false`
3. **Use Azure Bastion**: For secure VM access
4. **Enable Azure Defender**: For threat detection
5. **Use Azure Key Vault**: For secrets management
6. **Implement Azure Policy**: For governance
7. **Enable disk encryption**: With customer-managed keys

## Monitoring and Alerting

### View Metrics in Azure Portal

```bash
# Open Azure Monitor
open $(terraform output -json azure_portal_links | jq -r '.monitoring')

# View VM metrics via CLI
RG=$(terraform output -raw resource_group_name)
VM_ID=$(terraform output -json vm_ids | jq -r '.[0]')

az monitor metrics list \
  --resource $VM_ID \
  --metric "Percentage CPU"
```

### Create Custom Alert

```hcl
# Add to your configuration
resource "azurerm_monitor_metric_alert" "disk_usage" {
  name                = "LauraDB High Disk Usage"
  resource_group_name = module.laura_db.resource_group_name
  scopes              = module.laura_db.vm_ids
  description         = "Alert when disk usage exceeds 85%"
  severity            = 2
  frequency           = "PT1M"
  window_size         = "PT5M"

  criteria {
    metric_namespace = "Microsoft.Compute/virtualMachines"
    metric_name      = "OS Disk Used Percent"
    aggregation      = "Average"
    operator         = "GreaterThan"
    threshold        = 85
  }
}
```

## Backup and Restore

### Automated Backups

Backups run automatically via cron (configured in cloud-init script).

### Manual Backup

```bash
# SSH to VM
ssh ubuntu@<vm-ip>

# Run backup manually
sudo /usr/local/bin/laura-db-backup

# Verify backup in Blob Storage
az storage blob list \
  --account-name $(terraform output -raw storage_account) \
  --container-name laura-db-backups \
  --auth-mode login
```

### Restore from Backup

```bash
# List available backups
STORAGE=$(terraform output -raw storage_account)
az storage blob list \
  --account-name $STORAGE \
  --container-name laura-db-backups \
  --auth-mode login

# Download backup
az storage blob download \
  --account-name $STORAGE \
  --container-name laura-db-backups \
  --name backup-TIMESTAMP.tar.gz \
  --file backup.tar.gz \
  --auth-mode login

# Copy to VM
VM_IP=$(terraform output -json public_ips | jq -r '.[0]')
scp backup.tar.gz ubuntu@$VM_IP:/tmp/

# SSH to VM and restore
ssh ubuntu@$VM_IP
sudo systemctl stop laura-db
sudo tar -xzf /tmp/backup.tar.gz -C /var/lib/laura-db
sudo chown -R laura-db:laura-db /var/lib/laura-db
sudo systemctl start laura-db
```

## Support

- **Module Documentation**: [../../modules/azure/README.md](../../modules/azure/README.md)
- **Azure Deployment Guide**: [../../../docs/cloud/azure/README.md](../../../docs/cloud/azure/README.md)
- **Issues**: [GitHub Issues](https://github.com/mnohosten/laura-db/issues)

## References

- [Terraform Azure Provider](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs)
- [Azure Virtual Machines Best Practices](https://docs.microsoft.com/en-us/azure/virtual-machines/best-practices)
- [Azure Blob Storage Best Practices](https://docs.microsoft.com/en-us/azure/storage/blobs/storage-blobs-introduction)
- [LauraDB Documentation](../../../README.md)

## Cost Estimate Calculator

Use the [Azure Pricing Calculator](https://azure.microsoft.com/en-us/pricing/calculator/) with these inputs:

- **Virtual Machines**: Number of VMs × VM size × hours
- **Managed Disks**: Disk size × disk type
- **Blob Storage**: Backup size × replication type
- **Bandwidth**: Outbound traffic (typically 10-50GB/month)
- **Azure Monitor**: Log ingestion (typically 5-10GB/month)

Example production deployment (~$500/month):
- 3 × Standard_D4s_v3 VMs: ~$280/month
- 3 × 200GB Premium LRS disks: ~$77/month
- Blob Storage (500GB GRS): ~$42/month
- Load Balancer: ~$22/month
- Monitoring & Logging: ~$30/month
- Bandwidth (100GB): ~$9/month
