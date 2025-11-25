# LauraDB AWS Terraform Module

Terraform module for deploying LauraDB on Amazon Web Services (AWS).

## Features

- **EC2 Instances**: Configurable instance types and counts
- **VPC Networking**: Optional VPC creation or use existing
- **Auto Scaling**: Optional Auto Scaling Group support
- **Load Balancing**: Application Load Balancer for HA
- **S3 Backups**: Automated backups to S3 with lifecycle policies
- **CloudWatch Monitoring**: Comprehensive monitoring and logging
- **IAM Roles**: Least-privilege security
- **Elastic IPs**: Optional static IPs for instances

## Usage

### Basic Deployment

```hcl
module "laura_db" {
  source = "./modules/aws"

  project_name = "laura-db"
  environment  = "production"
  region       = "us-east-1"

  instance_type  = "t3.medium"
  instance_count = 2

  enable_backups    = true
  enable_monitoring = true
}
```

### Production Deployment with HA

```hcl
module "laura_db" {
  source = "./modules/aws"

  project_name = "laura-db"
  environment  = "production"
  region       = "us-east-1"

  # High availability
  instance_type  = "t3.large"
  instance_count = 3
  availability_zones = ["us-east-1a", "us-east-1b", "us-east-1c"]

  # Load balancer
  enable_load_balancer = true

  # Networking
  vpc_cidr = "10.0.0.0/16"
  allowed_cidr_blocks = ["10.0.0.0/8", "172.16.0.0/12"]

  # Storage
  volume_type = "gp3"
  volume_size = 200

  # Backups
  enable_backups = true
  backup_retention_days = 90

  # Monitoring
  enable_monitoring = true
  log_retention_days = 90
  alert_email = "ops@example.com"

  # Tags
  tags = {
    Team        = "Platform"
    CostCenter  = "Engineering"
    Compliance  = "HIPAA"
  }
}
```

### Auto-Scaling Deployment

```hcl
module "laura_db" {
  source = "./modules/aws"

  project_name = "laura-db"
  environment  = "production"
  region       = "us-east-1"

  # Auto-scaling configuration
  enable_auto_scaling = true
  instance_type       = "t3.medium"
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
| aws | ~> 5.0 |

## Providers

| Name | Version |
|------|---------|
| aws | ~> 5.0 |

## Inputs

### Required

| Name | Description | Type |
|------|-------------|------|
| project_name | Name of the project | string |

### Optional

| Name | Description | Type | Default |
|------|-------------|------|---------|
| environment | Environment name | string | `"production"` |
| region | AWS region | string | `"us-east-1"` |
| instance_type | EC2 instance type | string | `"t3.medium"` |
| instance_count | Number of instances | number | `1` |
| volume_type | EBS volume type | string | `"gp3"` |
| volume_size | EBS volume size (GB) | number | `100` |
| create_vpc | Create new VPC | bool | `true` |
| vpc_cidr | VPC CIDR block | string | `"10.0.0.0/16"` |
| enable_load_balancer | Enable ALB | bool | `false` |
| enable_auto_scaling | Enable Auto Scaling | bool | `false` |
| enable_backups | Enable S3 backups | bool | `true` |
| enable_monitoring | Enable CloudWatch | bool | `true` |
| laura_db_port | LauraDB HTTP port | number | `8080` |

See [variables.tf](./variables.tf) for complete list.

## Outputs

| Name | Description |
|------|-------------|
| instance_ids | EC2 instance IDs |
| public_ips | Public IP addresses |
| private_ips | Private IP addresses |
| load_balancer_dns | Load balancer DNS name |
| load_balancer_endpoint | Full LB endpoint URL |
| backup_bucket_name | S3 bucket name |
| security_group_id | Security group ID |
| vpc_id | VPC ID |
| connection_info | Connection details |
| deployment_summary | Deployment summary |

See [outputs.tf](./outputs.tf) for complete list.

## Examples

### 1. Single Instance (Development)

```hcl
module "laura_db_dev" {
  source = "./modules/aws"

  project_name  = "laura-db-dev"
  environment   = "development"
  instance_type = "t3.small"

  enable_backups    = false
  enable_monitoring = true
}
```

### 2. Multi-Instance with Load Balancer

```hcl
module "laura_db_prod" {
  source = "./modules/aws"

  project_name   = "laura-db-prod"
  environment    = "production"
  instance_count = 3
  instance_type  = "t3.large"

  enable_load_balancer = true
  enable_backups       = true
  enable_monitoring    = true
}
```

### 3. Using Existing VPC

```hcl
module "laura_db" {
  source = "./modules/aws"

  project_name = "laura-db"

  # Use existing VPC
  create_vpc = false
  vpc_id     = "vpc-1234567890abcdef0"
  subnet_ids = [
    "subnet-1234567890abcdef0",
    "subnet-0987654321fedcba0"
  ]

  instance_count = 2
}
```

### 4. With Custom Security

```hcl
module "laura_db" {
  source = "./modules/aws"

  project_name = "laura-db"

  # Restrict access
  allowed_cidr_blocks = ["10.0.0.0/8"]

  # Use SSH key
  ssh_public_key = file("~/.ssh/id_rsa.pub")

  # Use Elastic IPs
  enable_elastic_ips = true

  instance_count = 2
}
```

## Post-Deployment

### Connect to LauraDB

```bash
# Get endpoint from outputs
terraform output connection_info

