#!/bin/bash
# LauraDB installation and setup script
# This script is used as cloud-init/user-data across AWS, GCP, and Azure

set -e
set -o pipefail

# Configuration variables (will be templated by Terraform)
PROJECT_NAME="${project_name}"
ENVIRONMENT="${environment}"
LAURA_DB_VERSION="${laura_db_version}"
LAURA_DB_PORT="${laura_db_port}"
DATA_DIR="${data_dir}"
LOG_LEVEL="${log_level}"

# Logging setup
LOG_FILE="/var/log/laura-db-setup.log"
exec 1> >(tee -a "$LOG_FILE")
exec 2>&1

log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $*"
}

log "Starting LauraDB setup"
log "Project: $PROJECT_NAME"
log "Environment: $ENVIRONMENT"
log "Version: $LAURA_DB_VERSION"

# Update system
log "Updating system packages..."
export DEBIAN_FRONTEND=noninteractive
apt-get update -qq
apt-get upgrade -y -qq

# Install required packages
log "Installing required packages..."
apt-get install -y -qq \
    curl \
    wget \
    git \
    build-essential \
    ca-certificates \
    apt-transport-https \
    software-properties-common \
    jq

# Install Go (required for LauraDB)
log "Installing Go..."
GO_VERSION="1.21.5"
wget -q https://go.dev/dl/go$GO_VERSION.linux-amd64.tar.gz
rm -rf /usr/local/go
tar -C /usr/local -xzf go$GO_VERSION.linux-amd64.tar.gz
rm go$GO_VERSION.linux-amd64.tar.gz

# Setup Go environment
export PATH=$PATH:/usr/local/go/bin
echo 'export PATH=$PATH:/usr/local/go/bin' >> /etc/profile
echo 'export GOPATH=/opt/go' >> /etc/profile
export GOPATH=/opt/go
mkdir -p $GOPATH

# Create laura-db user
log "Creating laura-db user..."
if ! id -u laura-db > /dev/null 2>&1; then
    useradd -r -s /bin/bash -d /opt/laura-db -m laura-db
fi

# Create data directory
log "Creating data directory: $DATA_DIR"
mkdir -p $DATA_DIR
chown -R laura-db:laura-db $DATA_DIR
chmod 700 $DATA_DIR

# Clone and build LauraDB
log "Cloning LauraDB repository..."
LAURA_DB_HOME="/opt/laura-db"
if [ ! -d "$LAURA_DB_HOME/laura-db" ]; then
    cd $LAURA_DB_HOME
    sudo -u laura-db git clone https://github.com/mnohosten/laura-db.git
fi

cd $LAURA_DB_HOME/laura-db

# Checkout specific version if not latest
if [ "$LAURA_DB_VERSION" != "latest" ]; then
    log "Checking out version: $LAURA_DB_VERSION"
    sudo -u laura-db git checkout $LAURA_DB_VERSION
else
    log "Using latest version"
    sudo -u laura-db git pull origin main
fi

# Build LauraDB
log "Building LauraDB..."
sudo -u laura-db make build

# Install binary
log "Installing LauraDB binary..."
cp bin/laura-server /usr/local/bin/laura-server
chmod +x /usr/local/bin/laura-server

# Create configuration file
log "Creating configuration file..."
cat > /etc/laura-db.conf <<EOF
# LauraDB Configuration
# Managed by Terraform

PROJECT_NAME="$PROJECT_NAME"
ENVIRONMENT="$ENVIRONMENT"
LAURA_DB_PORT=$LAURA_DB_PORT
DATA_DIR="$DATA_DIR"
LOG_LEVEL="$LOG_LEVEL"
EOF

chown laura-db:laura-db /etc/laura-db.conf
chmod 640 /etc/laura-db.conf

# Create systemd service
log "Creating systemd service..."
cat > /etc/systemd/system/laura-db.service <<'EOF'
[Unit]
Description=LauraDB Server
Documentation=https://github.com/mnohosten/laura-db
After=network.target

[Service]
Type=simple
User=laura-db
Group=laura-db
EnvironmentFile=/etc/laura-db.conf
ExecStart=/usr/local/bin/laura-server -port $LAURA_DB_PORT -data-dir $DATA_DIR
Restart=always
RestartSec=10
StandardOutput=journal
StandardError=journal
SyslogIdentifier=laura-db

