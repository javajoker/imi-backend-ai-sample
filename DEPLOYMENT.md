# Deployment Guide

This guide covers deploying the IP Marketplace Backend across different environments, from local development to production cloud deployments.

## 📋 Table of Contents

- [Prerequisites](#prerequisites)
- [Environment Setup](#environment-setup)
- [Local Development](#local-development)
- [Docker Deployment](#docker-deployment)
- [Cloud Deployment](#cloud-deployment)
- [Production Deployment](#production-deployment)
- [Monitoring & Logging](#monitoring--logging)
- [Security Considerations](#security-considerations)
- [Troubleshooting](#troubleshooting)

## Prerequisites

### System Requirements

#### Minimum Requirements
- **CPU**: 2 cores
- **RAM**: 4GB
- **Storage**: 20GB SSD
- **Network**: Stable internet connection

#### Recommended Requirements (Production)
- **CPU**: 4+ cores
- **RAM**: 8GB+
- **Storage**: 100GB+ SSD
- **Network**: High-speed internet with redundancy

### Software Dependencies
- **Go**: 1.21+
- **PostgreSQL**: 12+
- **Redis**: 6+
- **Docker**: 20.10+ (optional but recommended)
- **Docker Compose**: 2.0+

### External Services
- **AWS Account**: For S3 storage and other services
- **Stripe Account**: For payment processing
- **SMTP Service**: For email notifications
- **Domain & SSL Certificate**: For production deployment

## Environment Setup

### 1. Environment Variables

Create environment-specific configuration files:

```bash
# Development
cp .env.example .env.development

# Staging  
cp .env.example .env.staging

# Production
cp .env.example .env.production
```

### 2. Security Configuration

#### Development Environment
```bash
# .env.development
ENVIRONMENT=development
JWT_SECRET=dev-secret-key-not-for-production
DB_PASSWORD=dev_password
```

#### Production Environment
```bash
# .env.production
ENVIRONMENT=production
JWT_SECRET=your-super-secure-256-bit-secret-key
DB_PASSWORD=your-very-secure-database-password
DB_SSL_MODE=require
```

### 3. Database Configuration

#### PostgreSQL Setup
```sql
-- Create database and user
CREATE DATABASE ip_marketplace;
CREATE USER ip_marketplace_user WITH PASSWORD 'secure_password';
GRANT ALL PRIVILEGES ON DATABASE ip_marketplace TO ip_marketplace_user;

-- Enable required extensions
\c ip_marketplace;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
```

## Local Development

### Quick Start
```bash
# Clone repository
git clone <repository-url>
cd imi-backend

# Setup environment
cp .env.example .env
# Edit .env with your configuration

# Install dependencies
go mod download

# Run database migrations
make migrate

# Start development server
make dev
```

### Development with Docker
```bash
# Start all services
docker-compose -f docker-compose.dev.yml up -d

# View logs
docker-compose logs -f backend

# Stop services
docker-compose down
```

### Live Reload Setup
```bash
# Install Air for live reload
go install github.com/cosmtrek/air@latest

# Start with live reload
air
```

## Docker Deployment

### 1. Single Container Deployment

#### Build Image
```bash
# Build production image
docker build -t imi-backend:latest .

# Or with specific version
docker build -t imi-backend:v1.0.0 .
```

#### Run Container
```bash
docker run -d \
  --name imi-backend \
  -p 8080:8080 \
  --env-file .env.production \
  -v $(pwd)/uploads:/app/uploads \
  imi-backend:latest
```

### 2. Docker Compose Deployment

#### Production Docker Compose
```yaml
# docker-compose.prod.yml
version: '3.8'

services:
  postgres:
    image: postgres:15-alpine
    environment:
      POSTGRES_DB: ip_marketplace
      POSTGRES_USER: ${DB_USER}
      POSTGRES_PASSWORD: ${DB_PASSWORD}
    volumes:
      - postgres_data:/var/lib/postgresql/data
      - ./backup:/backup
    restart: unless-stopped
    networks:
      - imi-network

  redis:
    image: redis:7-alpine
    command: redis-server --requirepass ${REDIS_PASSWORD}
    volumes:
      - redis_data:/data
    restart: unless-stopped
    networks:
      - imi-network

  backend:
    image: imi-backend:latest
    environment:
      - ENVIRONMENT=production
      - DB_HOST=postgres
      - REDIS_HOST=redis
    env_file:
      - .env.production
    ports:
      - "8080:8080"
    depends_on:
      - postgres
      - redis
    volumes:
      - ./uploads:/app/uploads
      - ./logs:/app/logs
    restart: unless-stopped
    networks:
      - imi-network

  nginx:
    image: nginx:alpine
    ports:
      - "80:80"
      - "443:443"
    volumes:
      - ./nginx/nginx.conf:/etc/nginx/nginx.conf
      - ./nginx/ssl:/etc/nginx/ssl
      - ./uploads:/var/www/uploads
    depends_on:
      - backend
    restart: unless-stopped
    networks:
      - imi-network

volumes:
  postgres_data:
  redis_data:

networks:
  imi-network:
    driver: bridge
```

#### Deploy with Docker Compose
```bash
# Production deployment
docker-compose -f docker-compose.prod.yml up -d

# Check status
docker-compose -f docker-compose.prod.yml ps

# View logs
docker-compose -f docker-compose.prod.yml logs -f backend
```

## Cloud Deployment

### AWS Deployment

#### 1. EC2 Instance Setup

```bash
# Launch EC2 instance (Ubuntu 22.04 LTS)
# Instance type: t3.medium or larger for production

# Connect to instance
ssh -i your-key.pem ubuntu@your-ec2-ip

# Update system
sudo apt update && sudo apt upgrade -y

# Install Docker
curl -fsSL https://get.docker.com -o get-docker.sh
sudo sh get-docker.sh
sudo usermod -aG docker ubuntu

# Install Docker Compose
sudo curl -L "https://github.com/docker/compose/releases/latest/download/docker-compose-$(uname -s)-$(uname -m)" -o /usr/local/bin/docker-compose
sudo chmod +x /usr/local/bin/docker-compose
```

#### 2. RDS Database Setup

```bash
# Create RDS PostgreSQL instance
aws rds create-db-instance \
  --db-instance-identifier imi-db \
  --db-instance-class db.t3.micro \
  --engine postgres \
  --engine-version 15.4 \
  --master-username postgres \
  --master-user-password your-secure-password \
  --allocated-storage 20 \
  --storage-type gp2 \
  --vpc-security-group-ids sg-xxxxxxxxx \
  --backup-retention-period 7 \
  --storage-encrypted
```

#### 3. ElastiCache Redis Setup

```bash
# Create ElastiCache Redis cluster
aws elasticache create-cache-cluster \
  --cache-cluster-id imi-redis \
  --cache-node-type cache.t3.micro \
  --engine redis \
  --num-cache-nodes 1 \
  --security-group-ids sg-xxxxxxxxx
```

#### 4. S3 Bucket Setup

```bash
# Create S3 bucket for file storage
aws s3 mb s3://imi-assets-prod

# Set bucket policy for public read access to certain paths
aws s3api put-bucket-policy \
  --bucket imi-assets-prod \
  --policy file://s3-bucket-policy.json
```

#### 5. CloudFront Distribution

```bash
# Create CloudFront distribution for S3
aws cloudfront create-distribution \
  --distribution-config file://cloudfront-config.json
```

### AWS ECS Deployment

#### 1. Task Definition

```json
{
  "family": "imi-backend",
  "networkMode": "awsvpc",
  "requiresCompatibilities": ["FARGATE"],
  "cpu": "512",
  "memory": "1024",
  "executionRoleArn": "arn:aws:iam::account:role/ecsTaskExecutionRole",
  "taskRoleArn": "arn:aws:iam::account:role/ecsTaskRole",
  "containerDefinitions": [
    {
      "name": "backend",
      "image": "your-account.dkr.ecr.region.amazonaws.com/imi-backend:latest",
      "portMappings": [
        {
          "containerPort": 8080,
          "protocol": "tcp"
        }
      ],
      "environment": [
        {
          "name": "ENVIRONMENT",
          "value": "production"
        }
      ],
      "secrets": [
        {
          "name": "DB_PASSWORD",
          "valueFrom": "arn:aws:secretsmanager:region:account:secret:db-password"
        }
      ],
      "logConfiguration": {
        "logDriver": "awslogs",
        "options": {
          "awslogs-group": "/ecs/imi-backend",
          "awslogs-region": "us-east-1",
          "awslogs-stream-prefix": "ecs"
        }
      }
    }
  ]
}
```

#### 2. Service Configuration

```bash
# Create ECS service
aws ecs create-service \
  --cluster imi-cluster \
  --service-name imi-backend-service \
  --task-definition imi-backend \
  --desired-count 2 \
  --launch-type FARGATE \
  --network-configuration "awsvpcConfiguration={subnets=[subnet-xxx,subnet-yyy],securityGroups=[sg-xxxxxxxxx],assignPublicIp=ENABLED}" \
  --load-balancers "targetGroupArn=arn:aws:elasticloadbalancing:region:account:targetgroup/imi-tg,containerName=backend,containerPort=8080"
```

### Google Cloud Platform (GCP)

#### 1. Cloud Run Deployment

```bash
# Build and push image to Container Registry
gcloud builds submit --tag gcr.io/PROJECT_ID/imi-backend

# Deploy to Cloud Run
gcloud run deploy imi-backend \
  --image gcr.io/PROJECT_ID/imi-backend \
  --platform managed \
  --region us-central1 \
  --allow-unauthenticated \
  --set-env-vars ENVIRONMENT=production \
  --set-env-vars DB_HOST=CLOUD_SQL_IP \
  --memory 1Gi \
  --cpu 1 \
  --max-instances 10
```

#### 2. Cloud SQL Setup

```bash
# Create Cloud SQL PostgreSQL instance
gcloud sql instances create imi-db \
  --database-version POSTGRES_15 \
  --tier db-f1-micro \
  --region us-central1 \
  --storage-type SSD \
  --storage-size 20GB
```

### Azure Deployment

#### 1. Container Instances

```bash
# Create resource group
az group create --name imi-rg --location eastus

# Deploy container
az container create \
  --resource-group imi-rg \
  --name imi-backend \
  --image your-registry/imi-backend:latest \
  --cpu 1 \
  --memory 2 \
  --ports 8080 \
  --environment-variables ENVIRONMENT=production \
  --secure-environment-variables DB_PASSWORD=your-password
```

#### 2. Azure Database for PostgreSQL

```bash
# Create PostgreSQL server
az postgres server create \
  --resource-group imi-rg \
  --name imi-db-server \
  --location eastus \
  --admin-user postgres \
  --admin-password your-secure-password \
  --sku-name B_Gen5_1
```

## Production Deployment

### Pre-deployment Checklist

- [ ] Environment variables configured
- [ ] Database backup created
- [ ] SSL certificates obtained
- [ ] Domain DNS configured
- [ ] Monitoring setup verified
- [ ] Security scan completed
- [ ] Load testing performed
- [ ] Rollback plan prepared

### Deployment Steps

#### 1. Database Migration

```bash
# Backup current database
pg_dump -h $DB_HOST -U $DB_USER $DB_NAME > backup-$(date +%Y%m%d).sql

# Run migrations
./scripts/migrate.sh

# Verify migration
./scripts/verify-migration.sh
```

#### 2. Blue-Green Deployment

```bash
# Deploy to staging environment first
docker-compose -f docker-compose.staging.yml up -d

# Run health checks
curl -f http://staging.yourdomain.com/health

# Switch traffic to new version
# Update load balancer or DNS

# Monitor for issues
# Rollback if necessary
```

#### 3. Rolling Deployment

```bash
# For ECS/Kubernetes deployments
# Update service with new image
# ECS will gradually replace instances

# Monitor deployment progress
aws ecs describe-services --cluster your-cluster --services imi-backend
```

### Post-deployment Verification

```bash
# Health check
curl -f https://api.yourdomain.com/health

# Functionality tests
curl -f https://api.yourdomain.com/v1/ip-assets?limit=5

# Performance check
ab -n 100 -c 10 https://api.yourdomain.com/health

# Monitor logs
docker-compose logs -f backend
```

## Monitoring & Logging

### Application Monitoring

#### Health Checks
```bash
# Kubernetes health check
apiVersion: v1
kind: Pod
spec:
  containers:
  - name: backend
    livenessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 30
      periodSeconds: 10
    readinessProbe:
      httpGet:
        path: /health
        port: 8080
      initialDelaySeconds: 5
      periodSeconds: 5
```

#### Metrics Collection

**Prometheus Configuration:**
```yaml
# prometheus.yml
global:
  scrape_interval: 15s

scrape_configs:
  - job_name: 'imi-backend'
    static_configs:
      - targets: ['backend:8080']
    metrics_path: /metrics
    scrape_interval: 5s
```

**Grafana Dashboard:**
- API response times
- Error rates
- Database connection pool
- Memory and CPU usage
- Request volume

### Log Management

#### Structured Logging
```bash
# Configure log output format
export LOG_FORMAT=json
export LOG_LEVEL=info

# Centralized logging with ELK stack
# Elasticsearch + Logstash + Kibana
```

#### Log Aggregation

**Fluentd Configuration:**
```yaml
<source>
  @type tail
  path /var/log/imi/*.log
  pos_file /var/log/fluentd/imi.log.pos
  tag imi
  format json
</source>

<match imi>
  @type elasticsearch
  host elasticsearch.logging.svc.cluster.local
  port 9200
  index_name imi
</match>
```

### Alerting

#### Critical Alerts
- API response time > 1 second
- Error rate > 5%
- Database connection failures
- Disk space > 90%
- Memory usage > 90%

#### Alert Configuration (Prometheus)
```yaml
groups:
  - name: imi
    rules:
      - alert: HighErrorRate
        expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.1
        for: 5m
        labels:
          severity: critical
        annotations:
          summary: High error rate detected
```

## Security Considerations

### Production Security Checklist

#### Network Security
- [ ] HTTPS/TLS 1.3 enabled
- [ ] Security headers configured
- [ ] CORS properly configured
- [ ] Rate limiting enabled
- [ ] DDoS protection active
- [ ] VPN/VPC access for admin functions

#### Application Security
- [ ] JWT secrets properly secured
- [ ] Database credentials encrypted
- [ ] File upload validation active
- [ ] Input sanitization enabled
- [ ] SQL injection prevention verified
- [ ] XSS protection headers set

#### Infrastructure Security
- [ ] Security groups configured
- [ ] Database encryption at rest
- [ ] Backup encryption enabled
- [ ] Access logs enabled
- [ ] Intrusion detection active
- [ ] Regular security scans scheduled

### Security Headers

```nginx
# nginx.conf security headers
add_header X-Frame-Options DENY;
add_header X-Content-Type-Options nosniff;
add_header X-XSS-Protection "1; mode=block";
add_header Strict-Transport-Security "max-age=31536000; includeSubDomains";
add_header Content-Security-Policy "default-src 'self'";
add_header Referrer-Policy "strict-origin-when-cross-origin";
```

### SSL/TLS Configuration

```bash
# Let's Encrypt SSL certificate
certbot --nginx -d api.yourdomain.com

# Or using cert-manager in Kubernetes
kubectl apply -f https://github.com/jetstack/cert-manager/releases/download/v1.13.0/cert-manager.yaml
```

## Troubleshooting

### Common Issues

#### 1. Database Connection Issues

**Problem:** `connection refused` or `timeout`

**Solutions:**
```bash
# Check database status
systemctl status postgresql

# Verify connection settings
psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME

# Check firewall rules
sudo ufw status
```

#### 2. Memory Issues

**Problem:** Out of memory errors

**Solutions:**
```bash
# Check memory usage
free -h
docker stats

# Increase container memory limits
docker run -m 2g your-image

# Optimize database queries
EXPLAIN ANALYZE SELECT * FROM your_table;
```

#### 3. File Upload Issues

**Problem:** File uploads failing

**Solutions:**
```bash
# Check S3 credentials
aws s3 ls s3://your-bucket

# Verify file permissions
ls -la uploads/

# Check nginx upload limits
client_max_body_size 50M;
```

#### 4. Performance Issues

**Problem:** Slow API responses

**Solutions:**
```bash
# Check database performance
SELECT * FROM pg_stat_activity;

# Monitor Redis cache hit rate
redis-cli info stats

# Analyze slow queries
tail -f /var/log/postgresql/postgresql.log
```

### Debug Commands

```bash
# Container debugging
docker exec -it container_name /bin/sh

# Check application logs
docker logs -f container_name

# Database debugging
docker exec -it postgres_container psql -U postgres -d ip_marketplace

# Redis debugging
docker exec -it redis_container redis-cli
```

### Recovery Procedures

#### Database Recovery
```bash
# Restore from backup
pg_restore -h $DB_HOST -U $DB_USER -d $DB_NAME backup.sql

# Point-in-time recovery (if available)
aws rds restore-db-instance-to-point-in-time \
  --source-db-instance-identifier original-db \
  --target-db-instance-identifier restored-db \
  --restore-time 2024-01-15T10:00:00Z
```

#### Application Recovery
```bash
# Rollback deployment
docker-compose -f docker-compose.prod.yml down
docker-compose -f docker-compose.prod.yml up -d --no-deps backend

# Scale down/up for restart
docker-compose -f docker-compose.prod.yml scale backend=0
docker-compose -f docker-compose.prod.yml scale backend=2
```

### Support and Maintenance

#### Regular Maintenance Tasks
- Database vacuum and analyze (weekly)
- Log rotation and cleanup (daily)
- Security updates (monthly)
- Backup verification (weekly)
- Performance monitoring review (weekly)
- SSL certificate renewal (automatic with Let's Encrypt)

#### Emergency Contacts
- **DevOps Team**: devops@imi.infoecos.ai
- **Database Admin**: dba@imi.infoecos.ai
- **Security Team**: security@imi.infoecos.ai
- **On-call Engineer**: +1-xxx-xxx-xxxx

For additional support, refer to the [README.md](README.md) or create an issue in the project repository.