# LauraDB Deployment on Google Kubernetes Engine (GKE)

This guide provides detailed instructions for deploying LauraDB on Google Kubernetes Engine (GKE).

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture](#architecture)
- [Creating GKE Cluster](#creating-gke-cluster)
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

GKE is Google's managed Kubernetes service offering:
- **Autopilot mode**: Fully managed, hands-off Kubernetes
- **Standard mode**: More control over node configuration
- **Regional clusters**: Multi-zone high availability
- **Node auto-repair**: Automatic node health maintenance
- **Node auto-upgrade**: Automatic Kubernetes version updates
- **Workload Identity**: Secure service account integration

## Prerequisites

### Required Tools

```bash
# Install gcloud CLI
curl https://sdk.cloud.google.com | bash

# Install kubectl
gcloud components install kubectl

# Install Helm
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Configure gcloud
gcloud init
gcloud config set project PROJECT_ID
gcloud config set compute/region us-central1
```

### Required APIs

```bash
# Enable required APIs
gcloud services enable container.googleapis.com
gcloud services enable compute.googleapis.com
gcloud services enable storage-api.googleapis.com
gcloud services enable logging.googleapis.com
gcloud services enable monitoring.googleapis.com
```

### Required Permissions

- `roles/container.admin` - GKE Admin
- `roles/iam.serviceAccountUser` - Service Account User
- `roles/compute.admin` - Compute Admin

## Architecture

### Production GKE Architecture

```
┌────────────────────────────────────┐
│    Cloud Load Balancer (HTTPS)     │
└──────────────┬─────────────────────┘
               │
       ┌───────┴────────┐
       │  Ingress GCE   │
       └───────┬────────┘
               │
    ┌──────────┴──────────┐
    │                     │
┌───▼────────┐    ┌──────▼─────┐
│  Zone A    │    │   Zone B   │
│ GKE Node   │    │  GKE Node  │
│  Pod 1     │    │   Pod 2    │
│  +PD       │    │   +PD      │
└────┬───────┘    └──────┬─────┘
     │                   │
     └────────┬──────────┘
          ┌───▼───┐
          │Filestore│
          └───────┘
```

### Components

- **GKE Control Plane**: Fully managed by Google
- **Worker Nodes**: e2, n2, or c2 machine types
- **Persistent Disks**: For pod storage
- **Filestore**: For shared storage
- **Cloud Load Balancer**: External traffic ingress
- **Workload Identity**: Secure GCP service access

## Creating GKE Cluster

### Option 1: Autopilot Cluster (Recommended for Simplicity)

```bash
# Create Autopilot cluster
gcloud container clusters create-auto laura-db-autopilot \
  --region=us-central1 \
  --release-channel=regular \
  --enable-private-nodes \
  --enable-private-endpoint \
  --enable-master-authorized-networks \
  --master-authorized-networks=YOUR_IP/32 \
  --network=default \
  --subnetwork=default

# Get credentials
gcloud container clusters get-credentials laura-db-autopilot --region=us-central1
```

**Autopilot Benefits**:
- No node management
- Automatic scaling
- Optimized resource allocation
- Lower operational overhead

### Option 2: Standard Cluster (More Control)

```bash
# Create Standard cluster
gcloud container clusters create laura-db-standard \
  --region=us-central1 \
  --num-nodes=1 \
  --machine-type=e2-standard-4 \
  --disk-type=pd-standard \
  --disk-size=100 \
  --enable-autoscaling \
  --min-nodes=1 \
  --max-nodes=10 \
  --enable-autorepair \
  --enable-autoupgrade \
  --enable-ip-alias \
  --network=default \
  --subnetwork=default \
  --enable-stackdriver-kubernetes \
  --addons=HorizontalPodAutoscaling,HttpLoadBalancing,GcePersistentDiskCsiDriver \
  --workload-pool=PROJECT_ID.svc.id.goog \
  --enable-shielded-nodes \
  --shielded-secure-boot \
  --shielded-integrity-monitoring \
  --release-channel=regular

# Get credentials
gcloud container clusters get-credentials laura-db-standard --region=us-central1
```

### Option 3: Using gcloud Config File

Create `cluster-config.yaml`:

```yaml
apiVersion: container.cnrm.cloud.google.com/v1beta1
kind: ContainerCluster
metadata:
  name: laura-db-cluster
spec:
  location: us-central1
  initialNodeCount: 1

  nodeConfig:
    machineType: e2-standard-4
    diskSizeGb: 100
    diskType: pd-standard
    oauthScopes:
      - "https://www.googleapis.com/auth/cloud-platform"
    shieldedInstanceConfig:
      enableSecureBoot: true
      enableIntegrityMonitoring: true

  autoscaling:
    enabled: true
    minNodeCount: 1
    maxNodeCount: 10

  addonsConfig:
    horizontalPodAutoscaling:
      disabled: false
    httpLoadBalancing:
      disabled: false
    gcePersistentDiskCsiDriver:
      enabled: true

  workloadIdentityConfig:
    workloadPool: PROJECT_ID.svc.id.goog

  releaseChannel:
    channel: REGULAR
```

### Verify Cluster

```bash
# Check cluster status
gcloud container clusters describe laura-db-standard --region=us-central1

# Verify kubectl connection
kubectl cluster-info
kubectl get nodes
```

## Deploying with Helm

### Step 1: Build and Push Docker Image

```bash
# Authenticate to Container Registry
gcloud auth configure-docker

# Or use Artifact Registry (recommended)
gcloud auth configure-docker us-central1-docker.pkg.dev

# Build image
cd /path/to/laura-db
docker build -t gcr.io/PROJECT_ID/laura-db:latest .

# Or for Artifact Registry
docker build -t us-central1-docker.pkg.dev/PROJECT_ID/laura-db/laura-db:latest .

# Push image
docker push gcr.io/PROJECT_ID/laura-db:latest
```

### Step 2: Install with Helm

```bash
# Create namespace
kubectl create namespace laura-db

# Install with default values
helm install laura-db ./helm/laura-db \
  --namespace laura-db \
  --set image.repository=gcr.io/PROJECT_ID/laura-db \
  --set image.tag=latest

# Or create custom values file
cat > gke-values.yaml <<EOF
replicaCount: 3

image:
  repository: gcr.io/PROJECT_ID/laura-db
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
  storageClass: standard-rwo
  size: 100Gi

service:
  type: LoadBalancer
  annotations:
    cloud.google.com/load-balancer-type: "External"

ingress:
  enabled: true
  className: gce
  annotations:
    kubernetes.io/ingress.class: gce
    kubernetes.io/ingress.global-static-ip-name: laura-db-ip
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
        topologyKey: topology.kubernetes.io/zone
EOF

# Install with custom values
helm install laura-db ./helm/laura-db \
  --namespace laura-db \
  -f gke-values.yaml
```

### Step 3: Verify Deployment

```bash
# Check pods
kubectl get pods -n laura-db

# Check service
kubectl get svc -n laura-db

# Get LoadBalancer IP
kubectl get svc laura-db -n laura-db -o jsonpath='{.status.loadBalancer.ingress[0].ip}'

# Test health endpoint
export LB_IP=$(kubectl get svc laura-db -n laura-db -o jsonpath='{.status.loadBalancer.ingress[0].ip}')
curl http://$LB_IP:8080/_health
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

### Option 1: Persistent Disks (Default)

GKE automatically provisions Persistent Disks when using PVCs.

**Standard StorageClass** (already available):

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: standard-rwo
provisioner: pd.csi.storage.gke.io
parameters:
  type: pd-standard
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

**SSD StorageClass**:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: ssd-rwo
provisioner: pd.csi.storage.gke.io
parameters:
  type: pd-ssd
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

**Balanced StorageClass** (recommended):

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: balanced-rwo
provisioner: pd.csi.storage.gke.io
parameters:
  type: pd-balanced
volumeBindingMode: WaitForFirstConsumer
allowVolumeExpansion: true
```

Apply:

```bash
kubectl apply -f balanced-storageclass.yaml

# Set as default
kubectl patch storageclass balanced-rwo -p '{"metadata": {"annotations":{"storageclass.kubernetes.io/is-default-class":"true"}}}'
```

### Option 2: Filestore for Shared Storage

**Create Filestore Instance**:

```bash
# Create Filestore
gcloud filestore instances create laura-db-filestore \
  --zone=us-central1-a \
  --tier=BASIC_HDD \
  --file-share=name=data,capacity=1TB \
  --network=name=default

# Get Filestore IP
FILESTORE_IP=$(gcloud filestore instances describe laura-db-filestore \
  --zone=us-central1-a \
  --format='value(networks[0].ipAddresses[0])')
```

**Install Filestore CSI Driver**:

```bash
kubectl apply -f https://raw.githubusercontent.com/kubernetes-sigs/gcp-filestore-csi-driver/master/deploy/kubernetes/overlays/stable/deploy.yaml
```

**Create StorageClass for Filestore**:

```yaml
apiVersion: storage.k8s.io/v1
kind: StorageClass
metadata:
  name: filestore
provisioner: filestore.csi.storage.gke.io
parameters:
  tier: standard
  network: default
volumeBindingMode: Immediate
allowVolumeExpansion: true
```

**Create PV and PVC**:

```yaml
apiVersion: v1
kind: PersistentVolume
metadata:
  name: laura-db-filestore-pv
spec:
  capacity:
    storage: 1Ti
  accessModes:
  - ReadWriteMany
  storageClassName: filestore
  csi:
    driver: filestore.csi.storage.gke.io
    volumeHandle: "modeInstance/us-central1-a/laura-db-filestore/data"
    volumeAttributes:
      ip: FILESTORE_IP
      volume: data
---
apiVersion: v1
kind: PersistentVolumeClaim
metadata:
  name: laura-db-filestore-pvc
  namespace: laura-db
spec:
  accessModes:
  - ReadWriteMany
  storageClassName: filestore
  resources:
    requests:
      storage: 1Ti
```

## Load Balancing

### Option 1: GCE Load Balancer (via Ingress)

**Reserve Static IP**:

```bash
gcloud compute addresses create laura-db-ip \
  --global

# Get IP address
gcloud compute addresses describe laura-db-ip --global
```

**Create Ingress**:

```yaml
apiVersion: networking.k8s.io/v1
kind: Ingress
metadata:
  name: laura-db
  namespace: laura-db
  annotations:
    kubernetes.io/ingress.class: "gce"
    kubernetes.io/ingress.global-static-ip-name: "laura-db-ip"
    ingress.gcp.kubernetes.io/pre-shared-cert: "laura-db-ssl-cert"
    networking.gke.io/managed-certificates: "laura-db-managed-cert"
spec:
  rules:
  - host: laura-db.example.com
    http:
      paths:
      - path: /*
        pathType: ImplementationSpecific
        backend:
          service:
            name: laura-db
            port:
              number: 8080
```

**Create Managed Certificate**:

```yaml
apiVersion: networking.gke.io/v1
kind: ManagedCertificate
metadata:
  name: laura-db-managed-cert
  namespace: laura-db
spec:
  domains:
    - laura-db.example.com
```

### Option 2: Network Load Balancer (via Service)

```yaml
apiVersion: v1
kind: Service
metadata:
  name: laura-db-nlb
  namespace: laura-db
  annotations:
    cloud.google.com/load-balancer-type: "External"
spec:
  type: LoadBalancer
  selector:
    app.kubernetes.io/name: laura-db
  ports:
  - port: 80
    targetPort: 8080
```

## Auto Scaling

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

### Cluster Autoscaler

Cluster autoscaler is enabled by default in GKE.

**Verify**:

```bash
kubectl get deployment cluster-autoscaler -n kube-system
```

**Configure node pool autoscaling**:

```bash
gcloud container clusters update laura-db-standard \
  --enable-autoscaling \
  --min-nodes=1 \
  --max-nodes=10 \
  --zone=us-central1-a
```

### Vertical Pod Autoscaler (VPA)

```bash
# Install VPA
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/vertical-pod-autoscaler/deploy/vpa-v1-crd-gen.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/vertical-pod-autoscaler/deploy/vpa-rbac.yaml
kubectl apply -f https://raw.githubusercontent.com/kubernetes/autoscaler/master/vertical-pod-autoscaler/deploy/vpa-deployment.yaml
```

**Create VPA**:

```yaml
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
```

## Secrets Management

### Option 1: Google Secret Manager with External Secrets

**Install External Secrets Operator**:

```bash
helm repo add external-secrets https://charts.external-secrets.io
helm install external-secrets \
  external-secrets/external-secrets \
  -n external-secrets-system \
  --create-namespace
```

**Create secrets in Secret Manager**:

```bash
# Create admin password
echo -n "admin-password" | gcloud secrets create laura-db-admin-password \
  --data-file=- \
  --replication-policy=automatic

# Create encryption key
echo -n "$(openssl rand -base64 32)" | gcloud secrets create laura-db-encryption-key \
  --data-file=- \
  --replication-policy=automatic
```

**Setup Workload Identity**:

```bash
# Create Google Service Account
gcloud iam service-accounts create laura-db-gsa \
  --display-name="LauraDB Service Account"

# Grant Secret Manager access
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:laura-db-gsa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Bind to Kubernetes Service Account
gcloud iam service-accounts add-iam-policy-binding \
  laura-db-gsa@PROJECT_ID.iam.gserviceaccount.com \
  --role roles/iam.workloadIdentityUser \
  --member "serviceAccount:PROJECT_ID.svc.id.goog[laura-db/laura-db]"

# Annotate K8s service account
kubectl annotate serviceaccount laura-db \
  -n laura-db \
  iam.gke.io/gcp-service-account=laura-db-gsa@PROJECT_ID.iam.gserviceaccount.com
```

**Create SecretStore**:

```yaml
apiVersion: external-secrets.io/v1beta1
kind: SecretStore
metadata:
  name: gcpsm-secret-store
  namespace: laura-db
spec:
  provider:
    gcpsm:
      projectID: "PROJECT_ID"
      auth:
        workloadIdentity:
          clusterLocation: us-central1
          clusterName: laura-db-standard
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
    name: gcpsm-secret-store
    kind: SecretStore
  target:
    name: laura-db-admin
    creationPolicy: Owner
  data:
  - secretKey: admin-password
    remoteRef:
      key: laura-db-admin-password
  - secretKey: encryption-key
    remoteRef:
      key: laura-db-encryption-key
```

## Monitoring and Logging

### Cloud Monitoring (Stackdriver)

GKE automatically sends metrics to Cloud Monitoring.

**View metrics in console**:
```
https://console.cloud.google.com/monitoring/dashboards
```

**Install Prometheus for custom metrics**:

```bash
helm repo add prometheus-community https://prometheus-community.github.io/helm-charts
helm install prometheus prometheus-community/kube-prometheus-stack \
  --namespace monitoring \
  --create-namespace
```

### Cloud Logging

**View logs**:

```bash
# View pod logs via gcloud
gcloud logging read "resource.type=k8s_container AND resource.labels.namespace_name=laura-db" \
  --limit=50 \
  --format=json

# Or use kubectl
kubectl logs -n laura-db -l app.kubernetes.io/name=laura-db -f
```

**Log-based metrics**:

```bash
# Create log metric for errors
gcloud logging metrics create laura_db_errors \
  --description="LauraDB error count" \
  --log-filter='resource.type="k8s_container"
    resource.labels.namespace_name="laura-db"
    severity>=ERROR'
```

## Security Best Practices

### 1. Enable Workload Identity

Already configured in cluster creation. Verify:

```bash
gcloud container clusters describe laura-db-standard \
  --region=us-central1 \
  --format="value(workloadIdentityConfig.workloadPool)"
```

### 2. Network Policies

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

### 3. Pod Security Standards

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

### 4. Binary Authorization

```bash
# Enable Binary Authorization
gcloud container clusters update laura-db-standard \
  --enable-binauthz \
  --region=us-central1
```

## Cost Optimization

### 1. Use Preemptible Nodes

```bash
gcloud container node-pools create preemptible-pool \
  --cluster=laura-db-standard \
  --preemptible \
  --machine-type=e2-standard-4 \
  --num-nodes=2 \
  --region=us-central1
```

**Savings**: Up to 80% compared to regular nodes

### 2. Use Spot Pods (Autopilot)

```yaml
apiVersion: v1
kind: Pod
metadata:
  labels:
    cloud.google.com/gke-spot: "true"
spec:
  nodeSelector:
    cloud.google.com/gke-spot: "true"
  tolerations:
  - key: cloud.google.com/gke-spot
    operator: Equal
    value: "true"
    effect: NoSchedule
```

### 3. Right-Size Pods

```bash
# Get VPA recommendations
kubectl describe vpa laura-db-vpa -n laura-db
```

### 4. Use Committed Use Discounts

Purchase 1 or 3-year commitments for GKE nodes:
- 1-year: 25% discount
- 3-year: 52% discount

### 5. Autopilot vs Standard Cost Comparison

| Workload Size | Standard (monthly) | Autopilot (monthly) | Winner |
|---------------|-------------------|---------------------|---------|
| Small (< 10 pods) | $150 | $120 | Autopilot |
| Medium (10-50 pods) | $300 | $280 | Autopilot |
| Large (> 50 pods) | $800 | $900 | Standard |

## Troubleshooting

### Pods Stuck in Pending

```bash
# Describe pod to see events
kubectl describe pod laura-db-0 -n laura-db

# Common causes:
# - Insufficient cluster capacity
# - PVC binding issues
# - Node affinity/taints not matching
```

### PVC Not Binding

```bash
# Check PVC status
kubectl get pvc -n laura-db

# Describe PVC
kubectl describe pvc data-laura-db-0 -n laura-db

# Check if StorageClass exists
kubectl get storageclass
```

### LoadBalancer Not Getting External IP

```bash
# Check service
kubectl describe svc laura-db -n laura-db

# Check events
kubectl get events -n laura-db --sort-by='.lastTimestamp'

# Verify firewall rules
gcloud compute firewall-rules list
```

### Workload Identity Issues

```bash
# Verify service account annotation
kubectl get serviceaccount laura-db -n laura-db -o yaml

# Check IAM binding
gcloud iam service-accounts get-iam-policy \
  laura-db-gsa@PROJECT_ID.iam.gserviceaccount.com

# Test from pod
kubectl run -it --rm debug \
  --image=gcr.io/google.com/cloudsdktool/cloud-sdk:slim \
  --serviceaccount=laura-db \
  -n laura-db \
  -- gcloud auth list
```

### High Network Latency

```bash
# Check if pods are in different zones
kubectl get pods -n laura-db -o wide

# Use topology-aware routing
kubectl annotate service laura-db \
  -n laura-db \
  service.kubernetes.io/topology-aware-hints=auto
```

## Best Practices

1. ✅ **Use Autopilot** for simplified operations (if workload fits)
2. ✅ **Enable Workload Identity** for secure GCP service access
3. ✅ **Use regional clusters** for high availability
4. ✅ **Enable Binary Authorization** for image security
5. ✅ **Implement Network Policies** for pod-to-pod security
6. ✅ **Use Managed Certificates** for TLS
7. ✅ **Enable Cloud Monitoring and Logging**
8. ✅ **Use VPA** for right-sizing recommendations
9. ✅ **Implement PodDisruptionBudgets** for availability
10. ✅ **Use preemptible nodes** for cost savings on non-critical workloads

## Next Steps

- [GCE Deployment](./gce-deployment.md)
- [Cloud Storage Backup Integration](./cloud-storage-backup.md)
- [Cloud Monitoring Setup](./cloud-monitoring.md)

## Additional Resources

- [GKE Documentation](https://cloud.google.com/kubernetes-engine/docs)
- [GKE Best Practices](https://cloud.google.com/kubernetes-engine/docs/best-practices)
- [GKE Pricing](https://cloud.google.com/kubernetes-engine/pricing)
- [Workload Identity](https://cloud.google.com/kubernetes-engine/docs/how-to/workload-identity)
