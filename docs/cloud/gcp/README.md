# LauraDB Google Cloud Platform Deployment Guides

Comprehensive guides for deploying and operating LauraDB on Google Cloud Platform (GCP).

## Available Guides

### Deployment Options

1. **[GCE Deployment](./gce-deployment.md)**
   - Single instance setup for development
   - Multi-instance with load balancing
   - Managed Instance Groups with auto-scaling
   - Storage configuration (Persistent Disks, Filestore)
   - Complete step-by-step instructions

2. **[GKE Deployment](./gke-deployment.md)**
   - Kubernetes deployment on GKE
   - Autopilot and Standard cluster modes
   - Helm chart installation
   - Storage with Persistent Disks and Filestore
   - Load balancing and ingress
   - Auto-scaling and monitoring

### Operations & Management

3. **[Cloud Storage Backup Integration](./cloud-storage-backup.md)**
   - Automated backup strategies
   - Full and incremental backups
   - Restore procedures
   - Lifecycle management
   - Cross-region replication
   - Cost optimization

4. **[Cloud Monitoring Setup](./cloud-monitoring.md)**
   - Metrics collection and analysis
   - Log aggregation with Cloud Logging
   - Alerting policies and notification channels
   - Custom dashboards
   - Uptime checks
   - Error reporting

## Quick Start

### Choose Your Deployment Method

#### For Development/Testing
**Recommended: GCE Single Instance**
- Fastest setup: ~10 minutes
- Lowest cost: ~$25/month
- Full control and easy debugging

```bash
# See gce-deployment.md for detailed instructions
```

#### For Production (Small to Medium)
**Recommended: GCE Managed Instance Group**
- Managed infrastructure
- Auto-scaling built-in
- Cost: ~$150-250/month
- Balance of control and automation

```bash
# See gce-deployment.md for detailed instructions
```

#### For Production (Large Scale/Enterprise)
**Recommended: GKE Autopilot**
- Fully managed Kubernetes
- Zero node management
- Maximum flexibility
- Cost: ~$200-400/month

```bash
# See gke-deployment.md for detailed instructions
```

## Deployment Comparison

| Factor | GCE | GCE MIG | GKE Standard | GKE Autopilot |
|--------|-----|---------|--------------|---------------|
| **Setup Time** | 10-15 min | 20-30 min | 30-45 min | 15-25 min |
| **Complexity** | Low | Medium | High | Low |
| **Management** | Manual | Semi-automated | Manual nodes | Fully managed |
| **Cost (small)** | $ | $$ | $$$ | $$ |
| **Cost (large)** | $ | $$ | $$ | $$$ |
| **Scaling** | Manual | Auto (VM-based) | Auto (pod-based) | Auto (pod-based) |
| **Flexibility** | High | High | Very High | Medium |
| **Best For** | Dev/Test | Production VMs | Complex apps | Simple apps |

## Architecture Patterns

### Pattern 1: Single Instance (Development)

```
┌─────────────────┐
│  External IP    │
└────────┬────────┘
         │
    ┌────▼─────┐
    │   GCE    │
    │ LauraDB  │
    │   +PD    │
    └──────────┘
```

**Use Case**: Development, testing, demos
**Cost**: ~$25-40/month
**Availability**: Single point of failure

### Pattern 2: Multi-Zone (Production)

```
┌───────────────────────────┐
│  Cloud Load Balancer       │
└──────┬────────────┬────────┘
       │            │
   ┌───▼──┐     ┌──▼───┐
   │Zone A│     │Zone B│
   │ GCE  │     │ GCE  │
   │ +PD  │     │ +PD  │
   └───┬──┘     └──┬───┘
       │           │
       └─────┬─────┘
         ┌───▼───┐
         │Filestore│
         └───────┘
```

**Use Case**: Production applications
**Cost**: ~$180-300/month
**Availability**: 99.9%+

### Pattern 3: GKE Multi-Region (High Availability)

```
┌──────────────┐         ┌──────────────┐
│ us-central1  │         │  us-east1    │
│              │         │              │
│  ┌────────┐  │         │  ┌────────┐  │
│  │  GKE   │  │◄────────┤  │  GKE   │  │
│  │Cluster │  │  Cloud  │  │Cluster │  │
│  └────────┘  │  Storage│  └────────┘  │
└──────────────┘    Sync  └──────────────┘
```

**Use Case**: Mission-critical, global applications
**Cost**: ~$500-800/month
**Availability**: 99.99%+

## Cost Estimates

### Monthly Cost Breakdown

#### Small Deployment (Dev/Test)
- **GCE**: e2-medium × 1 = $24.27
- **PD**: 50GB standard = $2.00
- **Network**: ~$5
- **Monitoring**: Free tier
- **Backups (Cloud Storage)**: $2
- **Total**: ~$33/month

