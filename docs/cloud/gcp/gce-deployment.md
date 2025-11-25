# LauraDB Deployment on Google Compute Engine (GCE)

This guide provides detailed instructions for deploying LauraDB on Google Compute Engine instances.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Architecture Options](#architecture-options)
- [Single Instance Deployment](#single-instance-deployment)
- [Multi-Instance with Load Balancing](#multi-instance-with-load-balancing)
- [Managed Instance Groups](#managed-instance-groups)
- [Storage Configuration](#storage-configuration)
- [Network Configuration](#network-configuration)
- [Security Best Practices](#security-best-practices)
- [Monitoring and Logging](#monitoring-and-logging)
- [Backup and Recovery](#backup-and-recovery)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

GCE provides flexible VM instances for running LauraDB with:
- **Custom machine types**: Optimize CPU and memory independently
- **Preemptible VMs**: Up to 80% cost savings
- **Live migration**: Zero-downtime maintenance
- **Per-second billing**: Pay only for what you use
- **Sustained use discounts**: Automatic discounts for long-running workloads

## Prerequisites

### Required Tools

```bash
# Install gcloud CLI
curl https://sdk.cloud.google.com | bash
exec -l $SHELL

# Initialize gcloud
gcloud init

# Set default project
gcloud config set project PROJECT_ID

# Set default region and zone
gcloud config set compute/region us-central1
gcloud config set compute/zone us-central1-a
```

### Required Permissions

IAM roles needed:
- `roles/compute.admin` - Compute Engine Admin
- `roles/iam.serviceAccountUser` - Service Account User
- `roles/storage.admin` - Storage Admin
- `roles/logging.admin` - Logging Admin

```bash
# Grant permissions to user
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="user:your-email@example.com" \
  --role="roles/compute.admin"
```

## Architecture Options

### Option 1: Single Instance (Development)

```
┌─────────────────────────────┐
│     External IP Address      │
└──────────────┬──────────────┘
               │
        ┌──────▼─────┐
        │    VPC     │
        │            │
        │  ┌──────┐  │
        │  │ GCE  │  │
        │  │Laura │  │
        │  │  DB  │  │
        │  │ +PD  │  │
        │  └──────┘  │
        └────────────┘
```

**Specs**:
- Machine Type: e2-medium (2 vCPU, 4 GB RAM)
- Boot Disk: 50 GB SSD persistent disk
- Estimated Cost: ~$25/month

### Option 2: Multi-Instance with Load Balancer

```
┌──────────────────────────────┐
│    Cloud Load Balancer        │
└────────┬─────────────┬────────┘
         │             │
    ┌────▼───┐    ┌───▼────┐
    │ Zone A │    │ Zone B │
    │  GCE   │    │  GCE   │
    │ +PD    │    │ +PD    │
    └────┬───┘    └───┬────┘
         │            │
         └─────┬──────┘
           ┌───▼───┐
           │Filestore│
           └───────┘
```

**Specs**:
- Machine Type: n2-standard-2 (2 vCPU, 8 GB RAM)
- Storage: 100 GB SSD PD + Filestore for shared data
- Load Balancer: HTTP(S) Load Balancer
- Estimated Cost: ~$180/month

## Single Instance Deployment

### Step 1: Create Service Account

```bash
# Create service account
gcloud iam service-accounts create laura-db-sa \
  --display-name="LauraDB Service Account"

# Grant necessary permissions
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/logging.logWriter"

gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/monitoring.metricWriter"

gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"
```

### Step 2: Create Firewall Rules

```bash
# Allow LauraDB HTTP API
gcloud compute firewall-rules create laura-db-http \
  --allow=tcp:8080 \
  --source-ranges=0.0.0.0/0 \
  --target-tags=laura-db \
  --description="Allow LauraDB HTTP API traffic"

# Allow SSH (optional, for debugging)
gcloud compute firewall-rules create laura-db-ssh \
  --allow=tcp:22 \
  --source-ranges=YOUR_IP/32 \
  --target-tags=laura-db \
  --description="Allow SSH from specific IP"
```

### Step 3: Create Startup Script

Create `startup-script.sh`:

```bash
#!/bin/bash

# Update system
apt-get update
apt-get upgrade -y

# Install dependencies
apt-get install -y wget git

# Install Go
cd /tmp
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

# Create data directory on persistent disk
mkdir -p /mnt/disks/data/laura-db
chown -R laura-db:laura-db /mnt/disks/data/laura-db
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
ExecStart=/opt/laura-db/bin/laura-server -port 8080 -data-dir /mnt/disks/data/laura-db
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
EOF

# Format and mount data disk
DISK_NAME=$(lsblk -o NAME,SERIAL | grep "data-disk" | awk '{print $1}')
if [ -n "$DISK_NAME" ]; then
  if ! blkid /dev/$DISK_NAME; then
    mkfs.ext4 -m 0 -F -E lazy_itable_init=0,lazy_journal_init=0,discard /dev/$DISK_NAME
  fi
  mkdir -p /mnt/disks/data
  mount -o discard,defaults /dev/$DISK_NAME /mnt/disks/data
  UUID=$(blkid /dev/$DISK_NAME -s UUID -o value)
  echo "UUID=$UUID /mnt/disks/data ext4 discard,defaults,nofail 0 2" >> /etc/fstab
fi

# Start service
systemctl daemon-reload
systemctl enable laura-db
systemctl start laura-db

# Install Cloud Ops Agent for monitoring
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
bash add-google-cloud-ops-agent-repo.sh --also-install

echo "LauraDB installation complete"
```

### Step 4: Create Instance

```bash
# Create persistent disk for data
gcloud compute disks create laura-db-data \
  --size=50GB \
  --type=pd-ssd \
  --zone=us-central1-a

# Create instance
gcloud compute instances create laura-db-instance \
  --zone=us-central1-a \
  --machine-type=e2-medium \
  --network-interface=network-tier=PREMIUM,subnet=default \
  --maintenance-policy=MIGRATE \
  --service-account=laura-db-sa@PROJECT_ID.iam.gserviceaccount.com \
  --scopes=https://www.googleapis.com/auth/cloud-platform \
  --tags=laura-db \
  --create-disk=auto-delete=yes,boot=yes,device-name=laura-db-instance,image=projects/ubuntu-os-cloud/global/images/ubuntu-2204-jammy-v20231213,mode=rw,size=20,type=pd-balanced \
  --disk=name=laura-db-data,device-name=data-disk,mode=rw,boot=no \
  --metadata-from-file=startup-script=startup-script.sh \
  --labels=app=laura-db,env=production
```

### Step 5: Verify Installation

```bash
# Get external IP
EXTERNAL_IP=$(gcloud compute instances describe laura-db-instance \
  --zone=us-central1-a \
  --format='get(networkInterfaces[0].accessConfigs[0].natIP)')

echo "LauraDB URL: http://$EXTERNAL_IP:8080"

# Test health endpoint
curl http://$EXTERNAL_IP:8080/_health

# SSH to instance
gcloud compute ssh laura-db-instance --zone=us-central1-a

# Check service status
sudo systemctl status laura-db
```

## Multi-Instance with Load Balancing

### Step 1: Create Instance Template

```bash
# Create instance template with startup script
gcloud compute instance-templates create laura-db-template \
  --machine-type=n2-standard-2 \
  --network-interface=network-tier=PREMIUM,subnet=default \
  --maintenance-policy=MIGRATE \
  --service-account=laura-db-sa@PROJECT_ID.iam.gserviceaccount.com \
  --scopes=https://www.googleapis.com/auth/cloud-platform \
  --tags=laura-db,http-server \
  --create-disk=auto-delete=yes,boot=yes,device-name=laura-db,image=projects/ubuntu-os-cloud/global/images/ubuntu-2204-jammy-v20231213,mode=rw,size=20,type=pd-balanced \
  --metadata-from-file=startup-script=startup-script-filestore.sh \
  --labels=app=laura-db
```

### Step 2: Create Filestore Instance

```bash
# Create Filestore for shared storage
gcloud filestore instances create laura-db-filestore \
  --zone=us-central1-a \
  --tier=BASIC_HDD \
  --file-share=name=data,capacity=1TB \
  --network=name=default
```

Update startup script to mount Filestore:

```bash
# Add to startup-script-filestore.sh
apt-get install -y nfs-common

# Mount Filestore
FILESTORE_IP=$(gcloud filestore instances describe laura-db-filestore \
  --zone=us-central1-a \
  --format='value(networks[0].ipAddresses[0])')

mkdir -p /mnt/filestore
mount $FILESTORE_IP:/data /mnt/filestore
echo "$FILESTORE_IP:/data /mnt/filestore nfs defaults 0 0" >> /etc/fstab

# Use Filestore for data
mkdir -p /mnt/filestore/laura-db
chown -R laura-db:laura-db /mnt/filestore/laura-db
```

### Step 3: Create Health Check

```bash
gcloud compute health-checks create http laura-db-health \
  --port=8080 \
  --request-path=/_health \
  --check-interval=30s \
  --timeout=5s \
  --healthy-threshold=2 \
  --unhealthy-threshold=3
```

### Step 4: Create Backend Service

```bash
gcloud compute backend-services create laura-db-backend \
  --protocol=HTTP \
  --port-name=http \
  --health-checks=laura-db-health \
  --global
```

### Step 5: Create Managed Instance Group

```bash
# Create instance group
gcloud compute instance-groups managed create laura-db-mig \
  --zone=us-central1-a \
  --template=laura-db-template \
  --size=2 \
  --health-check=laura-db-health \
  --initial-delay=300

# Set named port
gcloud compute instance-groups managed set-named-ports laura-db-mig \
  --zone=us-central1-a \
  --named-ports=http:8080

# Add to backend service
gcloud compute backend-services add-backend laura-db-backend \
  --instance-group=laura-db-mig \
  --instance-group-zone=us-central1-a \
  --balancing-mode=UTILIZATION \
  --max-utilization=0.8 \
  --global
```

### Step 6: Create Load Balancer

```bash
# Create URL map
gcloud compute url-maps create laura-db-lb \
  --default-service=laura-db-backend

# Create HTTP proxy
gcloud compute target-http-proxies create laura-db-http-proxy \
  --url-map=laura-db-lb

# Create forwarding rule
gcloud compute forwarding-rules create laura-db-forwarding-rule \
  --global \
  --target-http-proxy=laura-db-http-proxy \
  --ports=80

# Get load balancer IP
LB_IP=$(gcloud compute forwarding-rules describe laura-db-forwarding-rule \
  --global \
  --format='value(IPAddress)')

echo "Load Balancer IP: $LB_IP"
echo "Access LauraDB at: http://$LB_IP"
```

## Managed Instance Groups

### Auto Scaling Configuration

```bash
# Set autoscaling policy
gcloud compute instance-groups managed set-autoscaling laura-db-mig \
  --zone=us-central1-a \
  --max-num-replicas=10 \
  --min-num-replicas=2 \
  --target-cpu-utilization=0.7 \
  --cool-down-period=90
```

### Rolling Updates

```bash
# Update instance template
gcloud compute instance-templates create laura-db-template-v2 \
  --machine-type=n2-standard-4 \
  --source-instance-template=laura-db-template

# Perform rolling update
gcloud compute instance-groups managed rolling-action start-update laura-db-mig \
  --zone=us-central1-a \
  --version=template=laura-db-template-v2 \
  --max-surge=2 \
  --max-unavailable=0
```

## Storage Configuration

### Persistent Disk Types

| Type | IOPS | Throughput | Use Case | Cost/GB/month |
|------|------|------------|----------|---------------|
| pd-standard | Medium | Medium | Development | $0.040 |
| pd-balanced | Good | Good | Production | $0.100 |
| pd-ssd | High | High | High-performance | $0.170 |
| pd-extreme | Highest | Highest | Critical workloads | $0.125 + IOPS |

### Resize Disk

```bash
# Resize persistent disk
gcloud compute disks resize laura-db-data \
  --size=100GB \
  --zone=us-central1-a

# SSH to instance and resize filesystem
sudo resize2fs /dev/sdb
```

### Snapshot for Backup

```bash
# Create snapshot
gcloud compute disks snapshot laura-db-data \
  --zone=us-central1-a \
  --snapshot-names=laura-db-snapshot-$(date +%Y%m%d)

# Create snapshot schedule
gcloud compute resource-policies create snapshot-schedule laura-db-daily \
  --max-retention-days=7 \
  --on-source-disk-delete=keep-auto-snapshots \
  --daily-schedule \
  --start-time=02:00 \
  --storage-location=us-central1

# Attach schedule to disk
gcloud compute disks add-resource-policies laura-db-data \
  --zone=us-central1-a \
  --resource-policies=laura-db-daily
```

## Network Configuration

### VPC Setup

```bash
# Create custom VPC
gcloud compute networks create laura-db-vpc \
  --subnet-mode=custom

# Create subnet
gcloud compute networks subnets create laura-db-subnet \
  --network=laura-db-vpc \
  --range=10.0.1.0/24 \
  --region=us-central1

# Create Cloud Router for NAT
gcloud compute routers create laura-db-router \
  --network=laura-db-vpc \
  --region=us-central1

# Create Cloud NAT
gcloud compute routers nats create laura-db-nat \
  --router=laura-db-router \
  --region=us-central1 \
  --auto-allocate-nat-external-ips \
  --nat-all-subnet-ip-ranges
```

### Firewall Rules

```bash
# Allow internal traffic
gcloud compute firewall-rules create laura-db-internal \
  --network=laura-db-vpc \
  --allow=tcp,udp,icmp \
  --source-ranges=10.0.1.0/24

# Allow health checks
gcloud compute firewall-rules create laura-db-health-check \
  --network=laura-db-vpc \
  --allow=tcp:8080 \
  --source-ranges=35.191.0.0/16,130.211.0.0/22 \
  --target-tags=laura-db
```

## Security Best Practices

### 1. Use Service Accounts

```bash
# Create minimal service account
gcloud iam service-accounts create laura-db-minimal \
  --display-name="LauraDB Minimal Permissions"

# Grant only necessary roles
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:laura-db-minimal@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/logging.logWriter"
```

### 2. Enable Shielded VM

```bash
gcloud compute instances create laura-db-secure \
  --shielded-secure-boot \
  --shielded-vtpm \
  --shielded-integrity-monitoring \
  --machine-type=e2-medium
```

### 3. Use Secret Manager

```bash
# Store admin password
echo -n "admin-password" | gcloud secrets create laura-db-admin-password \
  --data-file=- \
  --replication-policy=automatic

# Grant access to service account
gcloud secrets add-iam-policy-binding laura-db-admin-password \
  --member="serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/secretmanager.secretAccessor"

# Access secret in startup script
ADMIN_PASSWORD=$(gcloud secrets versions access latest --secret=laura-db-admin-password)
```

### 4. Enable OS Patch Management

```bash
# Create patch policy
gcloud compute os-config patch-policies create laura-db-patches \
  --instance-filter-all \
  --patch-window-schedule="0 2 * * 0" \
  --reboot-config=always
```

## Monitoring and Logging

See [cloud-monitoring.md](./cloud-monitoring.md) for detailed monitoring setup.

### Quick Setup

```bash
# Install Cloud Ops Agent (already in startup script)
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
bash add-google-cloud-ops-agent-repo.sh --also-install
```

### View Logs

```bash
# View logs
gcloud logging read "resource.type=gce_instance AND resource.labels.instance_id=laura-db-instance" \
  --limit=50 \
  --format=json

# Tail logs
gcloud logging tail "resource.type=gce_instance AND resource.labels.instance_id=laura-db-instance"
```

## Backup and Recovery

See [cloud-storage-backup.md](./cloud-storage-backup.md) for detailed backup procedures.

### Quick Backup Script

```bash
#!/bin/bash
BACKUP_NAME="laura-db-backup-$(date +%Y%m%d-%H%M%S).tar.gz"
tar czf /tmp/$BACKUP_NAME /mnt/disks/data/laura-db
gsutil cp /tmp/$BACKUP_NAME gs://PROJECT_ID-laura-db-backups/
rm /tmp/$BACKUP_NAME
```

## Cost Optimization

### 1. Use Preemptible VMs for Dev/Test

```bash
gcloud compute instances create laura-db-preemptible \
  --preemptible \
  --machine-type=e2-medium \
  --zone=us-central1-a
```

**Savings**: Up to 80% compared to standard VMs

### 2. Use Committed Use Discounts

```bash
# Purchase 1-year commitment
gcloud compute commitments create laura-db-commitment \
  --resources=vcpu=4,memory=16GB \
  --plan=12-month \
  --region=us-central1
```

**Savings**: Up to 37% (1-year) or 55% (3-year)

### 3. Right-Size Instances

```bash
# Get recommendations
gcloud recommender recommendations list \
  --recommender=google.compute.instance.MachineTypeRecommender \
  --project=PROJECT_ID \
  --location=us-central1-a
```

### 4. Use pd-balanced Instead of pd-ssd

**Cost**: $0.100/GB vs $0.170/GB
**Performance**: Suitable for most workloads

## Troubleshooting

### Instance Not Starting

```bash
# Check serial console output
gcloud compute instances get-serial-port-output laura-db-instance \
  --zone=us-central1-a

# View startup script logs
gcloud compute instances get-serial-port-output laura-db-instance \
  --zone=us-central1-a | grep startup-script
```

### Service Not Running

```bash
# SSH to instance
gcloud compute ssh laura-db-instance --zone=us-central1-a

# Check service status
sudo systemctl status laura-db

# View logs
sudo journalctl -u laura-db -f
```

### Disk Mount Issues

```bash
# List disks
lsblk

# Check mount points
mount | grep /mnt/disks/data

# View fstab
cat /etc/fstab
```

### Load Balancer Issues

```bash
# Check backend health
gcloud compute backend-services get-health laura-db-backend --global

# View instance group status
gcloud compute instance-groups managed list-instances laura-db-mig \
  --zone=us-central1-a
```

## Best Practices

1. ✅ **Use managed instance groups** for production
2. ✅ **Enable auto-scaling** based on CPU/memory
3. ✅ **Use Filestore** for shared storage across instances
4. ✅ **Create snapshot schedules** for automatic backups
5. ✅ **Enable Cloud Ops Agent** for monitoring
6. ✅ **Use service accounts** with minimal permissions
7. ✅ **Enable Shielded VMs** for security
8. ✅ **Implement health checks** for automatic recovery
9. ✅ **Use labels** for cost tracking and organization
10. ✅ **Regular patch management** for security updates

## Next Steps

- [GKE Deployment](./gke-deployment.md)
- [Cloud Storage Backup Integration](./cloud-storage-backup.md)
- [Cloud Monitoring Setup](./cloud-monitoring.md)

## Additional Resources

- [GCE Documentation](https://cloud.google.com/compute/docs)
- [GCE Pricing](https://cloud.google.com/compute/all-pricing)
- [Instance Templates](https://cloud.google.com/compute/docs/instance-templates)
- [Load Balancing](https://cloud.google.com/load-balancing/docs)
