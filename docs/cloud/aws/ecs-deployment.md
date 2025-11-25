# LauraDB Deployment on AWS ECS/Fargate

This guide provides detailed instructions for deploying LauraDB on Amazon ECS (Elastic Container Service) using both EC2 and Fargate launch types.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture](#architecture)
- [Building Docker Image](#building-docker-image)
- [ECS Fargate Deployment](#ecs-fargate-deployment)
- [ECS EC2 Deployment](#ecs-ec2-deployment)
- [Service Discovery](#service-discovery)
- [Load Balancing](#load-balancing)
- [Storage with EFS](#storage-with-efs)
- [Auto Scaling](#auto-scaling)
- [Secrets Management](#secrets-management)
- [Monitoring](#monitoring)
- [Cost Comparison](#cost-comparison)
- [Troubleshooting](#troubleshooting)

## Overview

LauraDB can be deployed on ECS in two modes:

### Fargate (Serverless)
- **Pros**: No server management, automatic scaling, pay-per-use
- **Cons**: Higher per-task cost, limited customization
- **Best for**: Variable workloads, rapid deployment, minimal ops overhead

### EC2 (Managed Containers)
- **Pros**: Lower cost at scale, more control, better performance
- **Cons**: Requires cluster management, instance maintenance
- **Best for**: Steady-state workloads, cost optimization, custom configurations

## Prerequisites

### Required Tools

```bash
# Install AWS CLI
pip install awscli

# Install ECS CLI
sudo curl -Lo /usr/local/bin/ecs-cli https://amazon-ecs-cli.s3.amazonaws.com/ecs-cli-linux-amd64-latest
sudo chmod +x /usr/local/bin/ecs-cli

# Install Docker
# See: https://docs.docker.com/get-docker/

# Configure AWS CLI
aws configure
```

### Required Permissions

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "ecs:*",
        "ecr:*",
        "iam:PassRole",
        "ec2:*",
        "elasticloadbalancing:*",
        "logs:*",
        "cloudwatch:*"
      ],
      "Resource": "*"
    }
  ]
}
```

## Architecture

### Fargate Architecture

```
┌────────────────────────────────────────┐
│     Application Load Balancer          │
└───────────┬────────────────────────────┘
            │
    ┌───────┴───────┐
    │               │
┌───▼─────┐   ┌────▼────┐
│   AZ-1  │   │  AZ-2   │
│         │   │         │
│ Fargate │   │ Fargate │
│  Task   │   │  Task   │
│    +    │   │    +    │
│   EFS   │   │   EFS   │
└─────────┘   └─────────┘
      │             │
      └──────┬──────┘
         ┌───▼───┐
         │  EFS  │
         └───────┘
```

## Building Docker Image

### Step 1: Create Dockerfile

The Dockerfile is already available at the project root. Verify it:

```bash
cat /Users/krizos/code/mnohosten/laura-db/Dockerfile
```

### Step 2: Create ECR Repository

```bash
# Create repository
aws ecr create-repository \
  --repository-name laura-db \
  --image-scanning-configuration scanOnPush=true \
  --encryption-configuration encryptionType=AES256

# Get repository URI
export ECR_REPO=$(aws ecr describe-repositories \
  --repository-names laura-db \
  --query 'repositories[0].repositoryUri' \
  --output text)

echo "ECR Repository: $ECR_REPO"
```

### Step 3: Build and Push Image

```bash
# Navigate to project root
cd /Users/krizos/code/mnohosten/laura-db

# Authenticate with ECR
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin $ECR_REPO

# Build image
docker build -t laura-db:latest .

# Tag image
docker tag laura-db:latest $ECR_REPO:latest
docker tag laura-db:latest $ECR_REPO:v1.0.0

# Push image
docker push $ECR_REPO:latest
docker push $ECR_REPO:v1.0.0
```

## ECS Fargate Deployment

### Step 1: Create EFS File System

```bash
# Create EFS
aws efs create-file-system \
  --performance-mode generalPurpose \
  --throughput-mode bursting \
  --encrypted \
  --tags Key=Name,Value=laura-db-efs \
  --region us-east-1

# Get file system ID
export EFS_ID=$(aws efs describe-file-systems \
  --query 'FileSystems[?Name==`laura-db-efs`].FileSystemId' \
  --output text)

# Create mount targets in each subnet
for subnet in subnet-xxxxxxxx subnet-yyyyyyyy; do
  aws efs create-mount-target \
    --file-system-id $EFS_ID \
    --subnet-id $subnet \
    --security-groups sg-efs-xxxxxxxxx
done
```

### Step 2: Create ECS Cluster

```bash
# Create Fargate cluster
aws ecs create-cluster \
  --cluster-name laura-db-fargate \
  --capacity-providers FARGATE FARGATE_SPOT \
  --default-capacity-provider-strategy \
    capacityProvider=FARGATE,weight=1 \
    capacityProvider=FARGATE_SPOT,weight=4
```

### Step 3: Create Task Definition

Create `fargate-task-definition.json`:

```json
{
  "family": "laura-db-fargate",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "1024",
  "memory": "2048",
  "executionRoleArn": "arn:aws:iam::ACCOUNT_ID:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::ACCOUNT_ID:role/lauraDbTaskRole",
  "containerDefinitions": [
    {
      "name": "laura-db",
      "image": "ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/laura-db:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "PORT",
          "value": "8080"
        },
        {
          "name": "DATA_DIR",
          "value": "/data"
        },
        {
          "name": "BUFFER_SIZE",
          "value": "1000"
        },
        {
          "name": "DOC_CACHE",
          "value": "1000"
        },
        {
          "name": "WORKER_POOL_SIZE",
          "value": "4"
        },
        {
          "name": "LOG_LEVEL",
          "value": "info"
        }
      ],
      "secrets": [
        {
          "name": "ADMIN_USERNAME",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:ACCOUNT_ID:secret:laura-db/admin:username::"
        },
        {
          "name": "ADMIN_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:us-east-1:ACCOUNT_ID:secret:laura-db/admin:password::"
        }
      ],
      "mountPoints": [
        {
          "sourceVolume": "laura-db-data",
          "containerPath": "/data",
          "readOnly": false
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/laura-db-fargate",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "laura-db"
        }
      },
      "healthCheck": {
        "command": ["CMD-SHELL", "curl -f http://localhost:8080/_health || exit 1"],
        "interval": 30,
        "timeout": 5,
        "retries": 3,
        "startPeriod": 60
      }
    }
  ],
  "volumes": [
    {
      "name": "laura-db-data",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-xxxxxxxxx",
        "transitEncryption": "ENABLED",
        "authorizationConfig": {
          "iam": "ENABLED"
        }
      }
    }
  ]
}
```

Register the task definition:

```bash
# Create log group first
aws logs create-log-group --log-group-name /ecs/laura-db-fargate

# Register task definition
aws ecs register-task-definition \
  --cli-input-json file://fargate-task-definition.json
```

### Step 4: Create Application Load Balancer

```bash
# Create ALB
aws elbv2 create-load-balancer \
  --name laura-db-alb \
  --subnets subnet-xxxxxxxx subnet-yyyyyyyy \
  --security-groups sg-alb-xxxxxxxxx \
  --scheme internet-facing \
  --type application

# Create target group
aws elbv2 create-target-group \
  --name laura-db-fargate-tg \
  --protocol HTTP \
  --port 8080 \
  --vpc-id vpc-xxxxxxxxx \
  --target-type ip \
  --health-check-enabled \
  --health-check-path /_health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3

# Create listener
aws elbv2 create-listener \
  --load-balancer-arn arn:aws:elasticloadbalancing:... \
  --protocol HTTP \
  --port 80 \
  --default-actions Type=forward,TargetGroupArn=arn:aws:elasticloadbalancing:...
```

### Step 5: Create ECS Service

```bash
aws ecs create-service \
  --cluster laura-db-fargate \
  --service-name laura-db-service \
  --task-definition laura-db-fargate:1 \
  --desired-count 2 \
  --launch-type FARGATE \
  --platform-version LATEST \
  --network-configuration "awsvpcConfiguration={
    subnets=[subnet-xxxxxxxx,subnet-yyyyyyyy],
    securityGroups=[sg-xxxxxxxxx],
    assignPublicIp=ENABLED
  }" \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:...,containerName=laura-db,containerPort=8080" \
  --health-check-grace-period-seconds 60 \
  --enable-execute-command
```

## ECS EC2 Deployment

### Step 1: Create EC2 Cluster

```bash
# Create cluster
aws ecs create-cluster \
  --cluster-name laura-db-ec2 \
  --capacity-providers EC2 \
  --default-capacity-provider-strategy capacityProvider=EC2,weight=1
```

### Step 2: Launch EC2 Instances for ECS

```bash
# Create launch template
aws ec2 create-launch-template \
  --launch-template-name laura-db-ecs-template \
  --launch-template-data '{
    "ImageId": "ami-0c55b159cbfafe1f0",
    "InstanceType": "t3.large",
    "IamInstanceProfile": {
      "Name": "ecsInstanceRole"
    },
    "SecurityGroupIds": ["sg-xxxxxxxxx"],
    "UserData": "'$(base64 -w 0 <<'EOF'
#!/bin/bash
echo ECS_CLUSTER=laura-db-ec2 >> /etc/ecs/ecs.config
echo ECS_ENABLE_TASK_IAM_ROLE=true >> /etc/ecs/ecs.config
echo ECS_ENABLE_TASK_IAM_ROLE_NETWORK_HOST=true >> /etc/ecs/ecs.config
EOF
)'"
  }'

# Create auto scaling group
aws autoscaling create-auto-scaling-group \
  --auto-scaling-group-name laura-db-ecs-asg \
  --launch-template LaunchTemplateName=laura-db-ecs-template,Version='$Latest' \
  --min-size 2 \
  --max-size 10 \
  --desired-capacity 3 \
  --vpc-zone-identifier "subnet-xxxxxxxx,subnet-yyyyyyyy" \
  --health-check-type ELB \
  --health-check-grace-period 300
```

### Step 3: Create Task Definition for EC2

Create `ec2-task-definition.json`:

```json
{
  "family": "laura-db-ec2",
  "networkMode": "bridge",
  "requiresCompatibilities": ["EC2"],
  "cpu": "1024",
  "memory": "2048",
  "taskRoleArn": "arn:aws:iam::ACCOUNT_ID:role/lauraDbTaskRole",
  "containerDefinitions": [
    {
      "name": "laura-db",
      "image": "ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/laura-db:latest",
      "essential": true,
      "portMappings": [
        {
          "containerPort": 8080,
          "hostPort": 0,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "PORT",
          "value": "8080"
        },
        {
          "name": "DATA_DIR",
          "value": "/data"
        }
      ],
      "mountPoints": [
        {
          "sourceVolume": "laura-db-data",
          "containerPath": "/data"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/laura-db-ec2",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "laura-db"
        }
      }
    }
  ],
  "volumes": [
    {
      "name": "laura-db-data",
      "host": {
        "sourcePath": "/mnt/efs/laura-db"
      }
    }
  ]
}
```

### Step 4: Create EC2 Service

```bash
aws ecs create-service \
  --cluster laura-db-ec2 \
  --service-name laura-db-service \
  --task-definition laura-db-ec2:1 \
  --desired-count 3 \
  --launch-type EC2 \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:...,containerName=laura-db,containerPort=8080" \
  --health-check-grace-period-seconds 60
```

## Service Discovery

### Create Private Namespace

```bash
# Create Cloud Map namespace
aws servicediscovery create-private-dns-namespace \
  --name laura-db.local \
  --vpc vpc-xxxxxxxxx \
  --description "Private namespace for LauraDB"

# Create service
aws servicediscovery create-service \
  --name laura-db \
  --namespace-id ns-xxxxxxxxx \
  --dns-config "NamespaceId=ns-xxxxxxxxx,DnsRecords=[{Type=A,TTL=60}]" \
  --health-check-custom-config FailureThreshold=1
```

### Update ECS Service with Service Discovery

```bash
aws ecs update-service \
  --cluster laura-db-fargate \
  --service laura-db-service \
  --service-registries "registryArn=arn:aws:servicediscovery:..."
```

Now services can connect using:
```
http://laura-db.laura-db.local:8080
```

## Load Balancing

### Configure Health Checks

```bash
aws elbv2 modify-target-group \
  --target-group-arn arn:aws:elasticloadbalancing:... \
  --health-check-path /_health \
  --health-check-interval-seconds 30 \
  --health-check-timeout-seconds 5 \
  --healthy-threshold-count 2 \
  --unhealthy-threshold-count 3 \
  --matcher HttpCode=200
```

### Configure Stickiness

```bash
aws elbv2 modify-target-group-attributes \
  --target-group-arn arn:aws:elasticloadbalancing:... \
  --attributes \
    Key=stickiness.enabled,Value=true \
    Key=stickiness.type,Value=lb_cookie \
    Key=stickiness.lb_cookie.duration_seconds,Value=86400
```

## Storage with EFS

### EFS Security Group

```bash
# Create security group for EFS
aws ec2 create-security-group \
  --group-name laura-db-efs-sg \
  --description "Security group for LauraDB EFS" \
  --vpc-id vpc-xxxxxxxxx

# Allow NFS from ECS tasks
aws ec2 authorize-security-group-ingress \
  --group-id sg-efs-xxxxxxxxx \
  --protocol tcp \
  --port 2049 \
  --source-group sg-xxxxxxxxx
```

### EFS Access Point (Fargate)

```bash
# Create access point for better security
aws efs create-access-point \
  --file-system-id $EFS_ID \
  --posix-user Uid=1000,Gid=1000 \
  --root-directory "Path=/laura-db,CreationInfo={OwnerUid=1000,OwnerGid=1000,Permissions=755}"
```

Update task definition to use access point:

```json
{
  "volumes": [
    {
      "name": "laura-db-data",
      "efsVolumeConfiguration": {
        "fileSystemId": "fs-xxxxxxxxx",
        "transitEncryption": "ENABLED",
        "authorizationConfig": {
          "accessPointId": "fsap-xxxxxxxxx",
          "iam": "ENABLED"
        }
      }
    }
  ]
}
```

## Auto Scaling

### Target Tracking Scaling Policy

```bash
# Register scalable target
aws application-autoscaling register-scalable-target \
  --service-namespace ecs \
  --resource-id service/laura-db-fargate/laura-db-service \
  --scalable-dimension ecs:service:DesiredCount \
  --min-capacity 2 \
  --max-capacity 10

# Create scaling policy based on CPU
aws application-autoscaling put-scaling-policy \
  --service-namespace ecs \
  --resource-id service/laura-db-fargate/laura-db-service \
  --scalable-dimension ecs:service:DesiredCount \
  --policy-name cpu-scaling \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration '{
    "TargetValue": 70.0,
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ECSServiceAverageCPUUtilization"
    },
    "ScaleInCooldown": 300,
    "ScaleOutCooldown": 60
  }'

