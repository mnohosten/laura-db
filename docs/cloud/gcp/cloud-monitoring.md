# LauraDB Cloud Monitoring Setup

This guide provides comprehensive instructions for setting up Google Cloud Monitoring (formerly Stackdriver) for LauraDB deployments on GCP.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Cloud Ops Agent Setup](#cloud-ops-agent-setup)
- [Metrics Collection](#metrics-collection)
- [Log Management](#log-management)
- [Custom Metrics](#custom-metrics)
- [Alerting Policies](#alerting-policies)
- [Dashboards](#dashboards)
- [Uptime Checks](#uptime-checks)
- [Error Reporting](#error-reporting)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

Cloud Monitoring provides:
- **Metrics**: System and application performance metrics
- **Logging**: Centralized log aggregation and analysis
- **Alerting**: Automated notifications based on conditions
- **Dashboards**: Visual representation of health and performance
- **Uptime Monitoring**: External availability checks
- **Error Reporting**: Automatic error aggregation

### What to Monitor

1. **System Metrics**: CPU, memory, disk, network
2. **Application Metrics**: Request rate, latency, error rate
3. **Database Metrics**: Query performance, cache hit ratio
4. **Business Metrics**: Active users, data volume

## Prerequisites

### Required APIs

```bash
# Enable required APIs
gcloud services enable monitoring.googleapis.com
gcloud services enable logging.googleapis.com
gcloud services enable cloudtrace.googleapis.com
gcloud services enable cloudprofiler.googleapis.com
```

### IAM Permissions

```bash
# Grant monitoring permissions to service account
gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/monitoring.metricWriter"

gcloud projects add-iam-policy-binding PROJECT_ID \
  --member="serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com" \
  --role="roles/logging.logWriter"
```

## Cloud Ops Agent Setup

### Install on GCE Instance

```bash
# Download and install
curl -sSO https://dl.google.com/cloudagents/add-google-cloud-ops-agent-repo.sh
sudo bash add-google-cloud-ops-agent-repo.sh --also-install

# Verify installation
sudo systemctl status google-cloud-ops-agent
```

### Configure Agent

Create `/etc/google-cloud-ops-agent/config.yaml`:

```yaml
logging:
  receivers:
    syslog:
      type: files
      include_paths:
        - /var/log/syslog
    laura-db-server:
      type: files
      include_paths:
        - /var/log/laura-db/server.log
      exclude_paths:
        - /var/log/laura-db/*.tmp
    laura-db-access:
      type: files
      include_paths:
        - /var/log/laura-db/access.log
    laura-db-errors:
      type: files
      include_paths:
        - /var/log/laura-db/error.log

  processors:
    parse_json:
      type: parse_json
      field: message
      time_key: timestamp
      time_format: "%Y-%m-%dT%H:%M:%S.%LZ"

  exporters:
    google:
      type: google_cloud_logging

  service:
    pipelines:
      default_pipeline:
        receivers:
          - syslog
        exporters:
          - google
      laura_db_pipeline:
        receivers:
          - laura-db-server
          - laura-db-access
          - laura-db-errors
        processors:
          - parse_json
        exporters:
          - google

metrics:
  receivers:
    hostmetrics:
      type: hostmetrics
      collection_interval: 60s
      metrics:
        - cpu
        - disk
        - filesystem
        - load
        - memory
        - network
        - paging
        - processes
        - process

  processors:
    metrics_filter:
      type: exclude_metrics
      metrics_pattern:
        - system.network.dropped

  exporters:
    google:
      type: google_cloud_monitoring

  service:
    pipelines:
      default_pipeline:
        receivers:
          - hostmetrics
        processors:
          - metrics_filter
        exporters:
          - google
```

Restart agent:

```bash
sudo systemctl restart google-cloud-ops-agent
```

### Install on GKE

Cloud Ops is automatically enabled for GKE clusters. Verify:

```bash
kubectl get pods -n kube-system | grep stackdriver
```

For custom metrics, deploy the agent:

```bash
kubectl apply -f https://raw.githubusercontent.com/GoogleCloudPlatform/k8s-stackdriver/master/custom-metrics-stackdriver-adapter/deploy/production/adapter.yaml
```

## Metrics Collection

### System Metrics

GCE instances automatically send basic metrics. View in console:
```
https://console.cloud.google.com/monitoring/metrics-explorer
```

### Application Metrics

Create metrics publisher script `/usr/local/bin/laura-db-metrics.sh`:

```bash
#!/bin/bash

PROJECT_ID=$(gcloud config get-value project)
INSTANCE_ID=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/id" -H "Metadata-Flavor: Google")
ZONE=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/zone" -H "Metadata-Flavor: Google" | cut -d'/' -f4)

# Get LauraDB health status
HEALTH=$(curl -s http://localhost:8080/_health)
HEALTH_STATUS=$(echo $HEALTH | jq -r '.status // "unknown"')

# Get request metrics
REQUEST_COUNT=$(grep -c "$(date +%Y-%m-%d\ %H:%M)" /var/log/laura-db/access.log 2>/dev/null || echo 0)
ERROR_COUNT=$(grep -c "ERROR" /var/log/laura-db/error.log 2>/dev/null || echo 0)

# Create metric payload
cat > /tmp/metrics.json <<EOF
{
  "timeSeries": [
    {
      "metric": {
        "type": "custom.googleapis.com/laura_db/request_count",
        "labels": {
          "instance_id": "$INSTANCE_ID"
        }
      },
      "resource": {
        "type": "gce_instance",
        "labels": {
          "instance_id": "$INSTANCE_ID",
          "zone": "$ZONE"
        }
      },
      "points": [
        {
          "interval": {
            "endTime": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
          },
          "value": {
            "int64Value": "$REQUEST_COUNT"
          }
        }
      ]
    },
    {
      "metric": {
        "type": "custom.googleapis.com/laura_db/error_count",
        "labels": {
          "instance_id": "$INSTANCE_ID"
        }
      },
      "resource": {
        "type": "gce_instance",
        "labels": {
          "instance_id": "$INSTANCE_ID",
          "zone": "$ZONE"
        }
      },
      "points": [
        {
          "interval": {
            "endTime": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
          },
          "value": {
            "int64Value": "$ERROR_COUNT"
          }
        }
      ]
    },
    {
      "metric": {
        "type": "custom.googleapis.com/laura_db/health_status",
        "labels": {
          "instance_id": "$INSTANCE_ID"
        }
      },
      "resource": {
        "type": "gce_instance",
        "labels": {
          "instance_id": "$INSTANCE_ID",
          "zone": "$ZONE"
        }
      },
      "points": [
        {
          "interval": {
            "endTime": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
          },
          "value": {
            "int64Value": $([ "$HEALTH_STATUS" = "healthy" ] && echo 1 || echo 0)
          }
        }
      ]
    }
  ]
}
EOF

# Send metrics
curl -X POST \
  -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  -H "Content-Type: application/json" \
  -d @/tmp/metrics.json \
  "https://monitoring.googleapis.com/v3/projects/$PROJECT_ID/timeSeries"

rm /tmp/metrics.json
```

Make executable and run via cron:

```bash
chmod +x /usr/local/bin/laura-db-metrics.sh

# Add to crontab
* * * * * /usr/local/bin/laura-db-metrics.sh
```

## Log Management

### View Logs

```bash
# View logs via gcloud
gcloud logging read "resource.type=gce_instance AND resource.labels.instance_id=INSTANCE_ID" \
  --limit=50 \
  --format=json

# Filter by severity
gcloud logging read "resource.type=gce_instance AND severity>=ERROR" \
  --limit=50

# Tail logs
gcloud logging tail "resource.type=gce_instance"
```

### Log-Based Metrics

```bash
# Create metric for error rate
gcloud logging metrics create laura_db_errors \
  --description="LauraDB error count" \
  --log-filter='resource.type="gce_instance"
    logName:"laura-db"
    severity>=ERROR'

# Create metric for slow queries
gcloud logging metrics create laura_db_slow_queries \
  --description="LauraDB slow query count" \
  --log-filter='resource.type="gce_instance"
    logName:"laura-db"
    textPayload=~"slow query"'

# Create metric for failed requests
gcloud logging metrics create laura_db_failed_requests \
  --description="LauraDB failed request count" \
  --log-filter='resource.type="gce_instance"
    logName:"laura-db/access"
    jsonPayload.status>=400'
```

### Log Sinks

Export logs to Cloud Storage for long-term retention:

```bash
# Create log sink to Cloud Storage
gcloud logging sinks create laura-db-archive \
  gs://PROJECT_ID-laura-db-logs \
  --log-filter='resource.type="gce_instance" AND logName:"laura-db"'

# Create log sink to BigQuery for analysis
gcloud logging sinks create laura-db-bigquery \
  bigquery.googleapis.com/projects/PROJECT_ID/datasets/laura_db_logs \
  --log-filter='resource.type="gce_instance" AND logName:"laura-db"'
```

## Custom Metrics

### Database-Specific Metrics

Create `/usr/local/bin/laura-db-custom-metrics.sh`:

```bash
#!/bin/bash

PROJECT_ID=$(gcloud config get-value project)
INSTANCE_ID=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/id" -H "Metadata-Flavor: Google")
ZONE=$(curl -s "http://metadata.google.internal/computeMetadata/v1/instance/zone" -H "Metadata-Flavor: Google" | cut -d'/' -f4)

# Get database statistics (assuming LauraDB exposes these)
STATS=$(curl -s http://localhost:8080/_stats)

# Parse metrics
COLLECTION_COUNT=$(echo $STATS | jq -r '.collections.count // 0')
DOCUMENT_COUNT=$(echo $STATS | jq -r '.documents.total // 0')
INDEX_COUNT=$(echo $STATS | jq -r '.indexes.total // 0')
CACHE_HIT_RATIO=$(echo $STATS | jq -r '.cache.hit_ratio // 0')
BUFFER_USAGE=$(echo $STATS | jq -r '.buffer_pool.usage_percent // 0')

# Create metric payload
cat > /tmp/db-metrics.json <<EOF
{
  "timeSeries": [
    {
      "metric": {
        "type": "custom.googleapis.com/laura_db/collection_count"
      },
      "resource": {
        "type": "gce_instance",
        "labels": {
          "instance_id": "$INSTANCE_ID",
          "zone": "$ZONE"
        }
      },
      "points": [{
        "interval": {"endTime": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"},
        "value": {"int64Value": "$COLLECTION_COUNT"}
      }]
    },
    {
      "metric": {
        "type": "custom.googleapis.com/laura_db/document_count"
      },
      "resource": {
        "type": "gce_instance",
        "labels": {
          "instance_id": "$INSTANCE_ID",
          "zone": "$ZONE"
        }
      },
      "points": [{
        "interval": {"endTime": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"},
        "value": {"int64Value": "$DOCUMENT_COUNT"}
      }]
    },
    {
      "metric": {
        "type": "custom.googleapis.com/laura_db/cache_hit_ratio"
      },
      "resource": {
        "type": "gce_instance",
        "labels": {
          "instance_id": "$INSTANCE_ID",
          "zone": "$ZONE"
        }
      },
      "points": [{
        "interval": {"endTime": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"},
        "value": {"doubleValue": $CACHE_HIT_RATIO}
      }]
    }
  ]
}
EOF

# Send metrics
curl -X POST \
  -H "Authorization: Bearer $(gcloud auth print-access-token)" \
  -H "Content-Type: application/json" \
  -d @/tmp/db-metrics.json \
  "https://monitoring.googleapis.com/v3/projects/$PROJECT_ID/timeSeries"

rm /tmp/db-metrics.json
```

## Alerting Policies

### Create Notification Channels

```bash
# Create email notification channel
gcloud alpha monitoring channels create \
  --display-name="LauraDB Ops Team" \
  --type=email \
  --channel-labels=email_address=ops@example.com

# Create SMS notification channel
gcloud alpha monitoring channels create \
  --display-name="On-Call SMS" \
  --type=sms \
  --channel-labels=number=+1234567890

# Create Slack webhook
gcloud alpha monitoring channels create \
  --display-name="Slack Alerts" \
  --type=slack \
  --channel-labels=url=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# List channels to get IDs
gcloud alpha monitoring channels list
```

### Critical Alerts

**1. High CPU Utilization**

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="LauraDB High CPU" \
  --condition-display-name="CPU > 80%" \
  --condition-threshold-value=0.8 \
  --condition-threshold-duration=300s \
  --condition-filter='resource.type="gce_instance" AND metric.type="compute.googleapis.com/instance/cpu/utilization"' \
  --condition-comparison=COMPARISON_GT
```

**2. High Memory Usage**

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="LauraDB High Memory" \
  --condition-display-name="Memory > 85%" \
  --condition-threshold-value=85 \
  --condition-threshold-duration=300s \
  --condition-filter='resource.type="gce_instance" AND metric.type="agent.googleapis.com/memory/percent_used"' \
  --condition-comparison=COMPARISON_GT
```

**3. Disk Space Low**

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="LauraDB Low Disk Space" \
  --condition-display-name="Disk > 80%" \
  --condition-threshold-value=80 \
  --condition-threshold-duration=300s \
  --condition-filter='resource.type="gce_instance" AND metric.type="agent.googleapis.com/disk/percent_used"' \
  --condition-comparison=COMPARISON_GT
```

**4. High Error Rate**

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="LauraDB High Error Rate" \
  --condition-display-name="Errors > 100/min" \
  --condition-threshold-value=100 \
  --condition-threshold-duration=60s \
  --condition-filter='metric.type="logging.googleapis.com/user/laura_db_errors"' \
  --condition-comparison=COMPARISON_GT
```

**5. Service Down**

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="LauraDB Service Down" \
  --condition-display-name="Health check failed" \
  --condition-threshold-value=1 \
  --condition-threshold-duration=120s \
  --condition-filter='metric.type="custom.googleapis.com/laura_db/health_status"' \
  --condition-comparison=COMPARISON_LT
```

### Alert Policy YAML

For more complex policies, use YAML:

```yaml
# alert-policy.yaml
displayName: "LauraDB Critical Alert"
documentation:
  content: "LauraDB is experiencing critical issues. Check the dashboard and logs."
  mimeType: text/markdown
conditions:
  - displayName: "High CPU and Memory"
    conditionThreshold:
      filter: 'resource.type="gce_instance" AND metric.type="compute.googleapis.com/instance/cpu/utilization"'
      comparison: COMPARISON_GT
      thresholdValue: 0.8
      duration: 300s
      aggregations:
        - alignmentPeriod: 60s
          perSeriesAligner: ALIGN_MEAN
combiner: OR
notificationChannels:
  - projects/PROJECT_ID/notificationChannels/CHANNEL_ID
alertStrategy:
  autoClose: 604800s
```

Apply:

```bash
gcloud alpha monitoring policies create --policy-from-file=alert-policy.yaml
```

## Dashboards

### Create Custom Dashboard

```bash
# Create dashboard via gcloud
gcloud monitoring dashboards create --config-from-file=dashboard.json
```

Dashboard JSON (`dashboard.json`):

```json
{
  "displayName": "LauraDB Monitoring Dashboard",
  "mosaicLayout": {
    "columns": 12,
    "tiles": [
      {
        "width": 6,
        "height": 4,
        "widget": {
          "title": "CPU Utilization",
          "xyChart": {
            "dataSets": [
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "resource.type=\"gce_instance\" AND metric.type=\"compute.googleapis.com/instance/cpu/utilization\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_MEAN"
                    }
                  }
                },
                "plotType": "LINE"
              }
            ],
            "yAxis": {
              "scale": "LINEAR"
            }
          }
        }
      },
      {
        "xPos": 6,
        "width": 6,
        "height": 4,
        "widget": {
          "title": "Memory Usage",
          "xyChart": {
            "dataSets": [
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "resource.type=\"gce_instance\" AND metric.type=\"agent.googleapis.com/memory/percent_used\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_MEAN"
                    }
                  }
                },
                "plotType": "LINE"
              }
            ]
          }
        }
      },
      {
        "yPos": 4,
        "width": 6,
        "height": 4,
        "widget": {
          "title": "Request Count",
          "xyChart": {
            "dataSets": [
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "metric.type=\"custom.googleapis.com/laura_db/request_count\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_RATE"
                    }
                  }
                },
                "plotType": "LINE"
              }
            ]
          }
        }
      },
      {
        "xPos": 6,
        "yPos": 4,
        "width": 6,
        "height": 4,
        "widget": {
          "title": "Error Rate",
          "xyChart": {
            "dataSets": [
              {
                "timeSeriesQuery": {
                  "timeSeriesFilter": {
                    "filter": "metric.type=\"logging.googleapis.com/user/laura_db_errors\"",
                    "aggregation": {
                      "alignmentPeriod": "60s",
                      "perSeriesAligner": "ALIGN_RATE"
                    }
                  }
                },
                "plotType": "LINE"
              }
            ]
          }
        }
      },
      {
        "yPos": 8,
        "width": 12,
        "height": 4,
        "widget": {
          "title": "Recent Errors",
          "logsPanel": {
            "resourceNames": ["projects/PROJECT_ID"],
            "filter": "resource.type=\"gce_instance\" AND severity>=ERROR"
          }
        }
      }
    ]
  }
}
```

View dashboard:
```
https://console.cloud.google.com/monitoring/dashboards
```

## Uptime Checks

### Create Uptime Check

```bash
gcloud monitoring uptime-checks create \
  --display-name="LauraDB Health Check" \
  --resource-type=uptime-url \
  --monitored-resource=host=EXTERNAL_IP,project_id=PROJECT_ID \
  --http-check-path="/_health" \
  --port=8080 \
  --check-interval=60s \
  --timeout=10s
