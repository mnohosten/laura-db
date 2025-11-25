# LauraDB GCP Example

Complete example for deploying LauraDB on Google Cloud Platform using Terraform.

## Overview

This example demonstrates:
- Multi-instance deployment with Cloud Load Balancer
- Cloud Storage backups with lifecycle management
- Cloud Monitoring and logging
- Customizable configuration via variables

## Prerequisites

1. **GCP Project** with billing enabled
2. **gcloud CLI** configured with credentials
3. **Terraform** >= 1.0 installed

## Quick Start

### 1. Configure GCP Authentication

```bash
# Install gcloud SDK
curl https://sdk.cloud.google.com | bash
exec -l $SHELL

# Authenticate
gcloud auth login
gcloud auth application-default login

# Set project
gcloud config set project YOUR_PROJECT_ID
```

### 2. Enable Required APIs

```bash
# Enable necessary GCP APIs
gcloud services enable compute.googleapis.com
gcloud services enable storage.googleapis.com
gcloud services enable monitoring.googleapis.com
gcloud services enable logging.googleapis.com
```

### 3. Customize Variables

Create `terraform.tfvars`:

```hcl
project_id   = "my-gcp-project-id"
project_name = "my-laura-db"
environment  = "production"
region       = "us-central1"

# Instances
machine_type   = "e2-standard-4"
instance_count = 3

# Networking
allowed_cidr_blocks = ["10.0.0.0/8"]  # Restrict access
ssh_public_key      = "ssh-rsa AAAAB3... your-key"

# Features
enable_load_balancer = true
enable_backups       = true
enable_monitoring    = true

# Contact
alert_email = "ops@example.com"

# Labels
labels = {
  team        = "platform"
  cost_center = "engineering"
}
```

### 4. Deploy

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

### 5. Access LauraDB

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
project_id   = "my-project-id"
environment  = "development"
machine_type = "e2-small"
instance_count = 1

enable_load_balancer = false
enable_backups       = false
enable_monitoring    = true
```

**Estimated cost**: ~$20/month

### Production (High Availability)

```hcl
# terraform.tfvars
project_id     = "my-project-id"
environment    = "production"
machine_type   = "e2-standard-4"
instance_count = 3
zones          = ["us-central1-a", "us-central1-b", "us-central1-c"]

enable_load_balancer = true
enable_backups       = true
backup_retention_days = 90

enable_monitoring = true
alert_email       = "ops@example.com"

disk_type    = "pd-ssd"
disk_size_gb = 200
```

**Estimated cost**: ~$350/month

### Auto-Scaling (Dynamic Workloads)

```hcl
# terraform.tfvars
project_id   = "my-project-id"
environment  = "production"
machine_type = "e2-medium"

enable_auto_scaling = true
min_instances       = 2
max_instances       = 10

enable_load_balancer = true
enable_backups       = true
enable_monitoring    = true
```

**Estimated cost**: ~$150-600/month (depending on load)

## Outputs

After deployment, Terraform provides:

| Output | Description |
|--------|-------------|
| `laura_db_endpoints` | Connection information |
| `public_ips` | Instance public IPs |
| `load_balancer_endpoint` | Load balancer URL |
| `backup_bucket` | Cloud Storage bucket name |
| `monitoring_console` | Cloud Monitoring console URL |
| `logs_explorer` | Cloud Logging explorer URL |
| `gcloud_ssh_command` | Command to SSH into instance |
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

# SSH to instance
eval $(terraform output -raw gcloud_ssh_command)

# Check service status
sudo systemctl status laura-db
```

### View Logs

```bash
# Cloud Logging
gcloud logging read "resource.type=gce_instance AND resource.labels.instance_id:laura-db" --limit 50 --format=json

# Or visit Logs Explorer
open $(terraform output -raw logs_explorer)
```

### Backup Management

```bash
# List backups
gsutil ls gs://$(terraform output -raw backup_bucket)/

# Download backup
gsutil cp gs://$(terraform output -raw backup_bucket)/backup-latest.tar.gz ./

# Upload backup
gsutil cp backup.tar.gz gs://$(terraform output -raw backup_bucket)/
```

## Updating Infrastructure

### Scale Up

```hcl
# Update terraform.tfvars
instance_count = 5  # from 3

# Apply changes
terraform apply
```

### Change Machine Type

```hcl
# Update terraform.tfvars
machine_type = "e2-standard-8"  # from e2-standard-4

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
terraform destroy -target=module.laura_db.google_compute_instance.laura_db
```

**Warning**: This will delete all data including backups if `force_destroy = true`.

## Customization

### Use Existing VPC

```hcl
# In main.tf, add to module:
module "laura_db" {
  # ... other config

  create_network  = false
  network_name    = "my-existing-network"
  subnetwork_name = "my-existing-subnet"
}
```

### Private Instances with Cloud NAT

```hcl
# In terraform.tfvars
assign_external_ip = false
allowed_cidr_blocks = ["10.0.0.0/8"]  # Internal only

# Cloud NAT will be automatically created
```

### Custom LauraDB Version

```hcl
# In terraform.tfvars
laura_db_version = "v1.2.3"  # or "latest"
```

### Enable Disk Encryption with Cloud KMS

```bash
# Create KMS key ring and key
gcloud kms keyrings create laura-db-keyring --location=us-central1
gcloud kms keys create laura-db-key --location=us-central1 --keyring=laura-db-keyring --purpose=encryption

# Get key ID
KMS_KEY=$(gcloud kms keys describe laura-db-key --location=us-central1 --keyring=laura-db-keyring --format="value(name)")

# In terraform.tfvars
kms_key_id = "projects/my-project/locations/us-central1/keyRings/laura-db-keyring/cryptoKeys/laura-db-key"
```