# Create scaling policy based on memory
aws application-autoscaling put-scaling-policy \
  --service-namespace ecs \
  --resource-id service/laura-db-fargate/laura-db-service \
  --scalable-dimension ecs:service:DesiredCount \
  --policy-name memory-scaling \
  --policy-type TargetTrackingScaling \
  --target-tracking-scaling-policy-configuration '{
    "TargetValue": 80.0,
    "PredefinedMetricSpecification": {
      "PredefinedMetricType": "ECSServiceAverageMemoryUtilization"
    },
    "ScaleInCooldown": 300,
    "ScaleOutCooldown": 60
  }'
```

## Secrets Management

### Create Secrets in AWS Secrets Manager

```bash
# Create admin credentials secret
aws secretsmanager create-secret \
  --name laura-db/admin \
  --description "LauraDB admin credentials" \
  --secret-string '{
    "username": "admin",
    "password": "your-secure-password"
  }'

# Create encryption key secret
aws secretsmanager create-secret \
  --name laura-db/encryption-key \
  --description "LauraDB encryption key" \
  --secret-string "$(openssl rand -base64 32)"
```

### Grant ECS Task Access

Update task role policy:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "secretsmanager:GetSecretValue"
      ],
      "Resource": [
        "arn:aws:secretsmanager:us-east-1:ACCOUNT_ID:secret:laura-db/*"
      ]
    }
  ]
}
```

