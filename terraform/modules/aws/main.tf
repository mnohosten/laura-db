# AWS Terraform Module for LauraDB

terraform {
  required_version = ">= 1.0"

  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
  }
}

# Local variables
locals {
  name_prefix = "${var.project_name}-${var.environment}"

  common_tags = merge(
    {
      Project     = var.project_name
      Environment = var.environment
      ManagedBy   = "Terraform"
      Application = "LauraDB"
    },
    var.tags
  )

  # User data script
  user_data = templatefile("${path.module}/../common/user-data.sh", {
    project_name      = var.project_name
    environment       = var.environment
    laura_db_version  = var.laura_db_version
    laura_db_port     = var.laura_db_port
    data_dir          = var.data_dir
    log_level         = var.log_level
  })
}

# VPC
resource "aws_vpc" "main" {
  count = var.create_vpc ? 1 : 0

  cidr_block           = var.vpc_cidr
  enable_dns_hostnames = true
  enable_dns_support   = true

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-vpc"
    }
  )
}

# Internet Gateway
resource "aws_internet_gateway" "main" {
  count = var.create_vpc ? 1 : 0

  vpc_id = aws_vpc.main[0].id

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-igw"
    }
  )
}

# Public Subnet
resource "aws_subnet" "public" {
  count = var.create_vpc ? length(var.availability_zones) : 0

  vpc_id                  = aws_vpc.main[0].id
  cidr_block              = cidrsubnet(var.vpc_cidr, 8, count.index)
  availability_zone       = var.availability_zones[count.index]
  map_public_ip_on_launch = true

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-public-subnet-${count.index + 1}"
      Type = "Public"
    }
  )
}

# Route Table
resource "aws_route_table" "public" {
  count = var.create_vpc ? 1 : 0

  vpc_id = aws_vpc.main[0].id

  route {
    cidr_block = "0.0.0.0/0"
    gateway_id = aws_internet_gateway.main[0].id
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-public-rt"
    }
  )
}

# Route Table Association
resource "aws_route_table_association" "public" {
  count = var.create_vpc ? length(aws_subnet.public) : 0

  subnet_id      = aws_subnet.public[count.index].id
  route_table_id = aws_route_table.public[0].id
}

# Security Group
resource "aws_security_group" "laura_db" {
  name        = "${local.name_prefix}-sg"
  description = "Security group for LauraDB instances"
  vpc_id      = var.create_vpc ? aws_vpc.main[0].id : var.vpc_id

  # Allow LauraDB port
  ingress {
    description = "LauraDB HTTP port"
    from_port   = var.laura_db_port
    to_port     = var.laura_db_port
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  # Allow SSH
  ingress {
    description = "SSH access"
    from_port   = 22
    to_port     = 22
    protocol    = "tcp"
    cidr_blocks = var.allowed_cidr_blocks
  }

  # Allow all outbound
  egress {
    description = "Allow all outbound traffic"
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-sg"
    }
  )
}

# IAM Role for EC2 instances
resource "aws_iam_role" "laura_db" {
  name = "${local.name_prefix}-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "ec2.amazonaws.com"
        }
      }
    ]
  })

  tags = local.common_tags
}

# IAM Policy for CloudWatch
resource "aws_iam_role_policy" "cloudwatch" {
  count = var.enable_monitoring ? 1 : 0

  name = "${local.name_prefix}-cloudwatch-policy"
  role = aws_iam_role.laura_db.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "cloudwatch:PutMetricData",
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents",
          "logs:DescribeLogStreams"
        ]
        Resource = "*"
      }
    ]
  })
}

# IAM Policy for S3 backups
resource "aws_iam_role_policy" "s3_backups" {
  count = var.enable_backups ? 1 : 0

  name = "${local.name_prefix}-s3-backup-policy"
  role = aws_iam_role.laura_db.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:ListBucket",
          "s3:DeleteObject"
        ]
        Resource = [
          aws_s3_bucket.backups[0].arn,
          "${aws_s3_bucket.backups[0].arn}/*"
        ]
      }
    ]
  })
}

# IAM Instance Profile
resource "aws_iam_instance_profile" "laura_db" {
  name = "${local.name_prefix}-instance-profile"
  role = aws_iam_role.laura_db.name

  tags = local.common_tags
}

# SSH Key Pair
resource "aws_key_pair" "laura_db" {
  count = var.ssh_public_key != "" ? 1 : 0

  key_name   = "${local.name_prefix}-key"
  public_key = var.ssh_public_key

  tags = local.common_tags
}

