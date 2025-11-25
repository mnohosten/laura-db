# LauraDB Terraform Modules

Infrastructure as Code (IaC) for deploying LauraDB across AWS, GCP, and Azure using Terraform.

## Overview

This directory contains Terraform modules and examples for deploying LauraDB on major cloud providers:

- **AWS**: EC2, EKS, S3, CloudWatch
- **GCP**: GCE, GKE, Cloud Storage, Cloud Monitoring
- **Azure**: VMs, AKS, Blob Storage, Azure Monitor
- **Multi-Cloud**: Deploy across multiple clouds simultaneously

## Directory Structure

```
terraform/
├── modules/
│   ├── aws/           # AWS-specific resources
│   ├── gcp/           # GCP-specific resources
│   ├── azure/         # Azure-specific resources
│   └── common/        # Shared configuration
├── examples/
│   ├── aws/           # AWS deployment examples
│   ├── gcp/           # GCP deployment examples
│   ├── azure/         # Azure deployment examples
│   └── multi-cloud/   # Multi-cloud deployment examples
└── README.md          # This file
```

## Prerequisites

### Required Tools

```bash
# Install Terraform
# macOS
brew install terraform

# Linux
wget https://releases.hashicorp.com/terraform/1.6.0/terraform_1.6.0_linux_amd64.zip
unzip terraform_1.6.0_linux_amd64.zip
sudo mv terraform /usr/local/bin/

# Verify
terraform version
```

### Cloud Provider CLIs

```bash
# AWS CLI
curl "https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip" -o "awscliv2.zip"
unzip awscliv2.zip
sudo ./aws/install

# GCP CLI
curl https://sdk.cloud.google.com | bash
exec -l $SHELL
gcloud init

# Azure CLI
curl -sL https://aka.ms/InstallAzureCLIDeb | sudo bash
```

### Authentication

#### AWS
```bash
# Configure AWS credentials
aws configure

# Or set environment variables
export AWS_ACCESS_KEY_ID="your-access-key"
export AWS_SECRET_ACCESS_KEY="your-secret-key"
export AWS_DEFAULT_REGION="us-east-1"
```

#### GCP
```bash
# Authenticate with GCP
gcloud auth application-default login

# Set project
gcloud config set project YOUR_PROJECT_ID
```

#### Azure
```bash
# Login to Azure
az login

# Set subscription
az account set --subscription "Your Subscription Name"
```

## Quick Start

### AWS Deployment

```bash
cd examples/aws

# Initialize Terraform
terraform init

# Review planned changes
terraform plan

# Deploy infrastructure
terraform apply

# Get outputs
terraform output
```

### GCP Deployment

```bash
cd examples/gcp

terraform init
terraform plan
terraform apply
```

### Azure Deployment

```bash
cd examples/azure

terraform init
terraform plan
terraform apply
```

### Multi-Cloud Deployment

```bash
cd examples/multi-cloud

# Deploy to all clouds simultaneously
terraform init
terraform plan
terraform apply
```

## Module Usage

### AWS Module

```hcl
module "laura_db_aws" {
  source = "../../modules/aws"

  # Required variables
  project_name = "laura-db"
  environment  = "production"
  region       = "us-east-1"

  # VM configuration
  instance_type = "t3.medium"
  instance_count = 2

  # Storage
  volume_size = 100
  volume_type = "gp3"

  # Networking
  vpc_cidr = "10.0.0.0/16"

  # Backup
  enable_backups = true
  backup_retention_days = 30

  # Monitoring
  enable_monitoring = true

  # Tags
  tags = {
    Application = "LauraDB"
    Team        = "Platform"
  }
}
```

### GCP Module

```hcl
module "laura_db_gcp" {
  source = "../../modules/gcp"

  # Required variables
  project_id   = "my-project-id"
  project_name = "laura-db"
  environment  = "production"
  region       = "us-central1"

  # VM configuration
  machine_type   = "e2-medium"
  instance_count = 2

  # Storage
  disk_size_gb = 100
  disk_type    = "pd-ssd"

  # Networking
  network_cidr = "10.0.0.0/16"

  # Backup
  enable_backups = true
  backup_retention_days = 30

  # Monitoring
  enable_monitoring = true

  # Labels
  labels = {
    application = "laura-db"
    team        = "platform"
  }
}
```

### Azure Module

```hcl
module "laura_db_azure" {
  source = "../../modules/azure"

  # Required variables
  project_name = "laura-db"
  environment  = "production"
  location     = "eastus"

  # VM configuration
  vm_size        = "Standard_D2s_v3"
  instance_count = 2

  # Storage
  disk_size_gb = 100
  disk_type    = "Premium_LRS"

  # Networking
  vnet_address_space = ["10.0.0.0/16"]

  # Backup
  enable_backups = true
  backup_retention_days = 30

  # Monitoring
  enable_monitoring = true

  # Tags
  tags = {
    Application = "LauraDB"
    Team        = "Platform"
  }
}
```

