# LauraDB Deployment on AWS EKS

This guide provides detailed instructions for deploying LauraDB on Amazon Elastic Kubernetes Service (EKS).

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture](#architecture)
- [Creating EKS Cluster](#creating-eks-cluster)
- [Deploying with Helm](#deploying-with-helm)
- [Deploying with Kubernetes Manifests](#deploying-with-kubernetes-manifests)
- [Storage Configuration](#storage-configuration)
- [Load Balancing](#load-balancing)
- [Auto Scaling](#auto-scaling)
- [Secrets Management](#secrets-management)
- [Monitoring and Logging](#monitoring-and-logging)
- [Security Best Practices](#security-best-practices)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

EKS provides a managed Kubernetes service that eliminates the need to install and operate your own Kubernetes control plane. LauraDB can be deployed using:

- **Helm Charts** (recommended)
- **Raw Kubernetes manifests**
- **Kustomize overlays**

## Prerequisites

### Required Tools

```bash
# Install kubectl
curl -LO "https://dl.k8s.io/release/$(curl -L -s https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl"
sudo install -o root -g root -m 0755 kubectl /usr/local/bin/kubectl

# Install eksctl
curl --silent --location "https://github.com/weaveworks/eksctl/releases/latest/download/eksctl_$(uname -s)_amd64.tar.gz" | tar xz -C /tmp
sudo mv /tmp/eksctl /usr/local/bin

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Install AWS CLI
pip install awscli

# Configure AWS CLI
aws configure
```

### Required Permissions

Your IAM user/role needs:
- EKS full access
- EC2 full access
- CloudFormation access
- IAM role creation
- VPC management

## Architecture

### Production EKS Architecture

```
┌────────────────────────────────────────────┐
│          Internet Gateway                   │
└──────────────┬─────────────────────────────┘
               │
       ┌───────┴────────┐
       │  Public Subnet  │
       │   (NAT GW)      │
       └───────┬────────┘
               │
    ┌──────────┴──────────┐
    │                     │
┌───▼────────┐    ┌──────▼─────┐
│ Private    │    │  Private    │
│ Subnet 1   │    │  Subnet 2   │
│  (AZ-1)    │    │   (AZ-2)    │
│            │    │             │
│ ┌────────┐ │    │ ┌─────────┐│
│ │EKS Node│ │    │ │EKS Node ││
│ │  Pod1  │ │    │ │  Pod2   ││
│ │  +EBS  │ │    │ │  +EBS   ││
│ └────────┘ │    │ └─────────┘│
└────────────┘    └────────────┘
      │                 │
      └────────┬────────┘
           ┌───▼───┐
           │  EFS  │
           └───────┘
```

### Components

- **EKS Control Plane**: Managed by AWS
- **Worker Nodes**: EC2 instances in private subnets
- **EBS**: For pod persistent volumes
- **EFS**: For shared storage across pods
- **ALB**: Application Load Balancer for ingress
- **ECR**: Container registry for Docker images

## Creating EKS Cluster

### Option 1: Using eksctl (Recommended)

Create `cluster-config.yaml`:

```yaml
apiVersion: eksctl.io/v1alpha5
kind: ClusterConfig

metadata:
  name: laura-db-cluster
  region: us-east-1
  version: "1.28"

# IAM identity mappings
iam:
  withOIDC: true

# VPC configuration
vpc:
  cidr: 10.0.0.0/16
  nat:
    gateway: Single  # Use HighlyAvailable for production

# Managed node groups
managedNodeGroups:
  - name: laura-db-nodes
    instanceType: t3.large
    desiredCapacity: 3
    minSize: 2
    maxSize: 10
    volumeSize: 100
    volumeType: gp3
    privateNetworking: true
    labels:
      role: laura-db
    tags:
      k8s.io/cluster-autoscaler/enabled: "true"
      k8s.io/cluster-autoscaler/laura-db-cluster: "owned"
    iam:
      withAddonPolicies:
        autoScaler: true
        ebs: true
        efs: true
        albIngress: true
        cloudWatch: true

# CloudWatch logging
cloudWatch:
  clusterLogging:
    enableTypes:
      - api
      - audit
      - authenticator
      - controllerManager
      - scheduler

# Add-ons
addons:
  - name: vpc-cni
    version: latest
  - name: coredns
    version: latest
  - name: kube-proxy
    version: latest
  - name: aws-ebs-csi-driver
    version: latest
```

Create the cluster:

```bash
# Create cluster
eksctl create cluster -f cluster-config.yaml

# This takes 15-20 minutes
# Once complete, kubectl is automatically configured
```

### Option 2: Using AWS Console

1. Navigate to EKS in AWS Console
2. Click "Create cluster"
3. Configure:
   - Name: laura-db-cluster
   - Kubernetes version: 1.28
   - Cluster service role: Create new or select existing
   - VPC and subnets: Select or create
   - Security groups: Default or custom
4. Create node group:
   - Instance type: t3.large
   - Desired size: 3
   - Min: 2, Max: 10

### Option 3: Using Terraform

See `terraform/aws/eks/` directory for Terraform modules.

### Verify Cluster

```bash
# Check cluster status
eksctl get cluster

# Check nodes
kubectl get nodes

# Check system pods
kubectl get pods -n kube-system
```

## Deploying with Helm

### Step 1: Prepare Docker Image

```bash
# Build and push to ECR (see ECS deployment guide for ECR setup)
cd /Users/krizos/code/mnohosten/laura-db

# Authenticate
aws ecr get-login-password --region us-east-1 | \
  docker login --username AWS --password-stdin ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com

# Build and push
docker build -t ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/laura-db:latest .
docker push ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/laura-db:latest
```

### Step 2: Install with Helm

```bash
# Create namespace
kubectl create namespace laura-db

# Install with default values
helm install laura-db ./helm/laura-db \
  --namespace laura-db \
  --set image.repository=ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/laura-db \
  --set image.tag=latest

# Or create custom values file
cat > eks-values.yaml <<EOF
replicaCount: 3

image:
  repository: ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/laura-db
  tag: latest
  pullPolicy: Always

lauradb:
  bufferSize: 2000
  docCache: 2000
  workerPoolSize: 8
  logLevel: info

resources:
  limits:
    cpu: 2000m
    memory: 4Gi
  requests:
    cpu: 1000m
    memory: 2Gi

persistence:
  enabled: true
  storageClass: gp3
  size: 100Gi

service:
  type: LoadBalancer
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing

ingress:
  enabled: true
  className: alb
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/healthcheck-path: /_health
  hosts:
    - host: laura-db.example.com
      paths:
        - path: /
          pathType: Prefix

podDisruptionBudget:
  enabled: true
  minAvailable: 2

autoscaling:
  enabled: true
  minReplicas: 3
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70

metrics:
  enabled: true
  serviceMonitor:
    enabled: true

affinity:
  podAntiAffinity:
    preferredDuringSchedulingIgnoredDuringExecution:
    - weight: 100
      podAffinityTerm:
        labelSelector:
          matchExpressions:
          - key: app.kubernetes.io/name
            operator: In
            values:
            - laura-db
        topologyKey: kubernetes.io/hostname
EOF

# Install with custom values
helm install laura-db ./helm/laura-db \
  --namespace laura-db \
  -f eks-values.yaml
```

### Step 3: Verify Deployment

```bash
# Check pods
kubectl get pods -n laura-db

# Check service
kubectl get svc -n laura-db

# Get LoadBalancer URL
kubectl get svc laura-db -n laura-db -o jsonpath='{.status.loadBalancer.ingress[0].hostname}'

# Test health endpoint
export LB_URL=$(kubectl get svc laura-db -n laura-db -o jsonpath='{.status.loadBalancer.ingress[0].hostname}')
curl http://$LB_URL:8080/_health
```

## Deploying with Kubernetes Manifests

If you prefer raw Kubernetes manifests:

```bash
# Deploy using Kustomize overlays
kubectl apply -k k8s/overlays/prod

# Or deploy base manifests
kubectl apply -f k8s/base/
```

See the [Kubernetes README](../../../k8s/README.md) for detailed manifest documentation.

## Storage Configuration

### Option 1: EBS Volumes (Default)

EBS volumes are provisioned automatically when using PVCs.

**Install EBS CSI Driver** (if not already installed):

```bash
# Create IAM role for CSI driver
eksctl create iamserviceaccount \
  --name ebs-csi-controller-sa \
  --namespace kube-system \
  --cluster laura-db-cluster \
  --attach-policy-arn arn:aws:iam::aws:policy/service-role/AmazonEBSCSIDriverPolicy \
  --approve \
  --role-only \
  --role-name AmazonEKS_EBS_CSI_DriverRole

# Install EBS CSI driver
eksctl create addon \
  --name aws-ebs-csi-driver \
  --cluster laura-db-cluster \
  --service-account-role-arn arn:aws:iam::ACCOUNT_ID:role/AmazonEKS_EBS_CSI_DriverRole \
  --force
```

**Create gp3 StorageClass**:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: gp3
provisioner: ebs.csi.aws.com
parameters:
  type: gp3
  iops: "3000"
  throughput: "125"
  encrypted: "true"
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

Apply:

```bash
kubectl apply -f gp3-storageclass.yaml

# Set as default
kubectl patch storageclass gp3 -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

### Option 2: EFS for Shared Storage

**Create EFS File System**:

```bash
# Get VPC ID
VPC_ID=$(aws eks describe-cluster \
  --name laura-db-cluster \
  --query "cluster.resourcesVpcConfig.vpcId" \
  --output text)

# Create security group for EFS
aws ec2 create-security-group \
  --group-name laura-db-efs-sg \
  --description "Security group for LauraDB EFS" \
  --vpc-id $VPC_ID

# Allow NFS from worker nodes
NODE_SG=$(aws eks describe-cluster \
  --name laura-db-cluster \
  --query "cluster.resourcesVpcConfig.clusterSecurityGroupId" \
  --output text)

aws ec2 authorize-security-group-ingress \
  --group-id sg-efs-xxxxx \
  --protocol tcp \
  --port 2049 \
  --source-group $NODE_SG

# Create EFS
aws efs create-file-system \
  --performance-mode generalPurpose \
  --throughput-mode bursting \
  --encrypted \
  --tags Key=Name,Value=laura-db-efs

# Create mount targets in each subnet
for subnet in $(aws eks describe-cluster --name laura-db-cluster --query "cluster.resourcesVpcConfig.subnetIds" --output text); do
  aws efs create-mount-target \
    --file-system-id fs-xxxxx \
    --subnet-id $subnet \
    --security-groups sg-efs-xxxxx
done
```

**Install EFS CSI Driver**:

```bash
# Create IAM policy
cat > efs-csi-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "elasticfilesystem:DescribeAccessPoints",
        "elasticfilesystem:DescribeFileSystems",
        "elasticfilesystem:DescribeMountTargets",
        "ec2:DescribeAvailabilityZones"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "elasticfilesystem:CreateAccessPoint"
      ],
      "Resource": "*",
      "Condition": {
        "StringLike": {
          "aws:RequestTag/efs.csi.aws.com/cluster": "true"
        }
      }
    },
    {
      "Effect": "Allow",
      "Action": "elasticfilesystem:DeleteAccessPoint",
      "Resource": "*",
      "Condition": {
        "StringEquals": {
          "aws:ResourceTag/efs.csi.aws.com/cluster": "true"
        }
      }
    }
  ]
}
EOF

aws iam create-policy \
  --policy-name AmazonEKS_EFS_CSI_Driver_Policy \
  --policy-document file://efs-csi-policy.json

# Create service account
eksctl create iamserviceaccount \
  --cluster laura-db-cluster \
  --namespace kube-system \
  --name efs-csi-controller-sa \
  --attach-policy-arn arn:aws:iam::ACCOUNT_ID:policy/AmazonEKS_EFS_CSI_Driver_Policy \
  --approve

# Install EFS CSI driver
kubectl apply -k "github.com/kubernetes-sigs/aws-efs-csi-driver/deploy/kubernetes/overlays/stable/?ref=release-1.7"
```

**Create StorageClass for EFS**:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: efs-sc
provisioner: efs.csi.aws.com
parameters:
  provisioningMode: efs-ap
  fileSystemId: fs-xxxxx
  directoryPerms: "700"
```

## Load Balancing

### Option 1: AWS Load Balancer Controller (ALB)

**Install AWS Load Balancer Controller**:

```bash
# Create IAM policy
curl -o iam-policy.json https://raw.githubusercontent.com/kubernetes-sigs/aws-load-balancer-controller/v2.6.2/docs/install/iam_policy.json

aws iam create-policy \
  --policy-name AWSLoadBalancerControllerIAMPolicy \
  --policy-document file://iam-policy.json

# Create service account
eksctl create iamserviceaccount \
  --cluster=laura-db-cluster \
  --namespace=kube-system \
  --name=aws-load-balancer-controller \
  --attach-policy-arn=arn:aws:iam::ACCOUNT_ID:policy/AWSLoadBalancerControllerIAMPolicy \
  --approve

# Install controller using Helm
helm repo add eks https://aws.github.io/eks-charts
helm repo update

helm install aws-load-balancer-controller eks/aws-load-balancer-controller \
  -n kube-system \
  --set clusterName=laura-db-cluster \
  --set serviceAccount.create=false \
  --set serviceAccount.name=aws-load-balancer-controller
```

**Create Ingress**:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: laura-db
  namespace: laura-db
  annotations:
    alb.ingress.kubernetes.io/scheme: internet-facing
    alb.ingress.kubernetes.io/target-type: ip
    alb.ingress.kubernetes.io/healthcheck-path: /_health
    alb.ingress.kubernetes.io/healthcheck-interval-seconds: '30'
    alb.ingress.kubernetes.io/listen-ports: '[{"HTTP": 80}, {"HTTPS": 443}]'
    alb.ingress.kubernetes.io/ssl-redirect: '443'
spec:
  ingressClassName: alb
  rules:
  - host: laura-db.example.com
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: laura-db
            port:
              number: 8080
```

### Option 2: Network Load Balancer (NLB)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: laura-db-nlb
  namespace: laura-db
  annotations:
    service.beta.kubernetes.io/aws-load-balancer-type: nlb
    service.beta.kubernetes.io/aws-load-balancer-scheme: internet-facing
    service.beta.kubernetes.io/aws-load-balancer-cross-zone-load-balancing-enabled: "true"
spec:
  type: LoadBalancer
  selector:
    app.kubernetes.io/name: laura-db
  ports:
  - port: 80
    targetPort: 8080
```

## Auto Scaling

### Cluster Autoscaler

**Install Cluster Autoscaler**:

```bash
# Create IAM policy
cat > cluster-autoscaler-policy.json <<EOF
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "autoscaling:DescribeAutoScalingGroups",
        "autoscaling:DescribeAutoScalingInstances",
        "autoscaling:DescribeLaunchConfigurations",
        "autoscaling:DescribeTags",
        "autoscaling:SetDesiredCapacity",
        "autoscaling:TerminateInstanceInAutoScalingGroup",
        "ec2:DescribeLaunchTemplateVersions"
      ],
      "Resource": "*"
    }
  ]
}
EOF

aws iam create-policy \
  --policy-name AmazonEKSClusterAutoscalerPolicy \
  --policy-document file://cluster-autoscaler-policy.json

# Create service account
eksctl create iamserviceaccount \
  --cluster=laura-db-cluster \
  --namespace=kube-system \
  --name=cluster-autoscaler \
  --attach-policy-arn=arn:aws:iam::ACCOUNT_ID:policy/AmazonEKSClusterAutoscalerPolicy \
  --approve

# Deploy cluster autoscaler
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/cluster-autoscaler/cloudprovider/aws/examples/cluster-autoscaler-autodiscover.yaml

# Edit deployment to add cluster name
kubectl -n kube-system edit deployment cluster-autoscaler
# Add: --node-group-auto-discovery=asg:tag=k8s.io/cluster-autoscaler/enabled,k8s.io/cluster-autoscaler/laura-db-cluster
```

### Horizontal Pod Autoscaler (HPA)

HPA is automatically configured when using Helm with `autoscaling.enabled=true`.

Manual HPA:

```yaml
apiVersion: autoscaling/v2
kind: HorizontalPodAutoscaler
metadata:
  name: laura-db-hpa
  namespace: laura-db
spec:
  scaleTargetRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: laura-db
  minReplicas: 3
  maxReplicas: 10
  metrics:
  - type: Resource
    resource:
      name: cpu
      target:
        type: Utilization
        averageUtilization: 70
  - type: Resource
    resource:
      name: memory
      target:
        type: Utilization
        averageUtilization: 80
```

## Secrets Management

### AWS Secrets Manager Integration

**Install External Secrets Operator**:

```bash
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets \
  external-secrets/external-secrets \
  -n external-secrets-system \
  --create-namespace
```

**Create SecretStore**:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: aws-secretsmanager
  namespace: laura-db
spec:
  provider:
    aws:
      service: SecretsManager
      region: us-east-1
      auth:
        jwt:
          serviceAccountRef:
            name: laura-db
```

**Create ExternalSecret**:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: ExternalSecret
metadata:
  name: laura-db-credentials
  namespace: laura-db
spec:
  refreshInterval: 1h
  secretStoreRef:
    name: aws-secretsmanager
    kind: SecretStore
  target:
    name: laura-db-admin
    creationPolicy: Owner
  data:
  - secretKey: admin-username
    remoteRef:
      key: laura-db/admin
      property: username
  - secretKey: admin-password
    remoteRef:
      key: laura-db/admin
      property: password
```

## Monitoring and Logging

### CloudWatch Container Insights

```bash
# Enable Container Insights
eksctl utils update-cluster-logging \
  --cluster laura-db-cluster \
  --enable-types all \
  --approve

# Install CloudWatch agent
kubectl apply -f https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/quickstart/cwagent-fluentd-quickstart.yaml
```

### Prometheus and Grafana

```bash
# Install Prometheus
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace

# Access Grafana
kubectl port-forward -n monitoring svc/prometheus-grafana 3000:80
# Default credentials: admin/prom-operator
```

## Security Best Practices

### 1. Enable IRSA (IAM Roles for Service Accounts)

Already enabled via `--with-oidc` in cluster creation.

### 2. Enable Pod Security Standards

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: laura-db
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

### 3. Network Policies

```yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: laura-db-netpol
  namespace: laura-db
spec:
  podSelector:
    matchLabels:
      app.kubernetes.io/name: laura-db
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: laura-db
    ports:
    - protocol: TCP
      port: 8080
  egress:
  - to:
    - namespaceSelector: {}
    ports:
    - protocol: TCP
      port: 53
    - protocol: UDP
      port: 53
```

### 4. Encrypt Secrets

EKS encrypts secrets at rest by default using AWS KMS.

## Cost Optimization

### 1. Use Spot Instances

```yaml
managedNodeGroups:
  - name: laura-db-spot
    instanceTypes:
      - t3.large
      - t3a.large
    spot: true
    desiredCapacity: 2
    minSize: 1
    maxSize: 5
```

### 2. Use Fargate for Batch Jobs

```bash
eksctl create fargateprofile \
  --cluster laura-db-cluster \
  --name laura-db-batch \
  --namespace batch-jobs
```

### 3. Right-Size Pods

Monitor resource usage and adjust:

```bash
kubectl top pods -n laura-db
kubectl top nodes
```

### 4. Enable Cluster Autoscaler

Automatically scales nodes based on pod requirements.

## Troubleshooting

### Pods Stuck in Pending

```bash
# Check pod events
kubectl describe pod laura-db-0 -n laura-db

# Common causes:
# - Insufficient cluster capacity
# - PVC binding issues
# - Node affinity/taints
```

### PVC Not Binding

```bash
# Check PVC status
kubectl get pvc -n laura-db

# Check storage class
kubectl get storageclass

# Describe PVC for events
kubectl describe pvc data-laura-db-0 -n laura-db
```

### LoadBalancer Not Getting External IP

```bash
# Check service
kubectl describe svc laura-db -n laura-db

# Check AWS Load Balancer Controller logs
kubectl logs -n kube-system deployment/aws-load-balancer-controller
```

### High Network Latency

```bash
# Check pod network
kubectl exec -it laura-db-0 -n laura-db -- ping google.com

# Check DNS resolution
kubectl exec -it laura-db-0 -n laura-db -- nslookup kubernetes.default
```

## Next Steps

- [S3 Backup Integration](./s3-backup-integration.md)
- [CloudWatch Monitoring](./cloudwatch-monitoring.md)
- [RDS Alternative Comparison](./rds-comparison.md)
- [EC2 Deployment](./ec2-deployment.md)
- [ECS Deployment](./ecs-deployment.md)

## Additional Resources

- [EKS Documentation](https://docs.aws.amazon.com/eks/)
- [eksctl Documentation](https://eksctl.io/)
- [EKS Best Practices](https://aws.github.io/aws-eks-best-practices/)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
