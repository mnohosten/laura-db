# LauraDB GCP Terraform Module

Terraform module for deploying LauraDB on Google Cloud Platform (GCP).

## Features

- **Compute Engine Instances**: Configurable machine types and counts
- **VPC Networking**: Optional VPC creation or use existing
- **Managed Instance Groups**: Auto-scaling with health checks
- **Cloud Load Balancing**: Global HTTP(S) load balancer
- **Cloud Storage Backups**: Automated backups with lifecycle policies
- **Cloud Monitoring**: Comprehensive monitoring, logging, and alerting
- **Service Accounts**: Least-privilege IAM
- **Cloud NAT**: For private instances without external IPs

## Usage

### Basic Deployment

```hcl
module "laura_db" {
  source = "./modules/gcp"

  project_id   = "my-project-id"
  project_name = "laura-db"
  environment  = "production"
  region       = "us-central1"

  machine_type   = "e2-medium"
  instance_count = 2

  enable_backups    = true
  enable_monitoring = true
}
```

### Production Deployment with HA

```hcl
module "laura_db" {
  source = "./modules/gcp"

  project_id   = "my-project-id"
  project_name = "laura-db"
  environment  = "production"
  region       = "us-central1"

  # High availability
  machine_type   = "e2-standard-4"
  instance_count = 3
  zones          = ["us-central1-a", "us-central1-b", "us-central1-c"]

  # Load balancer
  enable_load_balancer = true

  # Networking
  network_cidr = "10.0.0.0/16"
  allowed_cidr_blocks = ["10.0.0.0/8"]

  # Storage
  disk_type    = "pd-ssd"
  disk_size_gb = 200

  # Encryption
  kms_key_id = "projects/my-project/locations/us-central1/keyRings/my-keyring/cryptoKeys/my-key"

  # Backups
  enable_backups        = true
  backup_retention_days = 90

  # Monitoring
  enable_monitoring = true
  alert_email       = "ops@example.com"

  # Labels
  labels = {
    team        = "platform"
    cost_center = "engineering"
    compliance  = "hipaa"
  }
}
```

### Auto-Scaling Deployment

```hcl
module "laura_db" {
  source = "./modules/gcp"

  project_id   = "my-project-id"
  project_name = "laura-db"
  environment  = "production"
  region       = "us-central1"

  # Auto-scaling configuration
  enable_auto_scaling = true
  machine_type        = "e2-medium"
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
| google | ~> 5.0 |

## Providers

| Name | Version |
|------|---------|
| google | ~> 5.0 |

## Inputs

### Required

| Name | Description | Type |
|------|-------------|------|
| project_id | GCP project ID | string |
| project_name | Name of the project | string |

### Optional

| Name | Description | Type | Default |
|------|-------------|------|---------|
| environment | Environment name | string | `"production"` |
| region | GCP region | string | `"us-central1"` |
| machine_type | GCE machine type | string | `"e2-medium"` |
| instance_count | Number of instances | number | `1` |
| disk_type | Persistent disk type | string | `"pd-balanced"` |
| disk_size_gb | Disk size (GB) | number | `100` |
| create_network | Create new VPC | bool | `true` |
| network_cidr | VPC CIDR block | string | `"10.0.0.0/16"` |
| enable_load_balancer | Enable Cloud Load Balancer | bool | `false` |
| enable_auto_scaling | Enable auto-scaling | bool | `false` |
| enable_backups | Enable Cloud Storage backups | bool | `true` |
| enable_monitoring | Enable Cloud Monitoring | bool | `true` |
| laura_db_port | LauraDB HTTP port | number | `8080` |

See [variables.tf](./variables.tf) for complete list.

## Outputs

| Name | Description |
|------|-------------|
| instance_ids | GCE instance IDs |
| instance_names | Instance names |
| public_ips | Public IP addresses |
| private_ips | Private IP addresses |
| load_balancer_ip | Load balancer IP |
| load_balancer_endpoint | Full LB endpoint URL |
| backup_bucket_name | Cloud Storage bucket name |
| service_account_email | Service account email |
| network_name | VPC network name |
| connection_info | Connection details |
| deployment_summary | Deployment summary |

See [outputs.tf](./outputs.tf) for complete list.

## Examples

### 1. Single Instance (Development)

```hcl
module "laura_db_dev" {
  source = "./modules/gcp"

