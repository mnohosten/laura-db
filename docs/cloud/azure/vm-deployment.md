# LauraDB Deployment on Azure Virtual Machines

This guide provides detailed instructions for deploying LauraDB on Azure Virtual Machines.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture Options](#architecture-options)
- [Single VM Deployment](#single-vm-deployment)
- [Multi-VM with Load Balancer](#multi-vm-with-load-balancer)
- [Virtual Machine Scale Sets](#virtual-machine-scale-sets)
- [Storage Configuration](#storage-configuration)
- [Network Configuration](#network-configuration)
- [Security Best Practices](#security-best-practices)
- [Monitoring and Logging](#monitoring-and-logging)
- [Backup and Recovery](#backup-and-recovery)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

Azure VMs provide flexible compute instances for running LauraDB with:
- **Flexible VM sizes**: Wide range of CPU/memory configurations
- **Spot VMs**: Up to 90% cost savings
- **Availability Zones**: Multi-zone high availability
- **Managed Disks**: Durable block storage
- **Azure Files**: Shared file storage
- **Proximity Placement Groups**: Low-latency deployments

## Prerequisites

### Required Tools

```bash
# Install Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Login to Azure
az login

# Set default subscription
az account set --subscription "SUBSCRIPTION_ID"

# Set default location
az configure --defaults location=eastus
```

### Required Permissions

- `Contributor` role on subscription or resource group
- `User Access Administrator` for role assignments

```bash
# Grant contributor role
az role assignment create \
  --assignee user@example.com \
  --role Contributor \
  --scope /subscriptions/SUBSCRIPTION_ID
```

## Architecture Options

### Option 1: Single VM (Development)

```
┌─────────────────────────┐
│     Public IP Address    │
└──────────┬──────────────┘
           │
    ┌──────▼─────┐
    │   VNet     │
    │            │
    │  ┌──────┐  │
    │  │  VM  │  │
    │  │Laura │  │
    │  │  DB  │  │
    │  │+Disk │  │
    │  └──────┘  │
    └────────────┘
```

**Specs**:
- VM Size: Standard_B2s (2 vCPU, 4 GB RAM)
- OS Disk: 30 GB Premium SSD
- Data Disk: 50 GB Premium SSD
- Estimated Cost: ~$35/month

### Option 2: Multi-VM with Load Balancer

```
┌──────────────────────────┐
│   Azure Load Balancer     │
└────────┬─────────┬────────┘
         │         │
    ┌────▼──┐  ┌──▼────┐
    │Zone 1 │  │Zone 2 │
    │  VM   │  │  VM   │
    │ +Disk │  │ +Disk │
    └───┬───┘  └───┬───┘
        │          │
        └────┬─────┘
         ┌───▼───┐
         │Azure  │
         │Files  │
         └───────┘
```

**Specs**:
- VM Size: Standard_D2s_v3 (2 vCPU, 8 GB RAM)
- Load Balancer: Standard tier
- Storage: Premium SSD + Azure Files Premium
- Estimated Cost: ~$250/month

## Single VM Deployment

### Step 1: Create Resource Group

```bash
# Create resource group
az group create \
  --name laura-db-rg \
  --location eastus \
  --tags application=laura-db environment=production
```

### Step 2: Create Virtual Network

```bash
# Create VNet
az network vnet create \
  --resource-group laura-db-rg \
  --name laura-db-vnet \
  --address-prefix 10.0.0.0/16 \
  --subnet-name laura-db-subnet \
  --subnet-prefix 10.0.1.0/24

# Create Network Security Group
az network nsg create \
  --resource-group laura-db-rg \
  --name laura-db-nsg

# Allow SSH
az network nsg rule create \
  --resource-group laura-db-rg \
  --nsg-name laura-db-nsg \
  --name allow-ssh \
  --priority 1000 \
  --source-address-prefixes Internet \
  --destination-port-ranges 22 \
  --access Allow \
  --protocol Tcp

# Allow LauraDB HTTP API
az network nsg rule create \
  --resource-group laura-db-rg \
  --nsg-name laura-db-nsg \
  --name allow-laura-db \
  --priority 1001 \
  --source-address-prefixes Internet \
  --destination-port-ranges 8080 \
  --access Allow \
  --protocol Tcp
```

### Step 3: Create Data Disk

```bash
# Create managed disk for data
az disk create \
  --resource-group laura-db-rg \
  --name laura-db-data-disk \
  --size-gb 50 \
  --sku Premium_LRS \
  --zone 1
```

### Step 4: Create Cloud-Init Configuration

Create `cloud-init.yml`:

```yaml
#cloud-config
package_update: true
package_upgrade: true

packages:
  - wget
  - git
  - jq

write_files:
  - path: /etc/systemd/system/laura-db.service
    content: |
      [Unit]
      Description=LauraDB Server
      After=network.target

      [Service]
      Type=simple
      User=laura-db
      Group=laura-db
      WorkingDirectory=/opt/laura-db
      ExecStart=/opt/laura-db/bin/laura-server -port 8080 -data-dir /mnt/data/laura-db
      Restart=always
      RestartSec=10
      StandardOutput=journal
      StandardError=journal

      [Install]
      WantedBy=multi-user.target

runcmd:
  # Install Go
  - cd /tmp
  - wget https://go.dev/dl/go1.25.4.linux-amd64.tar.gz
  - rm -rf /usr/local/go
  - tar -C /usr/local -xzf go1.25.4.linux-amd64.tar.gz
  - echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
  # Create laura-db user
  - useradd -m -s /bin/bash laura-db
  # Clone and build LauraDB
  - cd /opt
  - git clone https://github.com/mnohosten/laura-db.git
  - cd laura-db
  - /usr/local/go/bin/go build -o bin/laura-server cmd/server/main.go
  # Mount data disk
  - parted /dev/sdc --script mklabel gpt mkpart xfspart xfs 0% 100%
  - mkfs.xfs /dev/sdc1
  - mkdir -p /mnt/data
  - mount /dev/sdc1 /mnt/data
  - echo "/dev/sdc1 /mnt/data xfs defaults 0 0" >> /etc/fstab
  # Create data directory
  - mkdir -p /mnt/data/laura-db
  - chown -R laura-db:laura-db /mnt/data/laura-db
  - chown -R laura-db:laura-db /opt/laura-db
  # Install Azure Monitor agent
  - wget https://aka.ms/dependencyagentlinux -O InstallDependencyAgent-Linux64.bin
  - sh InstallDependencyAgent-Linux64.bin -s
  # Start service
  - systemctl daemon-reload
  - systemctl enable laura-db
  - systemctl start laura-db
```

### Step 5: Create Virtual Machine

```bash
# Create VM
az vm create \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --image Ubuntu2204 \
  --size Standard_B2s \
  --admin-username azureuser \
  --generate-ssh-keys \
  --custom-data cloud-init.yml \
  --vnet-name laura-db-vnet \
  --subnet laura-db-subnet \
  --nsg laura-db-nsg \
  --public-ip-sku Standard \
  --zone 1 \
  --tags application=laura-db

# Attach data disk
az vm disk attach \
  --resource-group laura-db-rg \
  --vm-name laura-db-vm \
  --name laura-db-data-disk
```

### Step 6: Verify Installation

```bash
# Get public IP
PUBLIC_IP=$(az vm show \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --show-details \
  --query publicIps \
  --output tsv)

echo "LauraDB URL: http://$PUBLIC_IP:8080"

# Test health endpoint
curl http://$PUBLIC_IP:8080/_health

# SSH to VM
ssh azureuser@$PUBLIC_IP

# Check service status
sudo systemctl status laura-db
```

## Multi-VM with Load Balancer

### Step 1: Create Load Balancer

```bash
# Create public IP for load balancer
az network public-ip create \
  --resource-group laura-db-rg \
  --name laura-db-lb-ip \
  --sku Standard \
  --zone 1 2

# Create load balancer
az network lb create \
  --resource-group laura-db-rg \
  --name laura-db-lb \
  --sku Standard \
  --public-ip-address laura-db-lb-ip \
  --frontend-ip-name laura-db-frontend \
  --backend-pool-name laura-db-backend

# Create health probe
az network lb probe create \
  --resource-group laura-db-rg \
  --lb-name laura-db-lb \
  --name laura-db-health \
  --protocol http \
  --port 8080 \
  --path /_health \
  --interval 30 \
  --threshold 2

# Create load balancing rule
az network lb rule create \
  --resource-group laura-db-rg \
  --lb-name laura-db-lb \
  --name laura-db-rule \
  --protocol tcp \
  --frontend-port 80 \
  --backend-port 8080 \
  --frontend-ip-name laura-db-frontend \
  --backend-pool-name laura-db-backend \
  --probe-name laura-db-health
```

### Step 2: Create Azure Files for Shared Storage

```bash
# Create storage account
az storage account create \
  --resource-group laura-db-rg \
  --name lauradbstorage$RANDOM \
  --location eastus \
  --sku Premium_LRS \
  --kind FileStorage

# Get storage account key
STORAGE_KEY=$(az storage account keys list \
  --resource-group laura-db-rg \
  --account-name lauradbstorage \
  --query '[0].value' \
  --output tsv)

# Create file share
az storage share create \
  --account-name lauradbstorage \
  --account-key $STORAGE_KEY \
  --name laura-db-data \
  --quota 100
```

### Step 3: Create Availability Set

```bash
az vm availability-set create \
  --resource-group laura-db-rg \
  --name laura-db-avset \
  --platform-fault-domain-count 2 \
  --platform-update-domain-count 5
```

### Step 4: Create Multiple VMs

```bash
# Create VMs in loop
for i in 1 2; do
  az vm create \
    --resource-group laura-db-rg \
    --name laura-db-vm-$i \
    --image Ubuntu2204 \
    --size Standard_D2s_v3 \
    --admin-username azureuser \
    --generate-ssh-keys \
    --custom-data cloud-init-azure-files.yml \
    --vnet-name laura-db-vnet \
    --subnet laura-db-subnet \
    --nsg laura-db-nsg \
    --public-ip-address "" \
    --availability-set laura-db-avset \
    --tags application=laura-db instance=$i

  # Add VM to load balancer backend pool
  NIC_ID=$(az vm show \
    --resource-group laura-db-rg \
    --name laura-db-vm-$i \
    --query 'networkProfile.networkInterfaces[0].id' \
    --output tsv)

  az network nic ip-config address-pool add \
    --resource-group laura-db-rg \
    --nic-name $(basename $NIC_ID) \
    --ip-config-name ipconfig1 \
    --lb-name laura-db-lb \
    --address-pool laura-db-backend
done
```

## Virtual Machine Scale Sets

### Create VM Scale Set

```bash
# Create VMSS
az vmss create \
  --resource-group laura-db-rg \
  --name laura-db-vmss \
  --image Ubuntu2204 \
  --vm-sku Standard_D2s_v3 \
  --instance-count 2 \
  --admin-username azureuser \
  --generate-ssh-keys \
  --custom-data cloud-init.yml \
  --vnet-name laura-db-vnet \
  --subnet laura-db-subnet \
  --lb laura-db-lb \
  --backend-pool-name laura-db-backend \
  --upgrade-policy-mode Automatic \
  --zones 1 2 \
  --tags application=laura-db

# Configure auto-scaling
az monitor autoscale create \
  --resource-group laura-db-rg \
  --resource laura-db-vmss \
  --resource-type Microsoft.Compute/virtualMachineScaleSets \
  --name laura-db-autoscale \
  --min-count 2 \
  --max-count 10 \
  --count 2

# Add CPU-based scale-out rule
az monitor autoscale rule create \
  --resource-group laura-db-rg \
  --autoscale-name laura-db-autoscale \
  --condition "Percentage CPU > 70 avg 5m" \
  --scale out 1

# Add CPU-based scale-in rule
az monitor autoscale rule create \
  --resource-group laura-db-rg \
  --autoscale-name laura-db-autoscale \
  --condition "Percentage CPU < 30 avg 5m" \
  --scale in 1
```

## Storage Configuration

### Managed Disk Types

| Type | IOPS | Throughput | Use Case | Cost/GB/month |
|------|------|------------|----------|---------------|
| Standard HDD | 500 | 60 MB/s | Development | $0.05 |
| Standard SSD | 500 | 60 MB/s | Web servers | $0.075 |
| Premium SSD | 5000 | 200 MB/s | Production | $0.135 |
| Ultra Disk | 160K+ | 2000+ MB/s | Critical | Variable |

### Resize Disk

```bash
# Deallocate VM
az vm deallocate \
  --resource-group laura-db-rg \
  --name laura-db-vm

# Resize disk
az disk update \
  --resource-group laura-db-rg \
  --name laura-db-data-disk \
  --size-gb 100

# Start VM
az vm start \
  --resource-group laura-db-rg \
  --name laura-db-vm

# SSH and resize filesystem
ssh azureuser@$PUBLIC_IP
sudo xfs_growfs /mnt/data
```

### Snapshot for Backup

```bash
# Create snapshot
az snapshot create \
  --resource-group laura-db-rg \
  --name laura-db-snapshot-$(date +%Y%m%d) \
  --source laura-db-data-disk

# List snapshots
az snapshot list \
  --resource-group laura-db-rg \
  --output table
```

## Network Configuration

### Virtual Network Setup

```bash
# Create VNet with subnets
az network vnet create \
  --resource-group laura-db-rg \
  --name laura-db-vnet \
  --address-prefix 10.0.0.0/16 \
  --subnet-name laura-db-subnet \
  --subnet-prefix 10.0.1.0/24

# Create additional subnet for bastion
az network vnet subnet create \
  --resource-group laura-db-rg \
  --vnet-name laura-db-vnet \
  --name AzureBastionSubnet \
  --address-prefix 10.0.2.0/27

# Create NAT Gateway
az network public-ip create \
  --resource-group laura-db-rg \
  --name laura-db-nat-ip \
  --sku Standard \
  --zone 1 2 3

az network nat gateway create \
  --resource-group laura-db-rg \
  --name laura-db-nat \
  --public-ip-addresses laura-db-nat-ip

az network vnet subnet update \
  --resource-group laura-db-rg \
  --vnet-name laura-db-vnet \
  --name laura-db-subnet \
  --nat-gateway laura-db-nat
```

### Application Gateway (Advanced)

```bash
# Create Application Gateway for layer 7 load balancing
az network application-gateway create \
  --resource-group laura-db-rg \
  --name laura-db-appgw \
  --location eastus \
  --sku Standard_v2 \
  --capacity 2 \
  --vnet-name laura-db-vnet \
  --subnet laura-db-appgw-subnet \
  --public-ip-address laura-db-appgw-ip \
  --http-settings-port 8080 \
  --http-settings-protocol Http \
  --servers 10.0.1.4 10.0.1.5
```

## Security Best Practices

### 1. Use Managed Identity

```bash
# Enable system-assigned managed identity
az vm identity assign \
  --resource-group laura-db-rg \
  --name laura-db-vm

# Grant permissions
IDENTITY_ID=$(az vm show \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --query identity.principalId \
  --output tsv)

az role assignment create \
  --assignee $IDENTITY_ID \
  --role "Storage Blob Data Contributor" \
  --scope /subscriptions/SUBSCRIPTION_ID/resourceGroups/laura-db-rg
```

### 2. Use Azure Key Vault

```bash
# Create Key Vault
az keyvault create \
  --resource-group laura-db-rg \
  --name laura-db-kv-$RANDOM \
  --enable-soft-delete true \
  --enable-purge-protection true

# Store admin password
az keyvault secret set \
  --vault-name laura-db-kv \
  --name admin-password \
  --value "your-secure-password"

# Grant VM access
az keyvault set-policy \
  --name laura-db-kv \
  --object-id $IDENTITY_ID \
  --secret-permissions get list
```

### 3. Enable Azure Disk Encryption

```bash
# Create Key Vault for encryption
az keyvault create \
  --resource-group laura-db-rg \
  --name laura-db-encryption-kv \
  --enable-for-disk-encryption true

# Enable encryption on VM
az vm encryption enable \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --disk-encryption-keyvault laura-db-encryption-kv
```

### 4. Use Azure Bastion

```bash
# Create Bastion
az network bastion create \
  --resource-group laura-db-rg \
  --name laura-db-bastion \
  --public-ip-address laura-db-bastion-ip \
  --vnet-name laura-db-vnet \
  --location eastus
```

## Monitoring and Logging

See [azure-monitor.md](./azure-monitor.md) for detailed monitoring setup.

### Quick Setup

```bash
# Enable VM Insights
az vm extension set \
  --resource-group laura-db-rg \
  --vm-name laura-db-vm \
  --name AzureMonitorLinuxAgent \
  --publisher Microsoft.Azure.Monitor \
  --enable-auto-upgrade true
```

## Backup and Recovery

See [blob-storage-backup.md](./blob-storage-backup.md) for detailed backup procedures.

### Quick Backup with Azure Backup

```bash
# Create Recovery Services vault
az backup vault create \
  --resource-group laura-db-rg \
  --name laura-db-vault \
  --location eastus

# Enable backup for VM
az backup protection enable-for-vm \
  --resource-group laura-db-rg \
  --vault-name laura-db-vault \
  --vm laura-db-vm \
  --policy-name DefaultPolicy
```

## Cost Optimization

### 1. Use Spot VMs for Dev/Test

```bash
az vm create \
  --resource-group laura-db-rg \
  --name laura-db-spot-vm \
  --priority Spot \
  --max-price -1 \
  --eviction-policy Deallocate \
  --image Ubuntu2204 \
  --size Standard_D2s_v3
```

**Savings**: Up to 90% compared to pay-as-you-go

### 2. Use Reserved Instances

```bash
# Purchase 1-year reservation
az reservations reservation-order purchase \
  --reserved-resource-type VirtualMachines \
  --sku Standard_D2s_v3 \
  --location eastus \
  --quantity 2 \
  --term P1Y
```

**Savings**: Up to 72% (1-year) or 82% (3-year)

### 3. Auto-Shutdown for Dev VMs

```bash
az vm auto-shutdown \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --time 1800 \
  --timezone "Eastern Standard Time"
```

### 4. Right-Size VMs

```bash
# Get VM recommendations
az advisor recommendation list \
  --category Cost \
  --output table
```

## Troubleshooting

### VM Not Starting

```bash
# Check boot diagnostics
az vm boot-diagnostics get-boot-log \
  --resource-group laura-db-rg \
  --name laura-db-vm

# Check VM status
az vm get-instance-view \
  --resource-group laura-db-rg \
  --name laura-db-vm
```

### Service Not Running

```bash
# SSH to VM
ssh azureuser@$PUBLIC_IP

# Check service
sudo systemctl status laura-db

# View logs
sudo journalctl -u laura-db -f
```

### Disk Mount Issues

```bash
# List disks
lsblk

# Check mounts
mount | grep /mnt/data

# View fstab
cat /etc/fstab
```

### Load Balancer Issues

```bash
# Check backend health
az network lb show \
  --resource-group laura-db-rg \
  --name laura-db-lb

# Test connectivity
az network lb probe show \
  --resource-group laura-db-rg \
  --lb-name laura-db-lb \
  --name laura-db-health
```

## Best Practices

1. ✅ **Use Availability Zones** for high availability
2. ✅ **Enable managed identity** for Azure service access
3. ✅ **Use Azure Key Vault** for secrets
4. ✅ **Enable disk encryption** for data at rest
5. ✅ **Implement auto-scaling** with VMSS
6. ✅ **Use Azure Bastion** for secure access
7. ✅ **Enable Azure Monitor** for observability
8. ✅ **Implement regular backups** with Azure Backup
9. ✅ **Use tags** for cost tracking
10. ✅ **Enable auto-shutdown** for dev environments

## Next Steps

- [AKS Deployment](./aks-deployment.md)
- [Blob Storage Backup Integration](./blob-storage-backup.md)
- [Azure Monitor Setup](./azure-monitor.md)

## Additional Resources

- [Azure VM Documentation](https://docs.microsoft.com/en-us/azure/virtual-machines/)
- [Azure Pricing Calculator](https://azure.microsoft.com/en-us/pricing/calculator/)
- [Azure Well-Architected Framework](https://docs.microsoft.com/en-us/azure/architecture/framework/)
- [Azure CLI Reference](https://docs.microsoft.com/en-us/cli/azure/)