## Module Outputs

All modules provide consistent outputs:

- `instance_ids` - IDs of created compute instances
- `public_ips` - Public IP addresses
- `private_ips` - Private IP addresses
- `load_balancer_ip` - Load balancer IP (if enabled)
- `storage_bucket` - Backup storage bucket/container name
- `connection_string` - LauraDB connection string

## Configuration Options

### Common Variables (All Modules)

| Variable | Type | Default | Description |
|----------|------|---------|-------------|
| `project_name` | string | - | Project name (required) |
| `environment` | string | `"production"` | Environment (dev, staging, production) |
| `instance_count` | number | `1` | Number of instances to create |
| `enable_monitoring` | bool | `true` | Enable cloud monitoring |
| `enable_backups` | bool | `true` | Enable automated backups |
| `backup_retention_days` | number | `30` | Backup retention period |

### Deployment Patterns

#### Single Instance (Development)

```hcl
module "laura_db" {
  source = "../../modules/aws"  # or gcp/azure

  project_name   = "laura-db-dev"
  environment    = "development"
  instance_count = 1
  instance_type  = "t3.small"  # Small instance for dev

  enable_backups    = false  # Disable backups for dev
  enable_monitoring = true
}
```

#### High Availability (Production)

```hcl
module "laura_db" {
  source = "../../modules/aws"  # or gcp/azure

  project_name   = "laura-db-prod"
  environment    = "production"
  instance_count = 3  # Multi-instance for HA
  instance_type  = "t3.large"

  enable_load_balancer = true
  enable_auto_scaling  = true
  min_instances        = 2
  max_instances        = 10

  enable_backups        = true
  backup_retention_days = 90

  enable_monitoring = true
  alert_email       = "ops@example.com"
}
```

## State Management

### Local State (Development)

By default, Terraform stores state locally in `terraform.tfstate`. This is suitable for development but **not recommended for production**.

### Remote State (Production)

For production, use remote state backends:

#### AWS S3 Backend

```hcl
terraform {
  backend "s3" {
    bucket         = "my-terraform-state"
    key            = "laura-db/terraform.tfstate"
    region         = "us-east-1"
    encrypt        = true
    dynamodb_table = "terraform-locks"
  }
}
```

#### GCP Cloud Storage Backend

```hcl
terraform {
  backend "gcs" {
    bucket = "my-terraform-state"
    prefix = "laura-db"
  }
}
```

#### Azure Blob Storage Backend

```hcl
terraform {
  backend "azurerm" {
    resource_group_name  = "terraform-state-rg"
    storage_account_name = "tfstate"
    container_name       = "tfstate"
    key                  = "laura-db.terraform.tfstate"
  }
}
```

## Cost Estimates

### Monthly Cost Estimates by Cloud

| Deployment | AWS | GCP | Azure |
|------------|-----|-----|-------|
| **Small (1 instance)** | ~$50 | ~$40 | ~$45 |
| **Medium (2 instances + LB)** | ~$200 | ~$180 | ~$190 |
| **Large (5 instances + LB + monitoring)** | ~$600 | ~$550 | ~$580 |

*Estimates based on standard pricing, may vary by region and usage*

## Best Practices

### 1. Use Workspaces for Multiple Environments

```bash
# Create workspaces
terraform workspace new development
terraform workspace new staging
terraform workspace new production

# Switch workspace
terraform workspace select production

# Use workspace in configuration
locals {
  env = terraform.workspace
  instance_count = terraform.workspace == "production" ? 3 : 1
}
```

### 2. Variable Files for Each Environment

```bash
# dev.tfvars
environment    = "development"
instance_count = 1
instance_type  = "t3.small"

# prod.tfvars
environment    = "production"
instance_count = 3
instance_type  = "t3.large"

# Apply with variable file
terraform apply -var-file="prod.tfvars"
```

### 3. Use Modules for Reusability

```hcl
# main.tf
module "database_primary" {
  source = "./modules/aws"
  region = "us-east-1"
}

module "database_dr" {
  source = "./modules/aws"
  region = "us-west-2"
}
```

### 4. Enable State Locking

Prevent concurrent modifications:

```hcl
# AWS DynamoDB for state locking
resource "aws_dynamodb_table" "terraform_locks" {
  name         = "terraform-locks"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "LockID"

  attribute {
    name = "LockID"
    type = "S"
  }
}
```

### 5. Use Data Sources for Existing Resources

```hcl
# Reference existing VPC
data "aws_vpc" "existing" {
  id = var.vpc_id
}

# Use in resource
resource "aws_subnet" "laura_db" {
  vpc_id = data.aws_vpc.existing.id
  # ...
}
```

## Security

### Secrets Management

**Never commit secrets to version control!**

#### Using Environment Variables