# Security settings
NoNewPrivileges=true
PrivateTmp=true
ProtectSystem=strict
ProtectHome=true
ReadWritePaths=$DATA_DIR

# Resource limits
LimitNOFILE=65536
LimitNPROC=4096

[Install]
WantedBy=multi-user.target
EOF

# Create log rotation
log "Setting up log rotation..."
cat > /etc/logrotate.d/laura-db <<'EOF'
/var/log/laura-db/*.log {
    daily
    rotate 14
    compress
    delaycompress
    notifempty
    create 0640 laura-db laura-db
    sharedscripts
    postrotate
        systemctl reload laura-db > /dev/null 2>&1 || true
    endscript
}
EOF

# Create log directory
mkdir -p /var/log/laura-db
chown laura-db:laura-db /var/log/laura-db
chmod 750 /var/log/laura-db

# Reload systemd and enable service
log "Enabling LauraDB service..."
systemctl daemon-reload
systemctl enable laura-db.service

# Start service
log "Starting LauraDB service..."
systemctl start laura-db.service

# Wait for service to start
sleep 5

# Verify service is running
if systemctl is-active --quiet laura-db.service; then
    log "LauraDB service started successfully"
    systemctl status laura-db.service --no-pager
else
    log "ERROR: LauraDB service failed to start"
    journalctl -u laura-db.service --no-pager -n 50
    exit 1
fi

# Test endpoint
log "Testing LauraDB endpoint..."
max_attempts=30
attempt=0

while [ $attempt -lt $max_attempts ]; do
    if curl -s http://localhost:$LAURA_DB_PORT/_health > /dev/null; then
        log "LauraDB is responding to health checks"
        curl -s http://localhost:$LAURA_DB_PORT/_health | jq .
        break
    fi

    attempt=$((attempt + 1))
    log "Waiting for LauraDB to respond... (attempt $attempt/$max_attempts)"
    sleep 2
done

if [ $attempt -eq $max_attempts ]; then
    log "WARNING: LauraDB health check timeout"
fi

# Create health check script
log "Creating health check script..."
cat > /usr/local/bin/laura-db-health-check <<'EOF'
#!/bin/bash
# LauraDB health check script

PORT=$(grep LAURA_DB_PORT /etc/laura-db.conf | cut -d'=' -f2)
RESPONSE=$(curl -s -w "%{http_code}" http://localhost:$PORT/_health)
HTTP_CODE=$(echo "$RESPONSE" | tail -n1)

if [ "$HTTP_CODE" = "200" ]; then
    echo "LauraDB is healthy"
    exit 0
else
    echo "LauraDB is unhealthy (HTTP $HTTP_CODE)"
    exit 1
fi
EOF

chmod +x /usr/local/bin/laura-db-health-check

# Create backup script placeholder
log "Creating backup script..."
cat > /usr/local/bin/laura-db-backup <<'EOF'
#!/bin/bash
# LauraDB backup script
# This script will be customized by cloud-specific configuration

DATA_DIR=$(grep DATA_DIR /etc/laura-db.conf | cut -d'"' -f2)
BACKUP_NAME="laura-db-backup-$(date +%Y%m%d-%H%M%S)"

echo "Backup script placeholder"
echo "Data directory: $DATA_DIR"
echo "Backup name: $BACKUP_NAME"
echo "Implement cloud-specific backup logic"
EOF

chmod +x /usr/local/bin/laura-db-backup

# Setup monitoring agent (cloud-specific, will be added by cloud modules)
log "Monitoring agent setup (cloud-specific)..."
# This section will be extended by cloud-specific user-data

# Final setup
log "Running final setup..."
chown -R laura-db:laura-db /opt/laura-db
chmod -R 755 /opt/laura-db

# Create success marker
touch /var/lib/cloud/instance/boot-finished
log "LauraDB setup completed successfully"

# Output summary
log "=== Setup Summary ==="
log "LauraDB Version: $(laura-server --version 2>&1 || echo 'unknown')"
log "Service Status: $(systemctl is-active laura-db.service)"
log "Data Directory: $DATA_DIR"
log "Port: $LAURA_DB_PORT"
log "Log Level: $LOG_LEVEL"
log "Configuration: /etc/laura-db.conf"
log "Log File: $LOG_FILE"
log "===================="

exit 0
