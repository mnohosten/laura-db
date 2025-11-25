# LauraDB AWS Deployment Guides

Comprehensive guides for deploying and operating LauraDB on Amazon Web Services (AWS).

## Available Guides

### Deployment Options

1. **[EC2 Deployment](./ec2-deployment.md)**
   - Single instance setup for development
   - Multi-instance with load balancing for production
   - Auto scaling configuration
   - Storage and networking setup
   - Complete step-by-step instructions

2. **[ECS/Fargate Deployment](./ecs-deployment.md)**
   - Containerized deployment on ECS
   - Fargate serverless option
   - EC2 launch type for cost optimization
   - Service discovery and load balancing
   - Auto scaling and secrets management

3. **[EKS Deployment](./eks-deployment.md)**
   - Kubernetes deployment on EKS
   - Helm chart installation
   - Storage with EBS and EFS
   - Load balancing with ALB/NLB
   - Cluster autoscaling and monitoring

### Operations & Management

4. **[S3 Backup Integration](./s3-backup-integration.md)**
   - Automated backup strategies
   - Full and incremental backups
   - Restore procedures
   - Lifecycle management
   - Cross-region replication
   - Cost optimization

5. **[CloudWatch Monitoring](./cloudwatch-monitoring.md)**
   - Metrics collection and analysis
   - Log aggregation and insights
   - Alarms and alerts
   - Custom dashboards
   - Container Insights
   - Performance monitoring

### Decision Support

6. **[RDS Alternative Comparison](./rds-comparison.md)**
   - LauraDB vs RDS for MongoDB
   - LauraDB vs DocumentDB
   - LauraDB vs DynamoDB
   - Cost-benefit analysis
   - Use case recommendations
   - Migration considerations

## Quick Start

### Choose Your Deployment Method

#### For Development/Testing
**Recommended: EC2 Single Instance**
- Fastest setup: ~15 minutes
- Lowest cost: ~$30/month
- Full control and easy debugging

```bash
# See ec2-deployment.md for detailed instructions
```

#### For Production (Small to Medium)
**Recommended: ECS Fargate**
- Managed infrastructure
- Auto scaling built-in
- Cost: ~$100-200/month
- No server management

```bash
# See ecs-deployment.md for detailed instructions
```

#### For Production (Large Scale/Enterprise)
**Recommended: EKS with Helm**
- Kubernetes native
- Maximum flexibility
- Advanced orchestration
- Cost: ~$200-500/month

```bash
# See eks-deployment.md for detailed instructions
```

## Deployment Comparison

| Factor | EC2 | ECS/Fargate | EKS |
|--------|-----|-------------|-----|
| **Setup Time** | 15-30 min | 20-40 min | 30-60 min |
| **Complexity** | Low | Medium | High |
| **Management** | Manual | Semi-managed | Managed control plane |
| **Cost (small)** | $ | $$ | $$$ |
| **Cost (large)** | $ | $$ | $$ |
| **Scaling** | Manual/ASG | Automatic | Automatic |
| **Flexibility** | High | Medium | Very High |
| **Best For** | Dev/Test | Production apps | Enterprise/Multi-service |

## Architecture Patterns

### Pattern 1: Single Region, Single AZ (Development)

```
┌─────────────────┐
│   Internet      │
│   Gateway       │
└────────┬────────┘
         │
    ┌────▼─────┐
    │Public    │
    │Subnet    │
    │          │
    │ ┌──────┐ │
    │ │ EC2  │ │
    │ │+LauraDB│
    │ └──────┘ │
    └──────────┘
```

**Use Case**: Development, testing, demos
**Cost**: ~$30-50/month
**Availability**: Single point of failure

### Pattern 2: Single Region, Multi-AZ (Production)

```
┌────────────────────────────┐
│   Application Load Balancer │
└──────┬─────────────┬────────┘
       │             │
   ┌───▼──┐      ┌──▼───┐
   │ AZ-1 │      │ AZ-2 │
   │ EC2  │      │ EC2  │
   │+LauraDB│    │+LauraDB│
   └───┬──┘      └──┬───┘
       │            │
       └─────┬──────┘
         ┌───▼───┐
         │  EFS  │
         └───────┘
```

**Use Case**: Production applications
**Cost**: ~$200-400/month
**Availability**: 99.9%+

### Pattern 3: Multi-Region (High Availability)

```
┌──────────────┐         ┌──────────────┐
│  us-east-1   │         │  us-west-2   │
│              │         │              │
│  ┌────────┐  │         │  ┌────────┐  │
│  │LauraDB │  │◄────────┤  │LauraDB │  │
│  │Cluster │  │  S3 CRR │  │Cluster │  │
│  └────────┘  │         │  └────────┘  │
└──────────────┘         └──────────────┘
```

**Use Case**: Mission-critical, global applications
**Cost**: ~$600-1000/month
**Availability**: 99.99%+

## Cost Estimates

### Monthly Cost Breakdown

#### Small Deployment (Dev/Test)
- **EC2**: t3.medium × 1 = $30
- **EBS**: 50GB gp3 = $4
- **Data Transfer**: ~$5
- **Backups (S3)**: $2
- **CloudWatch**: $5
- **Total**: ~$46/month

#### Medium Deployment (Production)
- **EC2**: t3.large × 2 = $121
- **ALB**: $22
- **EBS**: 200GB gp3 × 2 = $32
- **EFS**: 100GB = $30
- **Data Transfer**: ~$20
- **Backups (S3)**: $10
- **CloudWatch**: $15
- **Total**: ~$250/month

