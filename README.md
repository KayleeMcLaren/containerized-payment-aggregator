![header](https://capsule-render.vercel.app/api?type=waving&height=200&color=gradient&text=Resilient%20Payment%20Gateway%20Aggregator&fontSize=35&strokeWidth=0&desc=Go%20Microservice%20with%20Circuit%20Breakers%20on%20AWS%20Fargate&descAlign=50&descAlignY=62&reversal=false&fontAlign=50&fontAlignY=40)

## 🚀 Project Status

This project demonstrates a production-ready deployment pattern. All infrastructure was successfully deployed, validated, and **has been cleanly destroyed** via `terraform destroy` to prevent recurring costs.

**Deployment Pattern Endpoint:** `http://<YOUR-ALB-DNS-NAME>/v1/pay`

*(Note: The deployment has been destroyed; replace this placeholder with the output from a fresh 'terraform apply'.)*

---

## ✨ Core Technical Achievements (Focus on Resilience)

This project was built using **Go** to maximize performance and demonstrate expertise in core back-end stability and cloud deployment patterns, specifically moving beyond pure serverless to showcase resilient containerized systems.

| Category | Achievement | Implementation Details |
| :--- | :---: | :--- |
| **⚡ Resilience** | **Circuit Breaker Pattern** | Implemented `sony/gobreaker` to monitor provider failure rates (`>60%` threshold) and instantly return a **503 Service Unavailable** response, protecting the system from cascading failure. |
| **🔒 State & Idempotency** | **Redis-Backed Idempotency** | Integrated managed **AWS ElastiCache (Redis)** to store transaction status (`IN_PROGRESS`/`COMPLETED`) to prevent duplicate processing if a client retries a payment request. |
| **🧩 Extensibility** | **Multi-Provider Adapter** | Used a Go Interface (`PaymentProvider`) to seamlessly integrate and route traffic to multiple external services (**MTN** and **Airtel**). |
| **🐳 Containerization** | **Multi-Stage Docker Build** | Optimized the deployment artifact using a multi-stage Dockerfile (`golang:latest` $\rightarrow$ `alpine:latest`) to produce a minimal, secure, static binary. |
| **🧱 Infrastructure** | **ECS Fargate with ALB** | Provisioned and managed the entire architecture using **Terraform**, deploying the service to **Private Fargate** instances behind a Public **Application Load Balancer (ALB)** for stable, production-ready routing. |

---

## 📖 About This Project

The project is a high-performance **Payment Gateway Aggregator** designed to manage requests destined for multiple unreliable external payment APIs. The core function is to ensure system stability and predictable response handling under adverse conditions, a requirement critical for high-volume fintech platforms.

All infrastructure is provisioned and managed using **Terraform**, demonstrating **Infrastructure as Code (IaC)** best practices.

## 🏗️ Architecture Overview

The system follows a standard, robust, containerized microservice pattern:

1.  **Client $\rightarrow$ ALB:** Requests hit the stable public DNS of the Application Load Balancer.
2.  **ALB $\rightarrow$ Fargate:** The ALB routes traffic to the **private** ECS Fargate task.
3.  **Go Application:**
    * Checks **Idempotency** (Redis).
    * Checks **Circuit Breaker Status**.
    * Executes the provider's `ProcessPayment()` method.
4.  **Fargate $\leftrightarrow$ ElastiCache:** The application uses the private network to communicate with the managed Redis cluster for state storage.

## 🛠️ Tech Stack

| Component | Technology | Role |
| :--- | :--- | :--- |
| **Language** | **Go (Golang)** | High-concurrency backend service logic. |
| **Resilience** | `sony/gobreaker` | Implements the Circuit Breaker pattern. |
| **State** | **AWS ElastiCache (Redis)** | Idempotency store and transaction state. |
| **Container** | **Docker** | Packaging the application binary. |
| **Orchestration** | **AWS ECS Fargate** | Serverless container compute environment. |
| **Load Balancing**| **AWS ALB** | Stable public endpoint and routing. |
| **IaC** | **Terraform** | Provisioning and managing the entire cloud environment. |

---

## 📂 Project Structure

This project is a clean Go monorepo separating application code from the infrastructure definition.

```
payment-gateway-aggregator/
├── .gitignore
├──  README.md
├──  main.go                    # Core Aggregator Logic & Server Setup
├──  go.mod
├──  go.sum
├──  Dockerfile                 # Multi-stage build configuration
├──  cache/
│ ├── redis.go                  # Idempotency Store (Redis client logic)
├──  providers/
│ ├── provider.go               # PaymentProvider Interface (Adapter Pattern) 
│ ├── mtn.go                    # MTN Mock Provider (with 80% failure simulation) 
│ ├── airtel.go                 # Airtel Mock Provider (with 80% failure simulation)  
├──  terraform/
│ ├── main.tf                   # AWS Provider, ECR, and VPC Module definition
│ ├── variables.tf 
│ ├── ecs.tf                    # ECS Cluster, Task Definition, IAM, and ElastiCache
│ ├── alb.tf                    # Application Load Balancer (ALB) and Target Group
```

---

##  🚀 Deployment Instructions
**Prerequisites**
1.  An **[AWS Account](https://aws.amazon.com/)**
2.  **[AWS CLI](https://aws.amazon.com/cli/)** configured (run `aws configure`)
3.  **[Terraform](https://www.io/downloads.html)** installed
4.  **[Docker](https://www.docker.com/get-started)** installed
5.  A `git` client

**Deployment Steps:**
1. **Clone and Configure:**

```
git clone [YOUR REPO URL]
cd [YOUR REPO NAME]
```

2. **Build and Push Image:**
```
# Ensure you are logged into ECR via AWS CLI before this step
docker build -t payment-gateway-aggregator:latest .
docker tag payment-gateway-aggregator:latest YOUR_ECR_URI/aggregator-gateway:latest
docker push YOUR_ECR_URI/aggregator-gateway:latest
```

3. **Deploy Infrastructure:**

```
cd terraform
terraform init
terraform apply
```
*Note the aggregator_endpoint_dns output for live testing.*

**Cleanup (To avoid charges)**
```
terraform destroy
```

## 🧪 Testing Validation (Final Test)

The resilience features were validated against the live deployment by setting the provider failure rate to 80% and confirming the system's fail-fast mechanism.

**Validation Command (Run 10 times consecutively):**
```bash
ALB_DNS="<YOUR-ALB-DNS-NAME>"
for i in {1..10}; do \
  curl -s -X POST --max-time 10 "http://${ALB_DNS}/v1/pay" \
  -H "Content-Type: application/json" \
  -d '{"TransactionID":"TXN-ID-022-'$i'", "Amount":20.00, "Currency":"ZAR", "ProviderKey":"AIRTEL"}' \
  -o /dev/null -w "%{http_code}\n"; \
done
```

**Result:**
```bash
200      # Initial Success
500      # Provider Failure (Counted by CB)
500      # Provider Failure (Counted by CB)
503      # Circuit Breaker OPEN (Threshold reached)
503      # Fast Failure (System Protected)
# ... (all subsequent requests return 503 instantly)
```
**Validation:** The system successfully demonstrated that when the failure rate exceeds the 60% threshold, the Circuit Breaker immediately trips, preserving system stability.

---