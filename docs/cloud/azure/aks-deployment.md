# LauraDB Deployment on Azure Kubernetes Service (AKS)

Complete guide for deploying LauraDB on Azure Kubernetes Service (AKS) with production-ready configurations.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Deployment Options](#deployment-options)
- [Quick Start](#quick-start)
- [Detailed Setup](#detailed-setup)
- [Storage Configuration](#storage-configuration)
- [Networking](#networking)
- [Security](#security)
- [Monitoring & Logging](#monitoring--logging)
- [Scaling](#scaling)
- [Backup & Recovery](#backup--recovery)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

Azure Kubernetes Service (AKS) provides a managed Kubernetes platform for deploying containerized applications with built-in monitoring, scaling, and security features.

### Why AKS for LauraDB?

- **Managed Control Plane**: Azure manages the Kubernetes master nodes
- **Integrated Monitoring**: Azure Monitor and Container Insights built-in
- **Auto-scaling**: Both cluster and pod-level auto-scaling
- **Security**: Azure Active Directory integration, Azure Policy, Pod Security
- **Storage Options**: Azure Disks (Premium SSD, Ultra Disk) and Azure Files
- **Networking**: Advanced networking with Azure CNI, Network Policies
- **Cost Effective**: Pay only for worker nodes, not control plane

### Architecture

```
┌─────────────────────────────────────────┐
│      Azure Load Balancer / Ingress      │
└──────────────────┬──────────────────────┘
                   │
    ┌──────────────┴──────────────┐
    │        AKS Cluster           │
    │  ┌────────────────────────┐ │
    │  │   LauraDB StatefulSet  │ │
    │  │  ┌──────┐  ┌──────┐   │ │
    │  │  │ Pod0 │  │ Pod1 │   │ │
    │  │  └──┬───┘  └──┬───┘   │ │
    │  └─────┼─────────┼────────┘ │
    │     ┌──▼──┐   ┌──▼──┐       │
    │     │ PV0 │   │ PV1 │       │
    │     └─────┘   └─────┘       │
    └───────────────────────────────┘
              │
    ┌─────────▼────────────┐
    │  Azure Managed Disks │
    │    or Azure Files    │
    └──────────────────────┘
```

## Prerequisites

### Tools Required

```bash
# Install Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Install kubectl
az aks install-cli

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Verify installations
az --version
kubectl version --client
helm version
```

### Azure Setup

```bash
# Login to Azure
az login

# Set subscription
az account set --subscription "Your Subscription Name"

# Register required providers
az provider register --namespace Microsoft.ContainerService
az provider register --namespace Microsoft.Storage
az provider register --namespace Microsoft.Network
az provider register --namespace Microsoft.Compute

# Verify registration
az provider show -n Microsoft.ContainerService --query "registrationState"
```

## Deployment Options

### Option 1: Basic AKS Cluster (Development)

**Best for**: Development, testing, small workloads

**Specs**:
- 1-3 nodes
- Standard_B2s or Standard_D2s_v3
- Standard Load Balancer
- Azure CNI (optional)
- Cost: ~$150-300/month

### Option 2: Production AKS Cluster

**Best for**: Production workloads, high availability

**Specs**:
- 3-5 nodes across availability zones
- Standard_D4s_v3 or larger
- Standard Load Balancer
- Azure CNI networking
- Pod Security Standards
- Azure Monitor Container Insights
- Cost: ~$500-1000/month

### Option 3: Enterprise AKS Cluster

**Best for**: Mission-critical applications, large scale

**Specs**:
- 5+ nodes with node pools
- Standard_D8s_v3 or larger
- Application Gateway Ingress Controller
- Azure Private Link
- Azure Active Directory integration
- Azure Policy
- Azure Key Vault Provider
- Cluster Autoscaler
- Cost: ~$1500+/month

## Quick Start

### Create Basic AKS Cluster and Deploy LauraDB

```bash
# 1. Create resource group
az group create \
  --name laura-db-rg \
  --location eastus

# 2. Create AKS cluster
az aks create \
  --resource-group laura-db-rg \
  --name laura-db-aks \
  --node-count 2 \
  --node-vm-size Standard_D2s_v3 \
  --enable-managed-identity \
  --enable-cluster-autoscaler \
  --min-count 1 \
  --max-count 5 \
  --generate-ssh-keys

# 3. Get credentials
az aks get-credentials \
  --resource-group laura-db-rg \
  --name laura-db-aks

# 4. Verify connection
kubectl get nodes

# 5. Deploy using Helm
helm install laura-db ./helm/laura-db \
  --set image.tag=latest \
  --set persistence.enabled=true \
  --set persistence.size=20Gi \
  --set service.type=LoadBalancer

# 6. Wait for deployment
kubectl rollout status statefulset/laura-db

# 7. Get external IP
kubectl get service laura-db
```

## Detailed Setup

### 1. Create Resource Group

```bash
RESOURCE_GROUP="laura-db-rg"
LOCATION="eastus"

az group create \
  --name $RESOURCE_GROUP \
  --location $LOCATION \
  --tags application=laura-db environment=production
```

### 2. Create Virtual Network (Optional but Recommended)

```bash
VNET_NAME="laura-db-vnet"
AKS_SUBNET_NAME="aks-subnet"

# Create VNet
az network vnet create \
  --resource-group $RESOURCE_GROUP \
  --name $VNET_NAME \
  --address-prefixes 10.0.0.0/16 \
  --subnet-name $AKS_SUBNET_NAME \
  --subnet-prefix 10.0.1.0/24

# Get subnet ID
SUBNET_ID=$(az network vnet subnet show \
  --resource-group $RESOURCE_GROUP \
  --vnet-name $VNET_NAME \
  --name $AKS_SUBNET_NAME \
  --query id -o tsv)
```

### 3. Create AKS Cluster (Production Configuration)

```bash
CLUSTER_NAME="laura-db-aks"
NODE_COUNT=3
NODE_VM_SIZE="Standard_D4s_v3"

az aks create \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --location $LOCATION \
  --kubernetes-version 1.28 \
  --node-count $NODE_COUNT \
  --node-vm-size $NODE_VM_SIZE \
  --vnet-subnet-id $SUBNET_ID \
  --network-plugin azure \
  --network-policy azure \
  --enable-managed-identity \
  --enable-cluster-autoscaler \
  --min-count 2 \
  --max-count 10 \
  --enable-addons monitoring \
  --workspace-resource-id /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.OperationalInsights/workspaces/laura-db-logs \
  --enable-azure-rbac \
  --enable-aad \
  --aad-admin-group-object-ids YOUR_AAD_GROUP_ID \
  --zones 1 2 3 \
  --load-balancer-sku standard \
  --node-osdisk-type Managed \
  --node-osdisk-size 128 \
  --max-pods 110 \
  --generate-ssh-keys \
  --tags application=laura-db environment=production

# Enable auto-upgrade
az aks update \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --auto-upgrade-channel stable
```

### 4. Configure kubectl Access

```bash
# Get credentials
az aks get-credentials \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --overwrite-existing

# Verify connection
kubectl cluster-info
kubectl get nodes -o wide
```

### 5. Create Namespace

```bash
kubectl create namespace laura-db

# Set as default namespace (optional)
kubectl config set-context --current --namespace=laura-db
```

## Storage Configuration

### Option 1: Azure Disk (Default - Best for Single Pod)

Azure Managed Disks provide high-performance block storage.

#### Create Storage Class

```yaml
# azure-disk-sc.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: azure-disk-premium
provisioner: disk.csi.azure.com
parameters:
  skuName: Premium_LRS  # Premium_LRS, StandardSSD_LRS, or UltraSSD_LRS
  kind: Managed
  cachingMode: ReadWrite
reclaimPolicy: Retain
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
```

```bash
kubectl apply -f azure-disk-sc.yaml
```

#### Storage Performance Tiers

| Tier | Type | Max IOPS | Max Throughput | Use Case |
|------|------|----------|----------------|----------|
| **Premium_LRS** | Premium SSD | 20,000 | 900 MB/s | Production (default) |
| **StandardSSD_LRS** | Standard SSD | 6,000 | 750 MB/s | Dev/Test |
| **Premium_ZRS** | Premium SSD (Zone-redundant) | 20,000 | 900 MB/s | HA Production |
| **UltraSSD_LRS** | Ultra Disk | 160,000 | 4,000 MB/s | Extreme Performance |

### Option 2: Azure Files (Best for Multi-Pod ReadWriteMany)

Azure Files provides SMB/NFS shared storage.

#### Enable Azure Files CSI Driver

```bash
az aks update \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --enable-file-driver
```

#### Create Storage Class

```yaml
# azure-files-sc.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: azure-files-premium
provisioner: file.csi.azure.com
parameters:
  skuName: Premium_LRS  # Premium_LRS or Standard_LRS
  protocol: nfs          # nfs or smb
  networkEndpointType: privateEndpoint  # For private link
mountOptions:
  - dir_mode=0777
  - file_mode=0777
  - uid=0
  - gid=0
  - mfsymlinks
  - cache=strict
  - actimeo=30
reclaimPolicy: Retain
allowVolumeExpansion: true
volumeBindingMode: Immediate
```

```bash
kubectl apply -f azure-files-sc.yaml
```

### Option 3: Ultra Disk (Extreme Performance)

For workloads requiring > 20,000 IOPS or > 900 MB/s throughput.

#### Enable Ultra Disk Support

```bash
az aks nodepool update \
  --resource-group $RESOURCE_GROUP \
  --cluster-name $CLUSTER_NAME \
  --name nodepool1 \
  --enable-ultra-ssd
```

#### Create Ultra Disk Storage Class

```yaml
# azure-ultra-disk-sc.yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: azure-ultra-disk
provisioner: disk.csi.azure.com
parameters:
  skuName: UltraSSD_LRS
  cachingMode: None
  diskIOPSReadWrite: "50000"
  diskMBpsReadWrite: "1000"
reclaimPolicy: Retain
allowVolumeExpansion: true
volumeBindingMode: WaitForFirstConsumer
```

```bash
kubectl apply -f azure-ultra-disk-sc.yaml
```

## Deploy LauraDB using Helm

### 1. Review Default Values

```bash
# View default Helm values
helm show values ./helm/laura-db > values.yaml

# Edit as needed
vim values.yaml
```

### 2. Create Custom Values File

```yaml
# laura-db-values.yaml
replicaCount: 3

image:
  repository: lauradatabase/laura-db
  tag: "latest"
  pullPolicy: IfNotPresent

service:
  type: LoadBalancer
  port: 8080
  annotations:
    service.beta.kubernetes.io/azure-load-balancer-internal: "false"
    # For internal load balancer:
    # service.beta.kubernetes.io/azure-load-balancer-internal: "true"

persistence:
  enabled: true
  storageClass: "azure-disk-premium"
  accessMode: ReadWriteOnce
  size: 100Gi

resources:
  requests:
    memory: "2Gi"
    cpu: "1000m"
  limits:
    memory: "4Gi"
    cpu: "2000m"

securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  fsGroup: 1000
  seccompProfile:
    type: RuntimeDefault

podSecurityContext:
  runAsUser: 1000
  fsGroup: 1000

autoscaling:
  enabled: true
  minReplicas: 2
  maxReplicas: 10
  targetCPUUtilizationPercentage: 70
  targetMemoryUtilizationPercentage: 80

monitoring:
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
              - key: app
                operator: In
                values:
                  - laura-db
          topologyKey: kubernetes.io/hostname

nodeSelector:
  agentpool: nodepool1

tolerations: []

env:
  - name: LAURA_DB_DATA_DIR
    value: "/data"
  - name: LAURA_DB_LOG_LEVEL
    value: "info"
```

### 3. Install LauraDB

```bash
# Install with custom values
helm install laura-db ./helm/laura-db \
  --namespace laura-db \
  --values laura-db-values.yaml \
  --create-namespace

# Or install from chart repository (if published)
# helm repo add laura-db https://charts.laura-db.io
# helm install laura-db laura-db/laura-db -f laura-db-values.yaml

# Watch deployment
kubectl rollout status statefulset/laura-db -n laura-db
```

### 4. Verify Deployment

```bash
# Check pods
kubectl get pods -n laura-db -o wide

# Check persistent volumes
kubectl get pvc -n laura-db
kubectl get pv

# Check service
kubectl get svc laura-db -n laura-db

# Get external IP
EXTERNAL_IP=$(kubectl get svc laura-db -n laura-db -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
echo "LauraDB is accessible at: http://$EXTERNAL_IP:8080"

# Test connection
curl http://$EXTERNAL_IP:8080/_health
```

## Networking

### Load Balancer Configuration

#### Public Load Balancer (Default)

```yaml
# service-public-lb.yaml
apiVersion: v1
kind: Service
metadata:
  name: laura-db
  annotations:
    service.beta.kubernetes.io/azure-load-balancer-resource-group: "laura-db-rg"
spec:
  type: LoadBalancer
  selector:
    app: laura-db
  ports:
    - protocol: TCP
      port: 8080
      targetPort: 8080
```

#### Internal Load Balancer (Private Network)

```yaml
# service-internal-lb.yaml
apiVersion: v1
kind: Service
metadata:
  name: laura-db
  annotations:
    service.beta.kubernetes.io/azure-load-balancer-internal: "true"
    service.beta.kubernetes.io/azure-load-balancer-internal-subnet: "aks-subnet"
spec:
  type: LoadBalancer
  loadBalancerIP: 10.0.1.100  # Optional static IP
  selector:
    app: laura-db
  ports:
    - protocol: TCP
      port: 8080
      targetPort: 8080
```

### Application Gateway Ingress Controller (AGIC)

For advanced ingress with SSL termination, WAF, and URL routing.

#### Install AGIC

```bash
# Enable AGIC add-on
az aks enable-addons \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --addons ingress-appgw \
  --appgw-name laura-db-appgw \
  --appgw-subnet-cidr "10.0.2.0/24"
```

#### Create Ingress Resource

```yaml
# ingress-appgw.yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: laura-db-ingress
  annotations:
    kubernetes.io/ingress.class: azure/application-gateway
    appgw.ingress.kubernetes.io/ssl-redirect: "true"
    appgw.ingress.kubernetes.io/backend-protocol: "http"
    appgw.ingress.kubernetes.io/health-probe-path: "/_health"
spec:
  tls:
    - hosts:
        - laura-db.example.com
      secretName: laura-db-tls
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

```bash
kubectl apply -f ingress-appgw.yaml
```

### Network Policies

Restrict network traffic to LauraDB pods.

```yaml
# network-policy.yaml
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: laura-db-netpol
  namespace: laura-db
spec:
  podSelector:
    matchLabels:
      app: laura-db
  policyTypes:
    - Ingress
    - Egress
  ingress:
    # Allow from ingress controller
    - from:
        - namespaceSelector:
            matchLabels:
              name: ingress-nginx
      ports:
        - protocol: TCP
          port: 8080
    # Allow from same namespace
    - from:
        - podSelector: {}
  egress:
    # Allow DNS
    - to:
        - namespaceSelector: {}
          podSelector:
            matchLabels:
              k8s-app: kube-dns
      ports:
        - protocol: UDP
          port: 53
    # Allow outbound to Azure services
    - to:
        - namespaceSelector: {}
      ports:
        - protocol: TCP
          port: 443
```

```bash
kubectl apply -f network-policy.yaml
```

## Security

### 1. Azure Active Directory Integration

Enable AAD authentication for cluster access.

```bash
# Already enabled during cluster creation with --enable-aad
# Grant users access
az role assignment create \
  --assignee user@example.com \
  --role "Azure Kubernetes Service Cluster User Role" \
  --scope /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ContainerService/managedClusters/$CLUSTER_NAME
```

### 2. Workload Identity (Managed Identity for Pods)

Replace pod service account tokens with Azure Managed Identity.

#### Enable Workload Identity

```bash
# Enable OIDC issuer
az aks update \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --enable-oidc-issuer \
  --enable-workload-identity

# Get OIDC issuer URL
OIDC_ISSUER=$(az aks show \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --query "oidcIssuerProfile.issuerUrl" -o tsv)
```

#### Create Managed Identity and Role Assignment

```bash
# Create managed identity
az identity create \
  --name laura-db-identity \
  --resource-group $RESOURCE_GROUP

# Get identity info
IDENTITY_CLIENT_ID=$(az identity show \
  --name laura-db-identity \
  --resource-group $RESOURCE_GROUP \
  --query clientId -o tsv)

IDENTITY_PRINCIPAL_ID=$(az identity show \
  --name laura-db-identity \
  --resource-group $RESOURCE_GROUP \
  --query principalId -o tsv)

# Grant permissions (e.g., to access Azure Blob Storage)
az role assignment create \
  --assignee $IDENTITY_PRINCIPAL_ID \
  --role "Storage Blob Data Contributor" \
  --scope /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP

# Establish federated identity credential
az identity federated-credential create \
  --name laura-db-federated-identity \
  --identity-name laura-db-identity \
  --resource-group $RESOURCE_GROUP \
  --issuer $OIDC_ISSUER \
  --subject system:serviceaccount:laura-db:laura-db-sa
```

#### Create Service Account

```yaml
# service-account.yaml
apiVersion: v1
kind: ServiceAccount
metadata:
  name: laura-db-sa
  namespace: laura-db
  annotations:
    azure.workload.identity/client-id: "YOUR_IDENTITY_CLIENT_ID"
```

```bash
kubectl apply -f service-account.yaml
```

#### Update Deployment to Use Workload Identity

```yaml
# In your Helm values or deployment
spec:
  template:
    metadata:
      labels:
        azure.workload.identity/use: "true"
    spec:
      serviceAccountName: laura-db-sa
      containers:
        - name: laura-db
          # ... other config
```

### 3. Azure Key Vault Integration

Store secrets in Azure Key Vault and access from pods.

#### Install Azure Key Vault Provider for Secrets Store CSI Driver

```bash
# Enable add-on
az aks enable-addons \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --addons azure-keyvault-secrets-provider

# Verify installation
kubectl get pods -n kube-system -l app=secrets-store-csi-driver
kubectl get pods -n kube-system -l app=secrets-store-provider-azure
```

#### Create Key Vault and Secrets

```bash
# Create Key Vault
az keyvault create \
  --name laura-db-kv \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION

# Add secret
az keyvault secret set \
  --vault-name laura-db-kv \
  --name db-admin-password \
  --value "YourSecurePassword123!"

# Grant managed identity access to Key Vault
az keyvault set-policy \
  --name laura-db-kv \
  --object-id $IDENTITY_PRINCIPAL_ID \
  --secret-permissions get list
```

#### Create SecretProviderClass

```yaml
# secret-provider-class.yaml
apiVersion: secrets-store.csi.x-k8s.io/v1
kind: SecretProviderClass
metadata:
  name: laura-db-secrets
  namespace: laura-db
spec:
  provider: azure
  parameters:
    usePodIdentity: "false"
    useVMManagedIdentity: "false"
    clientID: "YOUR_IDENTITY_CLIENT_ID"
    keyvaultName: "laura-db-kv"
    cloudName: ""
    objects: |
      array:
        - |
          objectName: db-admin-password
          objectType: secret
          objectVersion: ""
    tenantId: "YOUR_TENANT_ID"
  secretObjects:
    - secretName: laura-db-secrets
      type: Opaque
      data:
        - objectName: db-admin-password
          key: password
```

```bash
kubectl apply -f secret-provider-class.yaml
```

#### Mount Secrets in Pod

```yaml
# In your deployment
spec:
  template:
    spec:
      serviceAccountName: laura-db-sa
      containers:
        - name: laura-db
          env:
            - name: DB_PASSWORD
              valueFrom:
                secretKeyRef:
                  name: laura-db-secrets
                  key: password
          volumeMounts:
            - name: secrets-store
              mountPath: "/mnt/secrets-store"
              readOnly: true
      volumes:
        - name: secrets-store
          csi:
            driver: secrets-store.csi.k8s.io
            readOnly: true
            volumeAttributes:
              secretProviderClass: "laura-db-secrets"
```

### 4. Pod Security Standards

Apply Pod Security Standards to namespace.

```yaml
# pod-security.yaml
apiVersion: v1
kind: Namespace
metadata:
  name: laura-db
  labels:
    pod-security.kubernetes.io/enforce: restricted
    pod-security.kubernetes.io/audit: restricted
    pod-security.kubernetes.io/warn: restricted
```

```bash
kubectl apply -f pod-security.yaml
```

### 5. Azure Policy for AKS

Enforce organizational policies.

```bash
# Enable Azure Policy add-on
az aks enable-addons \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --addons azure-policy

# Assign policy (example: enforce HTTPS ingress)
az policy assignment create \
  --name enforce-https-ingress \
  --policy /providers/Microsoft.Authorization/policyDefinitions/1a5b4dca-0b6f-4cf5-907c-56316bc1bf3d \
  --scope /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ContainerService/managedClusters/$CLUSTER_NAME
```

## Monitoring & Logging

### Azure Monitor Container Insights

Already enabled during cluster creation.

#### View Metrics

```bash
# In Azure Portal:
# AKS Cluster → Monitoring → Insights → Container Insights

# Query logs using Kusto Query Language (KQL)
# Example: Find error logs
ContainerLog
| where LogEntry contains "error"
| where Namespace == "laura-db"
| project TimeGenerated, LogEntry, Computer
| order by TimeGenerated desc
```

#### Create Log Alerts

```bash
# Create action group for notifications
az monitor action-group create \
  --name laura-db-alerts \
  --resource-group $RESOURCE_GROUP \
  --short-name laura-alerts \
  --email-receiver name=admin email=admin@example.com

# Create log alert (example: high error rate)
az monitor scheduled-query create \
  --name "LauraDB High Error Rate" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.ContainerService/managedClusters/$CLUSTER_NAME \
  --condition "count > 10" \
  --condition-query "ContainerLog | where Namespace == 'laura-db' and LogEntry contains 'ERROR' | summarize count() by bin(TimeGenerated, 5m)" \
  --description "Alert when error count exceeds 10 in 5 minutes" \
  --evaluation-frequency 5m \
  --window-size 5m \
  --severity 2 \
  --action-groups /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/microsoft.insights/actionGroups/laura-db-alerts
```

### Prometheus and Grafana (Alternative)

Install Prometheus and Grafana for monitoring.

```bash
# Add Helm repos
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm repo add grafana https://grafana.github.io/helm-charts
helm repo update

# Install kube-prometheus-stack (includes Prometheus, Grafana, and Alertmanager)
helm install kube-prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace \
  --set prometheus.prometheusSpec.storageSpec.volumeClaimTemplate.spec.resources.requests.storage=50Gi \
  --set prometheus.prometheusSpec.retention=30d \
  --set grafana.adminPassword=admin123

# Port-forward to access Grafana
kubectl port-forward -n monitoring svc/kube-prometheus-grafana 3000:80

# Access at http://localhost:3000 (admin/admin123)
```

## Scaling

### Horizontal Pod Autoscaling (HPA)

Scale pods based on CPU/memory usage.

```yaml
# hpa.yaml
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
  minReplicas: 2
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
  behavior:
    scaleDown:
      stabilizationWindowSeconds: 300
      policies:
        - type: Percent
          value: 50
          periodSeconds: 60
    scaleUp:
      stabilizationWindowSeconds: 0
      policies:
        - type: Percent
          value: 100
          periodSeconds: 30
        - type: Pods
          value: 2
          periodSeconds: 30
      selectPolicy: Max
```

```bash
kubectl apply -f hpa.yaml

# Monitor HPA
kubectl get hpa -n laura-db -w
```

### Cluster Autoscaler

Already enabled during cluster creation. Monitor with:

```bash
# View cluster autoscaler logs
kubectl logs -n kube-system -l app=cluster-autoscaler

# Check node count
kubectl get nodes
```

### Vertical Pod Autoscaling (VPA)

Automatically adjust resource requests/limits.

```bash
# Install VPA
git clone https://github.com/kubernetes/autoscaler.git
cd autoscaler/vertical-pod-autoscaler
./hack/vpa-up.sh

# Create VPA
cat <<EOF | kubectl apply -f -
apiVersion: autoscaling.k8s.io/v1
kind: VerticalPodAutoscaler
metadata:
  name: laura-db-vpa
  namespace: laura-db
spec:
  targetRef:
    apiVersion: apps/v1
    kind: StatefulSet
    name: laura-db
  updatePolicy:
    updateMode: "Auto"
  resourcePolicy:
    containerPolicies:
      - containerName: laura-db
        minAllowed:
          cpu: 500m
          memory: 1Gi
        maxAllowed:
          cpu: 4000m
          memory: 8Gi
EOF
```

## Backup & Recovery

### Backup with Velero

Velero provides Kubernetes cluster backup and disaster recovery.

#### Install Velero

```bash
# Create storage account for backups
STORAGE_ACCOUNT_NAME="lauradbbkp$(date +%s)"

az storage account create \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --sku Standard_GRS \
  --encryption-services blob \
  --https-only true \
  --kind BlobStorage \
  --access-tier Hot

# Create blob container
az storage container create \
  --name velero \
  --account-name $STORAGE_ACCOUNT_NAME

# Get storage account key
STORAGE_ACCOUNT_KEY=$(az storage account keys list \
  --resource-group $RESOURCE_GROUP \
  --account-name $STORAGE_ACCOUNT_NAME \
  --query "[0].value" -o tsv)

# Create credentials file
cat <<EOF > credentials-velero
AZURE_STORAGE_ACCOUNT_ACCESS_KEY=$STORAGE_ACCOUNT_KEY
AZURE_CLOUD_NAME=AzurePublicCloud
EOF

# Install Velero CLI
wget https://github.com/vmware-tanzu/velero/releases/download/v1.12.0/velero-v1.12.0-linux-amd64.tar.gz
tar -xvf velero-v1.12.0-linux-amd64.tar.gz
sudo mv velero-v1.12.0-linux-amd64/velero /usr/local/bin/

# Install Velero in cluster
velero install \
  --provider azure \
  --plugins velero/velero-plugin-for-microsoft-azure:v1.8.0 \
  --bucket velero \
  --secret-file ./credentials-velero \
  --backup-location-config resourceGroup=$RESOURCE_GROUP,storageAccount=$STORAGE_ACCOUNT_NAME \
  --snapshot-location-config apiTimeout=5m,resourceGroup=$RESOURCE_GROUP
```

#### Create Backup

```bash
# Backup entire namespace
velero backup create laura-db-backup-$(date +%Y%m%d-%H%M%S) \
  --include-namespaces laura-db \
  --storage-location default

# Backup with volume snapshots
velero backup create laura-db-full-backup \
  --include-namespaces laura-db \
  --snapshot-volumes=true

# Check backup status
velero backup describe laura-db-backup-TIMESTAMP
velero backup logs laura-db-backup-TIMESTAMP

# List backups
velero backup get
```

#### Restore from Backup

```bash
# Restore entire backup
velero restore create --from-backup laura-db-backup-TIMESTAMP

# Restore to different namespace
velero restore create --from-backup laura-db-backup-TIMESTAMP \
  --namespace-mappings laura-db:laura-db-restored

# Monitor restore
velero restore describe RESTORE_NAME
velero restore logs RESTORE_NAME
```

#### Schedule Automatic Backups

```bash
# Create backup schedule (daily at 2 AM)
velero schedule create laura-db-daily \
  --schedule="0 2 * * *" \
  --include-namespaces laura-db \
  --snapshot-volumes=true \
  --ttl 720h0m0s  # 30 days retention

# List schedules
velero schedule get

# Delete old backups
velero backup delete laura-db-backup-TIMESTAMP
```

### Application-Level Backups

For database-specific backups, see [Azure Blob Storage Backup Guide](./blob-storage-backup.md).

## Cost Optimization

### 1. Use Spot Node Pools

Save up to 80% with spot instances for fault-tolerant workloads.

```bash
# Add spot node pool
az aks nodepool add \
  --resource-group $RESOURCE_GROUP \
  --cluster-name $CLUSTER_NAME \
  --name spotpool \
  --priority Spot \
  --eviction-policy Delete \
  --spot-max-price -1 \
  --node-count 2 \
  --min-count 1 \
  --max-count 5 \
  --enable-cluster-autoscaler \
  --node-vm-size Standard_D4s_v3 \
  --node-taints kubernetes.azure.com/scalesetpriority=spot:NoSchedule

# Label spot nodes
kubectl label nodes -l agentpool=spotpool node-role.kubernetes.io/spot=""
```

#### Configure Pods for Spot Nodes

```yaml
# In deployment
spec:
  template:
    spec:
      nodeSelector:
        agentpool: spotpool
      tolerations:
        - key: kubernetes.azure.com/scalesetpriority
          operator: Equal
          value: spot
          effect: NoSchedule
```

### 2. Use Azure Reserved Instances

Save up to 72% with 1 or 3-year commitments.

```bash
# Purchase reserved instances via Azure Portal
# Reservations → Add → Virtual Machines
# Select region, VM size, and term
```

### 3. Right-Size Resources

Analyze and adjust resource requests/limits.

```bash
# Install metrics-server (if not already installed)
kubectl apply -f https://github.com/kubernetes-sigs/metrics-server/releases/latest/download/components.yaml

# View resource usage
kubectl top nodes
kubectl top pods -n laura-db

# Use VPA recommendations
kubectl describe vpa laura-db-vpa
```

### 4. Enable Cluster Auto-scaling

Scale down during low traffic periods.

```bash
# Already enabled during creation
# Configure min/max counts per workload needs
az aks nodepool update \
  --resource-group $RESOURCE_GROUP \
  --cluster-name $CLUSTER_NAME \
  --name nodepool1 \
  --min-count 1 \
  --max-count 5
```

### 5. Use Azure Cost Management

```bash
# Enable cost analysis
az consumption usage list \
  --start-date 2025-01-01 \
  --end-date 2025-01-31 \
  --query "[?contains(instanceName, 'laura-db')]"

# Set budget alerts in Azure Portal
# Cost Management → Budgets → Add
```

### 6. Optimize Storage

```bash
# Use Standard SSD instead of Premium SSD for dev/test
# Delete unused PVs
kubectl get pv | grep Released | awk '{print $1}' | xargs kubectl delete pv

# Use lifecycle policies on blob storage for backups
az storage blob service-properties update \
  --account-name $STORAGE_ACCOUNT_NAME \
  --enable-delete-retention true \
  --delete-retention-days 30
```

## Troubleshooting

### Issue: Pods in Pending State

```bash
# Check pod events
kubectl describe pod POD_NAME -n laura-db

# Common causes:
# 1. Insufficient resources
kubectl get nodes
kubectl describe node NODE_NAME

# 2. PVC not bound
kubectl get pvc -n laura-db
kubectl describe pvc PVC_NAME -n laura-db

# 3. Image pull errors
kubectl get events -n laura-db --sort-by='.lastTimestamp'

# Solutions:
# - Scale up node pool
# - Check storage class and provisioner
# - Verify image name and pull secrets
```

### Issue: Service External IP Stuck in Pending

```bash
# Check service
kubectl describe svc laura-db -n laura-db

# Check load balancer
az network lb list --resource-group MC_*

# Common causes:
# 1. Quota limit reached
# 2. Subnet full
# 3. Network policy blocking

# Solutions:
az vm list-usage --location $LOCATION
az aks update --resource-group $RESOURCE_GROUP --name $CLUSTER_NAME
```

### Issue: High Latency

```bash
# Check pod CPU/memory
kubectl top pods -n laura-db

# Check network policies
kubectl get networkpolicies -n laura-db

# Check Application Gateway (if using AGIC)
az network application-gateway show-backend-health \
  --resource-group $RESOURCE_GROUP \
  --name laura-db-appgw

# Solutions:
# - Increase pod resources
# - Enable pod autoscaling
# - Review network policies
# - Check backend pool health
```

### Issue: PVC Not Mounting

```bash
# Check PVC status
kubectl get pvc -n laura-db
kubectl describe pvc PVC_NAME -n laura-db

# Check storage class
kubectl get storageclass
kubectl describe storageclass STORAGE_CLASS_NAME

# Check CSI driver
kubectl get pods -n kube-system | grep csi

# Solutions:
# - Ensure storage class exists
# - Verify CSI driver is running
# - Check Azure RBAC permissions for cluster managed identity
az role assignment list --assignee $IDENTITY_PRINCIPAL_ID
```

### Issue: Unable to Access Application

```bash
# Check service
kubectl get svc -n laura-db

# Check ingress
kubectl get ingress -n laura-db
kubectl describe ingress INGRESS_NAME -n laura-db

# Check Application Gateway
az network application-gateway show \
  --resource-group $RESOURCE_GROUP \
  --name laura-db-appgw

# Test connectivity from pod
kubectl run -it --rm debug --image=nicolaka/netshoot --restart=Never -- bash
# Inside pod:
curl http://laura-db.laura-db.svc.cluster.local:8080/_health

# Solutions:
# - Verify network security groups
# - Check Application Gateway backend health
# - Review DNS resolution
```

### Issue: Workload Identity Not Working

```bash
# Check service account
kubectl describe sa laura-db-sa -n laura-db

# Check federated identity credential
az identity federated-credential list \
  --identity-name laura-db-identity \
  --resource-group $RESOURCE_GROUP

# Check pod labels
kubectl get pod POD_NAME -n laura-db -o yaml | grep azure.workload.identity

# View pod logs for auth errors
kubectl logs POD_NAME -n laura-db

# Solutions:
# - Verify client ID annotation on service account
# - Ensure pod has label azure.workload.identity/use: "true"
# - Check federated credential subject matches service account
# - Verify OIDC issuer is correct
```

### Useful Debugging Commands

```bash
# View all resources in namespace
kubectl get all -n laura-db

# Get recent events
kubectl get events -n laura-db --sort-by='.lastTimestamp' | tail -20

# Check cluster health
kubectl get cs
kubectl get nodes
kubectl cluster-info dump > cluster-dump.txt

# Check AKS diagnostics
az aks show --resource-group $RESOURCE_GROUP --name $CLUSTER_NAME
az aks check-acr --resource-group $RESOURCE_GROUP --name $CLUSTER_NAME --acr YOUR_ACR_NAME

# Enable diagnostics
az aks update \
  --resource-group $RESOURCE_GROUP \
  --name $CLUSTER_NAME \
  --enable-managed-identity \
  --enable-addons monitoring
```

## Next Steps

- Set up automated backups with Velero or Azure Backup
- Configure Azure Monitor alerts and dashboards
- Implement disaster recovery procedures
- Review [Azure Blob Storage Backup Guide](./blob-storage-backup.md)
- Review [Azure Monitor Integration Guide](./azure-monitor.md)
- Set up CI/CD pipeline for deployments
- Configure custom domains and SSL certificates
- Implement multi-region deployment for global availability

## References

- [AKS Documentation](https://docs.microsoft.com/en-us/azure/aks/)
- [Azure Storage Documentation](https://docs.microsoft.com/en-us/azure/storage/)
- [Helm Documentation](https://helm.sh/docs/)
- [LauraDB Main Documentation](../../README.md)
- [Kubernetes Documentation](https://kubernetes.io/docs/)
- [Velero Documentation](https://velero.io/docs/)

---

**Cost Estimate for Production AKS Deployment**:
- AKS Cluster (3 x Standard_D4s_v3): ~$432/month
- Azure Load Balancer: ~$22/month
- Premium SSD (100GB x 3): ~$51/month
- Application Gateway (optional): ~$140/month
- Log Analytics Workspace: ~$30/month
- Outbound data transfer (100GB): ~$9/month
- **Total**: ~$544-684/month (depending on options)

*Prices based on East US region, standard pricing, as of 2025*
