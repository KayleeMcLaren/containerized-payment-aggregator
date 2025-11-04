# terraform/alb.tf (Load Balancer Setup)

# 1. Application Load Balancer (ALB)
resource "aws_lb" "aggregator_alb" {
  name               = "${var.app_name}-alb"
  internal           = false
  load_balancer_type = "application"
  security_groups    = [aws_security_group.alb_sg.id]
  subnets            = module.vpc.public_subnets # ALB sits in PUBLIC subnets

  enable_deletion_protection = false # Easier cleanup for testing

  tags = {
    Name = "${var.app_name}-alb"
  }
}

# 2. Target Group (Where ALB sends traffic)
resource "aws_lb_target_group" "aggregator_tg" {
  name        = "${var.app_name}-tg"
  port        = 8080
  protocol    = "HTTP"
  vpc_id      = module.vpc.vpc_id
  target_type = "ip"

  health_check {
    path                = "/v1/pay" # Use your pay endpoint for health check
    protocol            = "HTTP"
    matcher             = "405"     # Expect Method Not Allowed (405) on a GET to a POST endpoint
    interval            = 30
    timeout             = 5
    healthy_threshold   = 2
    unhealthy_threshold = 2
  }
}

# 3. ALB Listener (Listens on port 80 and forwards to Target Group)
resource "aws_lb_listener" "http_listener" {
  load_balancer_arn = aws_lb.aggregator_alb.arn
  port              = "80"
  protocol          = "HTTP"

  default_action {
    type             = "forward"
    target_group_arn = aws_lb_target_group.aggregator_tg.arn
  }
}

# 4. ALB Security Group (Allows Inbound traffic from Internet)
resource "aws_security_group" "alb_sg" {
  name        = "${var.app_name}-alb-sg"
  description = "Allows HTTP traffic from the internet"
  vpc_id      = module.vpc.vpc_id

  ingress {
    description = "Allow HTTP from anywhere"
    from_port   = 80
    to_port     = 80
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
  }

  # Allow all outbound traffic
  egress {
    from_port   = 0
    to_port     = 0
    protocol    = "-1"
    cidr_blocks = ["0.0.0.0/0"]
  }
}

# 5. Output the stable DNS endpoint
output "aggregator_endpoint_dns" {
  description = "The stable public DNS endpoint for the Aggregator Service."
  value       = aws_lb.aggregator_alb.dns_name
}