# Using load balancer
curl http://<load-balancer-dns>:8080/_health

# Direct instance connection
curl http://<public-ip>:8080/_health
```

### SSH Access

```bash
# Get public IP
INSTANCE_IP=$(terraform output -json public_ips | jq -r '.[0]')

# SSH into instance
ssh ubuntu@$INSTANCE_IP

# Check LauraDB status
sudo systemctl status laura-db
```

### View Logs

```bash
# CloudWatch Logs
aws logs tail $(terraform output -raw cloudwatch_log_group) --follow

# Or via AWS Console
# Visit the monitoring_dashboard URL from outputs
```

### Backup Management

```bash
# List backups
aws s3 ls s3://$(terraform output -raw backup_bucket_name)/

# Download backup
aws s3 cp s3://$(terraform output -raw backup_bucket_name)/backup.tar.gz ./

# Upload backup
aws s3 cp backup.tar.gz s3://$(terraform output -raw backup_bucket_name)/
```

## Monitoring

### CloudWatch Metrics

The module automatically publishes metrics to CloudWatch:

- CPU Utilization
- Memory Usage
- Disk I/O
- Network Traffic
- LauraDB-specific metrics

### Alarms

Configure CloudWatch alarms:

```hcl
resource "aws_cloudwatch_metric_alarm" "cpu" {
  alarm_name          = "${module.laura_db.deployment_summary.project_name}-high-cpu"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "CPUUtilization"
  namespace           = "AWS/EC2"
  period              = "300"
  statistic           = "Average"
  threshold           = "80"

  dimensions = {
    InstanceId = module.laura_db.instance_ids[0]
  }

  alarm_actions = [aws_sns_topic.alerts.arn]
}
```

## Backup and Recovery

### Manual Backup

```bash
# SSH to instance
ssh ubuntu@<instance-ip>

# Run backup
sudo /usr/local/bin/laura-db-backup

# Backups are automatically uploaded to S3
```

### Restore from Backup

```bash
# Download backup
aws s3 cp s3://<bucket>/backup.tar.gz ./

# SSH to instance
scp backup.tar.gz ubuntu@<instance-ip>:/tmp/

# On instance
sudo systemctl stop laura-db
sudo tar -xzf /tmp/backup.tar.gz -C /var/lib/laura-db
sudo systemctl start laura-db
```

## Scaling

### Vertical Scaling (Resize Instance)

```hcl
# Update instance_type in your config
instance_type = "t3.xlarge"  # from t3.large

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

### 1. Use Spot Instances

```hcl
# Add to launch template
resource "aws_launch_template" "laura_db_spot" {
  instance_market_options {
    market_type = "spot"
    spot_options {
      max_price = "0.05"
    }
  }
}
```

### 2. Right-Size Instances

Monitor CloudWatch metrics and adjust `instance_type` based on actual usage.

### 3. Use S3 Intelligent-Tiering

```hcl
resource "aws_s3_bucket_intelligent_tiering_configuration" "backups" {
  bucket = module.laura_db.backup_bucket_name
  name   = "EntireBucket"

  tiering {
    access_tier = "ARCHIVE_ACCESS"
    days        = 90
  }
}
```

## Security Best Practices

1. **Restrict CIDR blocks**: Don't use `0.0.0.0/0` in production
2. **Use private subnets**: Deploy in private subnets with NAT gateway
3. **Enable encryption**: All volumes and S3 buckets are encrypted by default
4. **IAM least privilege**: Module creates minimal IAM policies
5. **Use AWS Secrets Manager**: Store sensitive configuration
6. **Enable AWS GuardDuty**: For threat detection
7. **Use VPC endpoints**: For private S3 and CloudWatch access

## Troubleshooting

### Instance Won't Start

```bash
# Check instance status
aws ec2 describe-instances --instance-ids <instance-id>

# View system log
aws ec2 get-console-output --instance-id <instance-id>

# Check user-data log
ssh ubuntu@<ip> sudo cat /var/log/laura-db-setup.log
```

### Can't Connect

```bash
# Check security group
aws ec2 describe-security-groups --group-ids <sg-id>

# Verify port is open
nc -zv <instance-ip> 8080

# Check service status
ssh ubuntu@<ip> sudo systemctl status laura-db
```

### High Costs

```bash
# Review resources
terraform state list

# Check EBS volumes
aws ec2 describe-volumes

# Review S3 bucket size
aws s3 ls s3://<bucket> --recursive --human-readable --summarize
```

## Migration

### From Manual Setup

```bash
# Import existing resources
terraform import module.laura_db.aws_instance.laura_db[0] i-1234567890abcdef0
terraform import module.laura_db.aws_security_group.laura_db sg-1234567890abcdef0

# Run plan to see differences
terraform plan
```

## References

- [AWS EC2 Documentation](https://docs.aws.amazon.com/ec2/)
- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
- [AWS CloudWatch Documentation](https://docs.aws.amazon.com/cloudwatch/)
- [LauraDB Documentation](../../../README.md)
- [LauraDB AWS Deployment Guide](../../../docs/cloud/aws/)

## Support

For issues or questions:

- [GitHub Issues](https://github.com/mnohosten/laura-db/issues)
- [AWS Deployment Guide](../../../docs/cloud/aws/README.md)

## License

Same as LauraDB project.
