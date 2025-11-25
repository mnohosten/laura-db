# LauraDB Multi-Cloud Deployment

Deploy LauraDB simultaneously across **AWS**, **GCP**, and **Azure** with a single Terraform configuration.

## Overview

This example demonstrates:
- **Multi-cloud deployment** with consistent configuration
- **Load balancers** in all three clouds
- **Automated backups** to cloud-native storage
- **Monitoring** with each cloud's native tools
- **Unified management** through Terraform

## Why Multi-Cloud?

### Benefits

1. **Avoid Vendor Lock-in**: Flexibility to move workloads between clouds
2. **Geographic Distribution**: Serve users from the closest region
3. **High Availability**: Survive entire cloud provider outages
4. **Cost Optimization**: Use best pricing from each provider
5. **Compliance**: Meet data residency requirements
6. **Risk Mitigation**: Reduce dependency on single provider

### Trade-offs

1. **Complexity**: More infrastructure to manage
2. **Cost**: Running resources in multiple clouds
3. **Data Consistency**: Requires application-level coordination
4. **Network Latency**: Cross-cloud communication slower than intra-cloud
5. **Learning Curve**: Need expertise in all three clouds

## Prerequisites

### Required Accounts

1. **AWS Account** with IAM credentials
2. **GCP Project** with billing enabled
3. **Azure Subscription** with contributor access

### Required Tools

```bash
# Terraform
brew install terraform  # macOS
# or download from https://www.terraform.io/downloads

# Cloud CLIs
brew install awscli     # AWS
brew install --cask google-cloud-sdk  # GCP
brew install azure-cli  # Azure

# Verify installations
terraform --version
aws --version
gcloud version
az --version
```

### Authentication Setup

#### AWS

```bash
# Configure AWS credentials
aws configure

# Or set environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"
```

#### GCP

```bash
# Authenticate with GCP
gcloud auth application-default login

# Set project
gcloud config set project YOUR_PROJECT_ID

# Enable required APIs
gcloud services enable compute.googleapis.com
gcloud services enable storage.googleapis.com
```

#### Azure

```bash
# Login to Azure
az login

# Set subscription
az account set --subscription "Your Subscription Name"

# Verify
az account show
```

## Quick Start

### 1. Create Configuration File

Create `terraform.tfvars`:

```hcl
# Common settings
project_name = "laura-db-multi"
environment  = "production"
ssh_public_key = "ssh-rsa AAAAB3... your-key"

# Security (IMPORTANT: Change for production!)
allowed_cidr_blocks = ["0.0.0.0/0"]  # Restrict to your IPs

# Features
enable_load_balancer = true
enable_backups       = true
enable_monitoring    = true

# GCP Project ID (required)
gcp_project_id = "my-gcp-project-id"

# Instance counts (adjust as needed)
aws_instance_count   = 2
gcp_instance_count   = 2
azure_instance_count = 2

# Tags
common_tags = {
  Team        = "Platform"
  Environment = "Production"
  MultiCloud  = "true"
}
```

### 2. Deploy to All Clouds

```bash
# Initialize Terraform (downloads all provider plugins)
terraform init

# Review the deployment plan
terraform plan

# Deploy to all three clouds simultaneously
terraform apply

# Confirm with 'yes' when prompted
```

### 3. Verify Deployments

```bash
# Get all endpoints
terraform output all_endpoints

# Test health checks
terraform output health_check_commands

# Test each cloud
curl $(terraform output -json all_endpoints | jq -r '.aws')/_health
curl $(terraform output -json all_endpoints | jq -r '.gcp')/_health
curl $(terraform output -json all_endpoints | jq -r '.azure')/_health
```

## Configuration Examples

### Minimal Multi-Cloud (Development)

```hcl
# terraform.tfvars
project_name       = "laura-db-dev"
environment        = "development"
gcp_project_id     = "my-project-id"

# Small instances
aws_instance_type  = "t3.small"
aws_instance_count = 1
gcp_machine_type   = "e2-small"
gcp_instance_count = 1
azure_vm_size      = "Standard_B2s"
azure_instance_count = 1

# Minimal features
enable_load_balancer = false
enable_backups       = false
enable_monitoring    = true
```

