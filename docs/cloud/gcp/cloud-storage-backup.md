# LauraDB Cloud Storage Backup Integration

This guide provides comprehensive instructions for integrating LauraDB with Google Cloud Storage for backups, disaster recovery, and long-term data archiving.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Cloud Storage Bucket Setup](#cloud-storage-bucket-setup)
- [IAM Configuration](#iam-configuration)
- [Backup Strategies](#backup-strategies)
- [Manual Backup](#manual-backup)
- [Automated Backup](#automated-backup)
- [Backup Verification](#backup-verification)
- [Restore Procedures](#restore-procedures)
- [Lifecycle Management](#lifecycle-management)
- [Cross-Region Replication](#cross-region-replication)
- [Encryption](#encryption)
- [Monitoring and Alerts](#monitoring-and-alerts)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

Cloud Storage backup integration provides:
- **Durability**: 99.999999999% (11 nines) durability
- **Availability**: 99.95% availability (Standard class)
- **Cost-effective**: Multiple storage classes for different use cases
- **Scalability**: Unlimited storage capacity
- **Versioning**: Keep multiple backup versions
- **Lifecycle policies**: Automatic tiering to cheaper storage classes

### Backup Types

1. **Full Backup**: Complete database copy
2. **Incremental Backup**: Only changed data since last backup
3. **Snapshot Backup**: Point-in-time copy
4. **Continuous Backup**: WAL (Write-Ahead Log) shipping

## Prerequisites

### Required Tools

```bash
# Install gsutil (part of gcloud SDK)
gcloud components install gsutil

# Configure authentication
gcloud auth login
gcloud config set project PROJECT_ID
```

### Required Permissions

IAM roles needed:
- `roles/storage.objectAdmin` - Storage Object Admin
- `roles/storage.admin` - Storage Admin (for bucket creation)

```bash
# Grant permissions to service account
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/storage.objectAdmin"
```

## Cloud Storage Bucket Setup

### Step 1: Create Storage Bucket

```bash
# Create bucket
gsutil mb -p PROJECT_ID \
  -c STANDARD \
  -l US-CENTRAL1 \
  -b on \
  gs://PROJECT_ID-laura-db-backups

# Add labels
gsutil label ch -l purpose:database-backups gs://PROJECT_ID-laura-db-backups
gsutil label ch -l application:lauradb gs://PROJECT_ID-laura-db-backups
```

### Step 2: Enable Versioning

```bash
gsutil versioning set on gs://PROJECT_ID-laura-db-backups
```

### Step 3: Set Default Encryption

```bash
# Use Google-managed encryption keys (default)
gsutil encryption set \
  -d gs://PROJECT_ID-laura-db-backups

# Or use customer-managed encryption keys (CMEK)
gcloud kms keyrings create laura-db-keyring \
  --location=us-central1

gcloud kms keys create laura-db-key \
  --location=us-central1 \
  --keyring=laura-db-keyring \
  --purpose=encryption

gsutil encryption set \
  -k projects/PROJECT_ID/locations/us-central1/keyRings/laura-db-keyring/cryptoKeys/laura-db-key \
  gs://PROJECT_ID-laura-db-backups
```

### Step 4: Configure Uniform Bucket-Level Access

```bash
# Enable uniform bucket-level access
gsutil uniformbucketlevelaccess set on gs://PROJECT_ID-laura-db-backups
```

### Step 5: Set Bucket Policy

```bash
# Create policy to deny unencrypted uploads
cat > bucket-policy.json <<'EOF'
{
  "bindings": [
    {
      "role": "roles/storage.objectAdmin",
      "members": [
        "serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com"
      ]
    }
  ],
  "conditions": [
    {
      "title": "DenyUnencryptedUploads",
      "expression": "!request.header['x-goog-encryption-algorithm']"
    }
  ]
}
EOF

gsutil iam set bucket-policy.json gs://PROJECT_ID-laura-db-backups
```

## IAM Configuration

### Create Service Account for Backups

```bash
# Create service account
gcloud iam service-accounts create laura-db-backup \
  --display-name="LauraDB Backup Service Account"

# Grant storage permissions
gsutil iam ch \
  serviceAccount:laura-db-backup@PROJECT_ID.iam.gserviceaccount.com:objectAdmin \
  gs://PROJECT_ID-laura-db-backups

# For GCE instances, attach service account
gcloud compute instances set-service-account INSTANCE_NAME \
  --zone=us-central1-a \
  --service-account=laura-db-backup@PROJECT_ID.iam.gserviceaccount.com \
  --scopes=https://www.googleapis.com/auth/cloud-platform
```

## Backup Strategies

### Strategy 1: Full Daily Backup

**Best for**: Small to medium databases (< 100GB)

```
Daily:  Full backup
Retain: 7 days
Storage: Standard → Nearline after 7 days
```

### Strategy 2: Full + Incremental

**Best for**: Medium to large databases (100GB - 1TB)

```
Daily:     Incremental backup
Weekly:    Full backup
Monthly:   Full backup (archive)
Storage:   Standard → Nearline → Coldline → Archive
```

### Strategy 3: Continuous WAL Shipping

**Best for**: Critical databases requiring point-in-time recovery

```
Continuous: WAL segments to Cloud Storage
Daily:      Full backup
Retain:     30 days of WAL
```

## Manual Backup

### Full Backup Script

Create `/usr/local/bin/laura-db-backup.sh`:

```bash
#!/bin/bash
set -e

# Configuration
DATA_DIR="/var/lib/laura-db"
BUCKET="gs://PROJECT_ID-laura-db-backups"
BACKUP_NAME="laura-db-backup-$(date +%Y%m%d-%H%M%S)"
TEMP_DIR="/tmp/laura-db-backup"
RETENTION_DAYS=7

# Create temporary directory
mkdir -p $TEMP_DIR

echo "[$(date)] Starting backup: $BACKUP_NAME"

# Stop LauraDB (optional, for consistent backup)
# systemctl stop laura-db

# Create backup archive
tar czf $TEMP_DIR/$BACKUP_NAME.tar.gz \
  -C $DATA_DIR \
  --exclude='*.tmp' \
  --exclude='*.lock' \
  .

# Calculate checksum
sha256sum $TEMP_DIR/$BACKUP_NAME.tar.gz > $TEMP_DIR/$BACKUP_NAME.tar.gz.sha256

# Upload to Cloud Storage
gsutil -m cp $TEMP_DIR/$BACKUP_NAME.tar.gz \
  $BUCKET/full-backups/

gsutil -m cp $TEMP_DIR/$BACKUP_NAME.tar.gz.sha256 \
  $BUCKET/full-backups/

# Set storage class to Nearline for cost savings
gsutil setmeta -h "x-goog-storage-class:NEARLINE" \
  $BUCKET/full-backups/$BACKUP_NAME.tar.gz

# Add metadata
gsutil setmeta \
  -h "x-goog-meta-backup-type:full" \
  -h "x-goog-meta-timestamp:$(date -Iseconds)" \
  $BUCKET/full-backups/$BACKUP_NAME.tar.gz

# Restart LauraDB
# systemctl start laura-db

# Cleanup
rm -rf $TEMP_DIR

# Delete old backups
echo "[$(date)] Cleaning up old backups (older than $RETENTION_DAYS days)"
gsutil ls $BUCKET/full-backups/ | \
  while read -r backup; do
    backup_date=$(basename $backup | grep -oP '\d{8}' | head -1)
    if [ -n "$backup_date" ]; then
      days_old=$(( ( $(date +%s) - $(date -d "$backup_date" +%s) ) / 86400 ))
      if [ $days_old -gt $RETENTION_DAYS ]; then
        echo "Deleting old backup: $backup (${days_old} days old)"
        gsutil rm $backup
      fi
    fi
  done

echo "[$(date)] Backup completed: $BACKUP_NAME"
```

Make executable:

```bash
chmod +x /usr/local/bin/laura-db-backup.sh
```

Run manually:

```bash
sudo /usr/local/bin/laura-db-backup.sh
```

### Incremental Backup Script

Create `/usr/local/bin/laura-db-incremental-backup.sh`:

```bash
#!/bin/bash
set -e

DATA_DIR="/var/lib/laura-db"
BUCKET="gs://PROJECT_ID-laura-db-backups"
BACKUP_NAME="laura-db-incremental-$(date +%Y%m%d-%H%M%S)"
TEMP_DIR="/tmp/laura-db-incremental"
TIMESTAMP_FILE="/var/lib/laura-db/.last-backup-timestamp"

mkdir -p $TEMP_DIR

echo "[$(date)] Starting incremental backup: $BACKUP_NAME"

# Find files modified since last backup
if [ -f "$TIMESTAMP_FILE" ]; then
  find $DATA_DIR -type f -newer $TIMESTAMP_FILE -print0 | \
    tar czf $TEMP_DIR/$BACKUP_NAME.tar.gz --null -T -
else
  echo "No previous backup found, performing full backup"
  tar czf $TEMP_DIR/$BACKUP_NAME.tar.gz -C $DATA_DIR .
fi

# Upload to Cloud Storage
gsutil cp $TEMP_DIR/$BACKUP_NAME.tar.gz \
  $BUCKET/incremental-backups/

# Set storage class
gsutil setmeta -h "x-goog-storage-class:NEARLINE" \
  $BUCKET/incremental-backups/$BACKUP_NAME.tar.gz

# Update timestamp
date +%s > $TIMESTAMP_FILE

# Cleanup
rm -rf $TEMP_DIR

echo "[$(date)] Incremental backup completed: $BACKUP_NAME"
```

## Automated Backup

### Option 1: Cron Job

```bash
# Edit crontab
crontab -e

# Add daily backup at 2 AM
0 2 * * * /usr/local/bin/laura-db-backup.sh >> /var/log/laura-db-backup.log 2>&1

# Add hourly incremental backup
0 * * * * /usr/local/bin/laura-db-incremental-backup.sh >> /var/log/laura-db-incremental-backup.log 2>&1
```

### Option 2: Systemd Timer

Create `/etc/systemd/system/laura-db-backup.service`:

```ini
[Unit]
Description=LauraDB Backup to Cloud Storage
After=network.target

[Service]
Type=oneshot
ExecStart=/usr/local/bin/laura-db-backup.sh
User=root
StandardOutput=journal
StandardError=journal
```

Create `/etc/systemd/system/laura-db-backup.timer`:

```ini
[Unit]
Description=LauraDB Backup Timer
Requires=laura-db-backup.service

[Timer]
OnCalendar=daily
OnCalendar=02:00
Persistent=true

[Install]
WantedBy=timers.target
```

Enable timer:

```bash
systemctl daemon-reload
systemctl enable laura-db-backup.timer
systemctl start laura-db-backup.timer

# Check timer status
systemctl list-timers
```

### Option 3: Cloud Scheduler + Cloud Functions

**Create Cloud Function**:

```python
# backup_function.py
import os
import subprocess
from datetime import datetime
from google.cloud import storage

def backup_laura_db(request):
    """Trigger backup via HTTP request"""

    bucket_name = os.environ.get('BUCKET_NAME')
    instance_name = os.environ.get('INSTANCE_NAME')
    zone = os.environ.get('ZONE')

    timestamp = datetime.now().strftime('%Y%m%d-%H%M%S')
    backup_name = f'laura-db-backup-{timestamp}.tar.gz'

    # Trigger backup via gcloud compute ssh
    cmd = [
        'gcloud', 'compute', 'ssh', instance_name,
        '--zone', zone,
        '--command', f'/usr/local/bin/laura-db-backup.sh'
    ]

    result = subprocess.run(cmd, capture_output=True, text=True)

    if result.returncode == 0:
        return {'status': 'success', 'backup': backup_name}
    else:
        return {'status': 'error', 'message': result.stderr}, 500
```

**Deploy Cloud Function**:

```bash
gcloud functions deploy laura-db-backup \
  --runtime=python39 \
  --trigger-http \
  --entry-point=backup_laura_db \
  --set-env-vars=BUCKET_NAME=PROJECT_ID-laura-db-backups,INSTANCE_NAME=laura-db-instance,ZONE=us-central1-a
```

**Create Cloud Scheduler Job**:

```bash
gcloud scheduler jobs create http laura-db-daily-backup \
  --schedule="0 2 * * *" \
  --uri="https://us-central1-PROJECT_ID.cloudfunctions.net/laura-db-backup" \
  --http-method=POST \
  --time-zone="America/New_York"
```

## Backup Verification

### Verify Backup Integrity

```bash
#!/bin/bash
BUCKET="gs://PROJECT_ID-laura-db-backups"
BACKUP_FILE="$1"

if [ -z "$BACKUP_FILE" ]; then
  echo "Usage: $0 <backup-file>"
  exit 1
fi

echo "Verifying backup: $BACKUP_FILE"

# Download backup and checksum
gsutil cp $BUCKET/full-backups/$BACKUP_FILE /tmp/
gsutil cp $BUCKET/full-backups/$BACKUP_FILE.sha256 /tmp/

# Verify checksum
cd /tmp
if sha256sum -c $BACKUP_FILE.sha256; then
  echo "✓ Backup integrity verified"

  # Test extraction
  mkdir -p /tmp/restore-test
  tar xzf $BACKUP_FILE -C /tmp/restore-test

  if [ $? -eq 0 ]; then
    echo "✓ Backup can be extracted successfully"
    rm -rf /tmp/restore-test
  else
    echo "✗ Failed to extract backup"
    exit 1
  fi
else
  echo "✗ Backup checksum verification failed"
  exit 1
fi

# Cleanup
rm -f /tmp/$BACKUP_FILE /tmp/$BACKUP_FILE.sha256

echo "✓ Backup verification complete"
```

## Restore Procedures

### Full Restore

```bash
#!/bin/bash
set -e

BUCKET="gs://PROJECT_ID-laura-db-backups"
BACKUP_FILE="$1"
DATA_DIR="/var/lib/laura-db"

if [ -z "$BACKUP_FILE" ]; then
  echo "Usage: $0 <backup-file>"
  exit 1
fi

echo "[$(date)] Starting restore from: $BACKUP_FILE"

# Stop LauraDB
systemctl stop laura-db

# Download backup
gsutil cp $BUCKET/full-backups/$BACKUP_FILE /tmp/

# Backup current data (just in case)
if [ -d "$DATA_DIR" ]; then
  mv $DATA_DIR $DATA_DIR.backup-$(date +%Y%m%d-%H%M%S)
fi

# Extract backup
mkdir -p $DATA_DIR
tar xzf /tmp/$BACKUP_FILE -C $DATA_DIR

# Set permissions
chown -R laura-db:laura-db $DATA_DIR
chmod 755 $DATA_DIR

# Start LauraDB
systemctl start laura-db

# Wait for health check
sleep 10
if curl -f http://localhost:8080/_health; then
  echo "[$(date)] ✓ Restore completed successfully"
  rm -f /tmp/$BACKUP_FILE
else
  echo "[$(date)] ✗ Health check failed after restore"
  exit 1
fi
```

### Point-in-Time Recovery (PITR)

```bash
#!/bin/bash
set -e

BUCKET="gs://PROJECT_ID-laura-db-backups"
TARGET_TIME="$1"  # Format: YYYY-MM-DD HH:MM:SS
DATA_DIR="/var/lib/laura-db"

# Find last full backup before target time
LAST_BACKUP=$(gsutil ls $BUCKET/full-backups/ | \
  grep -oP 'laura-db-backup-\d{8}-\d{6}' | \
  sort -r | \
  while read backup; do
    backup_time=$(echo $backup | grep -oP '\d{8}-\d{6}')
    if [[ "$backup_time" < "$(date -d "$TARGET_TIME" +%Y%m%d-%H%M%S)" ]]; then
      echo $backup
      break
    fi
  done)

echo "Using base backup: $LAST_BACKUP"

# Restore full backup
# ... (same as full restore)

# Apply incremental backups up to target time
gsutil ls $BUCKET/incremental-backups/ | \
  grep -oP 'laura-db-incremental-\d{8}-\d{6}' | \
  sort | \
  while read incremental; do
    incr_time=$(echo $incremental | grep -oP '\d{8}-\d{6}')
    if [[ "$incr_time" > "$(echo $LAST_BACKUP | grep -oP '\d{8}-\d{6}')" ]] && \
       [[ "$incr_time" < "$(date -d "$TARGET_TIME" +%Y%m%d-%H%M%S)" ]]; then
      echo "Applying incremental: $incremental"
      gsutil cp $BUCKET/incremental-backups/$incremental.tar.gz /tmp/
      tar xzf /tmp/$incremental.tar.gz -C $DATA_DIR
      rm /tmp/$incremental.tar.gz
    fi
  done

echo "Point-in-time recovery to $TARGET_TIME completed"
```

## Lifecycle Management

### Configure Object Lifecycle

```bash
cat > lifecycle-policy.json <<'EOF'
{
  "lifecycle": {
    "rule": [
      {
        "action": {
          "type": "SetStorageClass",
          "storageClass": "NEARLINE"
        },
        "condition": {
          "age": 7,
          "matchesPrefix": ["full-backups/"]
        }
      },
      {
        "action": {
          "type": "SetStorageClass",
          "storageClass": "COLDLINE"
        },
        "condition": {
          "age": 30,
          "matchesPrefix": ["full-backups/"]
        }
      },
      {
        "action": {
          "type": "SetStorageClass",
          "storageClass": "ARCHIVE"
        },
        "condition": {
          "age": 90,
          "matchesPrefix": ["full-backups/"]
        }
      },
      {
        "action": {
          "type": "Delete"
        },
        "condition": {
          "age": 365,
          "matchesPrefix": ["full-backups/"]
        }
      },
      {
        "action": {
          "type": "Delete"
        },
        "condition": {
          "age": 30,
          "matchesPrefix": ["incremental-backups/"]
        }
      }
    ]
  }
}
EOF

gsutil lifecycle set lifecycle-policy.json gs://PROJECT_ID-laura-db-backups
```

### Storage Class Comparison

| Storage Class | Use Case | Retrieval Time | Cost (GB/month) |
|--------------|----------|----------------|-----------------|
| STANDARD | Hot data | Instant | $0.020 |
| NEARLINE | Backups 1-30 days | Instant | $0.010 |
| COLDLINE | Backups 30-90 days | Instant | $0.004 |
| ARCHIVE | Long-term archive | Hours | $0.0012 |

## Cross-Region Replication

### Enable Dual-Region Bucket

```bash
# Create dual-region bucket
gsutil mb -p PROJECT_ID \
  -c STANDARD \
  -l US \
  -b on \
  gs://PROJECT_ID-laura-db-backups-dr

# Enable versioning
gsutil versioning set on gs://PROJECT_ID-laura-db-backups-dr
```

### Setup Transfer Service

```bash
# Create transfer job
gcloud transfer jobs create \
  gs://PROJECT_ID-laura-db-backups \
  gs://PROJECT_ID-laura-db-backups-dr \
  --schedule-repeats-every=1d \
  --schedule-repeats-until=2025-12-31 \
  --delete-from=destination-if-unique \
  --overwrite-when=different
```

## Encryption

### Client-Side Encryption

```bash
#!/bin/bash
# Encrypt before upload
DATA_FILE="$1"
ENCRYPTED_FILE="${DATA_FILE}.enc"
KEY_FILE="/etc/laura-db/encryption.key"

# Encrypt using OpenSSL
openssl enc -aes-256-cbc \
  -salt \
  -in $DATA_FILE \
  -out $ENCRYPTED_FILE \
  -pass file:$KEY_FILE

# Upload encrypted file
gsutil cp $ENCRYPTED_FILE gs://PROJECT_ID-laura-db-backups/encrypted/

# Cleanup
rm $ENCRYPTED_FILE
```

### Customer-Managed Encryption Keys (CMEK)

Already configured in bucket setup. Verify:

```bash
gsutil encryption get gs://PROJECT_ID-laura-db-backups
```

## Monitoring and Alerts

### Cloud Monitoring Metrics

```bash
# Create log-based metric for backup success
gcloud logging metrics create laura_db_backup_success \
  --description="LauraDB backup success count" \
  --log-filter='textPayload=~"Backup completed"'

# Create log-based metric for backup failures
gcloud logging metrics create laura_db_backup_failure \
  --description="LauraDB backup failure count" \
  --log-filter='textPayload=~"Backup failed" OR severity>=ERROR'
```

### Create Alerts

```bash
# Create alert policy for backup failures
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="LauraDB Backup Failure" \
  --condition-display-name="Backup failed" \
  --condition-threshold-value=1 \
  --condition-threshold-duration=60s \
  --condition-filter='metric.type="logging.googleapis.com/user/laura_db_backup_failure"'
```

### Backup Monitoring Script

```bash
#!/bin/bash
BUCKET="gs://PROJECT_ID-laura-db-backups"
EXPECTED_BACKUP_AGE_HOURS=26

# Get latest backup timestamp
LATEST_BACKUP=$(gsutil ls -l $BUCKET/full-backups/ | \
  grep tar.gz | \
  sort -k2 -r | \
  head -n 1 | \
  awk '{print $2}')

LATEST_TIMESTAMP=$(date -d "$LATEST_BACKUP" +%s)
CURRENT_TIMESTAMP=$(date +%s)
AGE_HOURS=$(( ($CURRENT_TIMESTAMP - $LATEST_TIMESTAMP) / 3600 ))

if [ $AGE_HOURS -gt $EXPECTED_BACKUP_AGE_HOURS ]; then
  echo "⚠️  WARNING: Latest backup is $AGE_HOURS hours old"
  # Send alert
  gcloud logging write laura-db-backup \
    "Latest backup is $AGE_HOURS hours old. Expected: < $EXPECTED_BACKUP_AGE_HOURS hours" \
    --severity=ERROR
  exit 1
else
  echo "✓ Backup is current (${AGE_HOURS} hours old)"
fi
```

## Cost Optimization

### Storage Cost Calculation

```
Monthly Cost = (Storage Size × Storage Class Cost) + (API Operations × Operation Cost)
```

**Example: 500GB database, daily backups**

```
Full backup size: 500GB
Daily incremental: 50GB
Monthly data: 500GB + (50GB × 30) = 2,000GB

NEARLINE cost: 2,000GB × $0.010 = $20/month
Retrieval cost: Minimal (only when restoring)
Total: ~$20-25/month
```

### Optimization Tips

1. **Use compression**: Reduces size by 60-80%
2. **Lifecycle policies**: Move to cheaper storage after 7 days
3. **Delete old incrementals**: Keep only last 30 days
4. **Use Nearline/Coldline**: Much cheaper than Standard
5. **Deduplication**: Remove duplicate data before upload
6. **Composite uploads**: Faster for large files

## Troubleshooting

### Upload Failures

```bash
# Check permissions
gsutil iam get gs://PROJECT_ID-laura-db-backups

# Test upload
echo "test" > /tmp/test.txt
gsutil cp /tmp/test.txt gs://PROJECT_ID-laura-db-backups/test.txt

# Enable debug logging
gsutil -D cp file.tar.gz gs://PROJECT_ID-laura-db-backups/
```

### Slow Uploads

```bash
# Use parallel composite uploads for large files
gsutil -o GSUtil:parallel_composite_upload_threshold=150M cp \
  large-file.tar.gz gs://PROJECT_ID-laura-db-backups/

# Increase parallel threads
gsutil -m -o GSUtil:parallel_thread_count=24 cp \
  backup.tar.gz gs://PROJECT_ID-laura-db-backups/
```

### Restore Failures

```bash
# Verify backup exists
gsutil ls gs://PROJECT_ID-laura-db-backups/full-backups/ | grep backup-name

# Check file integrity
gsutil cat gs://PROJECT_ID-laura-db-backups/full-backups/backup.tar.gz | tar tz > /dev/null

# Check disk space
df -h /var/lib/laura-db
```

## Best Practices

1. ✅ **Test restores regularly** (monthly)
2. ✅ **Verify backups** after creation
3. ✅ **Use versioning** for protection
4. ✅ **Enable encryption** (CMEK or client-side)
5. ✅ **Monitor backup age** with alerts
6. ✅ **Document restore procedures**
7. ✅ **Use lifecycle policies** to manage costs
8. ✅ **Implement cross-region replication** for DR
9. ✅ **Automate everything** to reduce errors
10. ✅ **Use Object Versioning** for accidental deletion protection

## Next Steps

- [GCE Deployment Guide](./gce-deployment.md)
- [GKE Deployment Guide](./gke-deployment.md)
- [Cloud Monitoring Setup](./cloud-monitoring.md)

## Additional Resources

- [Cloud Storage Documentation](https://cloud.google.com/storage/docs)
- [Cloud Storage Pricing](https://cloud.google.com/storage/pricing)
- [Backup Best Practices](https://cloud.google.com/architecture/best-practices-for-using-cloud-storage-for-data-backup)
- [gsutil Tool](https://cloud.google.com/storage/docs/gsutil)
