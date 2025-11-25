# LauraDB CloudWatch Monitoring Setup

This guide provides comprehensive instructions for setting up Amazon CloudWatch monitoring, logging, and alerting for LauraDB deployments on AWS.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [CloudWatch Agent Setup](#cloudwatch-agent-setup)
- [Metrics Collection](#metrics-collection)
- [Log Management](#log-management)
- [Custom Metrics](#custom-metrics)
- [Alarms and Alerts](#alarms-and-alerts)
- [Dashboards](#dashboards)
- [Container Insights](#container-insights)
- [Performance Insights](#performance-insights)
- [Cost Optimization](#cost-optimization)
- [Troubleshooting](#troubleshooting)

## Overview

CloudWatch monitoring provides:
- **Metrics**: System and application performance metrics
- **Logs**: Centralized log aggregation and analysis
- **Alarms**: Automated alerts based on thresholds
- **Dashboards**: Visual representation of health and performance
- **Insights**: Advanced log analytics and anomaly detection

### What to Monitor

1. **System Metrics**: CPU, memory, disk, network
2. **Application Metrics**: Request rate, response time, error rate
3. **Database Metrics**: Query performance, transaction rate, cache hit ratio
4. **Business Metrics**: Active users, data volume, backup status

## Prerequisites

### IAM Permissions

Create IAM policy for CloudWatch access:

```json
{
  "Version": "2012-10-17",
  "Statement": [
    {
      "Effect": "Allow",
      "Action": [
        "cloudwatch:PutMetricData",
        "cloudwatch:GetMetricStatistics",
        "cloudwatch:ListMetrics"
      ],
      "Resource": "*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "logs:CreateLogGroup",
        "logs:CreateLogStream",
        "logs:PutLogEvents",
        "logs:DescribeLogStreams"
      ],
      "Resource": "arn:aws:logs:*:*:*"
    },
    {
      "Effect": "Allow",
      "Action": [
        "ec2:DescribeInstances",
        "ec2:DescribeTags",
        "ec2:DescribeVolumes"
      ],
      "Resource": "*"
    }
  ]
}
```

Attach to EC2 instance role:

```bash
aws iam attach-role-policy \
  --role-name LauraDB-EC2-Role \
  --policy-arn arn:aws:iam::ACCOUNT_ID:policy/LauraDB-CloudWatch-Policy
```

## CloudWatch Agent Setup

### Step 1: Install CloudWatch Agent

**For Amazon Linux 2 / CentOS:**

```bash
# Download agent
wget https://s3.amazonaws.com/amazoncloudwatch-agent/amazon_linux/amd64/latest/amazon-cloudwatch-agent.rpm

# Install
sudo rpm -U ./amazon-cloudwatch-agent.rpm

# Verify installation
/opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl \
  -a query \
  -m ec2 \
  -c default \
  -s
```

**For Ubuntu / Debian:**

```bash
wget https://s3.amazonaws.com/amazoncloudwatch-agent/ubuntu/amd64/latest/amazon-cloudwatch-agent.deb
sudo dpkg -i -E ./amazon-cloudwatch-agent.deb
```

### Step 2: Configure CloudWatch Agent

Create configuration file `/opt/aws/amazon-cloudwatch-agent/etc/config.json`:

```json
{
  "agent": {
    "metrics_collection_interval": 60,
    "run_as_user": "cwagent"
  },
  "logs": {
    "logs_collected": {
      "files": {
        "collect_list": [
          {
            "file_path": "/var/log/laura-db/server.log",
            "log_group_name": "/aws/lauradb/server",
            "log_stream_name": "{instance_id}",
            "timezone": "UTC"
          },
          {
            "file_path": "/var/log/laura-db/access.log",
            "log_group_name": "/aws/lauradb/access",
            "log_stream_name": "{instance_id}",
            "timezone": "UTC"
          },
          {
            "file_path": "/var/log/laura-db/error.log",
            "log_group_name": "/aws/lauradb/errors",
            "log_stream_name": "{instance_id}",
            "timezone": "UTC"
          }
        ]
      }
    }
  },
  "metrics": {
    "namespace": "LauraDB/EC2",
    "metrics_collected": {
      "cpu": {
        "measurement": [
          {
            "name": "cpu_usage_idle",
            "rename": "CPU_IDLE",
            "unit": "Percent"
          },
          {
            "name": "cpu_usage_iowait",
            "rename": "CPU_IOWAIT",
            "unit": "Percent"
          },
          "cpu_time_guest"
        ],
        "metrics_collection_interval": 60,
        "totalcpu": false
      },
      "disk": {
        "measurement": [
          {
            "name": "used_percent",
            "rename": "DISK_USED",
            "unit": "Percent"
          },
          "disk_free",
          "disk_used"
        ],
        "metrics_collection_interval": 60,
        "resources": [
          "*"
        ]
      },
      "diskio": {
        "measurement": [
          "io_time",
          "read_bytes",
          "write_bytes"
        ],
        "metrics_collection_interval": 60,
        "resources": [
          "*"
        ]
      },
      "mem": {
        "measurement": [
          {
            "name": "mem_used_percent",
            "rename": "MEM_USED",
            "unit": "Percent"
          },
          "mem_available",
          "mem_used"
        ],
        "metrics_collection_interval": 60
      },
      "netstat": {
        "measurement": [
          "tcp_established",
          "tcp_time_wait"
        ],
        "metrics_collection_interval": 60
      },
      "swap": {
        "measurement": [
          {
            "name": "swap_used_percent",
            "rename": "SWAP_USED",
            "unit": "Percent"
          }
        ],
        "metrics_collection_interval": 60
      }
    }
  }
}
```

### Step 3: Start CloudWatch Agent

```bash
# Start agent with configuration
sudo /opt/aws/amazon-cloudwatch-agent/bin/amazon-cloudwatch-agent-ctl \
  -a fetch-config \
  -m ec2 \
  -s \
  -c file:/opt/aws/amazon-cloudwatch-agent/etc/config.json

# Enable on boot
sudo systemctl enable amazon-cloudwatch-agent

# Check status
sudo systemctl status amazon-cloudwatch-agent
```

## Metrics Collection

### System Metrics (Default)

CloudWatch automatically collects basic EC2 metrics:

- **CPU Utilization**: Percentage of allocated CPU in use
- **Network In/Out**: Bytes received/sent
- **Disk Read/Write**: Bytes and operations

**Enable detailed monitoring** (1-minute intervals):

```bash
aws ec2 monitor-instances --instance-ids i-xxxxxxxxx
```

### Application Metrics

Create a metrics publisher script `/usr/local/bin/laura-db-metrics.sh`:

```bash
#!/bin/bash

NAMESPACE="LauraDB/Application"
INSTANCE_ID=$(ec2-metadata --instance-id | cut -d " " -f 2)
REGION=$(ec2-metadata --availability-zone | cut -d " " -f 2 | sed 's/[a-z]$//')

# Get LauraDB health status
HEALTH=$(curl -s http://localhost:8080/_health)
HEALTH_STATUS=$(echo $HEALTH | jq -r '.status')

# Get request metrics (from access logs)
REQUEST_COUNT=$(grep -c "$(date +%Y-%m-%d\ %H:%M)" /var/log/laura-db/access.log)
ERROR_COUNT=$(grep -c "ERROR" /var/log/laura-db/error.log)

# Publish metrics
aws cloudwatch put-metric-data \
  --namespace $NAMESPACE \
  --metric-name RequestCount \
  --value $REQUEST_COUNT \
  --unit Count \
  --dimensions Instance=$INSTANCE_ID

aws cloudwatch put-metric-data \
  --namespace $NAMESPACE \
  --metric-name ErrorCount \
  --value $ERROR_COUNT \
  --unit Count \
  --dimensions Instance=$INSTANCE_ID

aws cloudwatch put-metric-data \
  --namespace $NAMESPACE \
  --metric-name HealthStatus \
  --value $([ "$HEALTH_STATUS" = "healthy" ] && echo 1 || echo 0) \
  --unit None \
  --dimensions Instance=$INSTANCE_ID
```

Run via cron every minute:

```bash
* * * * * /usr/local/bin/laura-db-metrics.sh
```

## Log Management

### Log Groups Setup

```bash
# Create log groups
aws logs create-log-group --log-group-name /aws/lauradb/server
aws logs create-log-group --log-group-name /aws/lauradb/access
aws logs create-log-group --log-group-name /aws/lauradb/errors

# Set retention policy
aws logs put-retention-policy \
  --log-group-name /aws/lauradb/server \
  --retention-in-days 30

aws logs put-retention-policy \
  --log-group-name /aws/lauradb/access \
  --retention-in-days 7

aws logs put-retention-policy \
  --log-group-name /aws/lauradb/errors \
  --retention-in-days 90
```

### Log Insights Queries

**Query 1: Error Rate by Hour**

```sql
fields @timestamp, @message
| filter @message like /ERROR/
| stats count() as ErrorCount by bin(1h)
```

**Query 2: Slowest Queries**

```sql
fields @timestamp, query, duration
| filter duration > 1000
| sort duration desc
| limit 20
```

**Query 3: Request Rate by Endpoint**

```sql
fields @timestamp, endpoint, method
| stats count() as RequestCount by endpoint
| sort RequestCount desc
```

**Query 4: Failed Requests**

```sql
fields @timestamp, status_code, endpoint, error_message
| filter status_code >= 400
| sort @timestamp desc
| limit 100
```

### Log Metric Filters

Create metric filter for error rate:

```bash
aws logs put-metric-filter \
  --log-group-name /aws/lauradb/errors \
  --filter-name ErrorRateFilter \
  --filter-pattern '[timestamp, level=ERROR, ...]' \
  --metric-transformations \
    metricName=ErrorRate,\
metricNamespace=LauraDB/Logs,\
metricValue=1,\
unit=Count
```

## Custom Metrics

### Database-Specific Metrics

Create `/usr/local/bin/laura-db-custom-metrics.sh`:

```bash
#!/bin/bash

NAMESPACE="LauraDB/Database"
API_ENDPOINT="http://localhost:8080"

# Get database statistics (assuming LauraDB exposes these)
STATS=$(curl -s $API_ENDPOINT/_stats)

# Parse and publish metrics
COLLECTION_COUNT=$(echo $STATS | jq -r '.collections.count')
DOCUMENT_COUNT=$(echo $STATS | jq -r '.documents.total')
INDEX_COUNT=$(echo $STATS | jq -r '.indexes.total')
CACHE_HIT_RATIO=$(echo $STATS | jq -r '.cache.hit_ratio')
BUFFER_USAGE=$(echo $STATS | jq -r '.buffer_pool.usage_percent')

# Publish metrics
aws cloudwatch put-metric-data \
  --namespace $NAMESPACE \
  --metric-data \
    '[
      {
        "MetricName": "CollectionCount",
        "Value": '$COLLECTION_COUNT',
        "Unit": "Count"
      },
      {
        "MetricName": "DocumentCount",
        "Value": '$DOCUMENT_COUNT',
        "Unit": "Count"
      },
      {
        "MetricName": "IndexCount",
        "Value": '$INDEX_COUNT',
        "Unit": "Count"
      },
      {
        "MetricName": "CacheHitRatio",
        "Value": '$CACHE_HIT_RATIO',
        "Unit": "Percent"
      },
      {
        "MetricName": "BufferPoolUsage",
        "Value": '$BUFFER_USAGE',
        "Unit": "Percent"
      }
    ]'
```

### Business Metrics

```bash
#!/bin/bash

NAMESPACE="LauraDB/Business"

# Example: Count active users
ACTIVE_USERS=$(curl -s http://localhost:8080/api/stats/active-users | jq -r '.count')

# Example: Calculate data growth rate
CURRENT_SIZE=$(df -BG /var/lib/laura-db | tail -1 | awk '{print $3}' | sed 's/G//')
LAST_SIZE=$(cat /tmp/laura-db-last-size 2>/dev/null || echo $CURRENT_SIZE)
GROWTH_RATE=$(echo "scale=2; ($CURRENT_SIZE - $LAST_SIZE) / $LAST_SIZE * 100" | bc)
echo $CURRENT_SIZE > /tmp/laura-db-last-size

aws cloudwatch put-metric-data \
  --namespace $NAMESPACE \
  --metric-data \
    '[
      {
        "MetricName": "ActiveUsers",
        "Value": '$ACTIVE_USERS',
        "Unit": "Count"
      },
      {
        "MetricName": "DataGrowthRate",
        "Value": '$GROWTH_RATE',
        "Unit": "Percent"
      }
    ]'
```

## Alarms and Alerts

### Create SNS Topic for Alerts

```bash
# Create SNS topic
aws sns create-topic --name LauraDB-Alerts

# Subscribe email
aws sns subscribe \
  --topic-arn arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Alerts \
  --protocol email \
  --notification-endpoint ops@example.com

# Subscribe SMS (optional)
aws sns subscribe \
  --topic-arn arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Alerts \
  --protocol sms \
  --notification-endpoint +1234567890
```

### Critical Alarms

**1. High CPU Utilization**

```bash
aws cloudwatch put-metric-alarm \
  --alarm-name LauraDB-HighCPU \
  --alarm-description "CPU utilization above 80%" \
  --metric-name CPUUtilization \
  --namespace AWS/EC2 \
  --statistic Average \
  --period 300 \
  --evaluation-periods 2 \
  --threshold 80 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=InstanceId,Value=i-xxxxxxxxx \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Alerts
```

**2. High Memory Usage**

```bash
aws cloudwatch put-metric-alarm \
  --alarm-name LauraDB-HighMemory \
  --alarm-description "Memory usage above 85%" \
  --metric-name MEM_USED \
  --namespace LauraDB/EC2 \
  --statistic Average \
  --period 300 \
  --evaluation-periods 2 \
  --threshold 85 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=InstanceId,Value=i-xxxxxxxxx \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Alerts
```

**3. Disk Space Low**

```bash
aws cloudwatch put-metric-alarm \
  --alarm-name LauraDB-LowDiskSpace \
  --alarm-description "Disk usage above 80%" \
  --metric-name DISK_USED \
  --namespace LauraDB/EC2 \
  --statistic Average \
  --period 300 \
  --evaluation-periods 1 \
  --threshold 80 \
  --comparison-operator GreaterThanThreshold \
  --dimensions Name=InstanceId,Value=i-xxxxxxxxx Name=path,Value=/ \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Alerts
```

**4. High Error Rate**

```bash
aws cloudwatch put-metric-alarm \
  --alarm-name LauraDB-HighErrorRate \
  --alarm-description "Error rate above 100 per minute" \
  --metric-name ErrorRate \
  --namespace LauraDB/Logs \
  --statistic Sum \
  --period 60 \
  --evaluation-periods 3 \
  --threshold 100 \
  --comparison-operator GreaterThanThreshold \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Alerts
```

**5. Service Unhealthy**

```bash
aws cloudwatch put-metric-alarm \
  --alarm-name LauraDB-ServiceDown \
  --alarm-description "LauraDB health check failed" \
  --metric-name HealthStatus \
  --namespace LauraDB/Application \
  --statistic Minimum \
  --period 60 \
  --evaluation-periods 2 \
  --threshold 1 \
  --comparison-operator LessThanThreshold \
  --dimensions Name=Instance,Value=i-xxxxxxxxx \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Alerts \
  --treat-missing-data notBreaching
```

**6. ELB Target Unhealthy (if using ALB)**

```bash
aws cloudwatch put-metric-alarm \
  --alarm-name LauraDB-UnhealthyTargets \
  --alarm-description "Unhealthy targets in target group" \
  --metric-name UnHealthyHostCount \
  --namespace AWS/ApplicationELB \
  --statistic Average \
  --period 300 \
  --evaluation-periods 1 \
  --threshold 1 \
  --comparison-operator GreaterThanOrEqualToThreshold \
  --dimensions \
    Name=LoadBalancer,Value=app/laura-db-alb/xxxxxxxxx \
    Name=TargetGroup,Value=targetgroup/laura-db/xxxxxxxxx \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Alerts
```

### Composite Alarms

```bash
aws cloudwatch put-composite-alarm \
  --alarm-name LauraDB-CriticalIssue \
  --alarm-description "Multiple critical issues detected" \
  --alarm-rule "ALARM(LauraDB-HighCPU) OR ALARM(LauraDB-HighMemory) OR ALARM(LauraDB-ServiceDown)" \
  --alarm-actions arn:aws:sns:us-east-1:ACCOUNT_ID:LauraDB-Critical-Alerts
```

## Dashboards

### Create CloudWatch Dashboard

```bash
cat > dashboard.json <<'EOF'
{
  "widgets": [
    {
      "type": "metric",
      "properties": {
        "metrics": [
          ["AWS/EC2", "CPUUtilization", {"stat": "Average"}],
          ["LauraDB/EC2", "MEM_USED", {"stat": "Average"}]
        ],
        "period": 300,
        "stat": "Average",
        "region": "us-east-1",
        "title": "System Resources",
        "yAxis": {
          "left": {
            "min": 0,
            "max": 100
          }
        }
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          ["LauraDB/Application", "RequestCount", {"stat": "Sum"}],
          [".", "ErrorCount", {"stat": "Sum"}]
        ],
        "period": 60,
        "stat": "Sum",
        "region": "us-east-1",
        "title": "Request Metrics",
        "yAxis": {
          "left": {
            "min": 0
          }
        }
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          ["LauraDB/Database", "DocumentCount", {"stat": "Average"}],
          [".", "CollectionCount", {"stat": "Average"}]
        ],
        "period": 300,
        "stat": "Average",
        "region": "us-east-1",
        "title": "Database Statistics"
      }
    },
    {
      "type": "log",
      "properties": {
        "query": "SOURCE '/aws/lauradb/errors'\n| filter @message like /ERROR/\n| stats count() as ErrorCount by bin(5m)",
        "region": "us-east-1",
        "title": "Error Rate (5m)",
        "stacked": false
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          ["AWS/EC2", "DiskReadBytes", {"stat": "Sum"}],
          [".", "DiskWriteBytes", {"stat": "Sum"}]
        ],
        "period": 300,
        "stat": "Sum",
        "region": "us-east-1",
        "title": "Disk I/O",
        "yAxis": {
          "left": {
            "label": "Bytes"
          }
        }
      }
    },
    {
      "type": "metric",
      "properties": {
        "metrics": [
          ["LauraDB/Database", "CacheHitRatio", {"stat": "Average"}],
          [".", "BufferPoolUsage", {"stat": "Average"}]
        ],
        "period": 300,
        "stat": "Average",
        "region": "us-east-1",
        "title": "Cache Performance",
        "yAxis": {
          "left": {
            "min": 0,
            "max": 100
          }
        }
      }
    }
  ]
}
EOF

aws cloudwatch put-dashboard \
  --dashboard-name LauraDB-Monitoring \
  --dashboard-body file://dashboard.json
```

View dashboard:
```
https://console.aws.amazon.com/cloudwatch/home?region=us-east-1#dashboards:name=LauraDB-Monitoring
```

## Container Insights

### For ECS Deployments

Enable Container Insights:

```bash
aws ecs update-cluster-settings \
  --cluster laura-db-fargate \
  --settings name=containerInsights,value=enabled
```

View Container Insights:
```
https://console.aws.amazon.com/cloudwatch/home#container-insights:performance/ECS
```

### For EKS Deployments

Install CloudWatch agent as DaemonSet:

```bash
kubectl apply -f https://raw.githubusercontent.com/aws-samples/amazon-cloudwatch-container-insights/latest/k8s-deployment-manifest-templates/deployment-mode/daemonset/container-insights-monitoring/quickstart/cwagent-fluentd-quickstart.yaml
```

## Performance Insights

### Query Performance Monitoring

Create custom metric for slow queries:

```bash
# Parse slow query log
SLOW_QUERY_COUNT=$(grep "slow query" /var/log/laura-db/server.log | wc -l)

aws cloudwatch put-metric-data \
  --namespace LauraDB/Performance \
  --metric-name SlowQueryCount \
  --value $SLOW_QUERY_COUNT \
  --unit Count
```

### Transaction Monitoring

```bash
# Monitor transaction rate
TXN_RATE=$(curl -s http://localhost:8080/_stats | jq -r '.transactions.rate')

aws cloudwatch put-metric-data \
  --namespace LauraDB/Performance \
  --metric-name TransactionRate \
  --value $TXN_RATE \
  --unit "Count/Second"
```

## Cost Optimization

### CloudWatch Costs

| Component | Pricing | Optimization |
|-----------|---------|--------------|
| Metrics | $0.30 per metric/month | Use metric filters, aggregate similar metrics |
| Custom Metrics | $0.30 per metric/month | Limit custom metrics to essentials |
| API Requests | $0.01 per 1000 requests | Batch metric publishing |
| Log Ingestion | $0.50 per GB | Set appropriate retention, filter unnecessary logs |
| Log Storage | $0.03 per GB/month | Use lifecycle policies |
| Dashboards | $3 per dashboard/month | Consolidate dashboards |
| Alarms | $0.10 per alarm/month | Use composite alarms |

### Cost-Saving Tips

1. **Aggregate metrics**: Publish multiple metrics in single API call
2. **Filter logs**: Only send relevant log entries
3. **Set retention policies**: Delete old logs automatically
4. **Use metric math**: Calculate derived metrics instead of publishing
5. **Batch operations**: Use `put-metric-data` with multiple data points

### Example: Batch Metric Publishing

```bash
aws cloudwatch put-metric-data \
  --namespace LauraDB/Database \
  --metric-data \
    '[
      {"MetricName": "CollectionCount", "Value": 10, "Unit": "Count"},
      {"MetricName": "DocumentCount", "Value": 1000, "Unit": "Count"},
      {"MetricName": "IndexCount", "Value": 25, "Unit": "Count"}
    ]'
```

## Troubleshooting

### Metrics Not Appearing

```bash
# Check CloudWatch agent status
sudo systemctl status amazon-cloudwatch-agent

# View agent logs
sudo tail -f /opt/aws/amazon-cloudwatch-agent/logs/amazon-cloudwatch-agent.log

# Verify IAM permissions
aws iam get-role-policy \
  --role-name LauraDB-EC2-Role \
  --policy-name CloudWatch-Policy

# Test metric publishing
aws cloudwatch put-metric-data \
  --namespace Test \
  --metric-name TestMetric \
  --value 1
```

### Logs Not Streaming

```bash
# Check log group exists
aws logs describe-log-groups --log-group-name-prefix /aws/lauradb

# Check log streams
aws logs describe-log-streams \
  --log-group-name /aws/lauradb/server \
  --order-by LastEventTime \
  --descending

# Verify file permissions
ls -la /var/log/laura-db/

# Check CloudWatch agent config
cat /opt/aws/amazon-cloudwatch-agent/etc/config.json
```

### Alarms Not Triggering

```bash
# Check alarm state
aws cloudwatch describe-alarms --alarm-names LauraDB-HighCPU

# View alarm history
aws cloudwatch describe-alarm-history \
  --alarm-name LauraDB-HighCPU \
  --max-records 10

# Test alarm
aws cloudwatch set-alarm-state \
  --alarm-name LauraDB-HighCPU \
  --state-value ALARM \
  --state-reason "Testing alarm"
```

## Best Practices

1. ✅ **Enable detailed monitoring** for production instances
2. ✅ **Use metric filters** to create metrics from logs
3. ✅ **Set appropriate alarm thresholds** based on baseline
4. ✅ **Create composite alarms** to reduce noise
5. ✅ **Use anomaly detection** for dynamic thresholds
6. ✅ **Implement dashboard for each team** (ops, dev, business)
7. ✅ **Tag all resources** for cost allocation
8. ✅ **Set log retention policies** to manage costs
9. ✅ **Use CloudWatch Insights** for log analysis
10. ✅ **Monitor your monitoring** (check agent health)

## Next Steps

- [EC2 Deployment Guide](./ec2-deployment.md)
- [ECS Deployment Guide](./ecs-deployment.md)
- [EKS Deployment Guide](./eks-deployment.md)
- [S3 Backup Integration](./s3-backup-integration.md)
- [RDS Comparison](./rds-comparison.md)

## Additional Resources

- [CloudWatch Documentation](https://docs.aws.amazon.com/cloudwatch/)
- [CloudWatch Agent Documentation](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/Install-CloudWatch-Agent.html)
- [CloudWatch Logs Insights](https://docs.aws.amazon.com/AmazonCloudWatch/latest/logs/AnalyzingLogData.html)
- [Container Insights](https://docs.aws.amazon.com/AmazonCloudWatch/latest/monitoring/ContainerInsights.html)
- [CloudWatch Pricing](https://aws.amazon.com/cloudwatch/pricing/)