**Estimated cost**: ~$75/month (all clouds combined)

### Production Multi-Cloud (High Availability)

```hcl
# terraform.tfvars
project_name   = "laura-db-prod"
environment    = "production"
gcp_project_id = "my-project-id"

# Production instances
aws_instance_type    = "t3.large"
aws_instance_count   = 3
gcp_machine_type     = "e2-standard-4"
gcp_instance_count   = 3
azure_vm_size        = "Standard_D4s_v3"
azure_instance_count = 3

# Storage
aws_volume_type      = "gp3"
aws_volume_size      = 200
gcp_disk_type        = "pd-ssd"
gcp_disk_size_gb     = 200
azure_disk_type      = "Premium_LRS"
azure_disk_size_gb   = 200

# All features enabled
enable_load_balancer = true
enable_backups       = true
enable_monitoring    = true

# Security
allowed_cidr_blocks = ["10.0.0.0/8"]  # Internal only
```

**Estimated cost**: ~$1,500/month (all clouds combined)

### Geo-Distributed (Global Presence)

```hcl
# terraform.tfvars
project_name   = "laura-db-global"
environment    = "production"
gcp_project_id = "my-project-id"

# Different regions for each cloud
aws_region     = "us-east-1"      # North America
gcp_region     = "europe-west1"   # Europe
azure_location = "southeastasia"  # Asia

# Medium instances
aws_instance_count   = 2
gcp_instance_count   = 2
azure_instance_count = 2

enable_load_balancer = true
enable_backups       = true
enable_monitoring    = true
```

**Estimated cost**: ~$600/month

## Outputs

After deployment, Terraform provides comprehensive outputs:

### Endpoints

```bash
# View all endpoints
terraform output all_endpoints

# Individual cloud endpoints
terraform output aws_load_balancer
terraform output gcp_load_balancer
terraform output azure_load_balancer
```

### Health Checks

```bash
# Get health check commands
terraform output health_check_commands

# Test AWS
eval $(terraform output -json health_check_commands | jq -r '.aws')

# Test GCP
eval $(terraform output -json health_check_commands | jq -r '.gcp')

# Test Azure
eval $(terraform output -json health_check_commands | jq -r '.azure')
```

### Backup Locations

```bash
# View all backup locations
terraform output backup_locations

# AWS S3 bucket
terraform output aws_backup_bucket

# GCP Cloud Storage bucket
terraform output gcp_backup_bucket

# Azure Storage account
terraform output azure_storage_account
```

### Monitoring

```bash
# View all monitoring dashboards
terraform output monitoring_dashboards

# Open AWS CloudWatch
open $(terraform output -raw aws_monitoring_dashboard)

# Open GCP Monitoring
open $(terraform output -raw gcp_monitoring_console)

# Open Azure Monitor
open $(terraform output -json azure_portal_links | jq -r '.monitoring')
```

## Management Operations

### Scaling

#### Scale Up All Clouds

```hcl
# Update terraform.tfvars
aws_instance_count   = 5  # from 3
gcp_instance_count   = 5  # from 3
azure_instance_count = 5  # from 3

# Apply changes
terraform apply
```

#### Scale Individual Cloud

```hcl
# Scale only AWS
aws_instance_count = 10

# Keep others the same
gcp_instance_count   = 3
azure_instance_count = 3
```

### Updating LauraDB Version

```hcl
# Update terraform.tfvars
laura_db_version = "v1.2.3"  # or "latest"

# Apply to all clouds
terraform apply
```

### Selective Cloud Deployment

Deploy to specific clouds only:

```bash
# Deploy only to AWS
terraform apply -target=module.laura_db_aws

# Deploy only to GCP
terraform apply -target=module.laura_db_gcp

# Deploy only to Azure
terraform apply -target=module.laura_db_azure

# Deploy to AWS and GCP only
terraform apply -target=module.laura_db_aws -target=module.laura_db_gcp
```

### Destroying Resources

```bash
# Destroy all clouds
terraform destroy

# Destroy specific cloud
terraform destroy -target=module.laura_db_aws
terraform destroy -target=module.laura_db_gcp
terraform destroy -target=module.laura_db_azure
```