# EC2 Instances
resource "aws_instance" "laura_db" {
  count = var.enable_auto_scaling ? 0 : var.instance_count

  ami                    = var.ami_id != "" ? var.ami_id : data.aws_ami.ubuntu.id
  instance_type          = var.instance_type
  key_name               = var.ssh_public_key != "" ? aws_key_pair.laura_db[0].key_name : null
  vpc_security_group_ids = [aws_security_group.laura_db.id]
  subnet_id              = var.create_vpc ? aws_subnet.public[count.index % length(aws_subnet.public)].id : var.subnet_ids[count.index % length(var.subnet_ids)]
  iam_instance_profile   = aws_iam_instance_profile.laura_db.name
  user_data              = local.user_data

  root_block_device {
    volume_type           = var.volume_type
    volume_size           = var.volume_size
    delete_on_termination = true
    encrypted             = true

    tags = merge(
      local.common_tags,
      {
        Name = "${local.name_prefix}-root-volume-${count.index}"
      }
    )
  }

  monitoring = var.enable_monitoring

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-instance-${count.index}"
    }
  )

  lifecycle {
    create_before_destroy = true
  }
}

# Elastic IPs (optional)
resource "aws_eip" "laura_db" {
  count = var.enable_elastic_ips && !var.enable_auto_scaling ? var.instance_count : 0

  instance = aws_instance.laura_db[count.index].id
  domain   = "vpc"

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-eip-${count.index}"
    }
  )

  depends_on = [aws_internet_gateway.main]
}

# S3 Bucket for backups
resource "aws_s3_bucket" "backups" {
  count = var.enable_backups ? 1 : 0

  bucket = "${local.name_prefix}-backups-${data.aws_caller_identity.current.account_id}"

  tags = merge(
    local.common_tags,
    {
      Name    = "${local.name_prefix}-backups"
      Purpose = "Backups"
    }
  )
}

# S3 Bucket versioning
resource "aws_s3_bucket_versioning" "backups" {
  count = var.enable_backups ? 1 : 0

  bucket = aws_s3_bucket.backups[0].id

  versioning_configuration {
    status = "Enabled"
  }
}

# S3 Bucket encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "backups" {
  count = var.enable_backups ? 1 : 0

  bucket = aws_s3_bucket.backups[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
  }
}

# S3 Bucket lifecycle policy
resource "aws_s3_bucket_lifecycle_configuration" "backups" {
  count = var.enable_backups ? 1 : 0

  bucket = aws_s3_bucket.backups[0].id

  rule {
    id     = "backup-retention"
    status = "Enabled"

    expiration {
      days = var.backup_retention_days
    }

    noncurrent_version_expiration {
      noncurrent_days = 7
    }
  }
}

# S3 Bucket public access block
resource "aws_s3_bucket_public_access_block" "backups" {
  count = var.enable_backups ? 1 : 0

  bucket = aws_s3_bucket.backups[0].id

  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
}

# Load Balancer (optional)
resource "aws_lb" "laura_db" {
  count = var.enable_load_balancer ? 1 : 0

  name               = "${local.name_prefix}-lb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.laura_db.id]
  subnets            = var.create_vpc ? aws_subnet.public[*].id : var.subnet_ids

  enable_deletion_protection = var.environment == "production" ? true : false

  tags = merge(
    local.common_tags,
    {
      Name = "${local.name_prefix}-lb"
    }
  )
}

# Target Group
resource "aws_lb_target_group" "laura_db" {
  count = var.enable_load_balancer ? 1 : 0

  name     = "${local.name_prefix}-tg"
  port     = var.laura_db_port
  protocol = "HTTP"
  vpc_id   = var.create_vpc ? aws_vpc.main[0].id : var.vpc_id

  health_check {
    enabled             = true
    healthy_threshold   = 2
    unhealthy_threshold = 2
    timeout             = 5
    interval            = 30
    path                = "/_health"
    protocol            = "HTTP"
  }

  tags = local.common_tags
}

# Target Group Attachment
resource "aws_lb_target_group_attachment" "laura_db" {
  count = var.enable_load_balancer && !var.enable_auto_scaling ? var.instance_count : 0

  target_group_arn = aws_lb_target_group.laura_db[0].arn
  target_id        = aws_instance.laura_db[count.index].id
  port             = var.laura_db_port
}

# Load Balancer Listener
resource "aws_lb_listener" "laura_db" {
  count = var.enable_load_balancer ? 1 : 0

  load_balancer_arn = aws_lb.laura_db[0].arn
  port              = var.laura_db_port
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.laura_db[0].arn
  }
}

# CloudWatch Log Group
resource "aws_cloudwatch_log_group" "laura_db" {
  count = var.enable_monitoring ? 1 : 0

  name              = "/aws/laura-db/${local.name_prefix}"
  retention_in_days = var.log_retention_days

  tags = local.common_tags
}

# Data sources
data "aws_caller_identity" "current" {}

data "aws_ami" "ubuntu" {
  most_recent = true
  owners      = ["099720109477"] # Canonical

  filter {
    name   = "name"
    values = ["ubuntu/images/hvm-ssd/ubuntu-jammy-22.04-amd64-server-*"]
  }

  filter {
    name   = "virtualization-type"
    values = ["hvm"]
  }
}
