# LauraDB Backup Integration with Azure Blob Storage

Complete guide for implementing automated backup and disaster recovery for LauraDB using Azure Blob Storage.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Setup Azure Blob Storage](#setup-azure-blob-storage)
- [Authentication Methods](#authentication-methods)
- [Backup Strategies](#backup-strategies)
- [Backup Scripts](#backup-scripts)
- [Restore Procedures](#restore-procedures)
- [Automation](#automation)
- [Lifecycle Management](#lifecycle-management)
- [Monitoring & Alerts](#monitoring--alerts)
- [Cost Optimization](#cost-optimization)
- [Disaster Recovery](#disaster-recovery)
- [Best Practices](#best-practices)

## Overview

Azure Blob Storage provides durable, scalable object storage for LauraDB backups with built-in features for lifecycle management, encryption, and geo-redundancy.

### Key Features

- **Durability**: 99.999999999% (11 nines) durability
- **Redundancy Options**: LRS, ZRS, GRS, RA-GRS, GZRS, RA-GZRS
- **Access Tiers**: Hot, Cool, Cold, Archive for cost optimization
- **Lifecycle Management**: Automatic tier transitions and deletion
- **Versioning**: Protect against accidental deletion or modification
- **Soft Delete**: Recover deleted blobs within retention period
- **Immutability**: WORM (Write Once Read Many) policies for compliance
- **Encryption**: Automatic encryption at rest with Microsoft or customer-managed keys

### Backup Architecture

```
┌──────────────────┐
│   LauraDB VM     │
│   or AKS Pod     │
└────────┬─────────┘
         │ backup script
         │ (tar + compress)
         ▼
┌──────────────────┐      Lifecycle      ┌──────────────────┐
│  Blob Storage    │──────────────────▶  │  Cool/Archive    │
│   (Hot Tier)     │     Policies        │     Tier         │
│  Full Backups    │                     │  Old Backups     │
│  Incremental     │                     └──────────────────┘
└──────────────────┘
         │
         │ Replication
         ▼
┌──────────────────┐
│  Secondary       │
│  Region (GRS)    │
│  Geo-redundant   │
└──────────────────┘
```

## Prerequisites

### Tools Required

```bash
# Install Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Verify installation
az --version

# Login
az login

# Set subscription
az account set --subscription "Your Subscription Name"
```

### Azure Storage Tools

```bash
# Install azcopy (fast blob upload/download)
wget https://aka.ms/downloadazcopy-v10-linux
tar -xvf downloadazcopy-v10-linux
sudo cp ./azcopy_linux_amd64_*/azcopy /usr/local/bin/
sudo chmod 755 /usr/local/bin/azcopy

# Verify
azcopy --version

# Or use Azure Storage blob SDK (for Go applications)
# Already included in LauraDB if using Azure SDK
```

## Setup Azure Blob Storage

### 1. Create Storage Account

```bash
RESOURCE_GROUP="laura-db-rg"
LOCATION="eastus"
STORAGE_ACCOUNT_NAME="lauradbbkp$(date +%s)"  # Must be globally unique

# Create storage account with GRS (geo-redundant storage)
az storage account create \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --sku Standard_GRS \
  --kind StorageV2 \
  --access-tier Hot \
  --encryption-services blob file \
  --https-only true \
  --min-tls-version TLS1_2 \
  --allow-blob-public-access false \
  --tags application=laura-db purpose=backup

# Enable versioning
az storage account blob-service-properties update \
  --account-name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --enable-versioning true

# Enable soft delete (30 days retention)
az storage account blob-service-properties update \
  --account-name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --enable-delete-retention true \
  --delete-retention-days 30

# Enable container soft delete
az storage account blob-service-properties update \
  --account-name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --enable-container-delete-retention true \
  --container-delete-retention-days 30
```

### Storage Account SKU Options

| SKU | Redundancy | Durability | Use Case | Cost |
|-----|------------|------------|----------|------|
| **Standard_LRS** | Locally redundant | 11 nines | Dev/Test | $ |
| **Standard_ZRS** | Zone redundant | 12 nines | Production (single region) | $$ |
| **Standard_GRS** | Geo-redundant | 16 nines | Production (multi-region) | $$$ |
| **Standard_RA-GRS** | Geo-redundant + read access | 16 nines | Mission-critical | $$$$ |
| **Standard_GZRS** | Geo + zone redundant | 16 nines | Maximum availability | $$$$$ |

### 2. Create Blob Containers

```bash
# Get storage account key
STORAGE_KEY=$(az storage account keys list \
  --resource-group $RESOURCE_GROUP \
  --account-name $STORAGE_ACCOUNT_NAME \
  --query "[0].value" -o tsv)

# Create containers for different backup types
az storage container create \
  --name full-backups \
  --account-name $STORAGE_ACCOUNT_NAME \
  --account-key $STORAGE_KEY \
  --public-access off

az storage container create \
  --name incremental-backups \
  --account-name $STORAGE_ACCOUNT_NAME \
  --account-key $STORAGE_KEY \
  --public-access off

az storage container create \
  --name wal-backups \
  --account-name $STORAGE_ACCOUNT_NAME \
  --account-key $STORAGE_KEY \
  --public-access off

az storage container create \
  --name archives \
  --account-name $STORAGE_ACCOUNT_NAME \
  --account-key $STORAGE_KEY \
  --public-access off
```

### 3. Configure Encryption (Optional: Customer-Managed Keys)

```bash
# Create Key Vault
az keyvault create \
  --name laura-db-kv \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --enable-purge-protection true

# Create encryption key
az keyvault key create \
  --vault-name laura-db-kv \
  --name storage-encryption-key \
  --protection software \
  --kty RSA \
  --size 2048

# Get key vault identity
KEYVAULT_ID=$(az keyvault show \
  --name laura-db-kv \
  --resource-group $RESOURCE_GROUP \
  --query id -o tsv)

# Get storage account identity
az storage account update \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --assign-identity

STORAGE_IDENTITY=$(az storage account show \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --query identity.principalId -o tsv)

# Grant storage account access to key vault
az keyvault set-policy \
  --name laura-db-kv \
  --object-id $STORAGE_IDENTITY \
  --key-permissions get unwrapKey wrapKey

# Configure storage account to use customer-managed key
az storage account update \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --encryption-key-source Microsoft.Keyvault \
  --encryption-key-vault $KEYVAULT_ID \
  --encryption-key-name storage-encryption-key
```

## Authentication Methods

### Option 1: Storage Account Key (Simple)

```bash
# Get account key
STORAGE_KEY=$(az storage account keys list \
  --resource-group $RESOURCE_GROUP \
  --account-name $STORAGE_ACCOUNT_NAME \
  --query "[0].value" -o tsv)

# Export for scripts
export AZURE_STORAGE_ACCOUNT=$STORAGE_ACCOUNT_NAME
export AZURE_STORAGE_KEY=$STORAGE_KEY

# Test connection
az storage blob list \
  --container-name full-backups \
  --account-name $STORAGE_ACCOUNT_NAME \
  --account-key $STORAGE_KEY
```

### Option 2: Shared Access Signature (SAS) - Recommended

More secure with limited permissions and expiration.

```bash
# Generate SAS token (valid for 1 year, write and list only)
SAS_TOKEN=$(az storage container generate-sas \
  --account-name $STORAGE_ACCOUNT_NAME \
  --account-key $STORAGE_KEY \
  --name full-backups \
  --permissions rwl \
  --expiry $(date -u -d "1 year" '+%Y-%m-%dT%H:%MZ') \
  -o tsv)

# Use with azcopy
azcopy copy "/path/to/backup.tar.gz" \
  "https://${STORAGE_ACCOUNT_NAME}.blob.core.windows.net/full-backups?${SAS_TOKEN}"
```

### Option 3: Managed Identity (Best for Azure VMs/AKS)

No credentials stored in code or configuration files.

```bash
# Grant VM or AKS managed identity access to storage
# Get managed identity principal ID (for VM)
VM_IDENTITY=$(az vm show \
  --resource-group $RESOURCE_GROUP \
  --name laura-db-vm \
  --query identity.principalId -o tsv)

# Or for AKS workload identity
IDENTITY_PRINCIPAL_ID=$(az identity show \
  --name laura-db-identity \
  --resource-group $RESOURCE_GROUP \
  --query principalId -o tsv)

# Assign Storage Blob Data Contributor role
az role assignment create \
  --assignee $VM_IDENTITY \
  --role "Storage Blob Data Contributor" \
  --scope /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT_NAME

# Login with managed identity in script
az login --identity

# No credentials needed for az storage commands
az storage blob upload \
  --account-name $STORAGE_ACCOUNT_NAME \
  --container-name full-backups \
  --name backup.tar.gz \
  --file backup.tar.gz \
  --auth-mode login
```

## Backup Strategies

### Strategy 1: Full Backups Only

**Simple, suitable for smaller databases (<100GB)**

- Frequency: Daily
- Retention: 7-30 days
- Recovery Time: Fast (single file restore)
- Storage Cost: High (full copy each time)

### Strategy 2: Full + Incremental Backups

**Balanced approach for medium databases (100GB-1TB)**

- Full backup: Weekly
- Incremental: Daily
- Retention: Full (4 weeks), Incremental (7 days)
- Recovery Time: Moderate (restore full + incrementals)
- Storage Cost: Medium

### Strategy 3: Full + Differential Backups

**Best for large databases (>1TB)**

- Full backup: Weekly
- Differential: Daily (only changes since last full)
- Retention: Full (4 weeks), Differential (7 days)
- Recovery Time: Fast (restore full + latest differential)
- Storage Cost: Medium-High

### Strategy 4: Continuous WAL Archival + Periodic Full

**Point-in-time recovery (PITR)**

- Full backup: Weekly
- WAL archival: Continuous (every 5 minutes)
- Retention: Full (4 weeks), WAL (2 weeks)
- Recovery Time: Precise (restore to any point in time)
- Storage Cost: High (many small files)

## Backup Scripts

### Full Backup Script

```bash
#!/bin/bash
# full-backup.sh - Create full database backup and upload to Azure Blob Storage

set -e
set -o pipefail

# Configuration
DATA_DIR="/var/lib/laura-db"
BACKUP_DIR="/tmp/laura-db-backups"
STORAGE_ACCOUNT="lauradbbkp12345"
CONTAINER="full-backups"
BACKUP_NAME="laura-db-full-$(date +%Y%m%d-%H%M%S)"
RETENTION_DAYS=30

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
NC='\033[0m' # No Color

log() {
    echo -e "${GREEN}[$(date +'%Y-%m-%d %H:%M:%S')]${NC} $1"
}

error() {
    echo -e "${RED}[ERROR]${NC} $1" >&2
}

# Create backup directory
mkdir -p $BACKUP_DIR

log "Starting full backup: $BACKUP_NAME"

# Stop LauraDB for consistent backup (optional, depends on your setup)
# systemctl stop laura-db

# Create tarball (exclude temp files and locks)
log "Creating tarball..."
tar czf $BACKUP_DIR/$BACKUP_NAME.tar.gz \
  -C $DATA_DIR \
  --exclude='*.tmp' \
  --exclude='*.lock' \
  --exclude='*.pid' \
  .

# Generate checksum
log "Generating checksum..."
sha256sum $BACKUP_DIR/$BACKUP_NAME.tar.gz > $BACKUP_DIR/$BACKUP_NAME.tar.gz.sha256

# Get file size
BACKUP_SIZE=$(du -h $BACKUP_DIR/$BACKUP_NAME.tar.gz | cut -f1)
log "Backup size: $BACKUP_SIZE"

# Restart LauraDB
# systemctl start laura-db

# Upload to Azure Blob Storage using azcopy (fastest)
log "Uploading to Azure Blob Storage..."

# Authenticate with managed identity
azcopy login --identity

# Upload backup
azcopy copy \
  "$BACKUP_DIR/$BACKUP_NAME.tar.gz" \
  "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}/$BACKUP_NAME.tar.gz" \
  --overwrite=false \
  --check-md5=FailIfDifferent \
  --put-md5

# Upload checksum
azcopy copy \
  "$BACKUP_DIR/$BACKUP_NAME.tar.gz.sha256" \
  "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}/$BACKUP_NAME.tar.gz.sha256" \
  --overwrite=false

# Add metadata to blob
az storage blob metadata update \
  --account-name $STORAGE_ACCOUNT \
  --container-name $CONTAINER \
  --name $BACKUP_NAME.tar.gz \
  --metadata backup_type=full timestamp=$(date +%s) hostname=$(hostname) \
  --auth-mode login

log "Upload completed successfully"

# Clean up local backup
rm -f $BACKUP_DIR/$BACKUP_NAME.tar.gz $BACKUP_DIR/$BACKUP_NAME.tar.gz.sha256
log "Local backup files removed"

# Delete old backups (older than retention period)
log "Cleaning up old backups (retention: $RETENTION_DAYS days)..."

CUTOFF_DATE=$(date -u -d "$RETENTION_DAYS days ago" '+%Y-%m-%dT%H:%MZ')

az storage blob list \
  --account-name $STORAGE_ACCOUNT \
  --container-name $CONTAINER \
  --auth-mode login \
  --query "[?properties.creationTime<'$CUTOFF_DATE'].name" -o tsv | \
while read blob_name; do
  log "Deleting old backup: $blob_name"
  az storage blob delete \
    --account-name $STORAGE_ACCOUNT \
    --container-name $CONTAINER \
    --name "$blob_name" \
    --auth-mode login
done

log "Backup completed: $BACKUP_NAME"
log "View backup: az storage blob list --account-name $STORAGE_ACCOUNT --container-name $CONTAINER --auth-mode login"

# Send notification (optional)
# curl -X POST https://hooks.slack.com/services/YOUR/WEBHOOK/URL \
#   -H 'Content-Type: application/json' \
#   -d "{\"text\":\"LauraDB full backup completed: $BACKUP_NAME (Size: $BACKUP_SIZE)\"}"

exit 0
```

### Incremental Backup Script

```bash
#!/bin/bash
# incremental-backup.sh - Create incremental backup (changes since last full backup)

set -e
set -o pipefail

# Configuration
DATA_DIR="/var/lib/laura-db"
BACKUP_DIR="/tmp/laura-db-backups"
STORAGE_ACCOUNT="lauradbbkp12345"
CONTAINER="incremental-backups"
SNAPSHOT_FILE="/var/lib/laura-db/.backup-snapshot"
BACKUP_NAME="laura-db-incr-$(date +%Y%m%d-%H%M%S)"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

mkdir -p $BACKUP_DIR

log "Starting incremental backup: $BACKUP_NAME"

# Create incremental backup (files modified since last snapshot)
if [ -f "$SNAPSHOT_FILE" ]; then
    log "Creating incremental tarball (changes since last backup)..."
    tar czf $BACKUP_DIR/$BACKUP_NAME.tar.gz \
      -C $DATA_DIR \
      --listed-incremental=$SNAPSHOT_FILE \
      --exclude='*.tmp' \
      --exclude='*.lock' \
      .
else
    log "No snapshot file found, creating full backup as baseline..."
    tar czf $BACKUP_DIR/$BACKUP_NAME.tar.gz \
      -C $DATA_DIR \
      --listed-incremental=$SNAPSHOT_FILE \
      --exclude='*.tmp' \
      --exclude='*.lock' \
      .
fi

# Generate checksum
sha256sum $BACKUP_DIR/$BACKUP_NAME.tar.gz > $BACKUP_DIR/$BACKUP_NAME.tar.gz.sha256

BACKUP_SIZE=$(du -h $BACKUP_DIR/$BACKUP_NAME.tar.gz | cut -f1)
log "Incremental backup size: $BACKUP_SIZE"

# Upload to blob storage
log "Uploading to Azure Blob Storage..."

azcopy login --identity

azcopy copy \
  "$BACKUP_DIR/$BACKUP_NAME.tar.gz" \
  "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}/$BACKUP_NAME.tar.gz" \
  --overwrite=false

azcopy copy \
  "$BACKUP_DIR/$BACKUP_NAME.tar.gz.sha256" \
  "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}/$BACKUP_NAME.tar.gz.sha256" \
  --overwrite=false

# Add metadata
az storage blob metadata update \
  --account-name $STORAGE_ACCOUNT \
  --container-name $CONTAINER \
  --name $BACKUP_NAME.tar.gz \
  --metadata backup_type=incremental timestamp=$(date +%s) hostname=$(hostname) \
  --auth-mode login

# Clean up
rm -f $BACKUP_DIR/$BACKUP_NAME.tar.gz $BACKUP_DIR/$BACKUP_NAME.tar.gz.sha256

log "Incremental backup completed: $BACKUP_NAME (Size: $BACKUP_SIZE)"

exit 0
```

### WAL Continuous Archival Script

```bash
#!/bin/bash
# wal-archive.sh - Continuously archive WAL files to Azure Blob Storage

set -e

# Configuration
WAL_DIR="/var/lib/laura-db/wal"
STORAGE_ACCOUNT="lauradbbkp12345"
CONTAINER="wal-backups"
ARCHIVE_MARKER="/var/lib/laura-db/.wal-archive-marker"

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

# Find WAL files modified since last run
if [ -f "$ARCHIVE_MARKER" ]; then
    FIND_NEWER="-newer $ARCHIVE_MARKER"
else
    FIND_NEWER=""
fi

# Update marker
touch $ARCHIVE_MARKER

# Archive new WAL files
find $WAL_DIR -type f -name "*.wal" $FIND_NEWER | while read wal_file; do
    wal_basename=$(basename $wal_file)

    log "Archiving WAL: $wal_basename"

    azcopy copy \
      "$wal_file" \
      "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}/$wal_basename" \
      --overwrite=false \
      --check-md5=FailIfDifferent \
      --put-md5

    log "Archived: $wal_basename"
done

log "WAL archival completed"

exit 0
```

## Restore Procedures

### Restore from Full Backup

```bash
#!/bin/bash
# restore-full.sh - Restore database from full backup

set -e

# Configuration
BACKUP_NAME="$1"  # e.g., laura-db-full-20250115-020000
STORAGE_ACCOUNT="lauradbbkp12345"
CONTAINER="full-backups"
RESTORE_DIR="/tmp/laura-db-restore"
DATA_DIR="/var/lib/laura-db"

if [ -z "$BACKUP_NAME" ]; then
    echo "Usage: $0 <backup-name>"
    echo "Example: $0 laura-db-full-20250115-020000"
    exit 1
fi

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

log "Starting restore from backup: $BACKUP_NAME"

# Stop LauraDB
log "Stopping LauraDB..."
systemctl stop laura-db || true

# Create restore directory
mkdir -p $RESTORE_DIR

# Download backup
log "Downloading backup from Azure Blob Storage..."
azcopy login --identity

azcopy copy \
  "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}/$BACKUP_NAME.tar.gz" \
  "$RESTORE_DIR/" \
  --check-md5=FailIfDifferent

# Download checksum
azcopy copy \
  "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${CONTAINER}/$BACKUP_NAME.tar.gz.sha256" \
  "$RESTORE_DIR/"

# Verify checksum
log "Verifying backup integrity..."
cd $RESTORE_DIR
sha256sum -c $BACKUP_NAME.tar.gz.sha256 || {
    log "ERROR: Checksum verification failed!"
    exit 1
}

log "Checksum verified successfully"

# Backup existing data (just in case)
if [ -d "$DATA_DIR" ]; then
    log "Backing up existing data to ${DATA_DIR}.bak..."
    mv $DATA_DIR ${DATA_DIR}.bak.$(date +%Y%m%d-%H%M%S)
fi

# Create data directory
mkdir -p $DATA_DIR

# Extract backup
log "Extracting backup..."
tar xzf $RESTORE_DIR/$BACKUP_NAME.tar.gz -C $DATA_DIR

# Set correct permissions
chown -R laura-db:laura-db $DATA_DIR
chmod 700 $DATA_DIR

# Clean up
rm -rf $RESTORE_DIR

# Start LauraDB
log "Starting LauraDB..."
systemctl start laura-db

# Verify
sleep 5
systemctl status laura-db

log "Restore completed successfully from backup: $BACKUP_NAME"
log "LauraDB is now running with restored data"

exit 0
```

### Point-in-Time Recovery (PITR)

```bash
#!/bin/bash
# restore-pitr.sh - Restore to specific point in time using full backup + WAL replay

set -e

# Configuration
FULL_BACKUP="$1"       # e.g., laura-db-full-20250115-020000
TARGET_TIME="$2"       # e.g., "2025-01-16 14:30:00"
STORAGE_ACCOUNT="lauradbbkp12345"
FULL_CONTAINER="full-backups"
WAL_CONTAINER="wal-backups"
RESTORE_DIR="/tmp/laura-db-restore"
DATA_DIR="/var/lib/laura-db"

if [ -z "$FULL_BACKUP" ] || [ -z "$TARGET_TIME" ]; then
    echo "Usage: $0 <full-backup-name> <target-time>"
    echo "Example: $0 laura-db-full-20250115-020000 '2025-01-16 14:30:00'"
    exit 1
fi

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

log "Starting point-in-time recovery"
log "Full backup: $FULL_BACKUP"
log "Target time: $TARGET_TIME"

# Stop LauraDB
systemctl stop laura-db || true

mkdir -p $RESTORE_DIR

# Download and restore full backup (same as restore-full.sh)
log "Downloading full backup..."
azcopy login --identity
azcopy copy \
  "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${FULL_CONTAINER}/$FULL_BACKUP.tar.gz" \
  "$RESTORE_DIR/"

# Extract
log "Extracting full backup..."
if [ -d "$DATA_DIR" ]; then
    mv $DATA_DIR ${DATA_DIR}.bak.$(date +%Y%m%d-%H%M%S)
fi
mkdir -p $DATA_DIR
tar xzf $RESTORE_DIR/$FULL_BACKUP.tar.gz -C $DATA_DIR

# Download WAL files needed for replay
log "Downloading WAL files..."
WAL_DIR="$DATA_DIR/wal"
mkdir -p $WAL_DIR

TARGET_TIMESTAMP=$(date -d "$TARGET_TIME" +%s)

# List and download WAL files created after full backup and before target time
az storage blob list \
  --account-name $STORAGE_ACCOUNT \
  --container-name $WAL_CONTAINER \
  --auth-mode login \
  --query "[].{name:name,time:properties.creationTime}" -o tsv | \
while read name time; do
    wal_timestamp=$(date -d "$time" +%s)
    if [ $wal_timestamp -le $TARGET_TIMESTAMP ]; then
        log "Downloading WAL: $name"
        azcopy copy \
          "https://${STORAGE_ACCOUNT}.blob.core.windows.net/${WAL_CONTAINER}/$name" \
          "$WAL_DIR/"
    fi
done

# Create recovery configuration
log "Creating recovery configuration..."
cat > $DATA_DIR/recovery.conf <<EOF
recovery_target_time = '$TARGET_TIME'
recovery_target_action = promote
EOF

# Set permissions
chown -R laura-db:laura-db $DATA_DIR
chmod 700 $DATA_DIR

# Start LauraDB (will automatically enter recovery mode)
log "Starting LauraDB in recovery mode..."
systemctl start laura-db

log "Point-in-time recovery initiated"
log "LauraDB is replaying WAL files to target time: $TARGET_TIME"
log "Monitor progress: journalctl -u laura-db -f"

exit 0
```

## Automation

### Systemd Timer for Daily Backups

```ini
# /etc/systemd/system/laura-db-backup.service
[Unit]
Description=LauraDB Full Backup
After=network-online.target

[Service]
Type=oneshot
User=root
ExecStart=/usr/local/bin/full-backup.sh
StandardOutput=journal
StandardError=journal

[Install]
WantedBy=multi-user.target
```

```ini
# /etc/systemd/system/laura-db-backup.timer
[Unit]
Description=LauraDB Daily Backup Timer
Requires=laura-db-backup.service

[Timer]
OnCalendar=daily
OnCalendar=02:00
Persistent=true

[Install]
WantedBy=timers.target
```

```bash
# Enable and start timer
sudo systemctl daemon-reload
sudo systemctl enable laura-db-backup.timer
sudo systemctl start laura-db-backup.timer

# Check timer status
systemctl status laura-db-backup.timer
systemctl list-timers --all

# View logs
journalctl -u laura-db-backup.service
```

### Cron Job Alternative

```bash
# Add to root crontab
sudo crontab -e

# Full backup daily at 2 AM
0 2 * * * /usr/local/bin/full-backup.sh >> /var/log/laura-db-backup.log 2>&1

# Incremental backup every 6 hours
0 */6 * * * /usr/local/bin/incremental-backup.sh >> /var/log/laura-db-backup.log 2>&1

# WAL archival every 5 minutes
*/5 * * * * /usr/local/bin/wal-archive.sh >> /var/log/laura-db-wal-archive.log 2>&1
```

### Azure Automation Runbook

For centralized backup management across multiple VMs.

```powershell
# backup-runbook.ps1
param(
    [Parameter(Mandatory=$true)]
    [string]$ResourceGroupName,

    [Parameter(Mandatory=$true)]
    [string]$VMName
)

# Execute backup script on VM
$script = Get-Content '/usr/local/bin/full-backup.sh' -Raw

Invoke-AzVMRunCommand `
    -ResourceGroupName $ResourceGroupName `
    -VMName $VMName `
    -CommandId 'RunShellScript' `
    -ScriptString $script

Write-Output "Backup completed for $VMName"
```

## Lifecycle Management

### Configure Lifecycle Management Policy

Automatically transition backups to cooler storage tiers and delete old backups.

```bash
# Create lifecycle policy JSON
cat > lifecycle-policy.json <<'EOF'
{
  "rules": [
    {
      "enabled": true,
      "name": "MoveToCoolAfter30Days",
      "type": "Lifecycle",
      "definition": {
        "actions": {
          "baseBlob": {
            "tierToCool": {
              "daysAfterModificationGreaterThan": 30
            },
            "tierToArchive": {
              "daysAfterModificationGreaterThan": 90
            },
            "delete": {
              "daysAfterModificationGreaterThan": 365
            }
          },
          "snapshot": {
            "delete": {
              "daysAfterCreationGreaterThan": 90
            }
          }
        },
        "filters": {
          "blobTypes": [
            "blockBlob"
          ],
          "prefixMatch": [
            "full-backups/",
            "incremental-backups/"
          ]
        }
      }
    },
    {
      "enabled": true,
      "name": "DeleteOldWALFiles",
      "type": "Lifecycle",
      "definition": {
        "actions": {
          "baseBlob": {
            "delete": {
              "daysAfterModificationGreaterThan": 14
            }
          }
        },
        "filters": {
          "blobTypes": [
            "blockBlob"
          ],
          "prefixMatch": [
            "wal-backups/"
          ]
        }
      }
    }
  ]
}
EOF

# Apply lifecycle policy
az storage account management-policy create \
  --account-name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --policy @lifecycle-policy.json

# View current policy
az storage account management-policy show \
  --account-name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP
```

### Cost Savings with Lifecycle Management

| Tier | Days After Creation | Monthly Cost (per GB) | Savings |
|------|---------------------|----------------------|---------|
| **Hot** | 0-30 | $0.0184 | Baseline |
| **Cool** | 31-90 | $0.0100 | 46% |
| **Archive** | 91+ | $0.0020 | 89% |

*Example*: 1TB of backups over 1 year
- Hot only: $220/month
- With lifecycle: $80/month (first 30 days hot, next 60 cool, rest archive)
- **Savings**: ~$140/month

## Monitoring & Alerts

### Create Alert Rules

```bash
# Create action group for notifications
az monitor action-group create \
  --name laura-db-backup-alerts \
  --resource-group $RESOURCE_GROUP \
  --short-name backup-alert \
  --email-receiver name=admin email=admin@example.com \
  --sms-receiver name=oncall country-code=1 phone-number=5551234567

# Alert on backup failures (no backup in last 25 hours)
az monitor metrics alert create \
  --name "LauraDB Backup Missing" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT_NAME \
  --condition "total BlobCount < 1" \
  --window-size 24h \
  --evaluation-frequency 1h \
  --description "No new backup created in last 24 hours" \
  --severity 2 \
  --action laura-db-backup-alerts

# Alert on high blob storage usage
az monitor metrics alert create \
  --name "LauraDB Backup Storage High" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/$STORAGE_ACCOUNT_NAME \
  --condition "total UsedCapacity > 1000000000000" \
  --window-size 1h \
  --evaluation-frequency 1h \
  --description "Backup storage exceeds 1TB" \
  --severity 3 \
  --action laura-db-backup-alerts
```

### Monitor Backup Job Success

```bash
# Create log query alert for backup failures
az monitor scheduled-query create \
  --name "LauraDB Backup Failed" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/laura-db-vm \
  --condition "count > 0" \
  --condition-query "Syslog | where Facility == 'user' and SyslogMessage contains 'ERROR' and SyslogMessage contains 'backup' | summarize count()" \
  --description "Backup script reported errors" \
  --evaluation-frequency 15m \
  --window-size 15m \
  --severity 2 \
  --action-groups /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/microsoft.insights/actionGroups/laura-db-backup-alerts
```

### Create Backup Dashboard

```bash
# Create Application Insights dashboard (via Azure Portal)
# Add charts for:
# - Blob count over time
# - Storage usage over time
# - Backup job duration
# - Backup job success rate
# - Cost trends
```

## Cost Optimization

### 1. Use Appropriate Storage Tier

```bash
# For long-term archives, move to Archive tier
az storage blob set-tier \
  --account-name $STORAGE_ACCOUNT_NAME \
  --container-name archives \
  --name backup-archive-20240101.tar.gz \
  --tier Archive \
  --auth-mode login
```

### 2. Enable Compression

Already done in backup scripts (tar czf uses gzip compression)

### 3. Deduplicate Data (Application-Level)

For databases with high redundancy, consider block-level deduplication tools:
- restic (built-in deduplication)
- borg (deduplicated backups)

### 4. Use Read-Access Geo-Redundant Storage Strategically

```bash
# Use RA-GRS only for critical backups
# Use LRS or ZRS for dev/test backups
az storage account update \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --sku Standard_LRS  # Downgrade for cost savings
```

### 5. Monitor Costs

```bash
# View storage account costs
az consumption usage list \
  --start-date 2025-01-01 \
  --end-date 2025-01-31 \
  --query "[?contains(instanceName, '$STORAGE_ACCOUNT_NAME')]" \
  -o table

# Set budget alert
az consumption budget create \
  --amount 100 \
  --budget-name laura-db-backup-budget \
  --category cost \
  --time-grain monthly \
  --start-date 2025-01-01 \
  --end-date 2026-01-01 \
  --resource-group $RESOURCE_GROUP \
  --notifications "actual_GreaterThan_80_Percent={enabled:true,operator:GreaterThan,threshold:80,contact-emails:['admin@example.com']}"
```

## Disaster Recovery

### Multi-Region Replication

Enable geo-redundant storage for automatic replication.

```bash
# Already configured with GRS SKU
# Check replication status
az storage account show \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --query "{replication:sku.name,primary:primaryLocation,secondary:secondaryLocation}"

# For read access to secondary region, upgrade to RA-GRS
az storage account update \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --sku Standard_RA-GRS

# Access secondary endpoint
# https://lauradbbkp12345-secondary.blob.core.windows.net
```

### Cross-Region Copy

Manually replicate backups to different region for extra protection.

```bash
# Create storage account in secondary region
SECONDARY_REGION="westus"
SECONDARY_STORAGE="lauradbbkpwest$(date +%s)"

az storage account create \
  --name $SECONDARY_STORAGE \
  --resource-group $RESOURCE_GROUP \
  --location $SECONDARY_REGION \
  --sku Standard_LRS

# Copy backups to secondary region
azcopy copy \
  "https://${STORAGE_ACCOUNT_NAME}.blob.core.windows.net/full-backups/*" \
  "https://${SECONDARY_STORAGE}.blob.core.windows.net/full-backups/" \
  --recursive
```

### Disaster Recovery Testing

```bash
#!/bin/bash
# dr-test.sh - Test disaster recovery procedures

set -e

log() {
    echo "[DR-TEST] [$(date +'%Y-%m-%d %H:%M:%S')] $1"
}

log "Starting disaster recovery test"

# 1. List available backups
log "Available backups:"
az storage blob list \
  --account-name $STORAGE_ACCOUNT_NAME \
  --container-name full-backups \
  --auth-mode login \
  --query "[].{Name:name,Created:properties.creationTime,Size:properties.contentLength}" \
  -o table

# 2. Select latest backup
LATEST_BACKUP=$(az storage blob list \
  --account-name $STORAGE_ACCOUNT_NAME \
  --container-name full-backups \
  --auth-mode login \
  --query "sort_by([].{name:name,time:properties.creationTime}, &time)[-1].name" \
  -o tsv)

log "Latest backup: $LATEST_BACKUP"

# 3. Test restore to temporary location
TEST_RESTORE_DIR="/tmp/dr-test-restore-$(date +%s)"
mkdir -p $TEST_RESTORE_DIR

log "Downloading backup to test location..."
azcopy copy \
  "https://${STORAGE_ACCOUNT_NAME}.blob.core.windows.net/full-backups/$LATEST_BACKUP" \
  "$TEST_RESTORE_DIR/"

log "Extracting backup..."
tar xzf $TEST_RESTORE_DIR/$LATEST_BACKUP -C $TEST_RESTORE_DIR

# 4. Verify backup integrity
log "Verifying backup integrity..."
if [ -f "$TEST_RESTORE_DIR/data/collections.dat" ]; then
    log "✓ Collections file found"
else
    log "✗ Collections file missing - BACKUP MAY BE CORRUPTED"
    exit 1
fi

# 5. Clean up
rm -rf $TEST_RESTORE_DIR

log "Disaster recovery test completed successfully"
log "Latest backup is valid and restorable"

exit 0
```

## Best Practices

### 1. Follow 3-2-1 Backup Rule

- **3** copies of data (original + 2 backups)
- **2** different storage types (local + blob storage)
- **1** copy offsite (geo-redundant or cross-region)

### 2. Test Restores Regularly

```bash
# Schedule monthly restore test
0 3 1 * * /usr/local/bin/dr-test.sh >> /var/log/dr-test.log 2>&1
```

### 3. Encrypt Backups

```bash
# Encrypt before upload (additional security)
tar czf - /var/lib/laura-db | openssl enc -aes-256-cbc -salt -out backup.tar.gz.enc -k "your-encryption-password"

# Upload encrypted backup
azcopy copy backup.tar.gz.enc "https://..."

# Decrypt during restore
openssl enc -d -aes-256-cbc -in backup.tar.gz.enc -out backup.tar.gz -k "your-encryption-password"
```

### 4. Document Recovery Procedures

Create runbook with step-by-step instructions:
1. Latest backup location
2. Restore commands
3. Verification steps
4. Contact information

### 5. Monitor Backup Health

- Check backup completion daily
- Verify backup size trends
- Test restore monthly
- Review storage costs monthly

### 6. Secure Access

```bash
# Use managed identity when possible
# Rotate SAS tokens regularly
# Enable firewall rules on storage account
az storage account update \
  --name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --default-action Deny

# Allow specific VNet
az storage account network-rule add \
  --account-name $STORAGE_ACCOUNT_NAME \
  --resource-group $RESOURCE_GROUP \
  --vnet-name laura-db-vnet \
  --subnet aks-subnet
```

## Summary

Azure Blob Storage provides a robust, scalable, and cost-effective solution for LauraDB backups with:

- **Durability**: 99.999999999% (11 nines)
- **Multiple redundancy options**: LRS, ZRS, GRS, RA-GRS
- **Lifecycle management**: Automatic tier transitions
- **Security**: Encryption, soft delete, versioning, immutability
- **Cost optimization**: Pay only for what you use, with intelligent tiering

### Monthly Cost Estimate (1TB backups)

- **Hot tier (30 days)**: 30GB × $0.0184 = $0.55
- **Cool tier (60 days)**: 60GB × $0.0100 = $0.60
- **Archive tier (rest)**: 910GB × $0.0020 = $1.82
- **Operations**: ~$5
- **Replication (GRS)**: +50% = ~$12
- **Total**: ~$20/month for 1TB with intelligent lifecycle management

Compare with backup appliance: ~$10,000 upfront + $200/month maintenance

## Next Steps

- Implement automated backup scripts
- Test restore procedures
- Set up monitoring and alerts
- Configure lifecycle policies
- Document disaster recovery runbooks
- Schedule monthly DR tests

## References

- [Azure Blob Storage Documentation](https://docs.microsoft.com/en-us/azure/storage/blobs/)
- [AzCopy Documentation](https://docs.microsoft.com/en-us/azure/storage/common/storage-use-azcopy-v10)
- [Azure Storage Lifecycle Management](https://docs.microsoft.com/en-us/azure/storage/blobs/storage-lifecycle-management-concepts)
- [LauraDB Main Documentation](../../README.md)

---

**Remember**: The best backup is the one you've tested restoring from. Test your backups regularly!