## Troubleshooting

### Deployment Fails

```bash
# Check Terraform logs
export TF_LOG=DEBUG
terraform apply

# Verify gcloud authentication
gcloud auth list
gcloud config list

# Check API enablement
gcloud services list --enabled
```

### Can't Connect to Instances

```bash
# Check firewall rules
gcloud compute firewall-rules list --filter="name~laura-db"

# Test connectivity
INSTANCE_IP=$(terraform output -json public_ips | jq -r '.[0]')
nc -zv $INSTANCE_IP 8080

# Check instance status
INSTANCE_NAME=$(terraform output -json instance_names | jq -r '.[0]')
gcloud compute instances describe $INSTANCE_NAME --format="value(status)"
```

### High Costs

```bash
# Review resources
terraform state list

# Check persistent disks
gcloud compute disks list --filter="labels.project=laura-db"

# Cloud Storage usage
gsutil du -sh gs://$(terraform output -raw backup_bucket)

# View cost breakdown in console
gcloud alpha billing accounts list
```

### Quota Issues

```bash
# Check quotas
gcloud compute project-info describe --project=YOUR_PROJECT_ID

# Request quota increase
# Visit: https://console.cloud.google.com/iam-admin/quotas
```

## Cost Optimization

1. **Use Committed Use Discounts**: Save up to 57% with 1 or 3-year commitments
2. **Enable Sustained Use Discounts**: Automatic up to 30% discount
3. **Use Preemptible VMs**: Up to 80% savings for fault-tolerant workloads
4. **Right-size instances**: Monitor usage with Cloud Monitoring
5. **Use pd-balanced**: Cheaper than pd-ssd for non-critical workloads
6. **Implement lifecycle policies**: Automatic tiering for old backups (included)
7. **Use Regional resources**: Cheaper than multi-regional

## Security Hardening

1. **Restrict CIDR blocks**: Update `allowed_cidr_blocks`
2. **Use private IPs**: Set `assign_external_ip = false`
3. **Enable VPC Service Controls**: Additional security perimeter
4. **Use Cloud KMS**: Customer-managed encryption keys
5. **Enable OS Login**: Better SSH key management (enabled by default)
6. **Implement Cloud Armor**: DDoS protection for load balancer
7. **Use Secret Manager**: For sensitive configuration

## Monitoring and Alerting

### View Metrics in Console

```bash
# Open Cloud Monitoring
open $(terraform output -raw monitoring_console)

# View specific instance metrics
gcloud monitoring time-series list \
  --filter='metric.type="compute.googleapis.com/instance/cpu/utilization"'
```

### Create Custom Alert

```hcl
# Add to your configuration
resource "google_monitoring_alert_policy" "disk_usage" {
  display_name = "LauraDB High Disk Usage"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Disk usage above 85%"

    condition_threshold {
      filter          = "resource.type = \"gce_instance\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.85

      aggregations {
        alignment_period   = "60s"
        per_series_aligner = "ALIGN_MEAN"
      }
    }
  }
}
```

## Backup and Restore

### Automated Backups

Backups run automatically via cron (configured in user-data script).

### Manual Backup

```bash
# SSH to instance
gcloud compute ssh <instance-name> --zone=<zone>

# Run backup manually
sudo /usr/local/bin/laura-db-backup

# Verify backup in Cloud Storage
gsutil ls gs://$(terraform output -raw backup_bucket)/
```

### Restore from Backup

```bash
# List available backups
gsutil ls gs://$(terraform output -raw backup_bucket)/ | sort

# Download backup
gsutil cp gs://$(terraform output -raw backup_bucket)/backup-TIMESTAMP.tar.gz ./

# Copy to instance
gcloud compute scp backup-TIMESTAMP.tar.gz <instance-name>:/tmp/ --zone=<zone>

# SSH to instance and restore
gcloud compute ssh <instance-name> --zone=<zone>
sudo systemctl stop laura-db
sudo tar -xzf /tmp/backup-TIMESTAMP.tar.gz -C /var/lib/laura-db
sudo chown -R laura-db:laura-db /var/lib/laura-db
sudo systemctl start laura-db
```

## Support

- **Module Documentation**: [../../modules/gcp/README.md](../../modules/gcp/README.md)
- **GCP Deployment Guide**: [../../../docs/cloud/gcp/README.md](../../../docs/cloud/gcp/README.md)
- **Issues**: [GitHub Issues](https://github.com/mnohosten/laura-db/issues)

## References

- [Terraform Google Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs)
- [GCE Best Practices](https://cloud.google.com/compute/docs/best-practices)
- [Cloud Storage Best Practices](https://cloud.google.com/storage/docs/best-practices)
- [LauraDB Documentation](../../../README.md)

## Cost Estimate Calculator

Use the [GCP Pricing Calculator](https://cloud.google.com/products/calculator) with these inputs:

- **Compute Engine**: Number of instances × machine type × hours
- **Persistent Disk**: Disk size × disk type
- **Cloud Storage**: Backup size × retention
- **Network**: Outbound traffic (typically 10-50GB/month)
- **Cloud Monitoring**: Log ingestion (typically 5-10GB/month)

Example production deployment (~$350/month):
- 3 × e2-standard-4 instances: ~$180/month
- 3 × 200GB pd-ssd disks: ~$120/month
- Cloud Storage (500GB): ~$10/month
- Load Balancer: ~$20/month
- Monitoring & Logging: ~$20/month