## Monitoring

### CloudWatch Container Insights

```bash
# Enable Container Insights on cluster
aws ecs update-cluster-settings \
  --cluster laura-db-fargate \
  --settings name=containerInsights,value=enabled
```

### Custom Metrics

Add CloudWatch agent as sidecar:

```json
{
  "name": "cloudwatch-agent",
  "image": "amazon/cloudwatch-agent:latest",
  "essential": false,
  "secrets": [
    {
      "name": "CW_CONFIG_CONTENT",
      "valueFrom": "arn:aws:secretsmanager:...:secret:cloudwatch-config"
    }
  ]
}
```

### CloudWatch Alarms

```bash
# High CPU alarm
aws cloudwatch put-metric-alarm \
  --alarm-name laura-db-high-cpu \
  --alarm-description "LauraDB high CPU utilization" \
  --metric-name CPUUtilization \
  --namespace AWS/ECS \
  --statistic Average \
  --period 300 \
  --threshold 80 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=ServiceName,Value=laura-db-service Name=ClusterName,Value=laura-db-fargate \
  --evaluation-periods 2 \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:ops-alerts
```

## Cost Comparison

### Fargate Pricing (us-east-1)

**Configuration**: 1 vCPU, 2 GB memory
- Per hour: $0.04048 + $0.004445 = $0.044925
- Per day (24h): $1.08
- Per month (730h): $32.80
- **2 tasks**: $65.60/month

