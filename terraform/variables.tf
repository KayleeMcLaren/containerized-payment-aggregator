# terraform/variables.tf

variable "aws_region" {
  description = "The AWS region to deploy resources in."
  type        = string
  default     = "us-east-1"
}

variable "app_name" {
  description = "The base name for all application resources."
  type        = string
  default     = "aggregator-gateway"
}