  project_id   = "my-project-id"
  project_name = "laura-db-dev"
  environment  = "development"
  machine_type = "e2-small"

  enable_backups    = false
  enable_monitoring = true
}
```

### 2. Multi-Instance with Load Balancer

```hcl
module "laura_db_prod" {
  source = "./modules/gcp"

  project_id     = "my-project-id"
  project_name   = "laura-db-prod"
  environment    = "production"
  instance_count = 3
  machine_type   = "e2-standard-4"

  enable_load_balancer = true
  enable_backups       = true
  enable_monitoring    = true
}
```

### 3. Using Existing VPC

```hcl
module "laura_db" {
  source = "./modules/gcp"

  project_id   = "my-project-id"
  project_name = "laura-db"

  # Use existing VPC
  create_network  = false
  network_name    = "my-existing-network"
  subnetwork_name = "my-existing-subnet"

  instance_count = 2
}
```

### 4. Private Instances with Cloud NAT

```hcl
module "laura_db" {
  source = "./modules/gcp"

  project_id   = "my-project-id"
  project_name = "laura-db"

  # No external IPs
  assign_external_ip = false

  # Restrict access
  allowed_cidr_blocks = ["10.0.0.0/8"]

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

# Direct instance connection
curl http://$(terraform output -json public_ips | jq -r '.[0]'):8080/_health
```

### SSH Access

```bash
# Get instance name
INSTANCE_NAME=$(terraform output -json instance_names | jq -r '.[0]')
ZONE=$(gcloud compute instances list --filter="name=$INSTANCE_NAME" --format="value(zone)")

# SSH using gcloud (uses OS Login)
gcloud compute ssh $INSTANCE_NAME --zone=$ZONE

# Check LauraDB status
sudo systemctl status laura-db
```

### View Logs

```bash
# Cloud Logging
gcloud logging read "resource.type=gce_instance AND resource.labels.instance_id:laura-db" --limit 50

# Or visit Logs Explorer
open $(terraform output -raw logs_explorer)
```

### Backup Management

```bash
# List backups
gsutil ls gs://$(terraform output -raw backup_bucket_name)/

# Download backup
gsutil cp gs://$(terraform output -raw backup_bucket_name)/backup.tar.gz ./

# Upload backup
gsutil cp backup.tar.gz gs://$(terraform output -raw backup_bucket_name)/
```

## Monitoring

### Cloud Monitoring Metrics

The module automatically publishes metrics to Cloud Monitoring:

- CPU Utilization
- Memory Usage
- Disk I/O
- Network Traffic
- LauraDB-specific metrics

### Alerts

Configure additional Cloud Monitoring alerts:

```hcl
resource "google_monitoring_alert_policy" "memory" {
  display_name = "${module.laura_db.deployment_summary.project_name} - High Memory"
  project      = var.project_id
  combiner     = "OR"

  conditions {
    display_name = "Memory usage above 80%"

    condition_threshold {
      filter          = "resource.type = \"gce_instance\""
      duration        = "300s"
      comparison      = "COMPARISON_GT"
      threshold_value = 0.8
    }
  }
}
```

## Backup and Recovery

### Manual Backup

```bash
# SSH to instance
gcloud compute ssh <instance-name> --zone=<zone>

# Run backup
sudo /usr/local/bin/laura-db-backup

# Backups are automatically uploaded to Cloud Storage
```

### Restore from Backup

```bash
# Download backup
gsutil cp gs://<bucket>/backup.tar.gz ./

# Copy to instance
gcloud compute scp backup.tar.gz <instance-name>:/tmp/ --zone=<zone>

# On instance
sudo systemctl stop laura-db
sudo tar -xzf /tmp/backup.tar.gz -C /var/lib/laura-db
sudo systemctl start laura-db
```

## Scaling

### Vertical Scaling (Resize Instance)

```hcl
# Update machine_type in your config
machine_type = "e2-standard-8"  # from e2-standard-4

# Apply changes
terraform apply
```

### Horizontal Scaling (Add Instances)

```hcl
# Increase instance count
instance_count = 5  # from 3

# Enable load balancer if not already
enable_load_balancer = true

# Apply changes
terraform apply
```

## Cost Optimization

### 1. Use Committed Use Discounts

Purchase 1 or 3-year commitments for predictable workloads (up to 57% savings).

### 2. Use Preemptible VMs

For fault-tolerant workloads (up to 80% savings):

```hcl
# Add to instance template
resource "google_compute_instance_template" "laura_db_preemptible" {
  scheduling {
    preemptible       = true
    automatic_restart = false
  }
}
```

### 3. Right-Size Instances

Monitor Cloud Monitoring metrics and adjust `machine_type` based on actual usage.

### 4. Use Sustained Use Discounts

Automatically applied - up to 30% off for resources used > 25% of month.

### 5. Optimize Storage

```hcl
# Use pd-balanced instead of pd-ssd for non-critical workloads
disk_type = "pd-balanced"  # ~40% cheaper than pd-ssd
```

## Security Best Practices

1. **Restrict CIDR blocks**: Don't use `0.0.0.0/0` in production
2. **Use private IPs**: Set `assign_external_ip = false` and use Cloud NAT
3. **Enable encryption**: Use Cloud KMS for disk encryption
4. **IAM least privilege**: Module creates minimal service account permissions
5. **Use Secret Manager**: Store sensitive configuration
6. **Enable VPC Service Controls**: For additional security perimeter
7. **Use Organization Policies**: Enforce security requirements

## Troubleshooting

### Instance Won't Start

```bash
# Check instance status
gcloud compute instances describe <instance-name> --zone=<zone>

# View serial port output
gcloud compute instances get-serial-port-output <instance-name> --zone=<zone>

# Check startup script log
gcloud compute ssh <instance-name> --zone=<zone> -- sudo cat /var/log/laura-db-setup.log
```

### Can't Connect

```bash
# Check firewall rules
gcloud compute firewall-rules list --filter="name~laura-db"

# Verify port is open
nc -zv <instance-ip> 8080

# Check service status
gcloud compute ssh <instance-name> --zone=<zone> -- sudo systemctl status laura-db
```

### High Costs

```bash
# Review resources
terraform state list

# Check persistent disks
gcloud compute disks list --filter="labels.project=laura-db"

# Review Cloud Storage bucket size
gsutil du -sh gs://<bucket>
```

## Migration

### From Manual Setup

```bash
# Import existing resources
terraform import module.laura_db.google_compute_instance.laura_db[0] projects/<project>/zones/<zone>/instances/<name>
terraform import module.laura_db.google_compute_firewall.laura_db projects/<project>/global/firewalls/<name>

# Run plan to see differences
terraform plan
```

## References

- [GCE Documentation](https://cloud.google.com/compute/docs)
- [Cloud Storage Documentation](https://cloud.google.com/storage/docs)
- [Cloud Monitoring Documentation](https://cloud.google.com/monitoring/docs)
- [LauraDB Documentation](../../../README.md)
- [LauraDB GCP Deployment Guide](../../../docs/cloud/gcp/)

## Support

For issues or questions:

- [GitHub Issues](https://github.com/mnohosten/laura-db/issues)
- [GCP Deployment Guide](../../../docs/cloud/gcp/README.md)

## License

Same as LauraDB project.
