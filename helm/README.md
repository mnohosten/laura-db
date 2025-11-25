# LauraDB Helm Charts

This directory contains Helm charts for deploying LauraDB on Kubernetes.

## Available Charts

### laura-db

The main Helm chart for deploying LauraDB. See [laura-db/README.md](./laura-db/README.md) for detailed documentation.

## Quick Start

### Prerequisites

- Kubernetes 1.19+
- Helm 3.0+

### Installation

```bash
# Install with default values
helm install my-laura-db ./helm/laura-db

# Install with custom values
helm install my-laura-db ./helm/laura-db -f my-values.yaml

# Install in a specific namespace
helm install my-laura-db ./helm/laura-db --namespace laura-db --create-namespace
```

## Validating the Chart

Before installation, you can validate and test the chart:

```bash
# Lint the chart
helm lint ./helm/laura-db

# Dry run to see what would be deployed
helm install my-laura-db ./helm/laura-db --dry-run --debug

# Generate templates to review
helm template my-laura-db ./helm/laura-db > output.yaml
```

## Packaging the Chart

To package the chart for distribution:

```bash
# Package the chart
helm package ./helm/laura-db

# This creates a file like: laura-db-0.1.0.tgz
```

## Chart Repository (Future)

In the future, this chart may be published to a chart repository for easier installation:

```bash
# Add the repository
helm repo add laura-db https://charts.laura-db.io

# Install from repository
helm install my-laura-db laura-db/laura-db
```

## Development

### Chart Structure

```
laura-db/
├── Chart.yaml              # Chart metadata
├── values.yaml             # Default configuration values
├── README.md               # Chart documentation
├── .helmignore            # Files to ignore when packaging
└── templates/             # Kubernetes manifest templates
    ├── _helpers.tpl       # Template helpers
    ├── NOTES.txt          # Post-installation notes
    ├── configmap.yaml     # ConfigMap resource
    ├── secrets.yaml       # Secret resources
    ├── serviceaccount.yaml # ServiceAccount resource
    ├── service.yaml       # Service resources
    ├── statefulset.yaml   # StatefulSet resource
    ├── ingress.yaml       # Ingress resource
    ├── hpa.yaml           # HorizontalPodAutoscaler
    ├── poddisruptionbudget.yaml # PodDisruptionBudget
    ├── networkpolicy.yaml # NetworkPolicy
    ├── servicemonitor.yaml # Prometheus ServiceMonitor
    ├── prometheusrule.yaml # Prometheus rules
    └── tests/
        └── test-connection.yaml # Helm test
```

### Testing Changes

After making changes to the chart:

1. **Lint the chart**:
   ```bash
   helm lint ./helm/laura-db
   ```

2. **Test template rendering**:
   ```bash
   helm template test-release ./helm/laura-db
   ```

3. **Install in a test namespace**:
   ```bash
   helm install test-release ./helm/laura-db \
     --namespace test \
     --create-namespace \
     --dry-run --debug
   ```

4. **Run Helm tests** (after installation):
   ```bash
   helm test test-release -n test
   ```

### Best Practices

When modifying the chart:

- Always update the chart version in `Chart.yaml` for releases
- Document any new parameters in `values.yaml` with comments
- Update the README.md with new configuration options
- Test with different values configurations
- Validate against different Kubernetes versions
- Follow [Helm chart best practices](https://helm.sh/docs/chart_best_practices/)

## Common Use Cases

### Development Environment

```bash
helm install laura-dev ./helm/laura-db \
  --set replicaCount=1 \
  --set resources.requests.memory=128Mi \
  --set persistence.size=5Gi
```

### Production Environment

```bash
helm install laura-prod ./helm/laura-db \
  --set replicaCount=3 \
  --set resources.requests.memory=512Mi \
  --set resources.limits.memory=2Gi \
  --set persistence.size=50Gi \
  --set podDisruptionBudget.enabled=true
```

### With Monitoring

```bash
helm install laura-prod ./helm/laura-db \
  --set metrics.enabled=true \
  --set metrics.serviceMonitor.enabled=true
```

## Resources

- [Helm Documentation](https://helm.sh/docs/)
- [LauraDB Documentation](https://github.com/mnohosten/laura-db/tree/main/docs)
- [Kubernetes Documentation](https://kubernetes.io/docs/)

## Support

For issues specific to the Helm chart, please open an issue at:
https://github.com/mnohosten/laura-db/issues
