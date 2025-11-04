# 1. AWS Provider and Region Definition
provider "aws" {
  region = var.aws_region
}

# 2. AWS Elastic Container Registry (ECR)
resource "aws_ecr_repository" "aggregator_repo" {
  name                 = var.app_name
  image_tag_mutability = "MUTABLE"
  force_delete         = true 
}

# 3. Networking Setup (VPC module)
module "vpc" {
  source  = "terraform-aws-modules/vpc/aws"
  version = "5.0.0" 

  name = "${var.app_name}-vpc"
  cidr = "10.0.0.0/16"
  
  azs             = ["${var.aws_region}a", "${var.aws_region}b"]
  private_subnets = ["10.0.1.0/24", "10.0.2.0/24"]
  public_subnets  = ["10.0.101.0/24", "10.0.102.0/24"]
  
  enable_nat_gateway   = true
  single_nat_gateway   = true
  enable_dns_hostnames = true

  tags = {
    Name = "${var.app_name}-vpc"
  }
}

output "vpc_id" { 
  value = module.vpc.vpc_id
}