## Application-Level Multi-Cloud Setup

### DNS-Based Routing

Use DNS with health checks to route to nearest healthy endpoint:

```hcl
# Example with AWS Route 53 (add to your config)
resource "aws_route53_zone" "main" {
  name = "laura-db.example.com"
}

resource "aws_route53_health_check" "aws" {
  fqdn              = module.laura_db_aws.load_balancer_dns
  port              = var.laura_db_port
  type              = "HTTP"
  resource_path     = "/_health"
  failure_threshold = 3
  request_interval  = 30
}

resource "aws_route53_record" "aws" {
  zone_id = aws_route53_zone.main.zone_id
  name    = "aws.laura-db.example.com"
  type    = "A"
  ttl     = 60
  records = [module.laura_db_aws.load_balancer_ip]

  health_check_id = aws_route53_health_check.aws.id
}

# Similar for GCP and Azure
```

### Client-Side Load Balancing

Configure your application to use multiple endpoints:

```json
{
  "lauradb_endpoints": [
    "http://aws-lb-dns:8080",
    "http://gcp-lb-ip:8080",
    "http://azure-lb-ip:8080"
  ],
  "strategy": "round-robin",
  "health_check_interval": 30,
  "failover_timeout": 5
}
```

### Data Synchronization

For application-level replication:

```bash
# Backup from AWS
ssh ubuntu@aws-instance sudo /usr/local/bin/laura-db-backup

# Replicate to GCP
gsutil cp s3://aws-backup-bucket/backup.tar.gz gs://gcp-backup-bucket/

# Replicate to Azure
az storage blob copy start \
  --source-uri https://aws-backup-bucket.s3.amazonaws.com/backup.tar.gz \
  --destination-container backups \
  --destination-blob backup.tar.gz
```

## Cost Management

### View Costs by Cloud

```bash
# AWS (via AWS CLI)
aws ce get-cost-and-usage \
  --time-period Start=2025-01-01,End=2025-01-31 \
  --granularity MONTHLY \
  --metrics "UnblendedCost" \
  --filter file://aws-filter.json

# GCP (via gcloud)
gcloud billing accounts list
gcloud billing projects describe PROJECT_ID

# Azure (via Azure CLI)
az consumption usage list \
  --start-date 2025-01-01 \
  --end-date 2025-01-31
```

### Cost Optimization Tips

1. **Right-size instances**: Monitor actual usage
2. **Use Reserved Instances**: Commit for 1-3 years (up to 72% savings)
3. **Use Spot/Preemptible VMs**: For fault-tolerant workloads
4. **Implement auto-shutdown**: For dev/test environments
5. **Optimize storage**: Use appropriate tiers
6. **Review regularly**: Monthly cost review

### Estimated Monthly Costs

| Configuration | AWS | GCP | Azure | Total |
|--------------|-----|-----|-------|-------|
| **Dev (1 small VM per cloud)** | $25 | $20 | $30 | **$75** |
| **Small Prod (2 medium VMs)** | $200 | $180 | $220 | **$600** |
| **Medium Prod (3 large VMs)** | $400 | $350 | $500 | **$1,250** |
| **Large Prod (5 large VMs)** | $700 | $600 | $800 | **$2,100** |

*Prices are estimates based on standard pricing as of 2025*

## Monitoring All Clouds

### Unified Monitoring with Grafana

Set up Grafana to monitor all clouds:

```bash
# Install Grafana
docker run -d -p 3000:3000 grafana/grafana

# Add data sources
# - CloudWatch (AWS)
# - Google Cloud Monitoring (GCP)
# - Azure Monitor (Azure)

# Import dashboard template
# See monitoring/grafana/multi-cloud-dashboard.json
```

### Health Check Script

Monitor all endpoints:

```bash
#!/bin/bash
# check-all-clouds.sh

AWS_ENDPOINT=$(terraform output -raw aws_load_balancer)
GCP_ENDPOINT=$(terraform output -raw gcp_load_balancer)
AZURE_ENDPOINT=$(terraform output -raw azure_load_balancer)

echo "Checking AWS..."
curl -s "${AWS_ENDPOINT}/_health" | jq .

echo "Checking GCP..."
curl -s "${GCP_ENDPOINT}/_health" | jq .

echo "Checking Azure..."
curl -s "${AZURE_ENDPOINT}/_health" | jq .
```

## Troubleshooting

### Deployment Fails on Specific Cloud

```bash
# Check provider authentication
aws sts get-caller-identity    # AWS
gcloud auth list               # GCP
az account show                # Azure

# Re-authenticate if needed
aws configure
gcloud auth application-default login
az login

# Retry deployment for specific cloud
terraform apply -target=module.laura_db_aws
```

### Resource Quota Exceeded

```bash
# Check quotas
aws service-quotas list-service-quotas --service-code ec2
gcloud compute project-info describe --project YOUR_PROJECT
az vm list-usage --location eastus

# Request quota increase in respective cloud console
```

### High Costs

```bash
# Review Terraform state
terraform state list

# Check resources per cloud
terraform state list | grep aws
terraform state list | grep gcp
terraform state list | grep azure

# Identify expensive resources
# Scale down or destroy unused resources
```

## Security Best Practices

1. **Restrict CIDR blocks**: Don't use `0.0.0.0/0` in production
2. **Use VPN/Private connectivity**: Connect clouds privately
3. **Implement WAF**: Web Application Firewall on load balancers
4. **Enable encryption**: All storage and network traffic
5. **Rotate credentials**: Regular rotation of SSH keys and secrets
6. **Use secrets managers**: AWS Secrets Manager, GCP Secret Manager, Azure Key Vault
7. **Implement least privilege**: Minimal IAM/RBAC permissions
8. **Enable audit logging**: CloudTrail, Cloud Audit Logs, Activity Log

## Disaster Recovery

### Cross-Cloud Backup

```bash
# Backup from AWS to GCP
aws s3 sync s3://aws-backup-bucket gs://gcp-backup-bucket

# Backup from GCP to Azure
gsutil -m rsync -r gs://gcp-backup-bucket \
  az://azure-storage-account/backups

# Backup from Azure to AWS
az storage blob copy start-batch \
  --destination-container backups \
  --source-uri s3://aws-backup-bucket
```

### Failover Testing

```bash
# Simulate AWS failure
terraform destroy -target=module.laura_db_aws

# Verify GCP and Azure still serving
curl $(terraform output -raw gcp_load_balancer)/_health
curl $(terraform output -raw azure_load_balancer)/_health

# Restore AWS
terraform apply -target=module.laura_db_aws
```

## Clean Up

```bash
# Destroy all resources in all clouds
terraform destroy

# Confirm destruction (will prompt for each cloud)
# Type 'yes' to confirm

# Verify all resources are destroyed
terraform state list  # Should be empty
```

## Support

- **Module Documentation**:
  - [AWS Module](../../modules/aws/README.md)
  - [GCP Module](../../modules/gcp/README.md)
  - [Azure Module](../../modules/azure/README.md)
- **Cloud Deployment Guides**:
  - [AWS Guide](../../../docs/cloud/aws/README.md)
  - [GCP Guide](../../../docs/cloud/gcp/README.md)
  - [Azure Guide](../../../docs/cloud/azure/README.md)
- **Issues**: [GitHub Issues](https://github.com/mnohosten/laura-db/issues)

## References

- [Multi-Cloud Architecture Best Practices](https://cloud.google.com/architecture/hybrid-and-multi-cloud-architecture-patterns)
- [Terraform Multi-Provider](https://www.terraform.io/language/providers/requirements)
- [LauraDB Documentation](../../../README.md)

## Next Steps

1. Configure DNS for global routing
2. Set up cross-cloud monitoring dashboard
3. Implement automated failover
4. Configure backup synchronization
5. Set up cost alerts for all clouds
6. Document disaster recovery procedures
7. Test failover scenarios

---

**Note**: Multi-cloud deployments are complex. Start with single-cloud deployments and gradually expand to multi-cloud once you're comfortable with each provider's ecosystem.
