# AWS Module Outputs

output "instance_ids" {
  description = "IDs of EC2 instances"
  value       = var.enable_auto_scaling ? [] : aws_instance.laura_db[*].id
}

output "public_ips" {
  description = "Public IP addresses of instances"
  value       = var.enable_elastic_ips ? aws_eip.laura_db[*].public_ip : (var.enable_auto_scaling ? [] : aws_instance.laura_db[*].public_ip)
}

output "private_ips" {
  description = "Private IP addresses of instances"
  value       = var.enable_auto_scaling ? [] : aws_instance.laura_db[*].private_ip
}

output "load_balancer_dns" {
  description = "DNS name of the load balancer"
  value       = var.enable_load_balancer ? aws_lb.laura_db[0].dns_name : null
}

output "load_balancer_endpoint" {
  description = "Full endpoint URL of the load balancer"
  value       = var.enable_load_balancer ? "http://${aws_lb.laura_db[0].dns_name}:${var.laura_db_port}" : null
}

output "backup_bucket_name" {
  description = "Name of the S3 backup bucket"
  value       = var.enable_backups ? aws_s3_bucket.backups[0].id : null
}

output "backup_bucket_arn" {
  description = "ARN of the S3 backup bucket"
  value       = var.enable_backups ? aws_s3_bucket.backups[0].arn : null
}

output "security_group_id" {
  description = "ID of the security group"
  value       = aws_security_group.laura_db.id
}

output "iam_role_arn" {
  description = "ARN of the IAM role"
  value       = aws_iam_role.laura_db.arn
}

output "iam_role_name" {
  description = "Name of the IAM role"
  value       = aws_iam_role.laura_db.name
}

output "vpc_id" {
  description = "VPC ID"
  value       = var.create_vpc ? aws_vpc.main[0].id : var.vpc_id
}

output "subnet_ids" {
  description = "Subnet IDs"
  value       = var.create_vpc ? aws_subnet.public[*].id : var.subnet_ids
}

output "cloudwatch_log_group" {
  description = "CloudWatch log group name"
  value       = var.enable_monitoring ? aws_cloudwatch_log_group.laura_db[0].name : null
}

output "connection_info" {
  description = "Connection information for LauraDB"
  value = {
    endpoints = var.enable_load_balancer ? [
      "http://${aws_lb.laura_db[0].dns_name}:${var.laura_db_port}"
      ] : [
      for ip in (var.enable_elastic_ips ? aws_eip.laura_db[*].public_ip : (var.enable_auto_scaling ? [] : aws_instance.laura_db[*].public_ip)) :
      "http://${ip}:${var.laura_db_port}"
    ]
    port           = var.laura_db_port
    health_check   = "/_health"
    admin_console  = "/admin"
  }
}

output "monitoring_dashboard" {
  description = "CloudWatch dashboard URL"
  value       = var.enable_monitoring ? "https://console.aws.amazon.com/cloudwatch/home?region=${var.region}#logsV2:log-groups/log-group/${urlencode(aws_cloudwatch_log_group.laura_db[0].name)}" : null
}

# Summary output for easy reference
output "deployment_summary" {
  description = "Summary of the deployment"
  value = {
    project_name      = var.project_name
    environment       = var.environment
    region            = var.region
    instance_count    = var.enable_auto_scaling ? "auto-scaling" : var.instance_count
    instance_type     = var.instance_type
    laura_db_version  = var.laura_db_version
    backups_enabled   = var.enable_backups
    monitoring_enabled = var.enable_monitoring
    load_balancer_enabled = var.enable_load_balancer
  }
}
