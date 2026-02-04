# Database Operations Guide

Database migrations, backups, seeding, and operations for the Ride Hailing platform.

## Migration Management

We use [golang-migrate](https://github.com/golang-migrate/migrate) for schema migrations.

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
make migrate-create NAME=add_user_preferences

# Or manually:
migrate create -ext sql -dir db/migrations -seq add_user_preferences
```

### Running Migrations

```bash
make migrate-up              # Apply all pending migrations
make migrate-down            # Rollback last migration
make migrate-version         # Check current version
make migrate-force VERSION=5 # Force to specific version (use with caution!)
```

### Testing Migrations

```bash
./scripts/test-migrations.sh              # Test all migrations (including rollbacks)
./scripts/test-migrations.sh --verbose    # Verbose output
./scripts/test-migrations.sh --clean      # Clean database after test
```

### Troubleshooting Migrations

**"Dirty database" error:**
```bash
make migrate-version          # Check status
make migrate-force VERSION=5  # Force to last known good version
make migrate-up               # Re-run
```

**Lock conflicts during migration:**
```sql
SELECT * FROM pg_locks WHERE NOT granted;
SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE pid = 12345;
```

---

## Backup and Restore

### Full Backup

```bash
./scripts/backup-database.sh                          # Basic backup
./scripts/backup-database.sh --compress               # Compressed
./scripts/backup-database.sh --compress --encrypt      # Encrypted
./scripts/backup-database.sh --compress --storage s3   # Upload to S3
./scripts/backup-database.sh --compress --storage gcs  # Upload to GCS
./scripts/backup-database.sh --retention 90            # Custom retention
```

### Restore from Backup

```bash
./scripts/restore-database.sh                                                          # Restore latest
./scripts/restore-database.sh --file backups/backup_ridehailing_20240115_120000.sql.gz  # From file
./scripts/restore-database.sh --from-remote --storage s3 --latest                      # From S3
./scripts/restore-database.sh --timestamp "2024-01-15 12:00:00"                        # To timestamp
./scripts/restore-database.sh --file backups/backup.sql.gz --validate-only             # Validate only
./scripts/restore-database.sh --new-database --target-database ridehailing_restored    # To new DB
```

### Automated Backups

```bash
# Cron: daily backup at 2 AM
0 2 * * * /path/to/ride-hailing/scripts/backup-database.sh --compress --storage s3 >> /var/log/db-backup.log 2>&1

# Kubernetes CronJob
kubectl apply -f deploy/cronjobs/database-backup-cronjob.yaml
```

### Backup Verification

```bash
./scripts/check-backup-health.sh
./scripts/check-backup-health.sh --alert-email ops@ridehailing.com
./scripts/check-backup-health.sh --slack-webhook https://hooks.slack.com/services/YOUR/WEBHOOK
```

### Point-in-Time Recovery

See [DATABASE_PITR.md](./DATABASE_PITR.md) for full PITR documentation.

```bash
./scripts/pitr-restore.sh --timestamp "2024-01-15 14:30:00"
psql -c "SELECT pg_create_restore_point('before_migration');"
```

---

## Database Seeding

### Available Seed Scripts

| Level | Data | Command |
|-------|------|---------|
| **Light** (Dev) | 11 users, 9 rides | `make db-seed` or `psql -d ridehailing -f scripts/seed-database.sql` |
| **Medium** (Test) | 50 users, 200 rides | `psql -d ridehailing -f scripts/seed-medium.sql` |
| **Heavy** (Load) | 1000 users, 5000 rides | `psql -d ridehailing -f scripts/seed-heavy.sql` |

- **Light**: 5 riders, 5 drivers, 1 admin, 9 rides (mixed statuses), 5 payments. Password: `password123`
- **Medium**: 30 riders, 20 drivers, promo codes, referrals, driver location history
- **Heavy**: 500 riders, 200 drivers, 5000 full-lifecycle rides, ML ETA predictions, wallet transactions

### Reset and Seed

```bash
make db-reset  # Drops, recreates, migrates, and seeds
```

---

## Monitoring

### Health Check Endpoints

- `/healthz` - Basic liveness check
- `/health/live` - Kubernetes liveness probe
- `/health/ready` - Kubernetes readiness probe (checks dependencies)

### Useful Queries

```sql
-- Active connections
SELECT count(*) FROM pg_stat_activity WHERE state = 'active';

-- Long running queries (> 5 min)
SELECT pid, now() - query_start AS duration, query
FROM pg_stat_activity
WHERE state = 'active' AND now() - query_start > interval '5 minutes';

-- Database size
SELECT pg_size_pretty(pg_database_size('ridehailing'));
```

---

## References

- [golang-migrate Documentation](https://github.com/golang-migrate/migrate)
- [Point-in-Time Recovery Guide](./DATABASE_PITR.md)
- [Disaster Recovery Runbook](./DISASTER_RECOVERY.md)
