# LauraDB Deployment on AWS EC2

This guide provides detailed instructions for deploying LauraDB on Amazon EC2 instances.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture Options](#architecture-options)
- [Single Instance Deployment](#single-instance-deployment)
- [Multi-Instance Deployment](#multi-instance-deployment)
- [Auto Scaling Setup](#auto-scaling-setup)
- [Storage Configuration](#storage-configuration)
- [Network Configuration](#network-configuration)
- [Security Best Practices](#security-best-practices)
- [Monitoring and Logging](#monitoring-and-logging)
- [Backup and Recovery](#backup-and-recovery)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

LauraDB can be deployed on EC2 instances in several configurations:
- **Single Instance**: Development and testing
- **Multi-Instance**: Production with load balancing
- **Auto Scaling**: Dynamic scaling based on load

## Prerequisites

### AWS Account Setup

1. **AWS Account** with appropriate permissions
2. **IAM User** with the following permissions:
   - EC2 full access
   - VPC management
   - EBS volume management
   - Security group management
   - CloudWatch access

3. **AWS CLI** installed and configured:
   ```bash
   aws configure
   ```

### Local Requirements

- AWS CLI 2.x
- SSH key pair for EC2 access
- Basic understanding of EC2, VPC, and security groups

## Architecture Options

### Option 1: Single Instance (Development)

```
┌─────────────────────────────────┐
│         Internet Gateway         │
└────────────┬────────────────────┘
             │
      ┌──────▼──────┐
      │   Public    │
      │   Subnet    │
      │             │
      │  ┌───────┐  │
      │  │  EC2  │  │
      │  │LauraDB│  │
      │  │  +EBS │  │
      │  └───────┘  │
      └─────────────┘
```

**Use Case**: Development, testing, proof of concept

**Specs**:
- Instance Type: t3.medium (2 vCPU, 4 GB RAM)
- Storage: 50 GB EBS gp3
- Estimated Cost: ~$30/month

### Option 2: Multi-Instance with Load Balancer (Production)

```
┌──────────────────────────────────────────┐
│          Application Load Balancer        │
└──────┬──────────────────┬────────────────┘
       │                  │
   ┌───▼────┐        ┌───▼────┐
   │  AZ 1  │        │  AZ 2  │
   │        │        │        │
   │ ┌────┐ │        │ ┌────┐ │
   │ │EC2 │ │        │ │EC2 │ │
   │ │+EBS│ │        │ │+EBS│ │
   │ └────┘ │        │ └────┘ │
   └────────┘        └────────┘
         │                │
         └────────┬───────┘
              ┌───▼───┐
              │  EFS  │
              │(shared)│
              └───────┘
```

**Use Case**: Production workloads, high availability

**Specs**:
- Instance Type: t3.large or m5.large
- Storage: 100-500 GB EBS gp3 per instance + EFS for shared data
- Load Balancer: Application Load Balancer
- Estimated Cost: ~$200-400/month

## Single Instance Deployment

### Step 1: Create Security Group

```bash
# Create security group
aws ec2 create-security-group \
  --group-name laura-db-sg \
  --description "Security group for LauraDB" \
  --vpc-id vpc-xxxxxxxxx

# Add inbound rules
# SSH access
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxxxxxx \
  --protocol tcp \
  --port 22 \
  --cidr 0.0.0.0/0

# LauraDB HTTP API
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxxxxxx \
  --protocol tcp \
  --port 8080 \
  --cidr 0.0.0.0/0
```

### Step 2: Launch EC2 Instance

```bash
# Launch instance
aws ec2 run-instances \
  --image-id ami-0c55b159cbfafe1f0 \
  --instance-type t3.medium \
  --key-name your-key-pair \
  --security-group-ids sg-xxxxxxxxx \
  --subnet-id subnet-xxxxxxxxx \
  --block-device-mappings '[
    {
      "DeviceName": "/dev/xvda",
      "Ebs": {
        "VolumeSize": 50,
        "VolumeType": "gp3",
        "Iops": 3000,
        "Throughput": 125,
        "DeleteOnTermination": false
      }
    }
  ]' \
  --tag-specifications 'ResourceType=instance,Tags=[{Key=Name,Value=laura-db-server}]' \
  --user-data file://user-data.sh
```

### Step 3: Create User Data Script

Create `user-data.sh`:

```bash
#!/bin/bash

# Update system
yum update -y

# Install required packages
yum install -y wget git

# Install Go 1.25
wget https://go.dev/dl/go1.25.4.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go1.25.4.linux-amd64.tar.gz
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
source /etc/profile

# Create laura-db user
useradd -m -s /bin/bash laura-db

# Clone and build LauraDB
cd /opt
git clone https://github.com/mnohosten/laura-db.git
cd laura-db
/usr/local/go/bin/go build -o bin/laura-server cmd/server/main.go

# Create data directory
mkdir -p /var/lib/laura-db
chown -R laura-db:laura-db /var/lib/laura-db
chown -R laura-db:laura-db /opt/laura-db

# Create systemd service
cat > /etc/systemd/system/laura-db.service <<'EOF'
[Unit]
Description=LauraDB Server
After=network.target

[Service]
Type=simple
User=laura-db
Group=laura-db
WorkingDirectory=/opt/laura-db
ExecStart=/opt/laura-db/bin/laura-server -port 8080 -data-dir /var/lib/laura-db
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Start service
systemctl daemon-reload
systemctl enable laura-db
systemctl start laura-db

echo "LauraDB installation complete"
```

### Step 4: Connect and Verify

```bash
# Get instance public IP
aws ec2 describe-instances \
  --filters "Name=tag:Name,Values=laura-db-server" \
  --query 'Reservations[0].Instances[0].PublicIpAddress' \
  --output text

# SSH to instance
ssh -i your-key-pair.pem ec2-user@<public-ip>

# Check service status
sudo systemctl status laura-db

# Test API
curl http://localhost:8080/_health
```

### Step 5: Access LauraDB

```bash
# From your local machine
export LAURA_DB_HOST=<public-ip>

# Access admin console
open http://$LAURA_DB_HOST:8080/

# Test API
curl http://$LAURA_DB_HOST:8080/_health
```

## Multi-Instance Deployment

### Step 1: Create Application Load Balancer

```bash
# Create target group
aws elbv2 create-target-group \
  --name laura-db-targets \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-xxxxxxxxx \
  --health-check-enabled \
  --health-check-path /_health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3

# Create load balancer
aws elbv2 create-load-balancer \
  --name laura-db-alb \
  --subnets subnet-xxxxxxxx subnet-yyyyyyyy \
  --security-groups sg-xxxxxxxxx \
  --scheme internet-facing \
  --type application

# Create listener
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:... \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:...
```

### Step 2: Create EFS for Shared Storage

```bash
# Create EFS file system
aws efs create-file-system \
  --performance-mode generalPurpose \
  --throughput-mode bursting \
  --encrypted \
  --tags Key=Name,Value=laura-db-shared

# Create mount targets in each AZ
aws efs create-mount-target \
  --file-system-id fs-xxxxxxxxx \
  --subnet-id subnet-xxxxxxxx \
  --security-groups sg-xxxxxxxxx

aws efs create-mount-target \
  --file-system-id fs-xxxxxxxxx \
  --subnet-id subnet-yyyyyyyy \
  --security-groups sg-xxxxxxxxx
```

### Step 3: Update User Data for EFS

```bash
#!/bin/bash

# Previous setup steps...

# Install EFS utilities
yum install -y amazon-efs-utils

# Mount EFS
mkdir -p /mnt/efs
echo "fs-xxxxxxxxx:/ /mnt/efs efs _netdev,tls 0 0" >> /etc/fstab
mount -a

# Create data directory on EFS
mkdir -p /mnt/efs/laura-db
chown -R laura-db:laura-db /mnt/efs/laura-db

# Update service to use EFS
sed -i 's|/var/lib/laura-db|/mnt/efs/laura-db|g' /etc/systemd/system/laura-db.service

# Restart service
systemctl daemon-reload
systemctl restart laura-db
```

### Step 4: Launch Multiple Instances

```bash
# Launch instances in multiple AZs
for subnet in subnet-xxxxxxxx subnet-yyyyyyyy; do
  aws ec2 run-instances \
    --image-id ami-0c55b159cbfafe1f0 \
    --instance-type t3.large \
    --key-name your-key-pair \
    --security-group-ids sg-xxxxxxxxx \
    --subnet-id $subnet \
    --iam-instance-profile Name=LauraDB-EC2-Role \
    --user-data file://user-data-efs.sh \
    --tag-specifications "ResourceType=instance,Tags=[{Key=Name,Value=laura-db-server-$subnet}]"
done

# Register instances with target group
aws elbv2 register-targets \
  --target-group-arn arn:aws:elasticloadbalancing:... \
  --targets Id=i-xxxxxxxxx Id=i-yyyyyyyyy
```

## Auto Scaling Setup

### Step 1: Create Launch Template

```bash
aws ec2 create-launch-template \
  --launch-template-name laura-db-template \
  --version-description "LauraDB v1.0" \
  --launch-template-data '{
    "ImageId": "ami-0c55b159cbfafe1f0",
    "InstanceType": "t3.large",
    "KeyName": "your-key-pair",
    "SecurityGroupIds": ["sg-xxxxxxxxx"],
    "IamInstanceProfile": {
      "Name": "LauraDB-EC2-Role"
    },
    "BlockDeviceMappings": [{
      "DeviceName": "/dev/xvda",
      "Ebs": {
        "VolumeSize": 100,
        "VolumeType": "gp3",
        "DeleteOnTermination": true
      }
    }],
    "UserData": "'$(base64 -w 0 user-data-efs.sh)'"
  }'
```

### Step 2: Create Auto Scaling Group

```bash
aws autoscaling create-auto-scaling-group \
  --auto-scaling-group-name laura-db-asg \
  --launch-template LaunchTemplateName=laura-db-template,Version='$Latest' \
  --min-size 2 \
  --max-size 10 \
  --desired-capacity 3 \
  --vpc-zone-identifier "subnet-xxxxxxxx,subnet-yyyyyyyy" \
  --target-group-arns arn:aws:elasticloadbalancing:... \
  --health-check-type ELB \
  --health-check-grace-period 300 \
  --tags Key=Name,Value=laura-db-asg-instance
```

### Step 3: Create Scaling Policies

```bash
# Scale up policy
aws autoscaling put-scaling-policy \
  --auto-scaling-group-name laura-db-asg \
  --policy-name scale-up \
  --policy-type TargetTrackingScaling \
  --target-tracking-configuration '{
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ASGAverageCPUUtilization"
    },
    "TargetValue": 70.0
  }'

# Scale down policy
aws autoscaling put-scaling-policy \
  --auto-scaling-group-name laura-db-asg \
  --policy-name scale-down \
  --policy-type TargetTrackingScaling \
  --target-tracking-configuration '{
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ASGAverageCPUUtilization"
    },
    "TargetValue": 40.0
  }'
```

## Storage Configuration

### EBS Optimization

```bash
# Create optimized EBS volume
aws ec2 create-volume \
  --availability-zone us-east-1a \
  --volume-type gp3 \
  --size 500 \
  --iops 16000 \
  --throughput 1000 \
  --encrypted \
  --tag-specifications 'ResourceType=volume,Tags=[{Key=Name,Value=laura-db-data}]'

# Attach volume
aws ec2 attach-volume \
  --volume-id vol-xxxxxxxxx \
  --instance-id i-xxxxxxxxx \
  --device /dev/sdf
```

### Format and Mount

```bash
# SSH to instance
ssh ec2-user@<instance-ip>

# Format volume
sudo mkfs.ext4 /dev/sdf

# Create mount point
sudo mkdir -p /data

# Mount volume
sudo mount /dev/sdf /data

# Add to fstab for persistence
UUID=$(sudo blkid /dev/sdf -s UUID -o value)
echo "UUID=$UUID /data ext4 defaults,nofail 0 2" | sudo tee -a /etc/fstab

# Set ownership
sudo chown -R laura-db:laura-db /data
```

## Network Configuration

### VPC Setup

```bash
# Create VPC
aws ec2 create-vpc \
  --cidr-block 10.0.0.0/16 \
  --tag-specifications 'ResourceType=vpc,Tags=[{Key=Name,Value=laura-db-vpc}]'

# Create public subnets in multiple AZs
aws ec2 create-subnet \
  --vpc-id vpc-xxxxxxxxx \
  --cidr-block 10.0.1.0/24 \
  --availability-zone us-east-1a \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=laura-db-public-1a}]'

aws ec2 create-subnet \
  --vpc-id vpc-xxxxxxxxx \
  --cidr-block 10.0.2.0/24 \
  --availability-zone us-east-1b \
  --tag-specifications 'ResourceType=subnet,Tags=[{Key=Name,Value=laura-db-public-1b}]'

# Create and attach internet gateway
aws ec2 create-internet-gateway \
  --tag-specifications 'ResourceType=internet-gateway,Tags=[{Key=Name,Value=laura-db-igw}]'

aws ec2 attach-internet-gateway \
  --vpc-id vpc-xxxxxxxxx \
  --internet-gateway-id igw-xxxxxxxxx
```

## Security Best Practices

### 1. IAM Role for EC2

Create IAM role with minimal permissions:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:ListBucket"
      ],
      "Resource": [
        "arn:aws:s3:::laura-db-backups",
        "arn:aws:s3:::laura-db-backups/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "cloudwatch:PutMetricData",
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents"
      ],
      "Resource": "*"
    }
  ]
}
```

### 2. Security Group Configuration

```bash
# Restrict SSH access to your IP
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxxxxxx \
  --protocol tcp \
  --port 22 \
  --cidr YOUR_IP/32

# Allow HTTP only from ALB
aws ec2 authorize-security-group-ingress \
  --group-id sg-xxxxxxxxx \
  --protocol tcp \
  --port 8080 \
  --source-group sg-alb-xxxxxxxxx
```

### 3. Enable Encryption

- Enable EBS encryption by default
- Use encrypted EFS
- Store secrets in AWS Secrets Manager

### 4. Enable AWS Systems Manager Session Manager

```bash
# Install SSM agent (included in Amazon Linux 2)
sudo yum install -y amazon-ssm-agent
sudo systemctl enable amazon-ssm-agent
sudo systemctl start amazon-ssm-agent
```

## Monitoring and Logging

See [cloudwatch-monitoring.md](./cloudwatch-monitoring.md) for detailed CloudWatch setup.

### Quick CloudWatch Setup

```bash
# Install CloudWatch agent
wget https://s3.amazonaws.com/amazoncloudwatch-agent/amazon_linux/amd64/latest/amazon-cloudwatch-agent.rpm
sudo rpm -U ./amazon-cloudwatch-agent.rpm

# Configure agent
sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-config-wizard
```

## Backup and Recovery

See [s3-backup-integration.md](./s3-backup-integration.md) for detailed S3 backup setup.

### Quick Backup Script

```bash
#!/bin/bash
BACKUP_NAME="laura-db-backup-$(date +%Y%m%d-%H%M%S).tar.gz"
tar czf /tmp/$BACKUP_NAME /var/lib/laura-db
aws s3 cp /tmp/$BACKUP_NAME s3://laura-db-backups/
rm /tmp/$BACKUP_NAME
```

## Cost Optimization

### 1. Use Reserved Instances

- 1-year commitment: ~30% savings
- 3-year commitment: ~50% savings

### 2. Use Spot Instances for Dev/Test

```bash
aws ec2 run-instances \
  --instance-type t3.large \
  --instance-market-options '{
    "MarketType": "spot",
    "SpotOptions": {
      "MaxPrice": "0.05",
      "SpotInstanceType": "one-time"
    }
  }'
```

### 3. Right-Size Instances

Monitor CloudWatch metrics and adjust instance types:
- CPU utilization < 40% → downsize
- Memory pressure → upgrade

### 4. Use gp3 Instead of gp2

gp3 is ~20% cheaper with better performance.

## Troubleshooting

### Instance Not Starting

```bash
# Check system log
aws ec2 get-console-output --instance-id i-xxxxxxxxx

# Check user data execution
ssh ec2-user@<ip>
sudo cat /var/log/cloud-init-output.log
```

### Service Not Running

```bash
# Check service status
sudo systemctl status laura-db

# View logs
sudo journalctl -u laura-db -f

# Check ports
sudo netstat -tlnp | grep 8080
```

### High Memory Usage

```bash
# Check memory
free -h

# Reduce cache sizes
# Edit /etc/systemd/system/laura-db.service
# Add environment variables:
Environment="BUFFER_SIZE=500"
Environment="DOC_CACHE=500"
```

### EFS Mount Issues

```bash
# Test EFS connectivity
nc -zv fs-xxxxxxxxx.efs.us-east-1.amazonaws.com 2049

# Check mount
mount | grep efs

# Remount
sudo umount /mnt/efs
sudo mount -a
```

## Next Steps

- [ECS/Fargate Deployment](./ecs-deployment.md)
- [EKS Deployment](./eks-deployment.md)
- [S3 Backup Integration](./s3-backup-integration.md)
- [CloudWatch Monitoring](./cloudwatch-monitoring.md)
- [RDS Alternative Comparison](./rds-comparison.md)

## Additional Resources

- [AWS EC2 Documentation](https://docs.aws.amazon.com/ec2/)
- [AWS EFS Documentation](https://docs.aws.amazon.com/efs/)
- [AWS Auto Scaling Documentation](https://docs.aws.amazon.com/autoscaling/)
- [AWS Well-Architected Framework](https://aws.amazon.com/architecture/well-architected/)
