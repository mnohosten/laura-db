# LauraDB Kubernetes Deployment

This directory contains Kubernetes manifests for deploying LauraDB on Kubernetes clusters.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Quick Start](#quick-start)
- [Directory Structure](#directory-structure)
- [Configuration](#configuration)
- [Deployment](#deployment)
- [Scaling](#scaling)
- [Monitoring](#monitoring)
- [Backup and Recovery](#backup-and-recovery)
- [Troubleshooting](#troubleshooting)
- [Production Considerations](#production-considerations)

## Overview

LauraDB can be deployed on Kubernetes using StatefulSets for data persistence and high availability. The deployment includes:

- **StatefulSet**: Manages LauraDB pods with persistent storage
- **Services**: Exposes LauraDB internally and externally
- **ConfigMap**: Configuration management
- **Secret**: Sensitive data management
- **PersistentVolumeClaims**: Data persistence
- **Ingress**: External HTTP(S) access

## Prerequisites

- Kubernetes cluster (v1.20+)
- kubectl configured to access your cluster
- (Optional) Kustomize for managing configurations
- (Optional) Helm 3.0+ if using Helm charts
- Storage provisioner (for dynamic volume provisioning)

### Verify Prerequisites

```bash
# Check kubectl access
kubectl version

# Check available storage classes
kubectl get storageclass

# Check cluster nodes
kubectl get nodes
```

## Quick Start

### 1. Build and Push Docker Image

```bash
# Build the Docker image
docker build -t laura-db:latest .

# Tag for your registry
docker tag laura-db:latest your-registry/laura-db:latest

# Push to registry
docker push your-registry/laura-db:latest
```

### 2. Deploy with kubectl

```bash
# Deploy to Kubernetes
kubectl apply -f k8s/base/

# Check deployment status
kubectl get pods -n laura-db
kubectl get svc -n laura-db
```

### 3. Deploy with Kustomize

```bash
# Deploy development environment
kubectl apply -k k8s/overlays/dev/

# Or deploy production environment
kubectl apply -k k8s/overlays/prod/
```

### 4. Access LauraDB

```bash
# Port forward to access locally
kubectl port-forward -n laura-db svc/laura-db 8080:8080

# Access the admin console
open http://localhost:8080
```

## Directory Structure

```
k8s/
├── base/                      # Base Kubernetes manifests
│   ├── namespace.yaml         # Namespace definition
│   ├── configmap.yaml         # Configuration
│   ├── secret.yaml            # Secrets
│   ├── pvc.yaml               # Persistent Volume Claims
│   ├── statefulset.yaml       # StatefulSet deployment
│   ├── service.yaml           # Services (headless + LoadBalancer)
│   ├── ingress.yaml           # Ingress configuration
│   └── kustomization.yaml     # Kustomize configuration
├── overlays/
│   ├── dev/                   # Development environment
│   │   ├── kustomization.yaml
│   │   ├── statefulset-patch.yaml
│   │   └── configmap-patch.yaml
│   └── prod/                  # Production environment
│       ├── kustomization.yaml
│       ├── statefulset-patch.yaml
│       └── configmap-patch.yaml
└── README.md                  # This file
```

## Configuration

### ConfigMap

Edit `k8s/base/configmap.yaml` to configure LauraDB:

```yaml
data:
  PORT: "8080"
  DATA_DIR: "/data"
  BUFFER_SIZE: "1000"      # Buffer pool size (pages)
  DOC_CACHE: "1000"        # Document cache size
  WORKER_POOL_SIZE: "4"    # Number of worker threads
  LOG_LEVEL: "info"        # Logging level: debug, info, warn, error
```

### Secrets

Update `k8s/base/secret.yaml` with your credentials:

```bash
# Generate base64 encoded password
echo -n "your-secure-password" | base64

# Generate encryption key
openssl rand -base64 32

# Update secret.yaml with generated values
```

**Important**: In production, use external secret management (e.g., Sealed Secrets, Vault).

### Storage

Configure storage class in `statefulset.yaml`:

```yaml
volumeClaimTemplates:
  - metadata:
      name: data
    spec:
      accessModes:
        - ReadWriteOnce
      storageClassName: fast-ssd  # Change to your storage class
      resources:
        requests:
          storage: 10Gi
```

### Ingress

Update `k8s/base/ingress.yaml` with your domain:

```yaml
spec:
  rules:
  - host: laura-db.your-domain.com  # Change this
    http:
      paths:
      - path: /
        pathType: Prefix
        backend:
          service:
            name: laura-db
            port:
              name: http
```

## Deployment

### Development Environment

```bash
# Apply development configuration
kubectl apply -k k8s/overlays/dev/

# Watch deployment
kubectl get pods -n laura-db -w

# Check logs
kubectl logs -n laura-db -l app=laura-db -f
```

### Production Environment

```bash
# Apply production configuration
kubectl apply -k k8s/overlays/prod/

# Verify deployment
kubectl get statefulset -n laura-db
kubectl get pods -n laura-db
kubectl get svc -n laura-db

# Check persistent volumes
kubectl get pvc -n laura-db
kubectl get pv
```

### Verify Deployment

```bash
# Check all resources
kubectl get all -n laura-db

# Check pod status
kubectl describe pod -n laura-db laura-db-0

# Test health endpoint
kubectl exec -n laura-db laura-db-0 -- curl http://localhost:8080/_health
```

## Scaling

### Horizontal Scaling

```bash
# Scale to 3 replicas
kubectl scale statefulset/laura-db -n laura-db --replicas=3

# Verify scaling
kubectl get pods -n laura-db

# Check each pod
kubectl get pod -n laura-db -o wide
```

### Vertical Scaling

Edit resource limits in `statefulset.yaml`:

```yaml
resources:
  requests:
    memory: "512Mi"
    cpu: "500m"
  limits:
    memory: "2Gi"
    cpu: "2000m"
```

Apply changes:

```bash
kubectl apply -f k8s/base/statefulset.yaml
```

## Monitoring

### Health Checks

LauraDB includes built-in health checks:

```bash
# Check liveness probe
kubectl get pod -n laura-db laura-db-0 -o jsonpath='{.spec.containers[0].livenessProbe}'

# Check readiness probe
kubectl get pod -n laura-db laura-db-0 -o jsonpath='{.spec.containers[0].readinessProbe}'

# Test health endpoint
kubectl exec -n laura-db laura-db-0 -- curl http://localhost:8080/_health
```

### Logs

```bash
# View logs from all pods
kubectl logs -n laura-db -l app=laura-db --tail=100

# Follow logs
kubectl logs -n laura-db laura-db-0 -f

# View logs from previous pod instance
kubectl logs -n laura-db laura-db-0 --previous
```

### Metrics

```bash
# Get pod metrics (requires metrics-server)
kubectl top pod -n laura-db

# Get node metrics
kubectl top node

# Describe pod for detailed metrics
kubectl describe pod -n laura-db laura-db-0
```

### Integration with Prometheus

Add annotations to StatefulSet for Prometheus scraping:

```yaml
metadata:
  annotations:
    prometheus.io/scrape: "true"
    prometheus.io/port: "8080"
    prometheus.io/path: "/_metrics"
```

## Backup and Recovery

### Manual Backup

```bash
# Create backup directory
kubectl exec -n laura-db laura-db-0 -- mkdir -p /data/backups

# Trigger backup (if LauraDB supports backup command)
kubectl exec -n laura-db laura-db-0 -- /app/laura-server backup --output /data/backups/backup-$(date +%Y%m%d).db

# Copy backup to local machine
kubectl cp laura-db/laura-db-0:/data/backups/backup-20250124.db ./backup-20250124.db
```

### Automated Backups with CronJob

Create a CronJob for scheduled backups:

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: laura-db-backup
  namespace: laura-db
spec:
  schedule: "0 2 * * *"  # Daily at 2 AM
  jobTemplate:
    spec:
      template:
        spec:
          containers:
          - name: backup
            image: laura-db:latest
            command:
            - /bin/sh
            - -c
            - |
              DATE=$(date +%Y%m%d-%H%M%S)
              /app/laura-server backup --output /backups/backup-${DATE}.db
              # Upload to S3/GCS if needed
          volumeMounts:
          - name: backup-storage
            mountPath: /backups
          restartPolicy: OnFailure
          volumes:
          - name: backup-storage
            persistentVolumeClaim:
              claimName: laura-db-backups
```

### Disaster Recovery

```bash
# In case of data loss, restore from backup

# Copy backup to pod
kubectl cp ./backup-20250124.db laura-db/laura-db-0:/data/restore.db

# Stop the StatefulSet
kubectl scale statefulset/laura-db -n laura-db --replicas=0

# Restore data (implementation-specific)
kubectl exec -n laura-db laura-db-0 -- /app/laura-server restore --input /data/restore.db

# Restart the StatefulSet
kubectl scale statefulset/laura-db -n laura-db --replicas=3
```

## Troubleshooting

### Common Issues

#### Pod Not Starting

```bash
# Check pod events
kubectl describe pod -n laura-db laura-db-0

# Check pod logs
kubectl logs -n laura-db laura-db-0

# Common causes:
# - Image pull errors
# - Insufficient resources
# - Volume mount issues
# - Configuration errors
```

#### Storage Issues

```bash
# Check PVC status
kubectl get pvc -n laura-db

# Check PV status
kubectl get pv

# Describe PVC for events
kubectl describe pvc -n laura-db laura-db-data-laura-db-0

# Common causes:
# - Storage class not available
# - Insufficient storage quota
# - Volume provisioner issues
```

#### Service Not Accessible

```bash
# Check service endpoints
kubectl get endpoints -n laura-db

# Test service internally
kubectl run -it --rm debug --image=alpine --restart=Never -n laura-db -- sh
# Inside the pod:
# apk add curl
# curl http://laura-db:8080/_health

# Check ingress status
kubectl describe ingress -n laura-db laura-db

# Common causes:
# - Pod not ready
# - Service selector mismatch
# - Network policy blocking traffic
# - Ingress controller not installed
```

#### Performance Issues

```bash
# Check resource usage
kubectl top pod -n laura-db

# Check resource limits
kubectl describe pod -n laura-db laura-db-0 | grep -A 10 "Limits"

# Check for CPU throttling
kubectl describe pod -n laura-db laura-db-0 | grep -i throttl

# Solutions:
# - Increase resource limits
# - Scale horizontally
# - Optimize buffer pool size
# - Add node affinity rules
```

### Debug Commands

```bash
# Access pod shell
kubectl exec -it -n laura-db laura-db-0 -- /bin/sh

# Check disk usage
kubectl exec -n laura-db laura-db-0 -- df -h

# Check process status
kubectl exec -n laura-db laura-db-0 -- ps aux

# Network debugging
kubectl run -it --rm debug --image=nicolaka/netshoot --restart=Never -n laura-db -- bash
```

## Production Considerations

### High Availability

1. **Multiple Replicas**: Run at least 3 replicas for high availability
2. **Pod Disruption Budget**: Prevent all pods from being down simultaneously
3. **Anti-Affinity**: Spread pods across different nodes
4. **Health Checks**: Configure appropriate liveness and readiness probes

```yaml
# Pod Disruption Budget
apiVersion: policy/v1
kind: PodDisruptionBudget
metadata:
  name: laura-db-pdb
  namespace: laura-db
spec:
  minAvailable: 2
  selector:
    matchLabels:
      app: laura-db
```

### Security

1. **Use Secrets**: Never hardcode credentials
2. **RBAC**: Implement Role-Based Access Control
3. **Network Policies**: Restrict network access
4. **TLS**: Enable TLS for ingress
5. **Security Context**: Run as non-root user

```yaml
# Network Policy example
apiVersion: networking.k8s.io/v1
kind: NetworkPolicy
metadata:
  name: laura-db-network-policy
  namespace: laura-db
spec:
  podSelector:
    matchLabels:
      app: laura-db
  policyTypes:
  - Ingress
  - Egress
  ingress:
  - from:
    - namespaceSelector:
        matchLabels:
          name: frontend
    ports:
    - protocol: TCP
      port: 8080
```

### Performance

1. **Storage Class**: Use SSD storage class for better performance
2. **Resource Limits**: Set appropriate CPU and memory limits
3. **Buffer Pool**: Configure buffer pool size based on available memory
4. **Node Affinity**: Pin to high-performance nodes

### Monitoring and Alerting

1. **Metrics**: Integrate with Prometheus/Grafana
2. **Logging**: Aggregate logs with ELK/Loki
3. **Alerts**: Set up alerts for critical metrics
4. **Dashboards**: Create monitoring dashboards

### Backup Strategy

1. **Automated Backups**: Schedule regular backups
2. **Retention Policy**: Define backup retention periods
3. **Off-site Storage**: Store backups in S3/GCS
4. **Test Restores**: Regularly test backup restoration

### Cost Optimization

1. **Right-size Resources**: Don't over-provision
2. **Use Spot Instances**: For non-critical workloads
3. **Storage Tiers**: Use appropriate storage tiers
4. **Resource Quotas**: Set namespace quotas

## Additional Resources

- [LauraDB Documentation](../docs/)
- [Docker Deployment](../docker-compose.yml)
- [Performance Tuning](../docs/performance-tuning.md)
- [Kubernetes Best Practices](https://kubernetes.io/docs/concepts/configuration/overview/)

## Support

For issues and questions:
- GitHub Issues: https://github.com/mnohosten/laura-db/issues
- Documentation: ../docs/
