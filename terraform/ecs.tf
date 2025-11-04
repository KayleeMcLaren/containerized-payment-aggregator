locals {
  # This performs the conversion and indexing, creating a 'whole object' reference.
  # We use tolist to convert the set output into an indexable list.
  public_route_assoc_ref = tolist(module.vpc.public_route_table_association_ids)[0]
}

# 1. AWS IAM Role for ECS Fargate Tasks
# ECS tasks need this role to pull images, write logs, and access other AWS services.
resource "aws_iam_role" "ecs_task_execution_role" {
  name = "${var.app_name}-ecs-exec-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "ecs-tasks.amazonaws.com"
      }
    }]
  })
}

# Attach standard policies needed for Fargate execution
resource "aws_iam_role_policy_attachment" "ecs_task_execution_role_policy" {
  role       = aws_iam_role.ecs_task_execution_role.name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AmazonECSTaskExecutionRolePolicy"
}

# 2. ElastiCache (Managed Redis) Setup
# This is used for your Idempotency Store.
resource "aws_elasticache_subnet_group" "aggregator_redis_sg" {
  name       = "${var.app_name}-redis-sg"
  subnet_ids = module.vpc.private_subnets # Redis should only be in private subnets
}

resource "aws_elasticache_cluster" "aggregator_redis" {
  cluster_id           = "${var.app_name}-redis-cluster"
  engine               = "redis"
  node_type            = "cache.t3.micro" # Small, cost-effective node type
  num_cache_nodes      = 1
  parameter_group_name = "default.redis7" # Assuming Redis 7 or a similar modern version
  engine_version       = "7.1"
  port                 = 6379
  subnet_group_name    = aws_elasticache_subnet_group.aggregator_redis_sg.name
  security_group_ids   = [aws_security_group.redis_sg.id]
}

# 3. ECS Cluster Definition
resource "aws_ecs_cluster" "aggregator_cluster" {
  name = "${var.app_name}-cluster"
}

# 4. ECS Task Definition (The blueprint for your container)
resource "aws_ecs_task_definition" "aggregator_task" {
  family                   = "${var.app_name}-task"
  requires_compatibilities = ["FARGATE"]
  network_mode             = "awsvpc"
  cpu                      = 256    # 0.25 vCPU
  memory                   = 512    # 0.5 GB RAM
  execution_role_arn       = aws_iam_role.ecs_task_execution_role.arn

  container_definitions = jsonencode([
    {
      name      = var.app_name
      image     = "${aws_ecr_repository.aggregator_repo.repository_url}:latest" # Uses the ECR repo URL
      cpu       = 256
      memory    = 512
      essential = true
      portMappings = [
        {
          containerPort = 8080
          hostPort      = 8080
        }
      ]
      environment = [
        {
          # Pass the Redis endpoint to the Go application via environment variable
          name  = "REDIS_ADDR"
          value = "${aws_elasticache_cluster.aggregator_redis.cache_nodes[0].address}:6379"
        },
        {
          # NEW: Unique variable to force a Task Definition change on every run
          name  = "DEPLOYMENT_ID"
          value = timestamp() 
        }      
      ]
      logConfiguration = {
        logDriver = "awslogs"
        options = {
          "awslogs-group"         = aws_cloudwatch_log_group.aggregator_log_group.name
          "awslogs-region"        = var.aws_region
          "awslogs-stream-prefix" = "ecs"
        }
      }
    }
  ])
}

# 5. Security Groups
# SG for the Redis cluster (Allow traffic from ECS)
resource "aws_security_group" "redis_sg" {
  name        = "${var.app_name}-redis-sg"
  description = "Allow inbound traffic from ECS service to Redis"
  vpc_id      = module.vpc.vpc_id

  ingress {
    description     = "Allow traffic from ECS Fargate service"
    from_port       = 6379
    to_port         = 6379
    protocol        = "tcp"
    security_groups = [aws_security_group.ecs_service_sg.id] # Reference ECS SG before it's fully defined (Terraform handles this)
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# SG for the ECS Service (Allow public inbound traffic to 8080)
resource "aws_security_group" "ecs_service_sg" {
  name        = "${var.app_name}-ecs-sg"
  description = "Allow inbound HTTP traffic to ECS service"
  vpc_id      = module.vpc.vpc_id

  ingress {
    description = "HTTP access from anywhere"
    from_port   = 8080
    to_port     = 8080
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# 6. CloudWatch Log Group
resource "aws_cloudwatch_log_group" "aggregator_log_group" {
  name              = "/ecs/${var.app_name}"
  retention_in_days = 7
}

# 7. ECS Service (Deployed to Private Subnets, exposed via ALB)
resource "aws_ecs_service" "aggregator_service" {
  name            = "${var.app_name}-service"
  cluster         = aws_ecs_cluster.aggregator_cluster.name
  task_definition = aws_ecs_task_definition.aggregator_task.arn
  desired_count   = 1

  launch_type = "FARGATE"

  network_configuration {
    security_groups  = [aws_security_group.ecs_service_sg.id]
    subnets          = module.vpc.private_subnets # CRITICAL CHANGE: Deploy to PRIVATE subnets
    assign_public_ip = false                       # CRITICAL CHANGE: No public IP needed
  }

  load_balancer { # NEW BLOCK: Link to the ALB target group
    target_group_arn = aws_lb_target_group.aggregator_tg.arn
    container_name   = var.app_name # "aggregator-gateway"
    container_port   = 8080
  }

  depends_on = [
    module.vpc,
    # The depends_on syntax is no longer critical here as the Load Balancer creates the dependency.
  ]

  deployment_controller {
    type = "ECS"
  }
}

# 8. Output the ARN and the expected access method
output "aggregator_service_arn" {
  description = "The ARN of the deployed ECS Fargate Service."
  value       = aws_ecs_service.aggregator_service.arn
}

output "aggregator_service_endpoint_note" {
  description = "Note on access: The service is deployed to public subnets. Check the ECS console for the Public IP or configure a Load Balancer."
  value       = "http://<Fargate-Public-IP>:8080"
}