#### Large Deployment (Enterprise)
- **EKS**: Control plane = $73
- **EC2**: m5.xlarge × 3 = $374
- **EBS**: 500GB gp3 × 3 = $120
- **EFS**: 500GB = $150
- **ALB**: $22
- **Data Transfer**: ~$50
- **Backups (S3)**: $30
- **CloudWatch**: $30
- **Total**: ~$849/month

*Prices based on us-east-1, on-demand pricing, as of 2025*

## Getting Started

### Step 1: Choose Deployment Method

Review the comparison table and cost estimates above to select the appropriate deployment method for your use case.

### Step 2: Prepare Prerequisites

All deployment methods require:

1. **AWS Account** with appropriate permissions
2. **AWS CLI** installed and configured
3. **SSH Key Pair** for EC2 access (if using EC2/EKS)
4. **Domain name** (optional, for production)

### Step 3: Follow Deployment Guide

Navigate to the specific guide and follow step-by-step instructions:
- [EC2 Deployment Guide](./ec2-deployment.md)
- [ECS Deployment Guide](./ecs-deployment.md)
- [EKS Deployment Guide](./eks-deployment.md)

### Step 4: Configure Monitoring

Set up CloudWatch monitoring:
- Follow [CloudWatch Monitoring Guide](./cloudwatch-monitoring.md)
- Create dashboards and alarms
- Configure log aggregation

### Step 5: Set Up Backups

Configure automated backups:
- Follow [S3 Backup Integration Guide](./s3-backup-integration.md)
- Test restore procedures
- Set up lifecycle policies

## Best Practices

### Security

1. ✅ **Use IAM roles** instead of access keys
2. ✅ **Enable encryption** at rest and in transit
3. ✅ **Restrict security groups** to minimum required access
4. ✅ **Use AWS Secrets Manager** for credentials
5. ✅ **Enable CloudTrail** for audit logging
6. ✅ **Implement network segmentation** with private subnets
7. ✅ **Enable MFA** for production accounts
8. ✅ **Regular security updates** and patching

### Reliability

1. ✅ **Deploy across multiple AZs** for high availability
2. ✅ **Use Auto Scaling** for capacity management
3. ✅ **Implement health checks** and automatic recovery
4. ✅ **Test disaster recovery** procedures regularly
5. ✅ **Use managed services** where possible
6. ✅ **Implement circuit breakers** and retries
7. ✅ **Monitor and alert** on key metrics

### Performance

1. ✅ **Use appropriate instance types** for workload
2. ✅ **Enable enhanced networking** for high throughput
3. ✅ **Use gp3 volumes** with provisioned IOPS
4. ✅ **Implement caching** at multiple layers
5. ✅ **Optimize database configuration** (buffer size, cache)
6. ✅ **Use CloudFront** for global content delivery
7. ✅ **Monitor and optimize** based on metrics

### Cost Optimization

1. ✅ **Use Reserved Instances** for steady-state workload
2. ✅ **Leverage Spot Instances** for fault-tolerant workloads
3. ✅ **Right-size instances** based on utilization
4. ✅ **Use Auto Scaling** to match capacity to demand
5. ✅ **Implement S3 lifecycle policies** for old backups
6. ✅ **Delete unused resources** regularly
7. ✅ **Use AWS Cost Explorer** for analysis
8. ✅ **Set up billing alarms** to avoid surprises

## Support and Resources

### Documentation
- [LauraDB Main Documentation](../../README.md)
- [AWS Well-Architected Framework](https://aws.amazon.com/architecture/well-architected/)
- [AWS Documentation](https://docs.aws.amazon.com/)

### Tools
- [AWS Pricing Calculator](https://calculator.aws/)
- [AWS CLI Reference](https://docs.aws.amazon.com/cli/)
- [eksctl Documentation](https://eksctl.io/)

### Community
- [LauraDB GitHub Issues](https://github.com/mnohosten/laura-db/issues)
- [AWS Forums](https://forums.aws.amazon.com/)
- [AWS re:Post](https://repost.aws/)

## Troubleshooting

### Common Issues

#### Issue: Instance cannot connect to S3
**Solution**: Check VPC endpoints, NAT Gateway, and IAM role permissions

#### Issue: High latency between components
**Solution**: Ensure resources are in same region/AZ, check network configuration

#### Issue: Backup failures
**Solution**: Verify IAM permissions, S3 bucket policy, and disk space

#### Issue: Memory pressure
**Solution**: Reduce cache sizes, upgrade instance type, or add swap

#### Issue: Cost higher than expected
**Solution**: Review CloudWatch metrics, check for idle resources, review data transfer costs

For detailed troubleshooting, see individual guide troubleshooting sections.

## Migration Guides

### From Development to Production

1. Review [RDS Comparison](./rds-comparison.md) for alternatives
2. Scale up instance types and add redundancy
3. Implement Multi-AZ deployment
4. Set up automated backups and monitoring
5. Configure proper security groups and IAM roles
6. Enable encryption and audit logging
7. Test disaster recovery procedures

### From On-Premises to AWS

1. Export data from on-premises LauraDB
2. Upload data to S3
3. Deploy LauraDB on AWS using appropriate guide
4. Import data from S3
5. Update application connection strings
6. Test thoroughly before cutover
7. Implement gradual migration with DNS routing

## License

LauraDB and all documentation are available under the same license as the main project.

## Contributing

Contributions to improve these guides are welcome! Please submit issues or pull requests to the [LauraDB repository](https://github.com/mnohosten/laura-db).