```

### Create Alert on Uptime Check

```bash
gcloud alpha monitoring policies create \
  --notification-channels=CHANNEL_ID \
  --display-name="LauraDB Uptime Alert" \
  --condition-display-name="Uptime check failed" \
  --condition-threshold-value=1 \
  --condition-threshold-duration=120s \
  --condition-filter='metric.type="monitoring.googleapis.com/uptime_check/check_passed" AND metric.label.check_id="CHECK_ID"' \
  --condition-comparison=COMPARISON_LT
```

## Error Reporting

### Enable Error Reporting

Error Reporting automatically aggregates errors from Cloud Logging.

View errors:
```
https://console.cloud.google.com/errors
```

### Report Custom Errors

```bash
# From application logs, structure errors like this:
echo '{
  "severity": "ERROR",
  "message": "Database connection failed",
  "context": {
    "httpRequest": {
      "method": "POST",
      "url": "/api/query",
      "userAgent": "Mozilla/5.0",
      "responseStatusCode": 500
    },
    "user": "user@example.com"
  },
  "stack_trace": "Error: Connection timeout\n  at connect() line 42"
}' | gcloud logging write laura-db-errors --severity=ERROR --payload-type=json -
```

## Cost Optimization

### Cloud Monitoring Costs

| Component | Free Tier | Pricing | Optimization |
|-----------|-----------|---------|--------------|
| Metrics (first 150) | Free | $0.2580 per metric/month | Use metric filters |
| Metrics (151-100K) | - | $0.1030 per metric/month | Aggregate similar metrics |
| Logs ingestion (first 50GB) | Free | $0.50 per GB | Filter unnecessary logs |
| Logs storage | First 30 days free | $0.01 per GB after 30 days | Set retention policies |
| Monitoring API calls (first 1M) | Free | $0.01 per 1000 calls | Batch requests |

### Cost-Saving Tips

1. **Use log exclusion filters**:
```bash
gcloud logging sinks create _Default \
  --log-filter='NOT (resource.type="gce_instance" AND severity="DEBUG")'
