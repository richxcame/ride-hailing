# Deployment Guide

## Table of Contents

-   [Local Development](#local-development)
-   [Docker Deployment](#docker-deployment)
-   [Cloud Deployment](#cloud-deployment)
-   [Environment Variables](#environment-variables)
-   [Database Setup](#database-setup)
-   [Monitoring Setup](#monitoring-setup)

## Local Development

### Prerequisites

-   Go 1.22+
-   PostgreSQL 15+
-   Redis 7+
-   Make (optional)

### Setup Steps

1. **Clone the repository**

    ```bash
    git clone https://github.com/richxcame/ride-hailing.git
    cd ride-hailing
    ```

2. **Install dependencies**

    ```bash
    go mod download
    make install-tools
    ```

3. **Start dependencies**

    ```bash
    docker-compose up postgres redis -d
    ```

4. **Configure environment**

    ```bash
    cp .env.example .env
    # Edit .env with your local configuration
    ```

5. **Run migrations**

    ```bash
    make migrate-up
    ```

6. **Start services**

    ```bash
    # Terminal 1
    make run-auth

    # Terminal 2
    make run-rides

    # Terminal 3
    make run-geo
    ```

## Docker Deployment

### Using Docker Compose

1. **Build and start all services**

    ```bash
    docker-compose up -d --build
    ```

2. **Run migrations**

    ```bash
    docker exec ridehailing-auth migrate -path /app/db/migrations \
      -database "postgresql://postgres:postgres@postgres:5432/ridehailing?sslmode=disable" up
    ```

3. **View logs**

    ```bash
    docker-compose logs -f
    ```

4. **Stop services**
    ```bash
    docker-compose down
    ```

### Individual Service Deployment

```bash
# Build image
docker build --build-arg SERVICE_NAME=auth -t ridehailing-auth:latest .

# Run container
docker run -d \
  --name ridehailing-auth \
  -p 8081:8080 \
  -e DB_HOST=postgres \
  -e DB_PASSWORD=postgres \
  ridehailing-auth:latest
```

## Cloud Deployment

### Google Cloud Platform (Cloud Run)

1. **Set up GCP project**

    ```bash
    gcloud config set project YOUR_PROJECT_ID
    ```

2. **Enable required APIs**

    ```bash
    gcloud services enable run.googleapis.com
    gcloud services enable sqladmin.googleapis.com
    gcloud services enable redis.googleapis.com
    ```

3. **Create Cloud SQL instance**

    ```bash
    gcloud sql instances create ridehailing-db \
      --database-version=POSTGRES_15 \
      --tier=db-f1-micro \
      --region=us-central1
    ```

4. **Create database**

    ```bash
    gcloud sql databases create ridehailing \
      --instance=ridehailing-db
    ```

5. **Build and push images**

    ```bash
    gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/auth-service
    gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/rides-service
    gcloud builds submit --tag gcr.io/YOUR_PROJECT_ID/geo-service
    ```

6. **Deploy to Cloud Run**
    ```bash
    gcloud run deploy auth-service \
      --image gcr.io/YOUR_PROJECT_ID/auth-service \
      --platform managed \
      --region us-central1 \
      --allow-unauthenticated \
      --add-cloudsql-instances YOUR_PROJECT_ID:us-central1:ridehailing-db \
      --set-env-vars DB_HOST=/cloudsql/YOUR_PROJECT_ID:us-central1:ridehailing-db
    ```

### AWS (ECS/Fargate)

1. **Create ECR repositories**

    ```bash
    aws ecr create-repository --repository-name ridehailing-auth
    aws ecr create-repository --repository-name ridehailing-rides
    aws ecr create-repository --repository-name ridehailing-geo
    ```

2. **Build and push images**

    ```bash
    aws ecr get-login-password --region us-east-1 | docker login --username AWS \
      --password-stdin ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com

    docker build --build-arg SERVICE_NAME=auth \
      -t ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/ridehailing-auth:latest .
    docker push ACCOUNT_ID.dkr.ecr.us-east-1.amazonaws.com/ridehailing-auth:latest
    ```

3. **Create RDS instance**

    ```bash
    aws rds create-db-instance \
      --db-instance-identifier ridehailing-db \
      --db-instance-class db.t3.micro \
      --engine postgres \
      --allocated-storage 20
    ```

4. **Create ECS cluster and service**
    ```bash
    aws ecs create-cluster --cluster-name ridehailing-cluster
    # Create task definitions and services using AWS Console or CLI
    ```

### Kubernetes Deployment

1. **Create namespace**

    ```bash
    kubectl create namespace ridehailing
    ```

2. **Create secrets**

    ```bash
    kubectl create secret generic ridehailing-secrets \
      --from-literal=db-password=your-password \
      --from-literal=jwt-secret=your-jwt-secret \
      -n ridehailing
    ```

3. **Deploy PostgreSQL**

    ```bash
    kubectl apply -f k8s/postgres-deployment.yaml
    ```

4. **Deploy services**
    ```bash
    kubectl apply -f k8s/auth-deployment.yaml
    kubectl apply -f k8s/rides-deployment.yaml
    kubectl apply -f k8s/geo-deployment.yaml
    ```

## Environment Variables

### Required Variables

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=your-password
DB_NAME=ridehailing

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379

# JWT
JWT_SECRET=your-secret-key
JWT_EXPIRATION=24
```

### Optional Variables

```bash
# Server
PORT=8080
ENVIRONMENT=production
READ_TIMEOUT=10
WRITE_TIMEOUT=10

# Database Pool
DB_MAX_CONNS=25
DB_MIN_CONNS=5

# Pub/Sub
PUBSUB_PROJECT_ID=your-project
PUBSUB_ENABLED=true

# Firebase
FIREBASE_PROJECT_ID=your-project
FIREBASE_CREDENTIALS_PATH=/path/to/credentials.json
FIREBASE_ENABLED=true
```

## Database Setup

### Running Migrations

**Using Make:**

```bash
make migrate-up
```

**Manual:**

```bash
migrate -path db/migrations \
  -database "postgresql://user:pass@host:5432/dbname?sslmode=disable" \
  up
```

### Creating Migrations

```bash
make migrate-create NAME=add_new_table
```

### Rollback

```bash
make migrate-down
```

## Monitoring Setup

### Prometheus

1. **Configure scrape targets** in `monitoring/prometheus.yml`

2. **Access Prometheus UI**
    ```
    http://localhost:9090
    ```

### Grafana

1. **Access Grafana**

    ```
    http://localhost:3000
    Username: admin
    Password: admin
    ```

2. **Add Prometheus datasource**

    - URL: `http://prometheus:9090`

3. **Import dashboards**
    - Go Dashboard ID: 14061
    - Custom metrics dashboard (create based on your metrics)

## SSL/TLS Configuration

### Using Let's Encrypt

1. **Install certbot**

    ```bash
    sudo apt-get install certbot
    ```

2. **Obtain certificate**

    ```bash
    sudo certbot certonly --standalone -d your-domain.com
    ```

3. **Update service configuration**
   Add SSL configuration to your reverse proxy (nginx/traefik)

## Backup and Recovery

### Database Backup

```bash
# Backup
pg_dump -h localhost -U postgres ridehailing > backup.sql

# Restore
psql -h localhost -U postgres ridehailing < backup.sql
```

### Automated Backups

Use cron or cloud provider's automated backup features.

## Scaling

### Horizontal Scaling

1. **Increase replicas in docker-compose**

    ```yaml
    auth-service:
        deploy:
            replicas: 3
    ```

2. **Add load balancer**
   Use nginx, HAProxy, or cloud load balancer

### Vertical Scaling

Adjust resource limits in docker-compose or k8s deployments:

```yaml
resources:
    limits:
        memory: 512Mi
        cpu: '1'
```

## Security Checklist

-   [ ] Change default passwords
-   [ ] Use strong JWT secret
-   [ ] Enable SSL/TLS
-   [ ] Set up firewall rules
-   [ ] Enable database SSL
-   [ ] Regular security updates
-   [ ] Implement rate limiting
-   [ ] Enable logging and monitoring
-   [ ] Regular backups
-   [ ] Secrets management (Vault/AWS Secrets Manager)

## Troubleshooting

### Service won't start

-   Check logs: `docker-compose logs service-name`
-   Verify database connectivity
-   Check environment variables

### Database connection issues

-   Verify PostgreSQL is running
-   Check connection string
-   Ensure network connectivity

### Performance issues

-   Check Prometheus metrics
-   Review database query performance
-   Scale services horizontally
