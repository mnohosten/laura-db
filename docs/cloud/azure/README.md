# LauraDB Microsoft Azure Deployment Guides

Comprehensive guides for deploying and operating LauraDB on Microsoft Azure.

## Table of Contents

- [Available Guides](#available-guides)
- [Quick Start](#quick-start)
- [Deployment Comparison](#deployment-comparison)
- [Architecture Patterns](#architecture-patterns)
- [Cost Estimates](#cost-estimates)
- [Azure vs Other Clouds](#azure-vs-other-clouds)
- [Getting Started](#getting-started)
- [Best Practices](#best-practices)
- [Azure Advantages](#azure-advantages)
- [Support and Resources](#support-and-resources)
- [Troubleshooting](#troubleshooting)
- [Migration Guides](#migration-guides)
- [Pre-Production Checklist](#pre-production-checklist)

## Available Guides

### Deployment Options

1. **[VM Deployment](./vm-deployment.md)**
   - Single VM for development/testing
   - Multi-VM deployment with load balancing
   - Virtual Machine Scale Sets with auto-scaling
   - Azure Managed Disks and Azure Files storage
   - Network configuration (VNet, NSG, Application Gateway)
   - Complete step-by-step instructions

2. **[AKS Deployment](./aks-deployment.md)**
   - Azure Kubernetes Service deployment
   - Helm chart installation
   - Storage with Azure Disks and Azure Files
   - Azure CNI networking and Network Policies
   - Workload Identity for secure Azure service access
   - Application Gateway Ingress Controller
   - Auto-scaling and high availability

### Operations & Management

3. **[Blob Storage Backup Integration](./blob-storage-backup.md)**
   - Automated backup strategies (full, incremental, WAL)
   - Blob Storage configuration and security
   - Restore procedures and point-in-time recovery
   - Lifecycle management and cost optimization
   - Cross-region replication
   - Disaster recovery planning

4. **[Azure Monitor Integration](./azure-monitor.md)**
   - Comprehensive monitoring with Azure Monitor
   - VM Insights and Container Insights
   - Custom metrics and log collection
   - Alerting policies and notification channels
   - Log Analytics with KQL queries
   - Application Insights for APM
   - Network Watcher for network diagnostics

## Quick Start

### Choose Your Deployment Method

#### For Development/Testing
**Recommended: Azure VM (Single Instance)**
- Fastest setup: ~10 minutes
- Lowest cost: ~$25/month
- Full control and easy debugging
- Good for: POC, development, learning

```bash
# Quick VM setup - see vm-deployment.md for details
az vm create \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --image Ubuntu2204 \
  --size Standard_B2s \
  --generate-ssh-keys
```

#### For Production (Small to Medium)
**Recommended: VM Scale Sets**
- Managed auto-scaling
- Built-in load balancing
- Cost: ~$150-250/month
- Good for: Production workloads up to 10K req/sec

```bash
# See vm-deployment.md for complete setup
```

#### For Production (Large Scale/Cloud-Native)
**Recommended: AKS (Azure Kubernetes Service)**
- Container orchestration
- Advanced networking and security
- Maximum flexibility and scalability
- Cost: ~$200-500/month
- Good for: Microservices, high availability, global scale

```bash
# See aks-deployment.md for complete setup
```

## Deployment Comparison

| Factor | Single VM | VM Scale Set | AKS | Best Use Case |
|--------|-----------|--------------|-----|---------------|
| **Setup Time** | 10-15 min | 20-30 min | 30-45 min | VM for speed |
| **Complexity** | Low | Medium | High | VM for simplicity |
| **Management** | Manual | Semi-automated | Automated | AKS for automation |
| **Cost (small)** | $ | $$ | $$$ | VM for cost |
| **Cost (large)** | $$$ | $$ | $$ | VMSS/AKS for scale |
| **Scaling** | Manual | Auto (VM-based) | Auto (pod-based) | AKS for granularity |
| **Availability** | Single zone | Multi-zone | Multi-zone | VMSS/AKS for HA |
| **Flexibility** | High | Medium | Very High | AKS for containers |
| **Maintenance** | Manual | Semi-automated | Automated | AKS for DevOps |
| **Networking** | Simple | Advanced | Very Advanced | AKS for control |

**Decision Matrix**:
- **Start with VM** if: New to Azure, small workload, need full control
- **Use VM Scale Sets** if: Traditional VMs preferred, predictable scaling, Windows workloads
- **Choose AKS** if: Cloud-native, microservices, need maximum flexibility

## Architecture Patterns

### Pattern 1: Single VM (Development)

```
┌─────────────────┐
│  Public IP      │
└────────┬────────┘
         │
    ┌────▼─────┐
    │   VM     │
    │ LauraDB  │
    │ +Disk    │
    └──────────┘
```

**Specifications**:
- VM: Standard_B2s (2 vCPU, 4 GB RAM)
- Disk: 50 GB Premium SSD
- Network: Public IP, NSG
- Cost: ~$30/month

**Use Case**: Development, testing, demos, learning

**Pros**:
- Simple setup
- Low cost
- Full SSH access
- Easy debugging

**Cons**:
- Single point of failure
- Manual scaling
- No built-in HA

### Pattern 2: Multi-Zone with Load Balancer (Production)

```
┌───────────────────────────────────┐
│      Azure Load Balancer          │
└──────┬────────────────────┬───────┘
       │                    │
   ┌───▼──┐             ┌───▼──┐
   │Zone 1│             │Zone 2│
   │  VM  │             │  VM  │
   │ +Disk│             │ +Disk│
   └───┬──┘             └──┬───┘
       │                   │
       └─────────┬─────────┘
             ┌───▼────┐
             │ Azure  │
             │ Files  │
             └────────┘
```

**Specifications**:
- VMs: 2-3 × Standard_D4s_v3 (4 vCPU, 16 GB RAM)
- Disks: 200 GB Premium SSD per VM
- Storage: 1 TB Azure Files Premium
- Network: Load Balancer, NSG, Application Gateway
- Cost: ~$500-700/month

**Use Case**: Production applications, high availability

**Pros**:
- High availability (99.99% SLA)
- Auto-failover
- Shared storage
- Zone-redundant

**Cons**:
- Higher cost
- More complex setup
- Requires shared storage

### Pattern 3: AKS Multi-Region (Global/Mission-Critical)

```
┌─────────────────┐         ┌─────────────────┐
│  East US        │         │   West US       │
│  ┌──────────┐   │         │  ┌──────────┐   │
│  │   AKS    │   │◄────────┤  │   AKS    │   │
│  │ Cluster  │   │  Azure  │  │ Cluster  │   │
│  └────┬─────┘   │  Traffic│  └────┬─────┘   │
│       │         │  Manager│       │         │
│  ┌────▼─────┐   │         │  ┌────▼─────┐   │
│  │Azure Disk│   │         │  │Azure Disk│   │
│  └──────────┘   │         │  └──────────┘   │
└─────────────────┘         └─────────────────┘
         │                           │
         └─────────┬─────────────────┘
              ┌────▼────┐
              │  Blob   │
              │ Storage │
              │  (GRS)  │
              └─────────┘
```

**Specifications**:
- AKS: 2 clusters (3-5 nodes each, Standard_D4s_v3)
- Storage: Azure Disk Premium + Blob Storage GRS
- Network: Azure Traffic Manager, Application Gateway
- Monitoring: Azure Monitor, Application Insights
- Cost: ~$1,200-2,000/month

**Use Case**: Mission-critical, global applications, 99.99%+ availability

**Pros**:
- Maximum availability
- Global reach
- Automatic failover
- Disaster recovery

**Cons**:
- High cost
- Complex architecture
- Data consistency challenges
- Requires expertise

## Cost Estimates

### Monthly Cost Breakdown

#### Small Deployment (Dev/Test)

| Component | Details | Monthly Cost |
|-----------|---------|--------------|
| **VM** | Standard_B2s | $24.82 |
| **Managed Disk** | 50 GB Premium SSD | $9.60 |
| **Public IP** | Static | $3.00 |
| **Network** | Outbound data (10 GB) | $0.87 |
| **Monitoring** | Log Analytics (1 GB) | $2.30 |
| **Backup** | Blob Storage (10 GB) | $0.19 |
| **Total** | | **~$40/month** |

#### Medium Deployment (Production)

| Component | Details | Monthly Cost |
|-----------|---------|--------------|
| **VMs** | 2 × Standard_D4s_v3 | $280.32 |
| **Managed Disks** | 2 × 200 GB Premium SSD | $76.80 |
| **Azure Files** | 1 TB Premium | $204.80 |
| **Load Balancer** | Standard tier | $21.90 |
| **Application Gateway** | WAF v2 | $140.00 |
| **Network** | Outbound data (100 GB) | $8.70 |
| **Monitoring** | Log Analytics (10 GB) | $23.00 |
| **Backup** | Blob Storage (100 GB GRS) | $4.20 |
| **Total** | | **~$760/month** |

#### Large Deployment (AKS Enterprise)

| Component | Details | Monthly Cost |
|-----------|---------|--------------|
| **AKS Control Plane** | Free (cluster fee waived) | $0.00 |
| **Worker Nodes** | 5 × Standard_D8s_v3 | $1,168.00 |
| **Managed Disks** | 5 × 500 GB Premium SSD | $192.00 |
| **Azure Files** | 2 TB Premium | $409.60 |
| **Load Balancer** | Standard tier | $21.90 |
| **Application Gateway** | WAF v2 with autoscale | $300.00 |
| **Network** | Outbound data (500 GB) | $43.50 |
| **Monitoring** | Log Analytics (50 GB) + App Insights | $130.00 |
| **Backup** | Blob Storage (500 GB GRS) | $21.00 |
| **Container Registry** | Premium | $167.20 |
| **Total** | | **~$2,453/month** |

*Prices based on East US region, pay-as-you-go pricing, as of 2025*

### Cost Optimization Strategies

1. **Use Reserved Instances** - Save up to 72% with 1 or 3-year commitments
2. **Azure Hybrid Benefit** - Use existing Windows Server/SQL Server licenses
3. **Spot VMs** - Save up to 90% for fault-tolerant workloads
4. **Right-size VMs** - Monitor usage and adjust VM sizes
5. **Auto-shutdown** - Schedule shutdown for dev/test environments
6. **Blob Storage tiers** - Use Cool/Archive tiers for old backups
7. **Budget alerts** - Set up alerts to avoid surprises

## Azure vs Other Clouds

### Feature Comparison

| Feature | Azure | AWS | GCP | Winner |
|---------|-------|-----|-----|--------|
| **VM Pricing** | $0.0672/hour (D2s_v3) | $0.0752/hour (t3.large) | $0.0475/hour (n2-standard-2) | GCP |
| **Storage (SSD)** | $0.12/GB/month | $0.08/GB/month | $0.17/GB/month | AWS |
| **Load Balancer** | $21.90/month | $22.27/month | $18.00/month | GCP |
| **Kubernetes** | AKS (control plane free) | EKS ($73/month) | GKE ($73/month) | Azure |
| **Managed Identity** | Yes (native) | Yes (IAM roles) | Yes (Workload Identity) | Tie |
| **Backup Solution** | Azure Backup + Blob | AWS Backup + S3 | GCP Backup + GCS | Tie |
| **Monitoring** | Azure Monitor | CloudWatch | Cloud Monitoring | Azure |
| **Data Transfer Out** | $0.087/GB | $0.090/GB | $0.120/GB | Azure |

**Overall**: Azure is competitive on pricing and offers excellent integration across services. Best choice if:
- Already using Microsoft stack (Windows, .NET, SQL Server)
- Need enterprise support and compliance
- Want seamless hybrid cloud with Azure Arc
- Prefer unified Azure Portal experience

### Migration Comparison

| From | To Azure | Effort | Tools |
|------|----------|--------|-------|
| **AWS** | Medium | Azure Migrate, Database Migration Service |
| **GCP** | Medium | Azure Migrate, Azure Data Box |
| **On-Premises** | Low-Medium | Azure Migrate, Azure Site Recovery |

## Getting Started

### Prerequisites

1. **Azure Account**: [Create free account](https://azure.microsoft.com/free/) ($200 credit)
2. **Azure CLI**: Install command-line tools
3. **Knowledge**: Basic Linux, networking, databases

### Step 1: Setup Azure Environment

```bash
# Install Azure CLI (Linux/macOS)
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Or on macOS with Homebrew
brew install azure-cli

# Verify installation
az --version

# Login to Azure
az login

# List subscriptions
az account list --output table

# Set active subscription
az account set --subscription "Your Subscription Name"

# Set default location
az config set defaults.location=eastus
az config set defaults.group=laura-db-rg
```

### Step 2: Create Resource Group

```bash
RESOURCE_GROUP="laura-db-rg"
LOCATION="eastus"

az group create \
  --name $RESOURCE_GROUP \
  --location $LOCATION \
  --tags application=laura-db environment=production owner=team@example.com
```

### Step 3: Choose Deployment Path

- **Simple/Quick**: Follow [VM Deployment Guide](./vm-deployment.md)
- **Production/Scale**: Follow [AKS Deployment Guide](./aks-deployment.md)
- **Then setup**: [Backup Integration](./blob-storage-backup.md) and [Monitoring](./azure-monitor.md)

## Best Practices

### Security

1. ✅ **Use Managed Identity** instead of credentials in code
2. ✅ **Enable Azure Defender** for threat protection
3. ✅ **Use Azure Key Vault** for secrets management
4. ✅ **Enable disk encryption** on all VMs and disks
5. ✅ **Use Network Security Groups** restrictively (allow-list only)
6. ✅ **Enable Azure Bastion** for secure VM access (no public SSH)
7. ✅ **Use Azure Policy** to enforce organizational standards
8. ✅ **Enable Azure Firewall** or Application Gateway WAF
9. ✅ **Use Private Endpoints** for Azure services
10. ✅ **Enable Microsoft Defender for Cloud** (formerly Security Center)

### Reliability

1. ✅ **Deploy across Availability Zones** for 99.99% SLA
2. ✅ **Use Load Balancer** or Application Gateway for traffic distribution
3. ✅ **Implement health probes** on all load-balanced resources
4. ✅ **Use VM Scale Sets** or AKS for auto-healing
5. ✅ **Enable Azure Backup** for all VMs and databases
6. ✅ **Use geo-redundant storage** (GRS/RA-GRS) for critical data
7. ✅ **Test disaster recovery** procedures quarterly
8. ✅ **Document runbooks** for common failure scenarios
9. ✅ **Set up Azure Monitor alerts** proactively
10. ✅ **Use Azure Traffic Manager** for multi-region failover

### Performance

1. ✅ **Choose appropriate VM sizes** (Dsv3/Easv4 for memory, Fsv2 for compute)
2. ✅ **Use Premium SSD or Ultra Disk** for production workloads
3. ✅ **Enable Accelerated Networking** on VMs for lower latency
4. ✅ **Use Azure Cache for Redis** for session/data caching
5. ✅ **Enable Azure CDN** for static content delivery
6. ✅ **Use Proximity Placement Groups** for low-latency workloads
7. ✅ **Optimize disk throughput** with proper IOPS configuration
8. ✅ **Use Azure NetApp Files** for high-performance shared storage
9. ✅ **Implement caching** at application and database layers
10. ✅ **Monitor with Azure Monitor** and optimize based on metrics

### Cost Optimization

1. ✅ **Purchase Reserved Instances** for predictable workloads (up to 72% savings)
2. ✅ **Use Spot VMs** for fault-tolerant workloads (up to 90% savings)
3. ✅ **Right-size VMs** using Azure Advisor recommendations
4. ✅ **Auto-shutdown** dev/test VMs during off-hours
5. ✅ **Use Blob Storage lifecycle policies** to move data to Cool/Archive tiers
6. ✅ **Enable VM autoscaling** to match demand
7. ✅ **Delete unused resources** (disks, IPs, snapshots)
8. ✅ **Use Azure Hybrid Benefit** for Windows/SQL Server (save 40%)
9. ✅ **Set up budget alerts** in Azure Cost Management
10. ✅ **Review Azure Advisor** cost recommendations monthly

### Operations

1. ✅ **Use Infrastructure as Code** (ARM templates, Bicep, Terraform)
2. ✅ **Implement CI/CD** with Azure DevOps or GitHub Actions
3. ✅ **Use Azure DevTest Labs** for development environments
4. ✅ **Tag all resources** consistently (application, environment, owner, cost-center)
5. ✅ **Enable diagnostic logs** for all Azure resources
6. ✅ **Create Azure Automation runbooks** for routine tasks
7. ✅ **Use Azure Resource Graph** for inventory and compliance queries
8. ✅ **Implement Azure Blueprints** for repeatable deployments
9. ✅ **Use Azure Management Groups** for organization-wide governance
10. ✅ **Document everything** in Azure DevOps Wiki or Confluence

## Azure Advantages

### Why Choose Azure for LauraDB?

1. **Microsoft Integration**: Seamless integration with Active Directory, Office 365, Microsoft 365
2. **Hybrid Cloud**: Azure Arc for consistent management across on-premises and cloud
3. **Enterprise Support**: Microsoft Premier Support with SLAs
4. **Compliance**: Widest set of compliance certifications (90+ compliance offerings)
5. **Global Reach**: 60+ regions worldwide (more than AWS or GCP)
6. **Azure Hybrid Benefit**: Bring your own Windows/SQL Server licenses
7. **Managed Services**: Extensive PaaS offerings reduce operational overhead
8. **Azure AI/ML**: Advanced AI and machine learning services if needed
9. **Developer Tools**: Excellent Visual Studio, VS Code, Azure DevOps integration
10. **Cost Management**: Granular cost management and optimization tools

### Azure-Specific Features

1. **Azure Bastion**: Secure RDP/SSH without public IPs
2. **Azure Policy**: Enforce governance across subscriptions
3. **Azure Blueprints**: Repeatable, governed deployments
4. **Azure Arc**: Extend Azure management to any infrastructure
5. **Azure Defender**: Advanced threat protection
6. **Azure Sentinel**: Cloud-native SIEM
7. **Azure Lighthouse**: Multi-tenant management
8. **Azure Private Link**: Access Azure services privately
9. **Azure Front Door**: Global CDN + WAF + load balancing
10. **Azure Chaos Studio**: Resilience testing platform

## Support and Resources

### Documentation

- [LauraDB Main Documentation](../../README.md)
- [Azure Documentation](https://docs.microsoft.com/en-us/azure/)
- [Azure Architecture Center](https://docs.microsoft.com/en-us/azure/architecture/)
- [Azure Well-Architected Framework](https://docs.microsoft.com/en-us/azure/architecture/framework/)

### Tools

- [Azure Portal](https://portal.azure.com)
- [Azure Pricing Calculator](https://azure.microsoft.com/en-us/pricing/calculator/)
- [Azure CLI Reference](https://docs.microsoft.com/en-us/cli/azure/)
- [Azure Resource Manager Templates](https://docs.microsoft.com/en-us/azure/azure-resource-manager/templates/)
- [Azure Quickstart Templates](https://github.com/Azure/azure-quickstart-templates)

### Community

- [LauraDB GitHub Issues](https://github.com/mnohosten/laura-db/issues)
- [Azure Community](https://techcommunity.microsoft.com/t5/azure/ct-p/Azure)
- [Stack Overflow - Azure](https://stackoverflow.com/questions/tagged/azure)
- [Azure Friday Videos](https://azure.microsoft.com/en-us/resources/videos/azure-friday/)

### Learning Resources

- [Microsoft Learn - Azure](https://docs.microsoft.com/en-us/learn/azure/)
- [Azure Fundamentals Certification](https://docs.microsoft.com/en-us/learn/certifications/azure-fundamentals/)
- [AKS Workshop](https://docs.microsoft.com/en-us/learn/modules/aks-workshop/)
- [Azure DevOps Labs](https://azuredevopslabs.com/)

## Troubleshooting

### Common Issues

#### Issue: "Quota exceeded" error

**Symptoms**: Cannot create VMs or other resources

**Solution**:
```bash
# Check quota usage
az vm list-usage --location eastus --output table

# Request quota increase
# Azure Portal → Subscriptions → Usage + quotas → Request increase
```

#### Issue: Cannot connect to VM

**Symptoms**: SSH/RDP connection fails

**Solutions**:
1. Check NSG rules allow inbound traffic on port 22 (SSH) or 3389 (RDP)
2. Verify VM is running: `az vm get-instance-view`
3. Check if public IP is assigned: `az vm show --show-details`
4. Use Azure Bastion for secure access
5. Check Azure Firewall rules if deployed

```bash
# Verify VM status
az vm get-instance-view \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --query instanceView.statuses[1]

# Check public IP
az vm show \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --show-details \
  --query publicIps -o tsv
```

#### Issue: High costs

**Symptoms**: Azure bill higher than expected

**Solutions**:
```bash
# Review current costs
az consumption usage list \
  --start-date 2025-01-01 \
  --end-date 2025-01-31 \
  --query "[?contains(instanceName, 'laura-db')]" \
  -o table

# Check Azure Advisor cost recommendations
az advisor recommendation list \
  --category Cost \
  --output table

# Set budget alert
az consumption budget create \
  --amount 500 \
  --budget-name laura-db-budget \
  --category cost \
  --time-grain monthly \
  --start-date 2025-01-01 \
  --end-date 2026-01-01

# Use Azure Cost Management in portal for detailed analysis
```

#### Issue: Slow performance

**Symptoms**: High latency, slow queries

**Solutions**:
1. Check VM CPU/memory usage in Azure Monitor
2. Upgrade to higher VM SKU if needed
3. Switch to Premium SSD or Ultra Disk
4. Enable Accelerated Networking
5. Check disk IOPS/throughput limits
6. Review application logs for bottlenecks

```bash
# Check VM metrics
az monitor metrics list \
  --resource /subscriptions/YOUR_SUB_ID/resourceGroups/laura-db-rg/providers/Microsoft.Compute/virtualMachines/laura-db-vm \
  --metric "Percentage CPU" \
  --start-time 2025-01-15T00:00:00Z \
  --end-time 2025-01-15T23:59:59Z \
  --interval PT1H

# Resize VM
az vm resize \
  --resource-group laura-db-rg \
  --name laura-db-vm \
  --size Standard_D4s_v3
```

#### Issue: Backup failures

**Symptoms**: Backups not appearing in Blob Storage

**Solutions**:
1. Check Managed Identity permissions on Storage Account
2. Verify network connectivity to *.blob.core.windows.net
3. Check storage account firewall rules
4. Review backup script logs
5. Ensure sufficient disk space for temporary backup files

```bash
# Verify managed identity role assignment
az role assignment list \
  --assignee YOUR_MANAGED_IDENTITY_ID \
  --scope /subscriptions/YOUR_SUB_ID/resourceGroups/laura-db-rg/providers/Microsoft.Storage/storageAccounts/YOUR_STORAGE_ACCOUNT

# Test blob storage connection
az storage blob list \
  --account-name YOUR_STORAGE_ACCOUNT \
  --container-name full-backups \
  --auth-mode login
```

## Migration Guides

### From AWS to Azure

1. **Export data** from AWS using LauraDB backup procedures
2. **Transfer to Azure** using Azure Data Box or network transfer
3. **Deploy LauraDB** on Azure using this guide
4. **Import data** from Azure Blob Storage
5. **Update DNS** to point to new Azure deployment
6. **Test thoroughly** before decommissioning AWS
7. **Decommission** AWS resources

**Key Mappings**:
- EC2 → Azure VMs
- EKS → AKS
- S3 → Blob Storage
- EBS → Managed Disks
- CloudWatch → Azure Monitor
- IAM Roles → Managed Identity

### From GCP to Azure

Similar process to AWS migration.

**Key Mappings**:
- GCE → Azure VMs
- GKE → AKS
- Cloud Storage → Blob Storage
- Persistent Disks → Managed Disks
- Cloud Monitoring → Azure Monitor
- Service Accounts → Managed Identity

### From On-Premises to Azure

1. **Assess** current environment with Azure Migrate
2. **Plan** deployment architecture on Azure
3. **Backup** on-premises LauraDB data
4. **Deploy** Azure infrastructure (VMs or AKS)
5. **Transfer** data using Azure Data Box or VPN/ExpressRoute
6. **Cutover** with application connection string updates
7. **Decommission** on-premises infrastructure (or keep as DR)

**Tools**:
- Azure Migrate for assessment and migration
- Azure Site Recovery for VM replication
- Azure Database Migration Service for databases
- Azure ExpressRoute for high-speed private connectivity

## Pre-Production Checklist

### Before Going to Production

- [ ] Architecture reviewed and documented
- [ ] All resources deployed in multiple availability zones
- [ ] Load Balancer or Application Gateway configured
- [ ] Azure Monitor and Log Analytics enabled
- [ ] Custom metrics and logs configured
- [ ] Alert rules created for all critical scenarios
- [ ] Backup strategy implemented and tested
- [ ] Disaster recovery plan documented and tested
- [ ] Network Security Groups configured restrictively
- [ ] Managed Identity configured (no credentials in code)
- [ ] Azure Key Vault for all secrets
- [ ] Disk encryption enabled on all VMs/disks
- [ ] Azure Backup or Blob Storage backups tested
- [ ] Restore procedures documented and tested
- [ ] Cost budget alerts configured
- [ ] Tagging strategy applied to all resources
- [ ] Azure Policy implemented for governance
- [ ] Runbooks documented for common operations
- [ ] Team training completed
- [ ] Load testing performed

### Ongoing Operations

- [ ] Monitor dashboards daily
- [ ] Review logs for errors weekly
- [ ] Test backups monthly
- [ ] Review costs monthly
- [ ] Apply security patches monthly
- [ ] Conduct DR drill quarterly
- [ ] Review and update documentation quarterly
- [ ] Review Azure Advisor recommendations monthly
- [ ] Optimize resource sizes quarterly
- [ ] Review access controls quarterly
- [ ] Update runbooks as needed

## License

LauraDB and all documentation are available under the same license as the main project.

## Contributing

Contributions to improve these guides are welcome! Please submit issues or pull requests to the [LauraDB repository](https://github.com/mnohosten/laura-db).

---

## Quick Links

- **Deployment**: [VM](./vm-deployment.md) | [AKS](./aks-deployment.md)
- **Operations**: [Backups](./blob-storage-backup.md) | [Monitoring](./azure-monitor.md)
- **Azure Portal**: https://portal.azure.com
- **Azure Status**: https://status.azure.com
- **Support**: https://azure.microsoft.com/en-us/support/options/

---

**Getting Started**: Choose [VM Deployment](./vm-deployment.md) for simple setup or [AKS Deployment](./aks-deployment.md) for container-based deployment. Then configure [backups](./blob-storage-backup.md) and [monitoring](./azure-monitor.md).

**Need Help?** Check [Troubleshooting](#troubleshooting) section or file an issue on [GitHub](https://github.com/mnohosten/laura-db/issues).