```bash
export TF_VAR_db_password="secure-password"
terraform apply
```

#### Using Terraform Cloud

```hcl
variable "db_password" {
  type      = string
  sensitive = true
}
```

Mark variable as sensitive in Terraform Cloud UI.

#### Using HashiCorp Vault

```hcl
data "vault_generic_secret" "db_password" {
  path = "secret/laura-db"
}

resource "aws_instance" "laura_db" {
  user_data = templatefile("init.sh", {
    password = data.vault_generic_secret.db_password.data["password"]
  })
}
```

### Least Privilege IAM

Each module creates minimal IAM roles:

- **Compute**: Read-only access to necessary services
- **Storage**: Write access only to backup buckets
- **Monitoring**: Write access to metrics/logs

## Troubleshooting

### Common Issues

#### 1. Authentication Errors

```bash
# AWS
aws sts get-caller-identity

# GCP
gcloud auth list
gcloud config list

# Azure
az account show
```

#### 2. State Lock Errors

```bash
# Force unlock (use with caution!)
terraform force-unlock LOCK_ID
```

#### 3. Resource Already Exists

```bash
# Import existing resource
terraform import module.laura_db.aws_instance.main i-1234567890abcdef0
```

#### 4. Plan Shows Unexpected Changes

```bash
# Refresh state
terraform refresh

# Show current state
terraform show
```

### Debugging

```bash
# Enable detailed logging
export TF_LOG=DEBUG
export TF_LOG_PATH=./terraform.log

# Run command
terraform apply

# View logs
cat terraform.log
```

## Maintenance

### Updating Infrastructure

```bash
# Always review changes first
terraform plan

# Apply updates
terraform apply

# Target specific resource
terraform apply -target=module.laura_db.aws_instance.main
```

### Destroying Infrastructure

```bash
# Destroy everything
terraform destroy

# Destroy specific resource
terraform destroy -target=module.laura_db.aws_instance.main
```

### State Operations

```bash
# List resources in state
terraform state list

# Show resource details
terraform state show module.laura_db.aws_instance.main

# Move resource in state
terraform state mv module.old.resource module.new.resource

# Remove resource from state (doesn't delete actual resource)
terraform state rm module.laura_db.aws_instance.main
```

## CI/CD Integration

### GitHub Actions

```yaml
name: Terraform

on:
  push:
    branches: [main]
  pull_request:

jobs:
  terraform:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3

      - name: Setup Terraform
        uses: hashicorp/setup-terraform@v2

      - name: Terraform Init
        run: terraform init

      - name: Terraform Plan
        run: terraform plan

      - name: Terraform Apply
        if: github.ref == 'refs/heads/main'
        run: terraform apply -auto-approve
```

### GitLab CI

```yaml
terraform:
  image: hashicorp/terraform:latest
  before_script:
    - terraform init
  script:
    - terraform plan
    - terraform apply -auto-approve
  only:
    - main
```

## Migration from Manual Setup

### Import Existing Resources

```bash
# AWS EC2 instance
terraform import module.laura_db.aws_instance.main i-1234567890abcdef0

# GCP VM instance
terraform import module.laura_db.google_compute_instance.main projects/my-project/zones/us-central1-a/instances/laura-db-vm

# Azure VM
terraform import module.laura_db.azurerm_virtual_machine.main /subscriptions/xxx/resourceGroups/rg/providers/Microsoft.Compute/virtualMachines/vm
```

## Examples

See the `examples/` directory for complete working examples:

- **[AWS](./examples/aws/)** - Complete AWS deployment
- **[GCP](./examples/gcp/)** - Complete GCP deployment
- **[Azure](./examples/azure/)** - Complete Azure deployment
- **[Multi-Cloud](./examples/multi-cloud/)** - Deploy across all clouds

## Support

- **Documentation**: See module-specific READMEs in `modules/` directories
- **Cloud Deployment Guides**:
  - [AWS](../docs/cloud/aws/)
  - [GCP](../docs/cloud/gcp/)
  - [Azure](../docs/cloud/azure/)
- **Issues**: [GitHub Issues](https://github.com/mnohosten/laura-db/issues)

## Contributing

Contributions welcome! Please:

1. Test changes with `terraform plan`
2. Update documentation
3. Follow [Terraform best practices](https://www.terraform.io/docs/cloud/guides/recommended-practices/index.html)
4. Submit pull request

## License

Same license as LauraDB project.

## References

- [Terraform Documentation](https://www.terraform.io/docs/)
- [AWS Provider](https://registry.terraform.io/providers/hashicorp/aws/latest/docs)
- [GCP Provider](https://registry.terraform.io/providers/hashicorp/google/latest/docs)
- [Azure Provider](https://registry.terraform.io/providers/hashicorp/azurerm/latest/docs)
- [LauraDB Documentation](../README.md)

---

**Getting Started**: Choose a cloud provider from `examples/` directory and follow the README instructions.
