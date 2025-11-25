# Docker Compose Demo

This example demonstrates how to deploy and use LauraDB with Docker Compose.

## Prerequisites

- Docker 20.10+
- Docker Compose 1.29+
- curl (for API testing)

## Quick Start

### 1. Start LauraDB

From the project root:

```bash
cd ../..
make compose-up
```

This will:
- Build the LauraDB Docker image
- Start the database container
- Expose port 8080
- Create a persistent volume for data

### 2. Verify Service is Running

```bash
curl http://localhost:8080/_health
```

Expected response:
```json
{"status":"ok","timestamp":"2025-11-24T10:00:00Z"}
```

### 3. Insert Sample Data

```bash
# Insert a user
curl -X POST http://localhost:8080/users/_doc \
  -H "Content-Type: application/json" \
  -d '{"name": "Alice", "email": "alice@example.com", "age": 28}'

# Insert more users
curl -X POST http://localhost:8080/users/_doc \
  -H "Content-Type: application/json" \
  -d '{"name": "Bob", "email": "bob@example.com", "age": 32}'

curl -X POST http://localhost:8080/users/_doc \
  -H "Content-Type: application/json" \
  -d '{"name": "Charlie", "email": "charlie@example.com", "age": 25}'
```

### 4. Query Data

```bash
# Find all users
curl -X POST http://localhost:8080/users/_search \
  -H "Content-Type: application/json" \
  -d '{"filter": {}}'

# Find users older than 25
curl -X POST http://localhost:8080/users/_search \
  -H "Content-Type: application/json" \
  -d '{"filter": {"age": {"$gt": 25}}}'
```

### 5. Create an Index

```bash
curl -X POST http://localhost:8080/users/_index \
  -H "Content-Type: application/json" \
  -d '{"field": "email", "unique": true}'
```

### 6. Aggregate Data

```bash
curl -X POST http://localhost:8080/users/_aggregate \
  -H "Content-Type: application/json" \
  -d '{
    "pipeline": [
      {
        "$group": {
          "_id": null,
          "avgAge": {"$avg": "$age"},
          "count": {"$count": {}}
        }
      }
    ]
  }'
```

### 7. View Statistics

```bash
curl http://localhost:8080/_stats | jq
```

### 8. Access Admin Console

Open your browser to: http://localhost:8080/

## With Monitoring

Start LauraDB with Prometheus and Grafana:

```bash
make compose-up-monitoring
```

Access:
- **LauraDB**: http://localhost:8080
- **Prometheus**: http://localhost:9090
- **Grafana**: http://localhost:3000 (admin/admin)

### Grafana Setup

1. Login to Grafana at http://localhost:3000
2. Navigate to Configuration → Data Sources
3. Add Prometheus data source:
   - URL: http://prometheus:9090
   - Access: Server (default)
4. Navigate to Dashboards to view LauraDB metrics

### Query Prometheus Metrics

```bash
# Query request rate
curl -g 'http://localhost:9090/api/v1/query?query=rate(laura_db_queries_total[5m])'

# Query average latency
curl -g 'http://localhost:9090/api/v1/query?query=laura_db_query_duration_seconds'
```

## Production Deployment

Deploy in production mode with resource limits:

```bash
# Create production environment
cp .env.example .env
# Edit .env with production values

# Start in production mode
make compose-up-prod
```

## Management Commands

### View Logs

```bash
# LauraDB logs
make compose-logs

# All logs (with monitoring)
make compose-logs-all
```

### Restart Services

```bash
make compose-restart
```

### Stop Services

```bash
# Stop services (keep data)
make compose-down

# Stop and remove volumes
make compose-down-volumes
```

### Scale Services

```bash
# Run multiple LauraDB instances (requires load balancer)
docker-compose up -d --scale laura-db=3
```

## Data Persistence

Data is stored in Docker volumes:

```bash
# List volumes
docker volume ls | grep laura

# Inspect volume
docker volume inspect laura-db_laura-data

# Backup volume
docker run --rm \
  -v laura-db_laura-data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar czf /backup/laura-backup.tar.gz -C /data .

# Restore volume
docker run --rm \
  -v laura-db_laura-data:/data \
  -v $(pwd)/backup:/backup \
  alpine tar xzf /backup/laura-backup.tar.gz -C /data
```

## Troubleshooting

### Port Already in Use

```bash
# Change port in .env
echo "LAURA_PORT=9090" >> .env
make compose-restart
```

### Container Won't Start

```bash
# View logs
docker-compose logs laura-db

# Check container status
docker-compose ps
```

### Reset Everything

```bash
# Stop and remove everything
make compose-down-volumes

# Restart fresh
make compose-up
```

## Configuration

### Environment Variables

Create `.env` file:

```env
LAURA_PORT=8080
LAURA_BUFFER_SIZE=5000
LAURA_CORS_ORIGIN=*
GRAFANA_ADMIN_PASSWORD=secure-password
```

### Resource Limits

Edit `docker-compose.prod.yml` to adjust:

```yaml
deploy:
  resources:
    limits:
      cpus: '4'
      memory: 8G
```

## Example Workload

Run a realistic workload:

```bash
#!/bin/bash
# insert-workload.sh

BASE_URL="http://localhost:8080"

echo "Inserting 1000 documents..."
for i in {1..1000}; do
  curl -s -X POST "$BASE_URL/test/_doc" \
    -H "Content-Type: application/json" \
    -d "{\"id\": $i, \"value\": \"item-$i\", \"timestamp\": \"$(date -u +%Y-%m-%dT%H:%M:%SZ)\"}" \
    > /dev/null

  if [ $((i % 100)) -eq 0 ]; then
    echo "Inserted $i documents..."
  fi
done

echo "✓ Workload complete!"
```

Make executable and run:

```bash
chmod +x insert-workload.sh
./insert-workload.sh
```

## Next Steps

- Read the [Docker Compose Guide](../../docs/docker-compose.md)
- Explore the [HTTP API documentation](../../docs/http-api.md)
- Check out [Performance Tuning Guide](../../docs/performance-tuning.md)
- Try other [examples](../)

## Support

For issues or questions:
- GitHub Issues: https://github.com/mnohosten/laura-db/issues
- Documentation: https://github.com/mnohosten/laura-db/tree/main/docs
