# Docker Compose Guide

This guide covers deploying and managing LauraDB using Docker Compose for various scenarios including development, production, and monitoring setups.

## Table of Contents

- [Overview](#overview)
- [Quick Start](#quick-start)
- [Configuration](#configuration)
- [Deployment Modes](#deployment-modes)
- [Monitoring Stack](#monitoring-stack)
- [Production Deployment](#production-deployment)
- [Management Commands](#management-commands)
- [Troubleshooting](#troubleshooting)

## Overview

LauraDB's Docker Compose setup provides:

- **Multi-service orchestration**: LauraDB server with optional monitoring stack
- **Environment-based configuration**: Easy customization via `.env` file
- **Multiple deployment profiles**: Development, production, and monitoring
- **Persistent storage**: Docker volumes for data persistence
- **Health checks**: Automatic health monitoring and restart policies
- **Network isolation**: Dedicated Docker network for service communication

## Quick Start

### Prerequisites

- Docker 20.10+ installed
- Docker Compose 1.29+ installed (or Docker with Compose V2)
- At least 1GB RAM available
- At least 2GB disk space

### Basic Deployment

1. **Clone and navigate to the repository**:
   ```bash
   git clone https://github.com/mnohosten/laura-db.git
   cd laura-db
   ```

2. **Start LauraDB**:
   ```bash
   make compose-up
   # or
   docker-compose up -d
   ```

3. **Verify it's running**:
   ```bash
   curl http://localhost:8080/_health
   ```

4. **Access the admin console**:
   Open http://localhost:8080 in your browser

5. **Stop services**:
   ```bash
   make compose-down
   # or
   docker-compose down
   ```

## Configuration

### Environment Variables

Create a `.env` file in the project root (copy from `.env.example`):

```bash
cp .env.example .env
```

Available variables:

| Variable | Default | Description |
|----------|---------|-------------|
| `LAURA_PORT` | `8080` | Port for LauraDB server |
| `LAURA_BUFFER_SIZE` | `1000` | Buffer pool size (number of pages) |
| `LAURA_CORS_ORIGIN` | `*` | CORS allowed origins |
| `GRAFANA_ADMIN_USER` | `admin` | Grafana admin username |
| `GRAFANA_ADMIN_PASSWORD` | `admin` | Grafana admin password |

### Custom Configuration Example

```env
# .env file
LAURA_PORT=9090
LAURA_BUFFER_SIZE=5000
LAURA_CORS_ORIGIN=https://myapp.com
GRAFANA_ADMIN_PASSWORD=secure-password-here
```

## Deployment Modes

### Development Mode (Default)

Best for local development and testing:

```bash
make compose-up
```

Features:
- Single LauraDB instance
- Restart policy: `unless-stopped`
- No resource limits
- Data persisted in `laura-data` volume

### Production Mode

Optimized for production deployments:

```bash
make compose-up-prod
# or
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up -d
```

Features:
- Restart policy: `always`
- Resource limits (2 CPU, 2GB RAM)
- Resource reservations (1 CPU, 1GB RAM)
- Log rotation (10MB max, 3 files)
- Optimized buffer size (5000 pages)
- Production CORS settings

### Monitoring Mode

Includes Prometheus and Grafana for observability:

```bash
make compose-up-monitoring
# or
docker-compose --profile monitoring up -d
```

Access points:
- **LauraDB**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

## Monitoring Stack

### Prometheus

Prometheus automatically scrapes metrics from LauraDB at `/_metrics` endpoint.

**Configuration**: Edit `prometheus.yml` to customize scrape settings:

```yaml
scrape_configs:
  - job_name: 'laura-db'
    scrape_interval: 15s
    static_configs:
      - targets: ['laura-db:8080']
```

**Access Prometheus UI**: http://localhost:9090

### Grafana

Pre-configured Grafana dashboard for LauraDB metrics.

**Initial Setup**:
1. Access http://localhost:3000
2. Login with admin/admin (change password on first login)
3. Navigate to Dashboards → LauraDB Dashboard

**Dashboard Panels**:
- Query throughput and latency
- Insert/Update/Delete rates
- Cache hit rates
- Index performance
- Memory and resource usage
- Transaction metrics

### Setting Up Alerts

Edit `prometheus.yml` to add alerting rules:

```yaml
rule_files:
  - 'alerts.yml'

alerting:
  alertmanagers:
    - static_configs:
        - targets: ['alertmanager:9093']
```

## Production Deployment

### Recommended Production Setup

1. **Create production environment file**:
   ```bash
   cat > .env.prod <<EOF
   LAURA_PORT=8080
   LAURA_BUFFER_SIZE=10000
   LAURA_CORS_ORIGIN=https://yourdomain.com
   GRAFANA_ADMIN_USER=admin
   GRAFANA_ADMIN_PASSWORD=$(openssl rand -base64 32)
   EOF
   ```

2. **Enable TLS** (edit `docker-compose.prod.yml`):
   ```yaml
   services:
     laura-db:
       command:
         - "-tls"
         - "-tls-cert"
         - "/certs/server.crt"
         - "-tls-key"
         - "/certs/server.key"
       volumes:
         - ./certs:/certs:ro
   ```

3. **Deploy with monitoring**:
   ```bash
   docker-compose \
     -f docker-compose.yml \
     -f docker-compose.prod.yml \
     --profile monitoring \
     up -d
   ```

### Resource Planning

**Minimum Requirements**:
- CPU: 1 core
- RAM: 1GB
- Disk: 10GB

**Recommended Production**:
- CPU: 2-4 cores
- RAM: 4-8GB
- Disk: 50GB SSD

**High-Traffic Production**:
- CPU: 8+ cores
- RAM: 16-32GB
- Disk: 200GB+ NVMe SSD

### Backup Strategy

**Backup data volume**:
```bash
# Create backup
docker run --rm \
  -v laura-data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar czf /backup/laura-db-$(date +%Y%m%d).tar.gz -C /data .

# Restore backup
docker run --rm \
  -v laura-data:/data \
  -v $(pwd)/backups:/backup \
  alpine tar xzf /backup/laura-db-20250124.tar.gz -C /data
```

## Management Commands

### Makefile Commands

| Command | Description |
|---------|-------------|
| `make compose-up` | Start services (development) |
| `make compose-up-monitoring` | Start with monitoring stack |
| `make compose-up-prod` | Start in production mode |
| `make compose-down` | Stop all services |
| `make compose-down-volumes` | Stop services and remove volumes |
| `make compose-logs` | View LauraDB logs |
| `make compose-logs-all` | View all service logs |
| `make compose-restart` | Restart all services |
| `make compose` | Alias for `compose-up` |

### Direct Docker Compose Commands

```bash
# Start services
docker-compose up -d

# Start with monitoring
docker-compose --profile monitoring up -d

# Stop services
docker-compose down

# View logs
docker-compose logs -f laura-db
docker-compose logs -f  # all services

# Scale services (if needed)
docker-compose up -d --scale laura-db=3

# Execute commands in container
docker-compose exec laura-db /bin/sh

# View service status
docker-compose ps

# Restart specific service
docker-compose restart laura-db
```

## Troubleshooting

### Common Issues

#### Port Already in Use

**Error**: `Bind for 0.0.0.0:8080 failed: port is already allocated`

**Solution**:
```bash
# Change port in .env
echo "LAURA_PORT=9090" >> .env

# Or stop conflicting service
lsof -ti:8080 | xargs kill
```

#### Container Fails to Start

**Check logs**:
```bash
docker-compose logs laura-db
```

**Common causes**:
- Insufficient memory
- Port conflicts
- Volume permission issues

#### Data Persistence Issues

**Verify volume**:
```bash
docker volume inspect laura-data
```

**Recreate volume**:
```bash
docker-compose down -v
docker-compose up -d
```

#### Health Check Failing

**Check health status**:
```bash
docker-compose ps
```

**Manual health check**:
```bash
docker-compose exec laura-db wget -O- http://localhost:8080/_health
```

#### Network Issues

**Inspect network**:
```bash
docker network inspect laura-db_laura-network
```

**Recreate network**:
```bash
docker-compose down
docker-compose up -d
```

### Performance Tuning

#### Adjust Buffer Size

For high-traffic scenarios:
```env
LAURA_BUFFER_SIZE=10000
```

#### Resource Limits

Edit `docker-compose.prod.yml`:
```yaml
deploy:
  resources:
    limits:
      cpus: '4'
      memory: 8G
```

#### Monitor Resource Usage

```bash
docker stats laura-db
```

### Debugging

#### Access Container Shell

```bash
docker-compose exec laura-db /bin/sh
```

#### Inspect Configuration

```bash
docker-compose config
```

#### View Container Details

```bash
docker inspect laura-db
```

## Advanced Configuration

### Custom Network

To use an existing Docker network:

```yaml
networks:
  laura-network:
    external:
      name: my-existing-network
```

### External Volumes

To use pre-existing volumes:

```yaml
volumes:
  laura-data:
    external:
      name: my-data-volume
```

### Multiple Instances

For load balancing (requires external load balancer):

```bash
docker-compose up -d --scale laura-db=3
```

### Custom Dockerfile

Build with custom modifications:

```bash
docker-compose build --build-arg GO_VERSION=1.25.4
```

## Security Best Practices

1. **Change default passwords**:
   ```bash
   # Generate secure password
   openssl rand -base64 32
   ```

2. **Enable TLS/SSL** (see Production Deployment section)

3. **Restrict CORS origins**:
   ```env
   LAURA_CORS_ORIGIN=https://trusted-domain.com
   ```

4. **Use Docker secrets** (Swarm mode):
   ```yaml
   secrets:
     db_password:
       external: true
   ```

5. **Run with read-only root filesystem**:
   ```yaml
   services:
     laura-db:
       read_only: true
       tmpfs:
         - /tmp
   ```

6. **Network isolation**:
   ```yaml
   services:
     laura-db:
       networks:
         - backend
       # No ports exposed to host
   ```

## Integration with Other Services

### Nginx Reverse Proxy

```nginx
server {
    listen 80;
    server_name laura-db.example.com;

    location / {
        proxy_pass http://localhost:8080;
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
    }
}
```

### Docker Swarm

Deploy as a stack:

```bash
docker stack deploy -c docker-compose.yml laura-db-stack
```

### Kubernetes

Convert to Kubernetes manifests:

```bash
kompose convert -f docker-compose.yml
```

## References

- [Docker Compose Documentation](https://docs.docker.com/compose/)
- [LauraDB Docker Guide](./docker.md)
- [LauraDB Deployment Guide](./deployment.md)
- [Prometheus Configuration](./prometheus-grafana.md)
- [TLS/SSL Setup](./tls-ssl.md)

## Support

For issues or questions:
- GitHub Issues: https://github.com/mnohosten/laura-db/issues
- Documentation: https://github.com/mnohosten/laura-db/tree/main/docs
