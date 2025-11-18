## Self-Hosted Sentry Setup Guide

This guide explains how to set up and use the self-hosted Sentry error tracking platform included in your ride-hailing infrastructure.

## Table of Contents

1. [Why Self-Host Sentry?](#why-self-host-sentry)
2. [Architecture Overview](#architecture-overview)
3. [Quick Start](#quick-start)
4. [Detailed Setup](#detailed-setup)
5. [Creating a Project](#creating-a-project)
6. [Configuring Services](#configuring-services)
7. [Maintenance & Operations](#maintenance--operations)
8. [Troubleshooting](#troubleshooting)
9. [Production Deployment](#production-deployment)

---

## Why Self-Host Sentry?

### Benefits

-   **Complete Data Control** - All error data stays within your infrastructure
-   **No Usage Limits** - Unlimited errors, events, and team members
-   **Cost Effective** - No per-event or per-user pricing at scale
-   **Privacy & Compliance** - Full GDPR/HIPAA compliance control
-   **Customization** - Modify and extend Sentry as needed
-   **No External Dependencies** - Works in air-gapped environments

### Comparison: Self-Hosted vs SaaS

| Feature          | Self-Hosted         | Sentry SaaS       |
| ---------------- | ------------------- | ----------------- |
| Cost             | Infrastructure only | $26+/month        |
| Events/month     | Unlimited           | 50K-1M+           |
| Data retention   | Configurable        | 90 days           |
| Data privacy     | Complete control    | Sentry has access |
| Setup complexity | Moderate            | Very easy         |
| Maintenance      | Self-managed        | Fully managed     |
| Updates          | Manual              | Automatic         |

---

## Architecture Overview

The self-hosted Sentry stack consists of:

### Core Services

1. **sentry** - Main web application (Port 9000)
2. **sentry-worker** - Background task processor
3. **sentry-cron** - Scheduled task runner

### Data Stores

4. **sentry-postgres** - Primary database (user data, projects, issues)
5. **sentry-redis** - Cache and task queue
6. **sentry-clickhouse** - Event storage and analytics
7. **sentry-kafka** - Event streaming pipeline
8. **sentry-zookeeper** - Kafka coordination

### Data Flow

```
Services → Sentry SDK → Kafka → Worker → ClickHouse/PostgreSQL
                                            → Redis (cache)
                                            → Sentry Web UI
```

### Resource Requirements

**Minimum (Development)**:

-   CPU: 4 cores
-   RAM: 8GB
-   Disk: 20GB

**Recommended (Production)**:

-   CPU: 8+ cores
-   RAM: 16GB+
-   Disk: 100GB+ SSD

---

## Quick Start

### Prerequisites

-   Docker & Docker Compose installed
-   At least 8GB RAM available
-   20GB+ free disk space

### 1. Run Setup Script

```bash
./scripts/setup-sentry.sh
```

This script will:

1. Generate a secret key
2. Start Sentry dependencies
3. Run database migrations
4. Create admin user
5. Start all Sentry services

**Follow the prompts to create your admin account.**

### 2. Access Sentry

Open your browser: **http://localhost:9000**

Login with the credentials you just created.

### 3. Create Project

1. Click "Create Project"
2. Select platform: **Go**
3. Name: `ride-hailing`
4. Click "Create Project"

### 4. Get DSN

From the project settings, copy the DSN:

```
http://[public-key]@localhost:9000/[project-id]
```

### 5. Configure Services

Add to your `.env` file:

```bash
SENTRY_DSN=http://[your-public-key]@localhost:9000/1
```

### 6. Restart Services

```bash
docker-compose restart auth-service rides-service
# ... or restart all services
```

### 7. Test It!

Trigger a test error and check the Sentry dashboard:

```bash
curl http://localhost:8081/test/sentry
```

Within 5-10 seconds, you should see the error appear in Sentry!

---

## Detailed Setup

### Step 1: Generate Secret Key

The secret key is used for cryptographic operations. Generate one:

```bash
openssl rand -hex 32
```

Add to `.env`:

```bash
SENTRY_SECRET_KEY=your-generated-secret-key-here
```

### Step 2: Start Dependencies

```bash
docker-compose up -d sentry-postgres sentry-redis sentry-clickhouse sentry-zookeeper sentry-kafka
```

Wait for services to be healthy:

```bash
# Check PostgreSQL
docker-compose exec sentry-postgres pg_isready -U sentry

# Check Redis
docker-compose exec sentry-redis redis-cli ping
```

### Step 3: Run Migrations

Initialize the Sentry database:

```bash
docker-compose run --rm sentry upgrade --noinput
```

This takes 5-10 minutes on first run.

### Step 4: Create Superuser

```bash
docker-compose run --rm sentry createuser \
  --email admin@ridehailing.com \
  --password your-secure-password \
  --superuser \
  --no-input
```

### Step 5: Start Sentry Services

```bash
docker-compose up -d sentry sentry-worker sentry-cron
```

### Step 6: Verify Installation

Check logs:

```bash
docker-compose logs sentry | tail -50
```

Access web UI:

```bash
open http://localhost:9000
```

---

## Creating a Project

### Via Web UI

1. **Login** to http://localhost:9000
2. **Click** "Create Project"
3. **Select Platform**: Go
4. **Set Alert Frequency**: Default
5. **Name Project**: `ride-hailing`
6. **Assign Team**: Default
7. **Click** "Create Project"

### Get the DSN

After creating the project:

1. Go to **Settings** → **Client Keys (DSN)**
2. Copy the DSN URL
3. Format: `http://[public-key]@localhost:9000/[project-id]`

### Configure Teams & Projects

**Recommended Structure**:

```
Organization: RideHailing

Teams:
  - Backend      (backend services)
  - Frontend     (mobile/web apps)
  - DevOps       (infrastructure)

Projects:
  - auth-service
  - rides-service
  - payments-service
  - ... (one per service)
```

**Or use a single project** for all services (simpler):

```
Organization: RideHailing
Team: Engineering
Project: ride-hailing-platform
```

---

## Configuring Services

### Option 1: Single Project for All Services

Use the same DSN for all services:

```bash
# .env file
SENTRY_DSN=http://abc123@localhost:9000/1
```

Services are differentiated by the `SERVICE_NAME` tag automatically.

### Option 2: Separate Project Per Service

Create a project for each service and use service-specific DSNs:

```bash
# docker-compose.yml - auth-service
environment:
  SENTRY_DSN: http://abc123@localhost:9000/1

# docker-compose.yml - rides-service
environment:
  SENTRY_DSN: http://def456@localhost:9000/2
```

### Recommended: Single Project

For most use cases, **use a single project** and filter by service name in the Sentry UI.

Benefits:

-   Simpler configuration
-   Unified view of all errors
-   Easier to track cross-service issues
-   Better for distributed tracing correlation

---

## Maintenance & Operations

### Viewing Logs

```bash
# Sentry web server logs
docker-compose logs -f sentry

# Worker logs
docker-compose logs -f sentry-worker

# Cron logs
docker-compose logs -f sentry-cron

# All Sentry services
docker-compose logs -f sentry sentry-worker sentry-cron
```

### Database Backup

**PostgreSQL** (user data, projects, issues):

```bash
# Backup
docker-compose exec sentry-postgres pg_dump -U sentry sentry > sentry-backup-$(date +%Y%m%d).sql

# Restore
cat sentry-backup-20241112.sql | docker-compose exec -T sentry-postgres psql -U sentry sentry
```

**ClickHouse** (event data):

```bash
# Backup
docker-compose exec sentry-clickhouse clickhouse-client --query "BACKUP DATABASE sentry TO Disk('backups', 'backup-$(date +%Y%m%d).zip')"

# Or use volume snapshots
docker run --rm -v ridehailing_sentry_clickhouse_data:/data -v $(pwd):/backup alpine tar czf /backup/clickhouse-backup.tar.gz /data
```

### Data Retention

Configure data retention in Sentry:

1. Go to **Settings** → **Data & Privacy**
2. Set **Event Retention**: 30/60/90 days
3. Enable **Auto-cleanup**: Yes

Or manually via CLI:

```bash
# Delete events older than 30 days
docker-compose exec sentry sentry cleanup --days 30
```

### Updating Sentry

```bash
# Backup first!
./scripts/backup-sentry.sh

# Pull new image
docker-compose pull sentry sentry-worker sentry-cron

# Run migrations
docker-compose run --rm sentry upgrade

# Restart services
docker-compose up -d sentry sentry-worker sentry-cron
```

### Monitoring Sentry

**Health Checks**:

```bash
# Sentry web health
curl http://localhost:9000/_health/

# Database health
docker-compose exec sentry-postgres pg_isready -U sentry

# Redis health
docker-compose exec sentry-redis redis-cli ping
```

**Resource Usage**:

```bash
# Container stats
docker stats ridehailing-sentry ridehailing-sentry-worker ridehailing-sentry-cron

# Disk usage
docker system df -v | grep sentry
```

---

## Troubleshooting

### Sentry Web UI Not Loading

**Check container status**:

```bash
docker-compose ps sentry
```

**Check logs**:

```bash
docker-compose logs sentry | tail -100
```

**Common issues**:

-   Database not ready: Wait for PostgreSQL health check
-   Secret key not set: Check `SENTRY_SECRET_KEY` in .env
-   Port conflict: Port 9000 might be in use

### Errors Not Appearing

**Check worker logs**:

```bash
docker-compose logs sentry-worker | tail -50
```

**Verify DSN**:

-   Correct format: `http://[key]@localhost:9000/[project-id]`
-   Project ID matches created project
-   Network connectivity from services to Sentry

**Test connectivity**:

```bash
# From your service container
docker-compose exec auth-service curl http://sentry:9000/_health/
```

### High Memory Usage

**ClickHouse** is memory-intensive. Options:

1. **Increase memory limit**:

```yaml
# docker-compose.yml
sentry-clickhouse:
    mem_limit: 4g
```

2. **Configure ClickHouse**:

```yaml
# Create config file
sentry-clickhouse:
    volumes:
        - ./config/clickhouse-config.xml:/etc/clickhouse-server/config.d/docker.xml
```

3. **Reduce retention**:

-   Set shorter data retention (7-14 days)
-   Run cleanup more frequently

### Slow Performance

**Symptoms**: Slow web UI, delayed error processing

**Solutions**:

1. **Scale workers**:

```bash
docker-compose up -d --scale sentry-worker=3
```

2. **Increase resources**:

```yaml
sentry-worker:
    deploy:
        resources:
            limits:
                cpus: '2'
                memory: 2G
```

3. **Optimize database**:

```bash
# PostgreSQL vacuum
docker-compose exec sentry-postgres vacuumdb -U sentry -d sentry -z -v
```

### Database Connection Errors

**Check PostgreSQL logs**:

```bash
docker-compose logs sentry-postgres
```

**Check connections**:

```bash
docker-compose exec sentry-postgres psql -U sentry -c "SELECT count(*) FROM pg_stat_activity;"
```

**Increase connection limit** (if needed):

```yaml
sentry-postgres:
    command: postgres -c max_connections=200
```

---

## Production Deployment

### Security Hardening

#### 1. Change Default Passwords

```yaml
# docker-compose.yml
sentry-postgres:
  environment:
    POSTGRES_PASSWORD: ${SENTRY_DB_PASSWORD}  # Use strong password

# .env
SENTRY_DB_PASSWORD=your-very-strong-password-here
```

#### 2. Use Secrets Management

Store sensitive data in secrets:

```bash
# Use Docker secrets or Kubernetes secrets
echo "super-secret-key" | docker secret create sentry_secret_key -
```

#### 3. Enable HTTPS

Use a reverse proxy (Nginx, Traefik, Caddy):

```nginx
# nginx.conf
server {
    listen 443 ssl http2;
    server_name sentry.ridehailing.com;

    ssl_certificate /path/to/cert.pem;
    ssl_certificate_key /path/to/key.pem;

    location / {
        proxy_pass http://localhost:9000;
        proxy_set_header Host $host;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

Update Sentry DSN:

```bash
SENTRY_DSN=https://[key]@sentry.ridehailing.com/1
```

#### 4. Configure Email Notifications

```bash
# .env
SENTRY_MAIL_HOST=smtp.gmail.com
SENTRY_MAIL_PORT=587
SENTRY_MAIL_USERNAME=your-email@gmail.com
SENTRY_MAIL_PASSWORD=your-app-password
SENTRY_MAIL_USE_TLS=true
SENTRY_SERVER_EMAIL=sentry@ridehailing.com
```

#### 5. Set Up Backups

```bash
# Automated backup script
#!/bin/bash
# /opt/backup-sentry.sh

DATE=$(date +%Y%m%d_%H%M%S)
BACKUP_DIR=/backups/sentry

# PostgreSQL
docker-compose exec -T sentry-postgres pg_dump -U sentry sentry | gzip > $BACKUP_DIR/postgres-$DATE.sql.gz

# Sentry files
docker run --rm -v ridehailing_sentry_data:/data -v $BACKUP_DIR:/backup alpine tar czf /backup/files-$DATE.tar.gz /data

# Retention: keep last 30 days
find $BACKUP_DIR -name "*.gz" -mtime +30 -delete
```

Add to crontab:

```bash
# Daily backup at 2 AM
0 2 * * * /opt/backup-sentry.sh
```

### High Availability Setup

For production HA:

#### 1. External Databases

Use managed databases:

-   PostgreSQL: AWS RDS, Google Cloud SQL
-   Redis: AWS ElastiCache, Google Memorystore
-   ClickHouse: ClickHouse Cloud

#### 2. Load Balancing

```yaml
# docker-compose.yml
sentry:
    deploy:
        replicas: 3

sentry-worker:
    deploy:
        replicas: 5
```

#### 3. Persistent Volumes

Use network-attached storage:

```yaml
sentry:
    volumes:
        - type: nfs
          source: nfs-server:/sentry-data
          target: /var/lib/sentry/files
```

### Monitoring & Alerting

#### Setup Prometheus Exporter

```yaml
# docker-compose.yml
sentry-exporter:
    image: sentry-prometheus-exporter
    environment:
        SENTRY_DSN: ${SENTRY_DSN}
    ports:
        - '9091:9091'
```

#### Alert on Sentry Issues

```yaml
# prometheus/alerts.yml
- alert: SentryHighErrorRate
  expr: rate(sentry_events_total[5m]) > 100
  annotations:
      summary: 'High error rate in Sentry'

- alert: SentryDown
  expr: up{job="sentry"} == 0
  annotations:
      summary: 'Sentry is down'
```

---

## Performance Tuning

### For High Volume (>1M events/day)

#### 1. Increase Worker Count

```bash
docker-compose up -d --scale sentry-worker=10
```

#### 2. Optimize ClickHouse

```xml
<!-- clickhouse-config.xml -->
<yandex>
    <max_memory_usage>8589934592</max_memory_usage> <!-- 8GB -->
    <max_threads>8</max_threads>
</yandex>
```

#### 3. Tune Kafka

```yaml
sentry-kafka:
    environment:
        KAFKA_NUM_PARTITIONS: 10
        KAFKA_LOG_RETENTION_HOURS: 24
```

#### 4. Enable Redis Cluster

For very high volume, use Redis Cluster instead of single instance.

---

## Cost Analysis

### Self-Hosted Infrastructure Costs

**AWS Example (monthly)**:

-   EC2 instances (3x t3.large): $250
-   RDS PostgreSQL (db.t3.large): $150
-   ElastiCache Redis (cache.t3.medium): $75
-   EBS Storage (500GB): $50
-   Data transfer: $50

**Total: ~$575/month**

Compare to Sentry Business ($89/month for 50K events) or Enterprise ($1000+/month).

**Break-even**: ~100K-500K events/month

---

## Comparison Matrix

| Feature      | Self-Hosted      | Sentry SaaS      | Cloud Provider Errors |
| ------------ | ---------------- | ---------------- | --------------------- |
| Setup time   | 30 min - 1 hour  | 5 minutes        | Varies                |
| Monthly cost | Infrastructure   | $26-$1000+       | $0-$100               |
| Data privacy | Full control     | Sentry access    | Provider access       |
| Scalability  | Manual           | Automatic        | Automatic             |
| Maintenance  | Self-managed     | Fully managed    | Fully managed         |
| Features     | Full feature set | Full + managed   | Basic                 |
| Support      | Community        | Email/Chat/Phone | Email                 |
| SLA          | Self-managed     | 99.9% uptime     | Varies                |

---

## FAQ

### Q: Can I migrate from Sentry SaaS to self-hosted?

A: Yes, but data migration is not straightforward. You'll need to:

1. Export projects and settings
2. Set up self-hosted instance
3. Reconfigure SDK with new DSN
4. Historical data won't transfer automatically

### Q: How much disk space do I need?

A: Depends on event volume:

-   Low (1K events/day): 10GB
-   Medium (10K events/day): 50GB
-   High (100K events/day): 200GB+
-   Very High (1M+ events/day): 1TB+

Set retention policies to manage disk usage.

### Q: Can I run Sentry in Kubernetes?

A: Yes! Use the official Sentry Helm chart:

```bash
helm repo add sentry https://sentry-kubernetes.github.io/charts
helm install sentry sentry/sentry
```

### Q: What about GDPR compliance?

A: Self-hosted Sentry gives you full control:

-   Data stays in your infrastructure
-   You control retention policies
-   You manage data deletion requests
-   No data shared with third parties

### Q: Can I use this for production?

A: Yes! This setup is production-ready. For large scale:

-   Use external managed databases
-   Set up high availability
-   Implement proper backups
-   Add monitoring and alerting
-   Use HTTPS with proper SSL certs

---

## Support & Resources

-   **Sentry Official Docs**: https://docs.sentry.io/self-hosted/
-   **GitHub Repository**: https://github.com/getsentry/self-hosted
-   **Community Forum**: https://forum.sentry.io/
-   **Docker Hub**: https://hub.docker.com/_/sentry

---

## Summary

You now have a fully functional self-hosted Sentry instance!

**What you get**:

-   Complete error tracking platform
-   No usage limits or costs
-   Full data control
-   Privacy and compliance
-   Integration with all 13 microservices

**Next steps**:

1. Create a project
2. Copy the DSN
3. Add to .env file
4. Test error capture
5. Explore the Sentry UI

**For production**:

-   Enable HTTPS
-   Set up backups
-   Configure email notifications
-   Implement monitoring
-   Scale for your needs
