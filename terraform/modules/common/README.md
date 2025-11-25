# Common Terraform Module

This module contains shared variables, outputs, and utilities used across all cloud-specific LauraDB Terraform modules (AWS, GCP, Azure).

## Purpose

The common module provides:

1. **Standardized Variables**: Consistent variable definitions across all clouds
2. **Output Structure**: Unified output format for all deployments
3. **Naming Conventions**: Standard resource naming patterns
4. **Tags/Labels**: Common tagging strategy

## Usage

Cloud-specific modules inherit these definitions:

```hcl
# In AWS/GCP/Azure modules
module "common" {
  source = "../common"

  project_name   = var.project_name
  environment    = var.environment
  instance_count = var.instance_count
  # ... other common variables
}

# Use common locals
resource "aws_instance" "main" {
  tags = merge(
    module.common.common_tags,
    {
      Name = format(module.common.resource_name_format, "instance-${count.index}")
    }
  )
}
```

## Variables

See `variables.tf` for all common variables.

### Required Variables

- `project_name` - Name of the project (1-32 characters)

### Optional Variables

- `environment` - Environment name (dev, staging, production) [default: "production"]
- `instance_count` - Number of instances [default: 1]
- `enable_monitoring` - Enable monitoring [default: true]
- `enable_backups` - Enable backups [default: true]
- `backup_retention_days` - Backup retention period [default: 30]
- `laura_db_version` - LauraDB version [default: "latest"]
- `laura_db_port` - HTTP server port [default: 8080]

## Outputs

Each cloud module should provide these standard outputs:

- `instance_ids` - List of created instance IDs
- `public_ips` - Public IP addresses
- `private_ips` - Private IP addresses
- `load_balancer_endpoint` - Load balancer DNS/IP
- `backup_storage` - Backup storage location
- `monitoring_dashboard` - Monitoring dashboard URL
- `connection_info` - LauraDB connection details

## Naming Convention

Resources are named using the pattern:

```
{project_name}-{environment}-{resource_type}-{suffix}
```

Examples:
- `laura-db-production-vm-0`
- `laura-db-staging-bucket`
- `laura-db-dev-network`

## Tags/Labels

All resources receive these common tags:

- `Project` - Project name
- `Environment` - Environment name
- `ManagedBy` - "Terraform"
- `Application` - "LauraDB"

Additional cloud-specific or resource-specific tags can be merged:

```hcl
tags = merge(
  module.common.common_tags,
  {
    CloudProvider = "AWS"
    InstanceType  = "Primary"
  }
)
```

## Best Practices

1. **Always use common variables** where applicable
2. **Merge common tags** into all resources
3. **Follow naming conventions** for consistency
4. **Provide standard outputs** from cloud modules
5. **Validate inputs** using variable validation blocks

## Extending

To add new common variables:

1. Add variable definition to `variables.tf`
2. Update this README
3. Update all cloud modules to use the new variable
4. Add validation if appropriate

Example:

```hcl
variable "new_common_var" {
  description = "Description of the new variable"
  type        = string
  default     = "default_value"

  validation {
    condition     = length(var.new_common_var) > 0
    error_message = "Variable cannot be empty."
  }
}
```

## Testing

Test common module changes by running:

```bash
# Validate syntax
terraform fmt -check
terraform validate

# Test in each cloud module
cd ../aws && terraform validate
cd ../gcp && terraform validate
cd ../azure && terraform validate
```

## References

- [Terraform Variable Syntax](https://www.terraform.io/language/values/variables)
- [Terraform Output Syntax](https://www.terraform.io/language/values/outputs)
- [Terraform Module](https://www.terraform.io/language/modules)