#### Medium Deployment (Production)
- **GCE MIG**: n2-standard-2 × 2 = $132
- **Load Balancer**: $18
- **PD**: 200GB balanced × 2 = $40
- **Filestore**: 1TB Basic = $200
- **Network**: ~$20
- **Backups (Cloud Storage)**: $10
- **Monitoring**: $10
- **Total**: ~$430/month

#### Large Deployment (Enterprise - GKE)
- **GKE Autopilot**: ~$150 (compute)
- **Load Balancer**: $18
- **PD**: 500GB × 3 = $150
- **Filestore**: 2TB Basic = $400
- **Network**: ~$50
- **Backups**: $30
- **Monitoring**: $30
- **Total**: ~$828/month

*Prices based on us-central1, standard pricing, as of 2025*

### GCP vs AWS Cost Comparison

| Component | GCP | AWS | Winner |
|-----------|-----|-----|--------|
| Compute (e2-medium/t3.medium) | $24/month | $30/month | GCP |
| Storage (50GB SSD) | $8.50/month | $4.25/month | AWS |
| Load Balancer | $18/month | $22/month | GCP |
| Managed K8s | $73/month | $73/month | Tie |
| Data Transfer (100GB out) | $12/month | $9/month | AWS |

**Overall**: GCP is ~10% cheaper for compute, AWS is ~15% cheaper for storage.

## Getting Started

### Step 1: Setup GCP Account

1. **Create GCP Account**: https://console.cloud.google.com
2. **Create Project**:
   ```bash
   gcloud projects create laura-db-prod --name="LauraDB Production"
   gcloud config set project laura-db-prod
   ```
3. **Enable Billing**: Link billing account in console
4. **Set Default Region**:
   ```bash
   gcloud config set compute/region us-central1
   gcloud config set compute/zone us-central1-a
   ```

### Step 2: Install Tools

```bash
# Install gcloud SDK
curl https://sdk.cloud.google.com | bash
exec -l $SHELL

# Install kubectl
gcloud components install kubectl

# Install Helm (for GKE)
curl https://raw.githubusercontent.com/helm/helm/main/scripts/get-helm-3 | bash

# Authenticate
gcloud auth login
```

### Step 3: Choose Deployment Method

Navigate to the specific guide:
- [GCE Deployment Guide](./gce-deployment.md) - VM-based deployment
- [GKE Deployment Guide](./gke-deployment.md) - Kubernetes deployment

### Step 4: Configure Monitoring

Set up Cloud Monitoring:
- Follow [Cloud Monitoring Guide](./cloud-monitoring.md)
- Create dashboards and alerts
- Configure log aggregation

### Step 5: Set Up Backups

Configure automated backups:
- Follow [Cloud Storage Backup Guide](./cloud-storage-backup.md)
- Test restore procedures
- Set up lifecycle policies

## Best Practices

### Security

1. ✅ **Use service accounts** with minimal permissions
2. ✅ **Enable Shielded VMs** for additional security
3. ✅ **Use VPC Service Controls** for data perimeter
4. ✅ **Enable encryption at rest** (default in GCP)
5. ✅ **Use Secret Manager** for credentials
6. ✅ **Implement firewall rules** restrictively
7. ✅ **Enable Binary Authorization** for GKE
8. ✅ **Use Workload Identity** for GKE service access

### Reliability

1. ✅ **Deploy across multiple zones** for HA
2. ✅ **Use Managed Instance Groups** for auto-recovery
3. ✅ **Implement health checks** on all instances
4. ✅ **Test disaster recovery** procedures regularly
5. ✅ **Use Cloud Load Balancing** for traffic distribution
6. ✅ **Enable auto-scaling** for capacity management
7. ✅ **Monitor with Cloud Monitoring** proactively

### Performance

1. ✅ **Use appropriate machine types** (n2 for general, c2 for compute)
2. ✅ **Use SSD Persistent Disks** for better IOPS
3. ✅ **Enable Cloud CDN** for static content
4. ✅ **Use Regional resources** to reduce latency
5. ✅ **Implement caching** at multiple layers
6. ✅ **Optimize database configuration** (buffer, cache)
7. ✅ **Use Cloud Profiler** for performance analysis

### Cost Optimization

1. ✅ **Use Committed Use Discounts** (up to 55% savings)
2. ✅ **Leverage Preemptible VMs** for dev/test (up to 80% savings)
3. ✅ **Right-size instances** based on utilization
4. ✅ **Use Sustained Use Discounts** (automatic)
5. ✅ **Implement lifecycle policies** for old backups
6. ✅ **Delete unused resources** regularly
7. ✅ **Use GCP Pricing Calculator** for estimates
8. ✅ **Set up billing alerts** to avoid surprises
9. ✅ **Use Spot VMs in GKE** for fault-tolerant workloads

## GCP-Specific Advantages

### Why Choose GCP for LauraDB?

