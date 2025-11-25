# LauraDB S3 Backup Integration

This guide provides comprehensive instructions for integrating LauraDB with Amazon S3 for backups, disaster recovery, and long-term data archiving.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [S3 Bucket Setup](#s3-bucket-setup)
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

S3 backup integration provides:
- **Durability**: 99.999999999% (11 nines) durability
- **Availability**: 99.99% availability
- **Cost-effective**: Lower cost than EBS snapshots
- **Scalability**: Unlimited storage
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
# AWS CLI
pip install awscli

# Configure AWS credentials
aws configure

# Verify access
aws s3 ls
```

### Required Permissions

IAM policy for S3 backup operations:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "s3:PutObject",
        "s3:GetObject",
        "s3:DeleteObject",
        "s3:ListBucket",
        "s3:GetBucketLocation",
        "s3:ListBucketVersions",
        "s3:GetObjectVersion"
      ],
      "Resource": [
        "arn:aws:s3:::laura-db-backups",
        "arn:aws:s3:::laura-db-backups/*"
      ]
    },
    {
      "Effect": "Allow",
      "Action": [
        "kms:Decrypt",
        "kms:Encrypt",
        "kms:GenerateDataKey"
      ],
      "Resource": "arn:aws:kms:us-east-1:ACCOUNT_ID:key/*"
    }
  ]
}
```

## S3 Bucket Setup

### Step 1: Create S3 Bucket

```bash
# Create bucket
aws s3api create-bucket \
  --bucket laura-db-backups \
  --region us-east-1 \
  --create-bucket-configuration LocationConstraint=us-east-1

# Add tags
aws s3api put-bucket-tagging \
  --bucket laura-db-backups \
  --tagging 'TagSet=[{Key=Purpose,Value=DatabaseBackups},{Key=Application,Value=LauraDB}]'
```

### Step 2: Enable Versioning

```bash
aws s3api put-bucket-versioning \
  --bucket laura-db-backups \
  --versioning-configuration Status=Enabled
```

### Step 3: Enable Server-Side Encryption

```bash
# SSE-S3 (AWS-managed keys)
aws s3api put-bucket-encryption \
  --bucket laura-db-backups \
  --server-side-encryption-configuration '{
    "Rules": [{
      "ApplyServerSideEncryptionByDefault": {
        "SSEAlgorithm": "AES256"
      }
    }]
  }'

# Or SSE-KMS (Customer-managed keys)
aws s3api put-bucket-encryption \
  --bucket laura-db-backups \
  --server-side-encryption-configuration '{
    "Rules": [{
      "ApplyServerSideEncryptionByDefault": {
        "SSEAlgorithm": "aws:kms",
        "KMSMasterKeyID": "arn:aws:kms:us-east-1:ACCOUNT_ID:key/KEY-ID"
      }
    }]
  }'
```

### Step 4: Enable Access Logging

```bash
# Create logging bucket
aws s3api create-bucket \
  --bucket laura-db-backups-logs \
  --region us-east-1

# Enable logging
aws s3api put-bucket-logging \
  --bucket laura-db-backups \
  --bucket-logging-status '{
    "LoggingEnabled": {
      "TargetBucket": "laura-db-backups-logs",
      "TargetPrefix": "backup-access-logs/"
    }
  }'
```

### Step 5: Configure Bucket Policy

```bash
cat > bucket-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Sid": "DenyUnencryptedObjectUploads",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:PutObject",
      "Resource": "arn:aws:s3:::laura-db-backups/*",
      "Condition": {
        "StringNotEquals": {
          "s3:x-amz-server-side-encryption": "AES256"
        }
      }
    },
    {
      "Sid": "DenyInsecureTransport",
      "Effect": "Deny",
      "Principal": "*",
      "Action": "s3:*",
      "Resource": [
        "arn:aws:s3:::laura-db-backups",
        "arn:aws:s3:::laura-db-backups/*"
      ],
      "Condition": {
        "Bool": {
          "aws:SecureTransport": "false"
        }
      }
    }
  ]
}
EOF

aws s3api put-bucket-policy \
  --bucket laura-db-backups \
  --policy file://bucket-policy.json
```

## IAM Configuration

### Create IAM Role for EC2

```bash
# Create trust policy
cat > trust-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "ec2.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

# Create role
aws iam create-role \
  --role-name LauraDB-Backup-Role \
  --assume-role-policy-document file://trust-policy.json

# Attach policy
aws iam attach-role-policy \
  --role-name LauraDB-Backup-Role \
  --policy-arn arn:aws:iam::ACCOUNT_ID:policy/LauraDB-S3-Backup-Policy

# Create instance profile
aws iam create-instance-profile \
  --instance-profile-name LauraDB-Backup-Profile

# Add role to instance profile
aws iam add-role-to-instance-profile \
  --instance-profile-name LauraDB-Backup-Profile \
  --role-name LauraDB-Backup-Role
```

### Attach Role to EC2 Instance

```bash
aws ec2 associate-iam-instance-profile \
  --instance-id i-xxxxxxxxx \
  --iam-instance-profile Name=LauraDB-Backup-Profile
```

## Backup Strategies

### Strategy 1: Full Daily Backup

**Best for**: Small to medium databases (< 100GB)

```
Daily:  Full backup
Retain: 7 days
```

### Strategy 2: Full + Incremental

**Best for**: Medium to large databases (100GB - 1TB)

```
Daily:     Incremental backup
Weekly:    Full backup
Monthly:   Full backup (archive)
```

### Strategy 3: Continuous WAL Shipping

**Best for**: Critical databases requiring point-in-time recovery

```
Continuous: WAL segments to S3
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
S3_BUCKET="laura-db-backups"
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

# Upload to S3
aws s3 cp $TEMP_DIR/$BACKUP_NAME.tar.gz \
  s3://$S3_BUCKET/full-backups/ \
  --storage-class STANDARD_IA \
  --metadata "backup-type=full,timestamp=$(date -Iseconds)"

aws s3 cp $TEMP_DIR/$BACKUP_NAME.tar.gz.sha256 \
  s3://$S3_BUCKET/full-backups/

# Restart LauraDB
# systemctl start laura-db

# Cleanup
rm -rf $TEMP_DIR

# Delete old backups
echo "[$(date)] Cleaning up old backups (older than $RETENTION_DAYS days)"
aws s3 ls s3://$S3_BUCKET/full-backups/ | \
  awk '{print $4}' | \
  while read -r backup; do
    backup_date=$(echo $backup | sed 's/.*-\([0-9]\{8\}\).*/\1/')
    if [ -n "$backup_date" ]; then
      days_old=$(( ( $(date +%s) - $(date -d "$backup_date" +%s) ) / 86400 ))
      if [ $days_old -gt $RETENTION_DAYS ]; then
        echo "Deleting old backup: $backup (${days_old} days old)"
        aws s3 rm s3://$S3_BUCKET/full-backups/$backup
      fi
    fi
  done

echo "[$(date)] Backup completed: $BACKUP_NAME"
```

Make it executable:

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
S3_BUCKET="laura-db-backups"
BACKUP_NAME="laura-db-incremental-$(date +%Y%m%d-%H%M%S)"
TEMP_DIR="/tmp/laura-db-incremental"
TIMESTAMP_FILE="/var/lib/laura-db/.last-backup-timestamp"

mkdir -p $TEMP_DIR

echo "[$(date)] Starting incremental backup: $BACKUP_NAME"

# Find files modified since last backup
if [ -f "$TIMESTAMP_FILE" ]; then
  LAST_BACKUP=$(cat $TIMESTAMP_FILE)
  find $DATA_DIR -type f -newer $TIMESTAMP_FILE -print0 | \
    tar czf $TEMP_DIR/$BACKUP_NAME.tar.gz --null -T -
else
  echo "No previous backup found, performing full backup"
  tar czf $TEMP_DIR/$BACKUP_NAME.tar.gz -C $DATA_DIR .
fi

# Upload to S3
aws s3 cp $TEMP_DIR/$BACKUP_NAME.tar.gz \
  s3://$S3_BUCKET/incremental-backups/ \
  --storage-class STANDARD_IA

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
Description=LauraDB Backup to S3
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
# Run daily at 2 AM
OnCalendar=daily
OnCalendar=02:00
Persistent=true

[Install]
WantedBy=timers.target
```

Enable and start timer:

```bash
systemctl daemon-reload
systemctl enable laura-db-backup.timer
systemctl start laura-db-backup.timer

# Check timer status
systemctl list-timers
```

### Option 3: AWS Backup Service

Create backup plan using AWS Backup:

```bash
# Create backup vault
aws backup create-backup-vault \
  --backup-vault-name LauraDB-Vault

# Create backup plan
cat > backup-plan.json <<'EOF'
{
  "BackupPlanName": "LauraDB-Daily-Backup",
  "Rules": [
    {
      "RuleName": "DailyBackup",
      "TargetBackupVaultName": "LauraDB-Vault",
      "ScheduleExpression": "cron(0 2 * * ? *)",
      "StartWindowMinutes": 60,
      "CompletionWindowMinutes": 120,
      "Lifecycle": {
        "DeleteAfterDays": 30,
        "MoveToColdStorageAfterDays": 7
      }
    }
  ]
}
EOF

aws backup create-backup-plan --backup-plan file://backup-plan.json
```

## Backup Verification

### Verify Backup Integrity

```bash
#!/bin/bash
S3_BUCKET="laura-db-backups"
BACKUP_FILE="$1"

if [ -z "$BACKUP_FILE" ]; then
  echo "Usage: $0 <backup-file>"
  exit 1
fi

echo "Verifying backup: $BACKUP_FILE"

# Download backup and checksum
aws s3 cp s3://$S3_BUCKET/full-backups/$BACKUP_FILE /tmp/
aws s3 cp s3://$S3_BUCKET/full-backups/$BACKUP_FILE.sha256 /tmp/

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

### Automated Verification

Add to cron:

```bash
# Verify yesterday's backup at 3 AM
0 3 * * * /usr/local/bin/verify-backup.sh laura-db-backup-$(date -d yesterday +\%Y\%m\%d)-020000.tar.gz
```

## Restore Procedures

### Full Restore

```bash
#!/bin/bash
set -e

S3_BUCKET="laura-db-backups"
BACKUP_FILE="$1"
DATA_DIR="/var/lib/laura-db"
RESTORE_DIR="/var/lib/laura-db-restore"

if [ -z "$BACKUP_FILE" ]; then
  echo "Usage: $0 <backup-file>"
  exit 1
fi

echo "[$(date)] Starting restore from: $BACKUP_FILE"

# Stop LauraDB
systemctl stop laura-db

# Download backup
aws s3 cp s3://$S3_BUCKET/full-backups/$BACKUP_FILE /tmp/

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

S3_BUCKET="laura-db-backups"
TARGET_TIME="$1"  # Format: YYYY-MM-DD HH:MM:SS
DATA_DIR="/var/lib/laura-db"

# Find last full backup before target time
LAST_BACKUP=$(aws s3 ls s3://$S3_BUCKET/full-backups/ | \
  awk '{print $4}' | \
  sort -r | \
  while read backup; do
    backup_time=$(echo $backup | sed 's/.*-\([0-9]\{8\}-[0-9]\{6\}\).*/\1/')
    if [[ "$backup_time" < "$(date -d "$TARGET_TIME" +%Y%m%d-%H%M%S)" ]]; then
      echo $backup
      break
    fi
  done)

echo "Using base backup: $LAST_BACKUP"

# Restore full backup
# ... (same as full restore above)

# Apply incremental backups up to target time
aws s3 ls s3://$S3_BUCKET/incremental-backups/ | \
  awk '{print $4}' | \
  sort | \
  while read incremental; do
    incr_time=$(echo $incremental | sed 's/.*-\([0-9]\{8\}-[0-9]\{6\}\).*/\1/')
    if [[ "$incr_time" > "$(echo $LAST_BACKUP | sed 's/.*-\([0-9]\{8\}-[0-9]\{6\}\).*/\1/')" ]] && \
       [[ "$incr_time" < "$(date -d "$TARGET_TIME" +%Y%m%d-%H%M%S)" ]]; then
      echo "Applying incremental: $incremental"
      aws s3 cp s3://$S3_BUCKET/incremental-backups/$incremental /tmp/
      tar xzf /tmp/$incremental -C $DATA_DIR
      rm /tmp/$incremental
    fi
  done

echo "Point-in-time recovery to $TARGET_TIME completed"
```

## Lifecycle Management

### Configure S3 Lifecycle Rules

```bash
cat > lifecycle-policy.json <<'EOF'
{
  "Rules": [
    {
      "Id": "MoveToIA",
      "Status": "Enabled",
      "Filter": {
        "Prefix": "full-backups/"
      },
      "Transitions": [
        {
          "Days": 7,
          "StorageClass": "STANDARD_IA"
        },
        {
          "Days": 30,
          "StorageClass": "GLACIER"
        },
        {
          "Days": 90,
          "StorageClass": "DEEP_ARCHIVE"
        }
      ],
      "Expiration": {
        "Days": 365
      }
    },
    {
      "Id": "DeleteIncrementalBackups",
      "Status": "Enabled",
      "Filter": {
        "Prefix": "incremental-backups/"
      },
      "Expiration": {
        "Days": 30
      }
    }
  ]
}
EOF

aws s3api put-bucket-lifecycle-configuration \
  --bucket laura-db-backups \
  --lifecycle-configuration file://lifecycle-policy.json
```

### Storage Class Comparison

| Storage Class | Use Case | Retrieval Time | Cost (GB/month) |
|--------------|----------|----------------|-----------------|
| STANDARD | Hot data | Instant | $0.023 |
| STANDARD_IA | Backups 1-30 days | Instant | $0.0125 |
| GLACIER | Backups 30-90 days | Minutes-hours | $0.004 |
| DEEP_ARCHIVE | Long-term archive | 12-48 hours | $0.00099 |

## Cross-Region Replication

### Enable S3 CRR

```bash
# Create destination bucket in another region
aws s3api create-bucket \
  --bucket laura-db-backups-replica \
  --region us-west-2 \
  --create-bucket-configuration LocationConstraint=us-west-2

# Create IAM role for replication
cat > replication-role-trust-policy.json <<'EOF'
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Principal": {
        "Service": "s3.amazonaws.com"
      },
      "Action": "sts:AssumeRole"
    }
  ]
}
EOF

aws iam create-role \
  --role-name s3-replication-role \
  --assume-role-policy-document file://replication-role-trust-policy.json

# Configure replication
cat > replication-config.json <<'EOF'
{
  "Role": "arn:aws:iam::ACCOUNT_ID:role/s3-replication-role",
  "Rules": [
    {
      "Status": "Enabled",
      "Priority": 1,
      "Filter": {},
      "Destination": {
        "Bucket": "arn:aws:s3:::laura-db-backups-replica",
        "ReplicationTime": {
          "Status": "Enabled",
          "Time": {
            "Minutes": 15
          }
        },
        "Metrics": {
          "Status": "Enabled"
        }
      },
      "DeleteMarkerReplication": {
        "Status": "Enabled"
      }
    }
  ]
}
EOF

aws s3api put-bucket-replication \
  --bucket laura-db-backups \
  --replication-configuration file://replication-config.json
```

## Encryption

### Enable Encryption at Rest

Already covered in bucket setup. For additional client-side encryption:

```bash
#!/bin/bash
# Encrypt before upload
DATA_FILE="$1"
ENCRYPTED_FILE="${DATA_FILE}.encrypted"
ENCRYPTION_KEY="/etc/laura-db/encryption.key"

# Encrypt using OpenSSL
openssl enc -aes-256-cbc \
  -salt \
  -in $DATA_FILE \
  -out $ENCRYPTED_FILE \
  -pass file:$ENCRYPTION_KEY

# Upload encrypted file
aws s3 cp $ENCRYPTED_FILE s3://laura-db-backups/encrypted/

# Cleanup
rm $ENCRYPTED_FILE
```

### Decrypt on Restore

```bash
#!/bin/bash
ENCRYPTED_FILE="$1"
DECRYPTED_FILE="${ENCRYPTED_FILE%.encrypted}"
ENCRYPTION_KEY="/etc/laura-db/encryption.key"

# Download
aws s3 cp s3://laura-db-backups/encrypted/$ENCRYPTED_FILE /tmp/

# Decrypt
openssl enc -aes-256-cbc \
  -d \
  -in /tmp/$ENCRYPTED_FILE \
  -out /tmp/$DECRYPTED_FILE \
  -pass file:$ENCRYPTION_KEY
```

## Monitoring and Alerts

### CloudWatch Metrics

```bash
# Create CloudWatch alarm for failed backups
aws cloudwatch put-metric-alarm \
  --alarm-name laura-db-backup-failure \
  --alarm-description "Alert when backup fails" \
  --metric-name BackupJobsCompleted \
  --namespace AWS/Backup \
  --statistic Sum \
  --period 86400 \
  --threshold 1 \
  --comparison-operator LessThanThreshold \
  --evaluation-periods 1 \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:ops-alerts
```

### Backup Monitoring Script

```bash
#!/bin/bash
S3_BUCKET="laura-db-backups"
EXPECTED_BACKUP_AGE_HOURS=26  # Alert if no backup in 26 hours

# Get latest backup timestamp
LATEST_BACKUP=$(aws s3 ls s3://$S3_BUCKET/full-backups/ | \
  sort -r | \
  head -n 1 | \
  awk '{print $1" "$2}')

LATEST_TIMESTAMP=$(date -d "$LATEST_BACKUP" +%s)
CURRENT_TIMESTAMP=$(date +%s)
AGE_HOURS=$(( ($CURRENT_TIMESTAMP - $LATEST_TIMESTAMP) / 3600 ))

if [ $AGE_HOURS -gt $EXPECTED_BACKUP_AGE_HOURS ]; then
  echo "⚠️  WARNING: Latest backup is $AGE_HOURS hours old"
  # Send alert via SNS
  aws sns publish \
    --topic-arn arn:aws:sns:us-east-1:ACCOUNT_ID:ops-alerts \
    --subject "LauraDB Backup Alert" \
    --message "Latest backup is $AGE_HOURS hours old. Expected: < $EXPECTED_BACKUP_AGE_HOURS hours"
  exit 1
else
  echo "✓ Backup is current (${AGE_HOURS} hours old)"
fi
```

## Cost Optimization

### Storage Cost Calculation

```
Monthly Backup Cost = (Storage Size × Storage Class Cost) + (API Requests × Request Cost)
```

**Example: 500GB database, daily backups**

```
Full backup size: 500GB
Daily incremental: 50GB
Monthly data: 500GB + (50GB × 30) = 2,000GB

STANDARD_IA cost: 2,000GB × $0.0125 = $25/month
Transfer cost: ~$10/month
Total: ~$35/month
```

### Optimization Tips

1. **Use compression**: Reduces size by 60-80%
2. **Lifecycle policies**: Move to cheaper storage after 7 days
3. **Delete old incrementals**: Keep only last 30 days
4. **Use S3 Intelligent-Tiering**: Auto-moves to cheaper tiers
5. **Deduplication**: Remove duplicate data before upload

## Troubleshooting

### Upload Failures

```bash
# Check S3 permissions
aws s3 ls s3://laura-db-backups/

# Test upload
echo "test" > /tmp/test.txt
aws s3 cp /tmp/test.txt s3://laura-db-backups/test.txt

# Enable debug logging
aws s3 cp file.tar.gz s3://laura-db-backups/ --debug
```

### Slow Uploads

```bash
# Use multipart upload for large files
aws s3 cp large-file.tar.gz s3://laura-db-backups/ \
  --storage-class STANDARD_IA \
  --expected-size 10737418240  # 10GB

# Or use AWS CLI with parallel transfers
aws configure set default.s3.max_concurrent_requests 20
aws configure set default.s3.multipart_threshold 64MB
aws configure set default.s3.multipart_chunksize 16MB
```

### Restore Failures

```bash
# Verify backup exists
aws s3 ls s3://laura-db-backups/full-backups/ | grep backup-name

# Check file integrity
aws s3 cp s3://laura-db-backups/full-backups/backup.tar.gz /tmp/
tar tzf /tmp/backup.tar.gz > /dev/null

# Check disk space
df -h /var/lib/laura-db
```

## Best Practices

1. ✅ **Test restores regularly** (monthly)
2. ✅ **Verify backups** after creation
3. ✅ **Use versioning** for protection against accidental deletion
4. ✅ **Enable MFA delete** for critical backups
5. ✅ **Monitor backup age** with CloudWatch alarms
6. ✅ **Document restore procedures**
7. ✅ **Use lifecycle policies** to manage costs
8. ✅ **Encrypt sensitive data**
9. ✅ **Implement cross-region replication** for DR
10. ✅ **Automate everything** to reduce human error

## Next Steps

- [EC2 Deployment Guide](./ec2-deployment.md)
- [ECS Deployment Guide](./ecs-deployment.md)
- [EKS Deployment Guide](./eks-deployment.md)
- [CloudWatch Monitoring](./cloudwatch-monitoring.md)
- [RDS Comparison](./rds-comparison.md)

## Additional Resources

- [AWS S3 Documentation](https://docs.aws.amazon.com/s3/)
- [AWS Backup Documentation](https://docs.aws.amazon.com/aws-backup/)
- [S3 Best Practices](https://docs.aws.amazon.com/AmazonS3/latest/userguide/best-practices.html)
- [Backup and Recovery Best Practices](https://aws.amazon.com/backup-restore/)
