# Self-Hosted Sentry - Quick Start Guide

This guide covers setting up Sentry for both production (full `docker-compose.yml`) and development (lightweight `docker-compose.dev.yml`) environments.

## 🚀 Get Sentry Running in 5 Minutes

### Production Environment

For production setup with all services running in Docker:

### Step 1: Run Setup Script

```bash
./scripts/setup-sentry.sh
```

Follow the prompts to create your admin account.

### Step 2: Access Sentry

Open http://localhost:9000 and login.

---

### Development Environment

For development setup with optional Sentry profile:

### Step 1: Start Sentry with Profile

```bash
# Start Sentry infrastructure using the sentry profile
docker-compose -f docker-compose.dev.yml --profile sentry up -d
```

This starts only the Sentry components (Postgres, Redis, ClickHouse, Kafka, Zookeeper, Sentry server, worker, and cron).

### Step 2: Initialize Sentry

```bash
# Run the setup script to initialize Sentry database and create admin user
./scripts/setup-sentry.sh dev
```

Follow the prompts to create your admin account.

### Step 3: Access Sentry

Open http://localhost:9000 and login.

---

## Common Steps (Both Environments)

### Step 1: Create Project

1. Click "Create Project"
2. Select platform: **Go**
3. Name: `ride-hailing`
4. Click "Create Project"

### Step 2: Copy DSN

From Settings → Client Keys, copy the DSN:

```
http://[public-key]@localhost:9000/1
```

### Step 3: Configure Services

**For Production (docker-compose.yml):**

Add to `.env`:

```bash
SENTRY_DSN=http://[your-key]@localhost:9000/1
```

Then restart services:

```bash
docker-compose restart
```

**For Development (docker-compose.dev.yml):**

When running services natively with `make run-*`, export the DSN:

```bash
# Export for current terminal session
export SENTRY_DSN=http://[your-key]@localhost:9000/1

# Or add to .env file in project root
echo "SENTRY_DSN=http://[your-key]@localhost:9000/1" >> .env

# Run your services
make run-auth
make run-rides
# etc.
```

### Step 4: Test

```bash
curl http://localhost:8081/test/sentry
```

Check Sentry dashboard - error should appear!

## 📚 Full Documentation

See [docs/SELF_HOSTED_SENTRY.md](docs/SELF_HOSTED_SENTRY.md) for complete guide.

## 🎯 What You Get

-   ✅ Self-hosted error tracking
-   ✅ No usage limits
-   ✅ Complete data control
-   ✅ Zero cost (except infrastructure)
-   ✅ All 13 services integrated

## 🔧 Useful Commands

**For Production (docker-compose.yml):**

```bash
# View logs
docker-compose logs -f sentry

# Restart Sentry
docker-compose restart sentry sentry-worker sentry-cron

# Backup database
docker-compose exec sentry-postgres pg_dump -U sentry sentry > backup.sql

# Access shell
docker-compose exec sentry bash

# Clean old data
docker-compose exec sentry sentry cleanup --days 30
```

**For Development (docker-compose.dev.yml):**

```bash
# View logs
docker-compose -f docker-compose.dev.yml logs -f sentry

# Restart Sentry
docker-compose -f docker-compose.dev.yml restart sentry sentry-worker sentry-cron

# Stop Sentry profile
docker-compose -f docker-compose.dev.yml --profile sentry down

# Start Sentry profile
docker-compose -f docker-compose.dev.yml --profile sentry up -d

# Backup database
docker-compose -f docker-compose.dev.yml exec sentry-postgres pg_dump -U sentry sentry > backup.sql

# Access shell
docker-compose -f docker-compose.dev.yml exec sentry bash

# Clean old data
docker-compose -f docker-compose.dev.yml exec sentry sentry cleanup --days 30
```

## 🌐 Access Points

-   **Sentry UI**: http://localhost:9000
-   **Health Check**: http://localhost:9000/\_health/

## 📊 Resource Usage

-   **CPU**: 2-4 cores
-   **RAM**: 4-8GB
-   **Disk**: 20GB+ (grows with events)

## 💡 Tips

1. **Use one project** for all services - filter by service name
2. **Set data retention** to 30 days to save disk space
3. **Enable email notifications** for critical errors
4. **Back up regularly** - especially PostgreSQL and ClickHouse
5. **Monitor disk usage** - set up alerts

## 🆘 Troubleshooting

### Sentry not loading?

**Production:**

```bash
docker-compose logs sentry | tail -50
```

**Development:**

```bash
docker-compose -f docker-compose.dev.yml logs sentry | tail -50
```

### Errors not appearing?

**Production:**

```bash
# Check worker
docker-compose logs sentry-worker | tail -50

# Test connectivity from service container
docker-compose exec auth-service curl http://sentry:9000/_health/
```

**Development:**

```bash
# Check worker
docker-compose -f docker-compose.dev.yml logs sentry-worker | tail -50

# Test connectivity from host (services run natively)
curl http://localhost:9000/_health/
```

### High memory usage?

```bash
# Check stats (works for both environments)
docker stats ridehailing-sentry*

# Reduce retention
# In Sentry UI: Settings → Data & Privacy → Set to 14 days
```

### Port conflicts in development?

If port 9000 is already in use, you can change it in [docker-compose.dev.yml](docker-compose.dev.yml):

```yaml
sentry:
    ports:
        - '9001:9000' # Change external port
```

## 🎉 You're All Set!

Your self-hosted Sentry is ready to catch all those errors. Happy debugging!