### EC2 Pricing (us-east-1)

**Configuration**: t3.large (2 vCPU, 8 GB memory)
- On-Demand: $0.0832/hour = $60.74/month
- **3 instances**: $182.22/month
- Can run ~12 tasks (6 tasks/instance)

### Break-even Point

- **< 4 tasks**: Fargate is cheaper
- **> 4 tasks**: EC2 is cheaper
- **Variable load**: Use Fargate Spot (70% savings)

## Troubleshooting

### Task Fails to Start

```bash
# Check task status
aws ecs describe-tasks \
  --cluster laura-db-fargate \
  --tasks task-id

# View stopped task reason
aws ecs describe-tasks \
  --cluster laura-db-fargate \
  --tasks task-id \
  --query 'tasks[0].stoppedReason'
```

### Container Exits Immediately

```bash
# View CloudWatch logs
aws logs tail /ecs/laura-db-fargate --follow

# Execute command in running container
aws ecs execute-command \
  --cluster laura-db-fargate \
  --task task-id \
  --container laura-db \
  --interactive \
  --command "/bin/sh"
```

### EFS Mount Failures

```bash
# Check EFS mount targets
aws efs describe-mount-targets --file-system-id $EFS_ID

# Verify security group rules
aws ec2 describe-security-groups --group-ids sg-efs-xxxxxxxxx

# Test connectivity from task subnet
nc -zv fs-xxxxxxxxx.efs.us-east-1.amazonaws.com 2049
```

