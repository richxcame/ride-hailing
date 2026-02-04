# Self-Hosted Sentry Setup

Self-hosted Sentry for error tracking across all ride-hailing services.

**Requirements**: Docker, 8GB+ RAM, 20GB+ disk.

## Quick Start

```bash
# 1. Run the setup script (generates keys, runs migrations, creates admin user)
./scripts/setup-sentry.sh

# 2. Open http://localhost:9000 and login

# 3. Create a project: platform "Go", name "ride-hailing"

# 4. Copy the DSN and add to .env:
#    SENTRY_DSN=http://[public-key]@localhost:9000/[project-id]

# 5. Restart services
docker-compose restart

# 6. Test
curl http://localhost:8081/test/sentry
```

## Architecture

| Component | Purpose |
|-----------|---------|
| sentry (:9000) | Web UI |
| sentry-worker | Background task processor |
| sentry-cron | Scheduled tasks |
| sentry-postgres | User data, projects, issues |
| sentry-redis | Cache and task queue |
| sentry-clickhouse | Event storage |
| sentry-kafka + zookeeper | Event streaming |

## Service Configuration

Use a **single project** for all services (recommended). Services are differentiated automatically by the `SERVICE_NAME` tag.

```bash
# .env
SENTRY_DSN=http://abc123@localhost:9000/1
SENTRY_SAMPLE_RATE=1.0
SENTRY_TRACES_SAMPLE_RATE=1.0
SENTRY_DEBUG=false
```

## Maintenance

```bash
# View logs
docker-compose logs -f sentry sentry-worker sentry-cron

# Backup Postgres
docker-compose exec sentry-postgres pg_dump -U sentry sentry > sentry-backup-$(date +%Y%m%d).sql

# Cleanup old events
docker-compose exec sentry sentry cleanup --days 30

# Update Sentry
./scripts/backup-sentry.sh
docker-compose pull sentry sentry-worker sentry-cron
docker-compose exec sentry sentry upgrade
docker-compose up -d
```

## Troubleshooting

**UI not loading**: Check `docker-compose logs sentry` and ensure port 9000 is free.

**Errors not appearing**: Verify `SENTRY_DSN` is set, check `docker-compose logs sentry-worker`, and test with `curl http://localhost:8081/test/sentry`.

**High memory**: Reduce ClickHouse buffer sizes or limit Kafka retention in `docker-compose.yml`.

## Production Notes

- Change default passwords for sentry-postgres and sentry-redis
- Enable HTTPS via reverse proxy (nginx/traefik)
- Configure email notifications in Sentry settings
- Set up automated backups for sentry-postgres and ClickHouse volumes
- Consider using managed PostgreSQL/Redis for HA deployments

See also: [SENTRY_QUICKSTART.md](SENTRY_QUICKSTART.md) for the 5-minute setup version.