```

2. **Aggregate metrics**: Send fewer, aggregated custom metrics

3. **Set retention policies**:
```bash
gcloud logging buckets update _Default \
  --location=global \
  --retention-days=30
```

4. **Use sampling for high-volume logs**

## Troubleshooting

### Metrics Not Appearing

```bash
# Check agent status
sudo systemctl status google-cloud-ops-agent

# View agent logs
sudo journalctl -u google-cloud-ops-agent -f

# Verify service account permissions
gcloud projects get-iam-policy PROJECT_ID \
  --flatten="bindings[].members" \
  --filter="bindings.members:serviceAccount:laura-db-sa@PROJECT_ID.iam.gserviceaccount.com"

# Test metric write
gcloud monitoring time-series create \
  --project=PROJECT_ID \
  --time-series-data='[{
    "metric": {"type": "custom.googleapis.com/test_metric"},
    "resource": {"type": "global"},
    "points": [{
      "interval": {"endTime": "'$(date -u +%Y-%m-%dT%H:%M:%SZ)'"},
      "value": {"int64Value": "1"}
    }]
  }]'
```

### Logs Not Showing

```bash
# Check if logs are being written
sudo tail -f /var/log/laura-db/server.log

# Check agent configuration
sudo cat /etc/google-cloud-ops-agent/config.yaml