### Health Check Failures

```bash
# Check target health
aws elbv2 describe-target-health \
  --target-group-arn arn:aws:elasticloadbalancing:...

# Update health check settings
aws elbv2 modify-target-group \
  --target-group-arn arn:aws:elasticloadbalancing:... \
  --health-check-interval-seconds 60 \
  --healthy-threshold-count 3
```

## Best Practices

1. **Use Fargate Spot** for 70% cost savings on non-critical workloads
2. **Enable Container Insights** for detailed metrics
3. **Use EFS Access Points** for better security isolation
4. **Implement circuit breakers** to prevent failed deployments
5. **Use task placement strategies** for optimal distribution (EC2)
6. **Enable execute-command** for debugging
7. **Store secrets in Secrets Manager**, not environment variables
8. **Use multiple target groups** for blue/green deployments

## Next Steps

- [EKS Deployment](./eks-deployment.md)
- [S3 Backup Integration](./s3-backup-integration.md)
- [CloudWatch Monitoring](./cloudwatch-monitoring.md)
- [RDS Alternative Comparison](./rds-comparison.md)

## Additional Resources

- [AWS ECS Documentation](https://docs.aws.amazon.com/ecs/)
- [AWS Fargate Documentation](https://docs.aws.amazon.com/fargate/)
- [ECS Best Practices](https://docs.aws.amazon.com/AmazonECS/latest/bestpracticesguide/)
- [Container Insights Documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/ContainerInsights.html)
