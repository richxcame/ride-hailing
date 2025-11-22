# Database Operations Guide

Complete guide for database migrations, backups, and operations for the Ride Hailing platform.

## Table of Contents

- [Migration Management](#migration-management)
- [Backup and Restore](#backup-and-restore)
- [Point-in-Time Recovery](#point-in-time-recovery)
- [Database Seeding](#database-seeding)
- [Monitoring and Health](#monitoring-and-health)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## Migration Management

### Overview

We use [golang-migrate](https://github.com/golang-migrate/migrate) for database schema migrations.

### Migration File Structure

```
db/migrations/
├── 000001_init_schema.up.sql
├── 000001_init_schema.down.sql
├── 000002_add_missing_tables.up.sql
├── 000002_add_missing_tables.down.sql
└── ...
```

**Naming Convention**: `NNNNNN_description.up.sql` and `NNNNNN_description.down.sql`

### Creating a New Migration

```bash
# Using Makefile
make migrate-create NAME=add_user_preferences

# Manual creation
migrate create -ext sql -dir db/migrations -seq add_user_preferences
```

This creates:
- `000010_add_user_preferences.up.sql` (schema changes)
- `000010_add_user_preferences.down.sql` (rollback changes)

### Migration Template

**UP Migration** (`*.up.sql`):
```sql
-- Description: Add user preferences table
-- Author: Your Name
-- Date: 2024-01-15

BEGIN;

CREATE TABLE IF NOT EXISTS user_preferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    theme VARCHAR(20) DEFAULT 'light',
    language VARCHAR(10) DEFAULT 'en',
    notifications_enabled BOOLEAN DEFAULT true,
    created_at TIMESTAMP NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP NOT NULL DEFAULT NOW(),
    UNIQUE(user_id)
);

CREATE INDEX idx_user_preferences_user_id ON user_preferences(user_id);

COMMENT ON TABLE user_preferences IS 'User application preferences';

COMMIT;
```

**DOWN Migration** (`*.down.sql`):
```sql
-- Rollback: Remove user preferences table

BEGIN;

DROP TABLE IF EXISTS user_preferences CASCADE;

COMMIT;
```

### Running Migrations

```bash
# Apply all pending migrations
make migrate-up

# Rollback last migration
make migrate-down

# Check current version
make migrate-version

# Force to specific version (use with caution!)
make migrate-force VERSION=5
```

### Testing Migrations

```bash
# Test all migrations (including rollbacks)
./scripts/test-migrations.sh

# Test with verbose output
./scripts/test-migrations.sh --verbose

# Skip rollback tests
./scripts/test-migrations.sh --skip-rollback

# Clean database after test
./scripts/test-migrations.sh --clean
```

### Migration Best Practices

1. **Always create both UP and DOWN migrations**
2. **Use transactions** (BEGIN/COMMIT) for atomic changes
3. **Add comments** to document purpose and changes
4. **Test rollbacks** before deploying
5. **Use IF EXISTS/IF NOT EXISTS** for idempotency
6. **Avoid data modifications** in schema migrations (use separate data migration scripts)
7. **Create indexes CONCURRENTLY** in production to avoid locking

Example of concurrent index creation:
```sql
-- This can run without locking the table
CREATE INDEX CONCURRENTLY idx_rides_status ON rides(status);
```

### Migration Checklist

Before deploying migrations to production:

- [ ] Both UP and DOWN migrations created
- [ ] Migrations tested locally
- [ ] Rollback tested
- [ ] Peer reviewed
- [ ] No data loss in DOWN migration
- [ ] Indexes created CONCURRENTLY if needed
- [ ] Large tables analyzed for performance impact
- [ ] Backup created before deployment

---

## Backup and Restore

### Backup Types

1. **Full Backup**: Complete database dump
2. **Incremental**: Only changes since last backup (via WAL archiving)
3. **Point-in-Time**: Combination of full backup + WAL files

### Full Backup

```bash
# Basic backup
./scripts/backup-database.sh

# Compressed backup
./scripts/backup-database.sh --compress

# Encrypted backup
./scripts/backup-database.sh --compress --encrypt

# Upload to S3
./scripts/backup-database.sh --compress --storage s3

# Upload to GCS
./scripts/backup-database.sh --compress --storage gcs

# Custom retention period
./scripts/backup-database.sh --retention 90
```

### Restore from Backup

```bash
# Restore latest backup
./scripts/restore-database.sh

# Restore from specific file
./scripts/restore-database.sh --file backups/backup_ridehailing_20240115_120000.sql.gz

# Restore from S3
./scripts/restore-database.sh --from-remote --storage s3 --latest

# Restore to specific timestamp
./scripts/restore-database.sh --timestamp "2024-01-15 12:00:00"

# Validate backup without restoring
./scripts/restore-database.sh --file backups/backup.sql.gz --validate-only

# Restore to new database
./scripts/restore-database.sh --new-database --target-database ridehailing_restored
```

### Automated Backups

#### Using Cron (Traditional Servers)

```bash
# Install cron schedule
crontab -e

# Add daily backup at 2 AM
0 2 * * * /path/to/ride-hailing/scripts/backup-database.sh --compress --storage s3 >> /var/log/db-backup.log 2>&1
```

#### Using Kubernetes CronJob

```bash
# Deploy backup CronJob
kubectl apply -f deploy/cronjobs/database-backup-cronjob.yaml

# Check CronJob status
kubectl get cronjobs

# View recent backup jobs
kubectl get jobs | grep database-backup

# View logs from last backup
kubectl logs -l app=database-backup --tail=100
```

### Backup Verification

```bash
# Check backup health
./scripts/check-backup-health.sh

# Check with email alerts
./scripts/check-backup-health.sh --alert-email ops@ridehailing.com

# Check with Slack alerts
./scripts/check-backup-health.sh --slack-webhook https://hooks.slack.com/services/YOUR/WEBHOOK

# Verbose output
./scripts/check-backup-health.sh --verbose
```

---

## Point-in-Time Recovery

See [DATABASE_PITR.md](./DATABASE_PITR.md) for comprehensive PITR documentation.

### Quick PITR Commands

```bash
# Restore to specific timestamp
./scripts/pitr-restore.sh --timestamp "2024-01-15 14:30:00"

# Restore to named restore point
psql -c "SELECT pg_create_restore_point('before_migration');"
./scripts/pitr-restore.sh --restore-point "before_migration"

# Check WAL archiving status
psql -c "SELECT * FROM pg_stat_archiver;"
```

---

## Database Seeding

### Available Seed Scripts

1. **Light** (Development): 11 users, 9 rides
   ```bash
   psql -d ridehailing -f scripts/seed-database.sql
   # Or: make db-seed
   ```

2. **Medium** (Testing): 50 users, 200 rides
   ```bash
   psql -d ridehailing -f scripts/seed-medium.sql
   ```

3. **Heavy** (Load Testing): 1000 users, 5000 rides
   ```bash
   psql -d ridehailing -f scripts/seed-heavy.sql
   ```

### Seed Data Details

#### Light Seed
- 5 riders, 5 drivers, 1 admin
- 9 sample rides (completed, in-progress, requested, cancelled)
- 5 payments
- Basic wallet data
- Password: `password123` for all users

#### Medium Seed
- 30 riders, 20 drivers, 2 admins
- 200 rides with realistic distribution
- Promo codes, referrals
- Driver location history
- Notifications

#### Heavy Seed
- 500 riders, 200 drivers, 10 admins
- 5000 rides with full lifecycle
- Comprehensive promo/referral data
- Extensive location tracking
- ML ETA predictions
- Wallet transactions

### Custom Seeding

```bash
# Reset and seed
make db-reset  # Drops, recreates, migrates, and seeds

# Seed only
psql -d ridehailing -f scripts/seed-database.sql
```

---

## Monitoring and Health

### Health Check Endpoints

All services expose health endpoints:

- `/healthz` - Basic liveness check
- `/health/live` - Kubernetes liveness probe
- `/health/ready` - Kubernetes readiness probe (checks dependencies)

### Database Monitoring

```sql
-- Active connections
SELECT count(*) FROM pg_stat_activity WHERE state = 'active';

-- Long running queries
SELECT pid, now() - pg_stat_activity.query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' AND now() - pg_stat_activity.query_start > interval '5 minutes';

-- Database size
SELECT pg_size_pretty(pg_database_size('ridehailing'));

-- Table sizes
SELECT
    schemaname,
    tablename,
    pg_size_pretty(pg_total_relation_size(schemaname||'.'||tablename)) AS size
FROM pg_tables
WHERE schemaname = 'public'
ORDER BY pg_total_relation_size(schemaname||'.'||tablename) DESC
LIMIT 20;

-- Index usage
SELECT
    schemaname,
    tablename,
    indexname,
    idx_scan,
    pg_size_pretty(pg_relation_size(indexrelid)) AS size
FROM pg_stat_user_indexes
ORDER BY idx_scan ASC, pg_relation_size(indexrelid) DESC
LIMIT 20;

-- Vacuum and analyze status
SELECT
    schemaname,
    relname,
    last_vacuum,
    last_autovacuum,
    last_analyze,
    last_autoanalyze
FROM pg_stat_user_tables
ORDER BY last_autovacuum DESC NULLS LAST;
```

### Backup Monitoring

```bash
# Check last backup age
find backups/ -name "backup_*.sql.gz" -type f -mtime -1

# Check S3 backups
aws s3 ls s3://ridehailing-backups/database-backups/ --recursive | tail -5

# Monitor backup size trends
du -sh backups/backup_*.sql.gz | tail -10
```

---

## Best Practices

### Development

1. **Always use migrations** - Never modify database schema directly
2. **Test locally first** - Run migrations on local database before staging
3. **Keep seeds updated** - Update seed data when schema changes
4. **Document changes** - Add clear comments to migrations

### Staging

1. **Test migrations** - Run full migration test suite
2. **Backup before migrate** - Create safety backup
3. **Monitor performance** - Check query performance after migrations
4. **Test rollbacks** - Ensure DOWN migrations work correctly

### Production

1. **Scheduled maintenance** - Run migrations during low-traffic periods
2. **Create backup first** - Always backup before schema changes
3. **Monitor during deployment** - Watch logs and metrics
4. **Have rollback plan** - Know how to quickly revert changes
5. **Use CONCURRENTLY** - For index creation on large tables
6. **Test on staging first** - Never deploy untested migrations

### Backup Strategy

1. **Daily full backups** - Automated at 2 AM
2. **Continuous WAL archiving** - For point-in-time recovery
3. **Weekly retention** - Keep 7 days of daily backups locally
4. **Monthly archives** - Long-term storage in S3/GCS
5. **Test restores monthly** - Verify backup integrity
6. **Off-site storage** - Use cloud storage for disaster recovery

---

## Troubleshooting

### Migration Issues

**Issue**: Migration fails with "dirty database" error
```bash
# Check migration status
make migrate-version

# Force to last known good version
make migrate-force VERSION=5

# Then re-run migration
make migrate-up
```

**Issue**: Migration timeout
```sql
-- Increase statement timeout
SET statement_timeout = '30min';
```

**Issue**: Lock conflicts during migration
```sql
-- Check for locks
SELECT * FROM pg_locks WHERE NOT granted;

-- Kill blocking session (use with caution!)
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid = 12345;
```

### Backup/Restore Issues

**Issue**: Backup file is empty
```bash
# Check database connectivity
psql -d ridehailing -c '\conninfo'

# Check disk space
df -h

# Run backup with verbose output
./scripts/backup-database.sh --verbose
```

**Issue**: Restore fails with permission errors
```bash
# Restore without owner information
pg_restore --no-owner --no-acl -d ridehailing backup.dump
```

**Issue**: Cannot find WAL file for PITR
```bash
# Check WAL archive
ls -lh /var/lib/postgresql/wal_archive/

# Check S3 WAL archive
aws s3 ls s3://ridehailing-wal-archive/wal-files/

# Download missing WAL file
aws s3 cp s3://ridehailing-wal-archive/wal-files/000000010000000000000042 /var/lib/postgresql/wal_archive/
```

### Performance Issues

**Issue**: Slow queries after migration
```sql
-- Analyze tables to update statistics
ANALYZE;

-- Reindex if needed
REINDEX INDEX idx_rides_status;

-- Vacuum to reclaim space
VACUUM ANALYZE rides;
```

**Issue**: Connection pool exhaustion
```sql
-- Check active connections
SELECT count(*), state FROM pg_stat_activity GROUP BY state;

-- Increase max_connections (requires restart)
ALTER SYSTEM SET max_connections = 200;
```

---

## Quick Reference

```bash
# Migrations
make migrate-up              # Apply pending migrations
make migrate-down            # Rollback last migration
make migrate-create NAME=x   # Create new migration
./scripts/test-migrations.sh # Test all migrations

# Backups
./scripts/backup-database.sh --compress --storage s3
./scripts/restore-database.sh --latest
./scripts/check-backup-health.sh

# Seeding
make db-seed                 # Light seed
psql -f scripts/seed-medium.sql
psql -f scripts/seed-heavy.sql

# Reset
make db-reset                # Drop, create, migrate, seed

# Monitoring
make db-status               # Show database info
psql -c "SELECT * FROM pg_stat_activity;"
```

---

## References

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [PostgreSQL Backup Documentation](https://www.postgresql.org/docs/current/backup.html)
- [Point-in-Time Recovery Guide](./DATABASE_PITR.md)
- [Disaster Recovery Runbook](./DISASTER_RECOVERY.md)

---

**Last Updated**: 2024-01-15