# Verify logging is enabled
gcloud services list --enabled | grep logging

# Test log write
gcloud logging write test-log "Test message" --severity=INFO
```

### Alerts Not Triggering

```bash
# Check alert policy
gcloud alpha monitoring policies list

# Describe specific policy
gcloud alpha monitoring policies describe POLICY_ID

# Check notification channel
gcloud alpha monitoring channels list
gcloud alpha monitoring channels describe CHANNEL_ID

# Test notification
gcloud alpha monitoring channels verify CHANNEL_ID
```

## Best Practices

1. ✅ **Use Cloud Ops Agent** for comprehensive monitoring
2. ✅ **Create log-based metrics** for important events
3. ✅ **Set appropriate alert thresholds** based on baseline
4. ✅ **Use notification channels wisely** (email, SMS, Slack)
5. ✅ **Create dashboards for different audiences** (ops, dev, business)
6. ✅ **Set log retention policies** to manage costs
7. ✅ **Use uptime checks** for external monitoring
8. ✅ **Enable Error Reporting** for automated error aggregation
9. ✅ **Monitor your monitoring** (check agent health)
10. ✅ **Use labels** for resource organization

## Next Steps

- [GCE Deployment Guide](./gce-deployment.md)
- [GKE Deployment Guide](./gke-deployment.md)
- [Cloud Storage Backup Integration](./cloud-storage-backup.md)

## Additional Resources

- [Cloud Monitoring Documentation](https://cloud.google.com/monitoring/docs)
- [Cloud Logging Documentation](https://cloud.google.com/logging/docs)
- [Cloud Ops Agent Documentation](https://cloud.google.com/stackdriver/docs/solutions/agents/ops-agent)
- [Alerting Documentation](https://cloud.google.com/monitoring/alerts)
- [Pricing Calculator](https://cloud.google.com/products/calculator)