1. **Better Pricing**: Per-second billing (vs AWS per-minute)
2. **Sustained Use Discounts**: Automatic discounts (no reservation needed)
3. **Live Migration**: Zero-downtime VM maintenance
4. **Custom Machine Types**: Optimize CPU/memory ratio
5. **Global Network**: Fast inter-region connectivity
6. **Integrated Tools**: Cloud Shell, built-in editors
7. **BigQuery Integration**: Easy log analysis
8. **Anthos**: Hybrid/multi-cloud if needed

### GCP vs AWS Feature Comparison

| Feature | GCP | AWS | Winner |
|---------|-----|-----|--------|
| VM Migration | Live Migration | Scheduled downtime | GCP |
| Billing | Per-second | Per-minute | GCP |
| Custom VMs | Yes (any CPU/RAM) | Limited | GCP |
| Global LB | Built-in | Requires setup | GCP |
| Kubernetes | GKE (simpler) | EKS | GCP |
| Storage Speed | pd-extreme (very fast) | io2 Block Express | Tie |
| Network Egress | $0.12/GB | $0.09/GB | AWS |

## Support and Resources

### Documentation
- [LauraDB Main Documentation](../../README.md)
- [GCP Documentation](https://cloud.google.com/docs)
- [GCP Architecture Center](https://cloud.google.com/architecture)

### Tools
- [GCP Pricing Calculator](https://cloud.google.com/products/calculator)
- [gcloud CLI Reference](https://cloud.google.com/sdk/gcloud/reference)
- [GKE Documentation](https://cloud.google.com/kubernetes-engine/docs)

### Community
- [LauraDB GitHub Issues](https://github.com/mnohosten/laura-db/issues)
- [GCP Community](https://www.googlecloudcommunity.com/)
- [Stack Overflow - GCP](https://stackoverflow.com/questions/tagged/google-cloud-platform)

## Troubleshooting

### Common Issues

#### Issue: "Quota exceeded" error
**Solution**: Request quota increase in GCP Console → IAM & Admin → Quotas

#### Issue: Cannot connect to instance
**Solution**: Check firewall rules, verify external IP, ensure service is running

#### Issue: High costs
**Solution**: Review billing reports, check for idle resources, use Recommender

#### Issue: Slow performance
**Solution**: Check machine type, disk type, network configuration, enable monitoring

#### Issue: Backup failures
**Solution**: Verify service account permissions, check Cloud Storage bucket policy

For detailed troubleshooting, see individual guide troubleshooting sections.

## Migration Guides

### From AWS to GCP

1. **Export data** from AWS using backup procedures
2. **Transfer to GCS** using Storage Transfer Service
3. **Deploy LauraDB** on GCP using appropriate guide
4. **Import data** from GCS
5. **Update DNS** to point to new GCP deployment
6. **Verify** all functionality
7. **Decommission** AWS resources

### From On-Premises to GCP

1. **Backup** on-premises LauraDB data
2. **Upload to GCS** using gsutil
3. **Deploy LauraDB** on GCE or GKE
4. **Restore data** from GCS
5. **Test thoroughly** before cutover
6. **Update application** connection strings
7. **Monitor** for issues

### From Other GCP Services

If migrating from Cloud SQL or Firestore:

1. **Export data** in compatible format
2. **Transform** to LauraDB document format
3. **Import** using bulk API
4. **Verify** data integrity
5. **Update** application code
6. **Test** thoroughly

## Best Practices Checklist

### Before Going to Production

- [ ] Enable Cloud Monitoring and Logging
- [ ] Set up automated backups to Cloud Storage
- [ ] Configure alerting policies
- [ ] Implement health checks
- [ ] Enable auto-scaling
- [ ] Use multiple zones for HA
- [ ] Set up Cloud Load Balancer
- [ ] Enable encryption at rest
- [ ] Use Secret Manager for credentials
- [ ] Document runbooks
- [ ] Test disaster recovery
- [ ] Set up billing alerts
- [ ] Review security settings
- [ ] Enable Cloud Armor (if using HTTPS LB)
- [ ] Set up uptime checks

### Ongoing Operations

- [ ] Monitor dashboards daily
- [ ] Review logs for errors weekly
- [ ] Test backups monthly
- [ ] Review costs monthly
- [ ] Apply security patches regularly
- [ ] Update documentation
- [ ] Conduct DR drills quarterly
- [ ] Review and optimize performance
- [ ] Clean up unused resources
- [ ] Update runbooks

## License

LauraDB and all documentation are available under the same license as the main project.

## Contributing

Contributions to improve these guides are welcome! Please submit issues or pull requests to the [LauraDB repository](https://github.com/mnohosten/laura-db).

---

## Quick Links

- **Deployment**: [GCE](./gce-deployment.md) | [GKE](./gke-deployment.md)
- **Operations**: [Backups](./cloud-storage-backup.md) | [Monitoring](./cloud-monitoring.md)
- **GCP Console**: https://console.cloud.google.com
- **Status Page**: https://status.cloud.google.com
- **Support**: https://cloud.google.com/support
