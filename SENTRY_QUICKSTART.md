# Self-Hosted Sentry - Quick Start Guide

## 🚀 Get Sentry Running in 5 Minutes

### Step 1: Run Setup Script

```bash
./scripts/setup-sentry.sh
```

Follow the prompts to create your admin account.

### Step 2: Access Sentry

Open http://localhost:9000 and login.

### Step 3: Create Project

1. Click "Create Project"
2. Select platform: **Go**
3. Name: `ride-hailing`
4. Click "Create Project"

### Step 4: Copy DSN

From Settings → Client Keys, copy the DSN:

```
http://[public-key]@localhost:9000/1
```

### Step 5: Configure Services

Add to `.env`:

```bash
SENTRY_DSN=http://[your-key]@localhost:9000/1
```

### Step 6: Restart Services

```bash
docker-compose restart
```

### Step 7: Test

```bash
curl http://localhost:8081/test/sentry
```

Check Sentry dashboard - error should appear!

## 📚 Full Documentation

See [docs/SELF_HOSTED_SENTRY.md](docs/SELF_HOSTED_SENTRY.md) for complete guide.

## 🎯 What You Get

- ✅ Self-hosted error tracking
- ✅ No usage limits
- ✅ Complete data control
- ✅ Zero cost (except infrastructure)
- ✅ All 13 services integrated

## 🔧 Useful Commands

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

## 🌐 Access Points

- **Sentry UI**: http://localhost:9000
- **Health Check**: http://localhost:9000/_health/

## 📊 Resource Usage

- **CPU**: 2-4 cores
- **RAM**: 4-8GB
- **Disk**: 20GB+ (grows with events)

## 💡 Tips

1. **Use one project** for all services - filter by service name
2. **Set data retention** to 30 days to save disk space
3. **Enable email notifications** for critical errors
4. **Back up regularly** - especially PostgreSQL and ClickHouse
5. **Monitor disk usage** - set up alerts

## 🆘 Troubleshooting

### Sentry not loading?

```bash
docker-compose logs sentry | tail -50
```

### Errors not appearing?

```bash
# Check worker
docker-compose logs sentry-worker | tail -50

# Test connectivity
docker-compose exec auth-service curl http://sentry:9000/_health/
```

### High memory usage?

```bash
# Check stats
docker stats ridehailing-sentry*

# Reduce retention
# In Sentry UI: Settings → Data & Privacy → Set to 14 days
```

## 🎉 You're All Set!

Your self-hosted Sentry is ready to catch all those errors. Happy debugging!
