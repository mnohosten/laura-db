# LauraDB AWS Example

Complete example for deploying LauraDB on AWS using Terraform.

## Overview

This example demonstrates:
- Multi-instance deployment with load balancer
- S3 backups with lifecycle management
- CloudWatch monitoring and logging
- Customizable configuration via variables

## Prerequisites

1. **AWS Account** with appropriate permissions
2. **AWS CLI** configured with credentials
3. **Terraform** >= 1.0 installed

## Quick Start

### 1. Configure AWS Credentials

```bash
# Using AWS CLI
aws configure

# Or set environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"
```

### 2. Customize Variables

Create `terraform.tfvars`:

```hcl
project_name  = "my-laura-db"
environment   = "production"
region        = "us-east-1"

# Instances
instance_type  = "t3.large"
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
environment    = "development"
instance_type  = "t3.small"
instance_count = 1

enable_load_balancer = false
enable_backups       = false
enable_monitoring    = true
```

**Estimated cost**: ~$25/month

### Production (High Availability)

```hcl
# terraform.tfvars
environment    = "production"
instance_type  = "t3.large"
instance_count = 3

enable_load_balancer = true
enable_backups       = true
backup_retention_days = 90

enable_monitoring  = true
log_retention_days = 90

volume_type = "gp3"
volume_size = 200
```

**Estimated cost**: ~$400/month

## Outputs

After deployment, Terraform provides:

| Output | Description |
|--------|-------------|
| `laura_db_endpoints` | Connection information |
| `public_ips` | Instance public IPs |
| `load_balancer_endpoint` | Load balancer URL |
| `backup_bucket` | S3 bucket name |
| `monitoring_dashboard` | CloudWatch dashboard URL |
| `ssh_command` | Command to SSH into instance |
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
eval $(terraform output -raw ssh_command)

# Check service status
sudo systemctl status laura-db
```

### View Logs

```bash
# CloudWatch Logs
aws logs tail /aws/laura-db/$(terraform output -json deployment_summary | jq -r '.project_name')-$(terraform output -json deployment_summary | jq -r '.environment') --follow

# Or visit dashboard
open $(terraform output -raw monitoring_dashboard)
```

### Backup Management

```bash
# List backups
aws s3 ls s3://$(terraform output -raw backup_bucket)/

# Download backup
aws s3 cp s3://$(terraform output -raw backup_bucket)/backup-latest.tar.gz ./
```

## Updating Infrastructure

### Scale Up

```hcl
# Update terraform.tfvars
instance_count = 5  # from 3

# Apply changes
terraform apply
```

### Change Instance Type

```hcl
# Update terraform.tfvars
instance_type = "t3.xlarge"  # from t3.large

# Apply changes
terraform apply
```

## Cleanup

```bash
# Destroy all resources
terraform destroy

# Or destroy specific resources
terraform destroy -target=module.laura_db.aws_instance.laura_db
```

## Customization

### Use Existing VPC

```hcl
# In main.tf, add to module:
module "laura_db" {
  # ... other config

  create_vpc = false
  vpc_id     = "vpc-1234567890abcdef0"
  subnet_ids = [
    "subnet-1234567890abcdef0",
    "subnet-0987654321fedcba0"
  ]
}
```

### Add Auto-Scaling

```hcl
# In main.tf, add to module:
module "laura_db" {
  # ... other config

  enable_auto_scaling = true
  min_instances       = 2
  max_instances       = 10
}
```

### Custom LauraDB Version

```hcl
# In terraform.tfvars
laura_db_version = "v1.2.3"  # or "latest"
```

## Troubleshooting

### Deployment Fails

```bash
# Check Terraform logs
export TF_LOG=DEBUG
terraform apply

# Verify AWS credentials
aws sts get-caller-identity

# Check AWS service quotas
aws service-quotas list-service-quotas --service-code ec2
```

### Can't Connect to Instances

```bash
# Check security group
aws ec2 describe-security-groups \
  --group-ids $(terraform output -json deployment_summary | jq -r '.security_group_id')

# Test connectivity
nc -zv <instance-ip> 8080

# Check instance status
aws ec2 describe-instance-status \
  --instance-ids $(terraform output -json instance_ids | jq -r '.[0]')
```

### High Costs

```bash
# Review resources
terraform state list

# Check EBS volumes
aws ec2 describe-volumes --filters "Name=tag:Project,Values=$(terraform output -json deployment_summary | jq -r '.project_name')"

# S3 storage usage
aws s3 ls s3://$(terraform output -raw backup_bucket) --recursive --human-readable --summarize
```

## Cost Optimization

1. **Use Reserved Instances** for predictable workloads
2. **Enable auto-scaling** to match demand
3. **Use gp3 volumes** instead of gp2
4. **Implement S3 lifecycle policies** (included by default)
5. **Right-size instances** based on monitoring data
6. **Use Spot Instances** for non-production

## Security Hardening

1. **Restrict CIDR blocks**: Update `allowed_cidr_blocks`
2. **Use private subnets**: Deploy in private subnets with NAT gateway
3. **Enable AWS GuardDuty**: For threat detection
4. **Use AWS Secrets Manager**: For sensitive config
5. **Enable VPC Flow Logs**: For network monitoring
6. **Implement WAF**: Add AWS WAF to load balancer

## Support

- **Module Documentation**: [../../modules/aws/README.md](../../modules/aws/README.md)
- **AWS Deployment Guide**: [../../../docs/cloud/aws/README.md](../../../docs/cloud/aws/README.md)
- **Issues**: [GitHub Issues](https://github.com/mnohosten/laura-db/issues)

## References

- [Terraform AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [AWS EC2 Best Practices](https://docs.aws.amazon.com/AWSEC2/latest/UserGuide/ec2-best-practices.html)
- [LauraDB Documentation](../../../README.md)
