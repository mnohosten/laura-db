# LauraDB Monitoring with Azure Monitor

Complete guide for implementing comprehensive monitoring, logging, and alerting for LauraDB using Azure Monitor and related services.

## Table of Contents

- [Overview](#overview)
- [Prerequisites](#prerequisites)
- [Azure Monitor Components](#azure-monitor-components)
- [VM Monitoring Setup](#vm-monitoring-setup)
- [AKS Monitoring Setup](#aks-monitoring-setup)
- [Custom Metrics](#custom-metrics)
- [Log Analytics](#log-analytics)
- [Alerting](#alerting)
- [Dashboards](#dashboards)
- [Application Insights](#application-insights)
- [Network Monitoring](#network-monitoring)
- [Cost Monitoring](#cost-monitoring)
- [Best Practices](#best-practices)

## Overview

Azure Monitor provides comprehensive monitoring for LauraDB deployments with metrics collection, log aggregation, alerting, and visualization capabilities.

### Key Features

- **Metrics**: Collect and analyze time-series data
- **Logs**: Aggregate and query logs using Kusto Query Language (KQL)
- **Alerts**: Proactive notifications based on metrics and logs
- **Dashboards**: Customizable visualizations
- **Application Insights**: Application performance monitoring (APM)
- **VM Insights**: Deep VM and container monitoring
- **Network Watcher**: Network diagnostics and monitoring
- **Cost Management**: Track and optimize spending

### Monitoring Architecture

```
┌──────────────────────────────────────────┐
│         LauraDB Application              │
│  ┌────────────┐      ┌────────────┐     │
│  │    VM      │      │    AKS     │     │
│  │  Metrics   │      │   Metrics  │     │
│  └─────┬──────┘      └──────┬─────┘     │
│        │                    │           │
│        │    Azure Monitor   │           │
│        ▼        Agent        ▼           │
└────────┼─────────────────────┼───────────┘
         │                     │
         ▼                     ▼
┌────────────────────────────────────────┐
│       Log Analytics Workspace          │
│  ┌──────────────────────────────────┐ │
│  │  Metrics  │  Logs  │  Traces     │ │
│  └──────────────────────────────────┘ │
└───────────┬────────────────────────────┘
            │
     ┌──────┴──────┐
     │             │
     ▼             ▼
┌─────────┐  ┌──────────┐
│ Alerts  │  │Dashboards│
└─────────┘  └──────────┘
```

## Prerequisites

### Tools Required

```bash
# Install Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash

# Verify
az --version

# Login
az login

# Set subscription
az account set --subscription "Your Subscription Name"
```

### Variables

```bash
RESOURCE_GROUP="laura-db-rg"
LOCATION="eastus"
WORKSPACE_NAME="laura-db-logs"
VM_NAME="laura-db-vm"
AKS_CLUSTER="laura-db-aks"
```

## Azure Monitor Components

### 1. Create Log Analytics Workspace

Central repository for all logs and metrics.

```bash
# Create workspace
az monitor log-analytics workspace create \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --location $LOCATION \
  --sku PerGB2018 \
  --retention-time 90 \
  --tags application=laura-db environment=production

# Get workspace ID and key
WORKSPACE_ID=$(az monitor log-analytics workspace show \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --query customerId -o tsv)

WORKSPACE_KEY=$(az monitor log-analytics workspace get-shared-keys \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --query primarySharedKey -o tsv)

echo "Workspace ID: $WORKSPACE_ID"
echo "Workspace Key: $WORKSPACE_KEY"
```

### 2. Create Action Groups

Action groups define notification channels for alerts.

```bash
# Create action group with email
az monitor action-group create \
  --name laura-db-alerts \
  --resource-group $RESOURCE_GROUP \
  --short-name laura-ag \
  --email-receiver name=admin email=admin@example.com \
  --email-receiver name=oncall email=oncall@example.com

# Add SMS notification
az monitor action-group update \
  --name laura-db-alerts \
  --resource-group $RESOURCE_GROUP \
  --add-sms-receiver name=oncall country-code=1 phone-number=5551234567

# Add webhook (for Slack, PagerDuty, etc.)
az monitor action-group update \
  --name laura-db-alerts \
  --resource-group $RESOURCE_GROUP \
  --add-webhook-receiver name=slack service-uri=https://hooks.slack.com/services/YOUR/WEBHOOK/URL

# Add Azure Function
az monitor action-group update \
  --name laura-db-alerts \
  --resource-group $RESOURCE_GROUP \
  --add-azure-function-receiver name=custom-handler \
    function-resource-id=/subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Web/sites/your-function-app/functions/alert-handler
```

## VM Monitoring Setup

### 1. Install Azure Monitor Agent

```bash
# Install Azure Monitor Agent (AMA) on Linux VM
az vm extension set \
  --resource-group $RESOURCE_GROUP \
  --vm-name $VM_NAME \
  --name AzureMonitorLinuxAgent \
  --publisher Microsoft.Azure.Monitor \
  --enable-auto-upgrade true

# Verify installation
az vm extension show \
  --resource-group $RESOURCE_GROUP \
  --vm-name $VM_NAME \
  --name AzureMonitorLinuxAgent
```

### 2. Enable VM Insights

VM Insights provides comprehensive monitoring for VMs.

```bash
# Enable VM Insights
az vm extension set \
  --resource-group $RESOURCE_GROUP \
  --vm-name $VM_NAME \
  --name DependencyAgentLinux \
  --publisher Microsoft.Azure.Monitoring.DependencyAgent \
  --enable-auto-upgrade true

# Associate with Log Analytics workspace
az monitor log-analytics workspace pack enable \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --name "VMInsights"
```

### 3. Configure Data Collection Rules (DCR)

Define what data to collect and where to send it.

```bash
# Create DCR JSON configuration
cat > dcr-config.json <<'EOF'
{
  "location": "eastus",
  "properties": {
    "dataSources": {
      "performanceCounters": [
        {
          "name": "perfCounterDataSource",
          "streams": ["Microsoft-Perf"],
          "samplingFrequencyInSeconds": 60,
          "counterSpecifiers": [
            "\\Processor(_Total)\\% Processor Time",
            "\\Memory\\Available MBytes",
            "\\Memory\\% Used Memory",
            "\\Disk(_Total)\\% Disk Time",
            "\\Disk(_Total)\\Disk Read Bytes/sec",
            "\\Disk(_Total)\\Disk Write Bytes/sec",
            "\\Network Interface(*)\\Bytes Sent/sec",
            "\\Network Interface(*)\\Bytes Received/sec"
          ]
        }
      ],
      "syslog": [
        {
          "name": "syslogDataSource",
          "streams": ["Microsoft-Syslog"],
          "facilityNames": ["auth", "authpriv", "cron", "daemon", "kern", "syslog", "user"],
          "logLevels": ["Debug", "Info", "Notice", "Warning", "Error", "Critical", "Alert", "Emergency"]
        }
      ]
    },
    "destinations": {
      "logAnalytics": [
        {
          "workspaceResourceId": "/subscriptions/YOUR_SUB_ID/resourceGroups/laura-db-rg/providers/Microsoft.OperationalInsights/workspaces/laura-db-logs",
          "name": "laWorkspace"
        }
      ]
    },
    "dataFlows": [
      {
        "streams": ["Microsoft-Perf"],
        "destinations": ["laWorkspace"]
      },
      {
        "streams": ["Microsoft-Syslog"],
        "destinations": ["laWorkspace"]
      }
    ]
  }
}
EOF

# Create DCR
az monitor data-collection rule create \
  --resource-group $RESOURCE_GROUP \
  --name laura-db-dcr \
  --location $LOCATION \
  --rule-file dcr-config.json

# Associate DCR with VM
DCR_ID=$(az monitor data-collection rule show \
  --resource-group $RESOURCE_GROUP \
  --name laura-db-dcr \
  --query id -o tsv)

az monitor data-collection rule association create \
  --name laura-db-dcr-association \
  --rule-id $DCR_ID \
  --resource /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME
```

### 4. Configure Custom Log Collection

Collect LauraDB application logs.

```bash
# Create custom log table
cat > custom-log-table.json <<'EOF'
{
  "properties": {
    "schema": {
      "name": "LauraDBLogs_CL",
      "columns": [
        {
          "name": "TimeGenerated",
          "type": "datetime"
        },
        {
          "name": "Level",
          "type": "string"
        },
        {
          "name": "Message",
          "type": "string"
        },
        {
          "name": "InstanceId",
          "type": "string"
        },
        {
          "name": "ErrorCode",
          "type": "int"
        }
      ]
    }
  }
}
EOF

# Add custom log collection to DCR
# Update dcr-config.json to include:
cat > dcr-custom-logs.json <<'EOF'
{
  "location": "eastus",
  "properties": {
    "dataSources": {
      "logFiles": [
        {
          "name": "LauraDBLogFiles",
          "streams": ["Custom-LauraDBLogs_CL"],
          "filePatterns": ["/var/log/laura-db/*.log"],
          "format": "text",
          "settings": {
            "text": {
              "recordStartTimestampFormat": "ISO 8601"
            }
          }
        }
      ]
    },
    "destinations": {
      "logAnalytics": [
        {
          "workspaceResourceId": "/subscriptions/YOUR_SUB_ID/resourceGroups/laura-db-rg/providers/Microsoft.OperationalInsights/workspaces/laura-db-logs",
          "name": "laWorkspace"
        }
      ]
    },
    "dataFlows": [
      {
        "streams": ["Custom-LauraDBLogs_CL"],
        "destinations": ["laWorkspace"]
      }
    ]
  }
}
EOF

az monitor data-collection rule update \
  --resource-group $RESOURCE_GROUP \
  --name laura-db-dcr \
  --rule-file dcr-custom-logs.json
```

## AKS Monitoring Setup

### 1. Enable Container Insights

Already enabled during AKS creation with `--enable-addons monitoring`.

```bash
# If not enabled, enable it now
az aks enable-addons \
  --resource-group $RESOURCE_GROUP \
  --name $AKS_CLUSTER \
  --addons monitoring \
  --workspace-resource-id /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.OperationalInsights/workspaces/$WORKSPACE_NAME

# Verify
az aks show \
  --resource-group $RESOURCE_GROUP \
  --name $AKS_CLUSTER \
  --query addonProfiles.omsagent
```

### 2. Configure Container Insights

```bash
# Create ConfigMap for additional settings
kubectl apply -f - <<EOF
apiVersion: v1
kind: ConfigMap
metadata:
  name: container-azm-ms-agentconfig
  namespace: kube-system
data:
  schema-version: v1
  config-version: ver1
  log-data-collection-settings: |-
    [log_collection_settings]
       [log_collection_settings.stdout]
          enabled = true
          exclude_namespaces = ["kube-system", "kube-public"]
       [log_collection_settings.stderr]
          enabled = true
          exclude_namespaces = ["kube-system", "kube-public"]
       [log_collection_settings.env_var]
          enabled = true
  prometheus-data-collection-settings: |-
    [prometheus_data_collection_settings.cluster]
        interval = "1m"
        monitor_kubernetes_pods = true
    [prometheus_data_collection_settings.node]
        interval = "1m"
EOF
```

### 3. Enable Prometheus Metrics

```bash
# Install Prometheus operator (if not using Container Insights Prometheus)
kubectl apply -f https://raw.githubusercontent.com/prometheus-operator/prometheus-operator/main/bundle.yaml

# Create ServiceMonitor for LauraDB
kubectl apply -f - <<EOF
apiVersion: monitoring.coreos.com/v1
kind: ServiceMonitor
metadata:
  name: laura-db-metrics
  namespace: laura-db
  labels:
    app: laura-db
spec:
  selector:
    matchLabels:
      app: laura-db
  endpoints:
    - port: metrics
      interval: 30s
      path: /metrics
EOF
```

## Custom Metrics

### Publish Custom Metrics from LauraDB

#### Method 1: Azure Monitor REST API

```bash
#!/bin/bash
# publish-metrics.sh - Publish custom metrics to Azure Monitor

RESOURCE_ID="/subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME"
METRIC_NAMESPACE="LauraDB"
TIMESTAMP=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# Get access token
ACCESS_TOKEN=$(az account get-access-token --resource=https://monitoring.azure.com/ --query accessToken -o tsv)

# Get LauraDB metrics (example: query count)
QUERY_COUNT=$(curl -s http://localhost:8080/metrics | grep 'query_count' | awk '{print $2}')

# Publish metric
curl -X POST "https://monitoring.azure.com${RESOURCE_ID}/metrics" \
  -H "Authorization: Bearer $ACCESS_TOKEN" \
  -H "Content-Type: application/json" \
  -d @- <<EOF
{
  "time": "$TIMESTAMP",
  "data": {
    "baseData": {
      "metric": "QueryCount",
      "namespace": "$METRIC_NAMESPACE",
      "dimNames": ["Instance"],
      "series": [
        {
          "dimValues": ["$(hostname)"],
          "min": $QUERY_COUNT,
          "max": $QUERY_COUNT,
          "sum": $QUERY_COUNT,
          "count": 1
        }
      ]
    }
  }
}
EOF
```

#### Method 2: Application Insights SDK (Go)

```go
// Add to LauraDB application code
package main

import (
    "github.com/microsoft/ApplicationInsights-Go/appinsights"
    "os"
    "time"
)

var telemetryClient appinsights.TelemetryClient

func init() {
    instrumentationKey := os.Getenv("APPINSIGHTS_INSTRUMENTATIONKEY")
    telemetryClient = appinsights.NewTelemetryClient(instrumentationKey)

    // Enable auto-collection
    appinsights.NewDiagnosticsMessageListener(func(msg string) error {
        log.Printf("[AppInsights] %s\n", msg)
        return nil
    })
}

// Track custom metric
func trackQueryMetric(queryType string, duration time.Duration) {
    metric := appinsights.NewMetricTelemetry("QueryDuration", float64(duration.Milliseconds()))
    metric.Properties["QueryType"] = queryType
    telemetryClient.Track(metric)
}

// Track request
func trackRequest(name string, duration time.Duration, success bool) {
    request := appinsights.NewRequestTelemetry("HTTP", name, duration, "200")
    request.Success = success
    telemetryClient.Track(request)
}

// Track error
func trackError(err error) {
    trace := appinsights.NewTraceTelemetry(err.Error(), appinsights.Error)
    telemetryClient.Track(trace)
}
```

### Create Custom Metrics in Log Analytics

Use KQL queries to derive metrics from logs.

```kusto
// Create custom metric from logs
LauraDBLogs_CL
| where Level == "INFO"
| where Message contains "Query executed"
| summarize QueryCount = count() by bin(TimeGenerated, 5m)
| render timechart
```

## Log Analytics

### Kusto Query Language (KQL) Examples

#### Query VM Performance

```kusto
// CPU usage over time
Perf
| where ObjectName == "Processor" and CounterName == "% Processor Time"
| where Computer contains "laura-db"
| summarize AvgCPU = avg(CounterValue) by bin(TimeGenerated, 5m)
| render timechart

// Memory usage
Perf
| where ObjectName == "Memory" and CounterName == "Available MBytes"
| where Computer contains "laura-db"
| summarize AvgMemory = avg(CounterValue) by bin(TimeGenerated, 5m)
| render timechart

// Disk I/O
Perf
| where ObjectName == "Disk" and CounterName in ("Disk Read Bytes/sec", "Disk Write Bytes/sec")
| where Computer contains "laura-db"
| summarize sum(CounterValue) by CounterName, bin(TimeGenerated, 5m)
| render timechart
```

#### Query AKS Container Logs

```kusto
// View LauraDB container logs
ContainerLog
| where Namespace == "laura-db"
| where ContainerName == "laura-db"
| project TimeGenerated, LogEntry, Computer
| order by TimeGenerated desc
| take 100

// Error logs only
ContainerLog
| where Namespace == "laura-db"
| where LogEntry contains "ERROR"
| project TimeGenerated, LogEntry, Computer
| order by TimeGenerated desc

// Log patterns (find common errors)
ContainerLog
| where Namespace == "laura-db"
| where LogEntry contains "ERROR"
| summarize Count = count() by ErrorPattern = extract("ERROR: (.*?)\\n", 1, LogEntry)
| order by Count desc
```

#### Query Custom LauraDB Logs

```kusto
// Query rate by endpoint
LauraDBLogs_CL
| where Message contains "Request processed"
| extend Endpoint = extract("endpoint=([^ ]+)", 1, Message)
| summarize RequestCount = count() by Endpoint, bin(TimeGenerated, 5m)
| render timechart

// Error rate
LauraDBLogs_CL
| where Level in ("ERROR", "CRITICAL")
| summarize ErrorCount = count() by bin(TimeGenerated, 5m)
| render timechart

// Slow queries (> 1 second)
LauraDBLogs_CL
| where Message contains "Query executed"
| extend Duration = extract("duration=([0-9.]+)ms", 1, Message)
| where todouble(Duration) > 1000
| project TimeGenerated, Message, Duration
| order by Duration desc
```

#### Cross-Resource Queries

```kusto
// Correlate VM metrics with application logs
let errorTimes = LauraDBLogs_CL
| where Level == "ERROR"
| project TimeGenerated, Message;
Perf
| where Computer contains "laura-db"
| where ObjectName == "Processor"
| where CounterName == "% Processor Time"
| join kind=inner (errorTimes) on TimeGenerated
| project TimeGenerated, CPUUsage = CounterValue, ErrorMessage = Message
| order by TimeGenerated desc
```

### Create Log Analytics Solutions

```bash
# Install solutions for enhanced monitoring
az monitor log-analytics solution create \
  --resource-group $RESOURCE_GROUP \
  --workspace $WORKSPACE_NAME \
  --solution-type "ContainerInsights"

az monitor log-analytics solution create \
  --resource-group $RESOURCE_GROUP \
  --workspace $WORKSPACE_NAME \
  --solution-type "VMInsights"
```

## Alerting

### Metric-Based Alerts

#### High CPU Alert

```bash
az monitor metrics alert create \
  --name "LauraDB High CPU" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME \
  --condition "avg Percentage CPU > 80" \
  --window-size 5m \
  --evaluation-frequency 1m \
  --description "CPU usage exceeded 80% for 5 minutes" \
  --severity 2 \
  --action laura-db-alerts
```

#### High Memory Alert

```bash
az monitor metrics alert create \
  --name "LauraDB High Memory" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME \
  --condition "avg Available Memory Bytes < 536870912" \
  --window-size 5m \
  --evaluation-frequency 1m \
  --description "Available memory below 512MB" \
  --severity 2 \
  --action laura-db-alerts
```

#### High Disk Usage Alert

```bash
az monitor metrics alert create \
  --name "LauraDB High Disk Usage" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME \
  --condition "avg Disk Used Percentage > 85" \
  --window-size 15m \
  --evaluation-frequency 5m \
  --description "Disk usage exceeded 85%" \
  --severity 3 \
  --action laura-db-alerts
```

### Log-Based Alerts

#### Error Rate Alert

```bash
az monitor scheduled-query create \
  --name "LauraDB High Error Rate" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.OperationalInsights/workspaces/$WORKSPACE_NAME \
  --condition "count > 10" \
  --condition-query "LauraDBLogs_CL | where Level == 'ERROR' | summarize count()" \
  --description "More than 10 errors in 5 minutes" \
  --evaluation-frequency 5m \
  --window-size 5m \
  --severity 2 \
  --action-groups /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/microsoft.insights/actionGroups/laura-db-alerts
```

#### Slow Query Alert

```bash
az monitor scheduled-query create \
  --name "LauraDB Slow Queries" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.OperationalInsights/workspaces/$WORKSPACE_NAME \
  --condition "count > 5" \
  --condition-query "LauraDBLogs_CL | where Message contains 'Query executed' | extend Duration = extract('duration=([0-9.]+)ms', 1, Message) | where todouble(Duration) > 1000 | summarize count()" \
  --description "More than 5 queries took longer than 1 second" \
  --evaluation-frequency 10m \
  --window-size 10m \
  --severity 3 \
  --action-groups /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/microsoft.insights/actionGroups/laura-db-alerts
```

#### Application Unavailable Alert

```bash
az monitor scheduled-query create \
  --name "LauraDB Application Down" \
  --resource-group $RESOURCE_GROUP \
  --scopes /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.OperationalInsights/workspaces/$WORKSPACE_NAME \
  --condition "count == 0" \
  --condition-query "Heartbeat | where Computer contains 'laura-db' | summarize count()" \
  --description "No heartbeat received from LauraDB in last 5 minutes" \
  --evaluation-frequency 5m \
  --window-size 5m \
  --severity 1 \
  --action-groups /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/microsoft.insights/actionGroups/laura-db-alerts
```

### Smart Detection Alerts

Enable anomaly detection using Azure Monitor's machine learning.

```bash
# Enable smart detection for Application Insights
# This is automatic once Application Insights is configured
# View in Azure Portal: Application Insights → Smart Detection
```

## Dashboards

### Create Custom Dashboard

```bash
# Create dashboard via Azure Portal or ARM template
cat > dashboard-template.json <<'EOF'
{
  "$schema": "https://schema.management.azure.com/schemas/2015-01-01/deploymentTemplate.json#",
  "contentVersion": "1.0.0.0",
  "resources": [
    {
      "type": "Microsoft.Portal/dashboards",
      "apiVersion": "2020-09-01-preview",
      "name": "LauraDB-Monitoring-Dashboard",
      "location": "eastus",
      "properties": {
        "lenses": [
          {
            "order": 0,
            "parts": [
              {
                "position": {"x": 0, "y": 0, "rowSpan": 4, "colSpan": 6},
                "metadata": {
                  "type": "Extension/HubsExtension/PartType/MonitorChartPart",
                  "settings": {
                    "content": {
                      "title": "CPU Usage",
                      "chartType": "Line"
                    }
                  }
                }
              },
              {
                "position": {"x": 6, "y": 0, "rowSpan": 4, "colSpan": 6},
                "metadata": {
                  "type": "Extension/HubsExtension/PartType/MonitorChartPart",
                  "settings": {
                    "content": {
                      "title": "Memory Usage",
                      "chartType": "Line"
                    }
                  }
                }
              }
            ]
          }
        ]
      }
    }
  ]
}
EOF

# Deploy dashboard
az deployment group create \
  --resource-group $RESOURCE_GROUP \
  --template-file dashboard-template.json
```

### Pre-built Dashboards

Access pre-built dashboards in Azure Portal:
- VM Insights → Performance/Map
- Container Insights → Cluster/Nodes/Controllers/Containers
- Log Analytics → Workbooks

### Export Dashboard to Grafana

```bash
# Install Grafana Azure Monitor data source plugin
# Configure in Grafana:
# 1. Add Azure Monitor data source
# 2. Configure authentication (Service Principal or Managed Identity)
# 3. Import dashboard JSON

# Example Grafana dashboard JSON available in:
# ./monitoring/grafana/laura-db-dashboard.json
```

## Application Insights

### 1. Create Application Insights

```bash
az monitor app-insights component create \
  --app laura-db-appinsights \
  --location $LOCATION \
  --resource-group $RESOURCE_GROUP \
  --workspace /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.OperationalInsights/workspaces/$WORKSPACE_NAME

# Get instrumentation key
APPINSIGHTS_KEY=$(az monitor app-insights component show \
  --app laura-db-appinsights \
  --resource-group $RESOURCE_GROUP \
  --query instrumentationKey -o tsv)

echo "Instrumentation Key: $APPINSIGHTS_KEY"
```

### 2. Configure LauraDB with Application Insights

```bash
# Set environment variable for LauraDB
export APPINSIGHTS_INSTRUMENTATIONKEY=$APPINSIGHTS_KEY

# Or add to systemd service
sudo tee -a /etc/systemd/system/laura-db.service <<EOF
Environment="APPINSIGHTS_INSTRUMENTATIONKEY=$APPINSIGHTS_KEY"
EOF

sudo systemctl daemon-reload
sudo systemctl restart laura-db
```

### 3. View Application Insights Data

```bash
# Query Application Insights using KQL
az monitor app-insights query \
  --app laura-db-appinsights \
  --resource-group $RESOURCE_GROUP \
  --analytics-query "requests | where timestamp > ago(1h) | summarize count() by bin(timestamp, 5m) | render timechart"

# View in Azure Portal:
# Application Insights → Overview/Performance/Failures/Metrics
```

## Network Monitoring

### Enable Network Watcher

```bash
# Enable Network Watcher
az network watcher configure \
  --resource-group NetworkWatcherRG \
  --locations $LOCATION \
  --enabled true

# Enable flow logs for NSG
NSG_ID=$(az network nsg show \
  --resource-group $RESOURCE_GROUP \
  --name laura-db-nsg \
  --query id -o tsv)

# Create storage account for flow logs
FLOW_LOG_STORAGE="lauradbnsg$(date +%s)"
az storage account create \
  --name $FLOW_LOG_STORAGE \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --sku Standard_LRS

# Enable NSG flow logs
az network watcher flow-log create \
  --name laura-db-nsg-flow-log \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --nsg $NSG_ID \
  --storage-account $FLOW_LOG_STORAGE \
  --log-version 2 \
  --retention 30 \
  --workspace /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.OperationalInsights/workspaces/$WORKSPACE_NAME \
  --interval 10 \
  --traffic-analytics true
```

### Connection Monitor

```bash
# Create connection monitor
az network watcher connection-monitor create \
  --name laura-db-connectivity \
  --resource-group $RESOURCE_GROUP \
  --location $LOCATION \
  --endpoint-source-resource-id /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME \
  --endpoint-dest-address example.com \
  --test-config-protocol Tcp \
  --test-config-port 443
```

## Cost Monitoring

### Track Monitoring Costs

```bash
# View Log Analytics costs
az monitor log-analytics workspace get-schema \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME

# Query data ingestion
az monitor log-analytics query \
  --workspace $WORKSPACE_ID \
  --analytics-query "Usage | where TimeGenerated > ago(30d) | summarize DataIngested = sum(Quantity) by Solution | order by DataIngested desc"

# Set daily cap
az monitor log-analytics workspace update \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --quota 5  # 5 GB per day

# Create budget alert
az consumption budget create \
  --amount 100 \
  --budget-name laura-db-monitoring-budget \
  --category cost \
  --time-grain monthly \
  --start-date 2025-01-01 \
  --end-date 2026-01-01 \
  --resource-group $RESOURCE_GROUP \
  --notifications "actual_GreaterThan_80_Percent={enabled:true,operator:GreaterThan,threshold:80,contact-emails:['admin@example.com']}"
```

## Best Practices

### 1. Organize Logs and Metrics

- Use consistent naming conventions
- Tag all resources (application, environment, owner)
- Use namespaces for custom metrics
- Structure log messages with consistent formats (JSON recommended)

### 2. Optimize Data Collection

```bash
# Exclude unnecessary logs
# Update DCR to exclude verbose logs
# Reduce sampling frequency for non-critical metrics
# Use diagnostic settings to control what's collected
```

### 3. Set Appropriate Retention

```bash
# Different retention for different data types
az monitor log-analytics workspace update \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --retention-time 30  # 30 days for most logs

# Export old data to blob storage for long-term retention
az monitor log-analytics workspace data-export create \
  --resource-group $RESOURCE_GROUP \
  --workspace-name $WORKSPACE_NAME \
  --name archive-export \
  --destination /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Storage/storageAccounts/lauradbbkp12345 \
  --enable true \
  --tables LauraDBLogs_CL
```

### 4. Alert Fatigue Prevention

- Set appropriate thresholds (not too sensitive)
- Use dynamic thresholds for varying workloads
- Consolidate related alerts
- Use alert processing rules to suppress during maintenance
- Implement escalation policies

### 5. Dashboard Best Practices

- One dashboard per audience (ops, dev, management)
- Use consistent time ranges
- Include both metrics and logs
- Add context with annotations
- Keep dashboards focused (max 12 widgets)

### 6. Security Monitoring

```kusto
// Monitor authentication failures
Syslog
| where Facility == "auth" or Facility == "authpriv"
| where SyslogMessage contains "Failed password"
| summarize FailedLogins = count() by Computer, bin(TimeGenerated, 5m)
| where FailedLogins > 5
| render timechart

// Monitor unauthorized API access
LauraDBLogs_CL
| where Level == "WARNING"
| where Message contains "Unauthorized"
| project TimeGenerated, Message, InstanceId
```

## Troubleshooting

### Agent Not Collecting Data

```bash
# Check agent status
az vm extension show \
  --resource-group $RESOURCE_GROUP \
  --vm-name $VM_NAME \
  --name AzureMonitorLinuxAgent

# View agent logs on VM
sudo journalctl -u azuremonitoragent -f

# Restart agent
sudo systemctl restart azuremonitoragent
```

### No Logs in Log Analytics

```bash
# Verify DCR association
az monitor data-collection rule association list \
  --resource /subscriptions/YOUR_SUB_ID/resourceGroups/$RESOURCE_GROUP/providers/Microsoft.Compute/virtualMachines/$VM_NAME

# Check workspace connection
az monitor log-analytics query \
  --workspace $WORKSPACE_ID \
  --analytics-query "Heartbeat | where Computer contains 'laura-db' | take 10"

# Verify firewall allows outbound to *.ods.opinsights.azure.com:443
```

### High Costs

```bash
# Identify top data sources
az monitor log-analytics query \
  --workspace $WORKSPACE_ID \
  --analytics-query "Usage | where TimeGenerated > ago(7d) | summarize DataGB = sum(Quantity) / 1000 by DataType | order by DataGB desc"

# Review and optimize:
# - Reduce log collection frequency
# - Exclude unnecessary log types
# - Implement sampling for high-volume logs
# - Set daily cap
```

## Summary

Azure Monitor provides comprehensive monitoring for LauraDB with:

- **VM Insights**: Deep VM performance monitoring
- **Container Insights**: Kubernetes cluster and container monitoring
- **Log Analytics**: Centralized log aggregation and analysis
- **Application Insights**: APM with distributed tracing
- **Alerts**: Proactive notifications for issues
- **Dashboards**: Customizable visualizations

### Monthly Cost Estimate

- **Log Analytics**: ~$2.30/GB ingested
- **Application Insights**: ~$2.30/GB ingested
- **Alerts**: ~$0.10 per alert evaluation
- **Typical small deployment**: ~$30-50/month (5-10 GB ingestion)
- **Typical medium deployment**: ~$100-200/month (50-100 GB ingestion)

*Prices based on East US region, standard pricing, as of 2025*

## Next Steps

- Configure all monitoring components
- Create custom dashboards for your team
- Set up alert rules for critical scenarios
- Test alert notifications
- Document runbooks for common alerts
- Review and optimize data collection costs monthly

## References

- [Azure Monitor Documentation](https://docs.microsoft.com/en-us/azure/azure-monitor/)
- [Log Analytics KQL Reference](https://docs.microsoft.com/en-us/azure/data-explorer/kusto/query/)
- [Application Insights Documentation](https://docs.microsoft.com/en-us/azure/azure-monitor/app/app-insights-overview)
- [LauraDB Main Documentation](../../README.md)

---

**Remember**: Effective monitoring is not just about collecting data—it's about deriving actionable insights and responding to issues proactively.
