# Database Scripts Quick Reference

Quick reference for all database scripts in this directory.

## Migration Scripts

### test-migrations.sh
Test all database migrations including rollback validation.

```bash
./test-migrations.sh                # Test all migrations
./test-migrations.sh --verbose      # Detailed output
./test-migrations.sh --clean        # Clean database after test
./test-migrations.sh --skip-rollback  # Skip rollback tests
```

## Backup Scripts

### backup-database.sh
Create database backups with optional compression, encryption, and remote storage.

```bash
# Basic backup
./backup-database.sh

# Compressed backup
./backup-database.sh --compress

# Encrypted backup
./backup-database.sh --compress --encrypt

# Upload to S3
./backup-database.sh --compress --storage s3

# Upload to GCS
./backup-database.sh --compress --storage gcs

# Custom retention
./backup-database.sh --retention 90

# All options
./backup-database.sh --compress --encrypt --storage s3 --retention 30 --verbose
```

**Environment Variables:**
```bash
BACKUP_STORAGE_TYPE=s3              # s3, gcs, or azure
BACKUP_S3_BUCKET=your-bucket
BACKUP_S3_PREFIX=database-backups
BACKUP_GPG_RECIPIENT=backup@example.com
```

### restore-database.sh
Restore database from backup with validation.

```bash
# Restore latest backup
./restore-database.sh

# Restore from specific file
./restore-database.sh --file backups/backup_20240115.sql.gz

# Restore from S3
./restore-database.sh --from-remote --storage s3 --latest

# Restore to specific timestamp
./restore-database.sh --timestamp "2024-01-15 12:00:00"

# Validate only (no restore)
./restore-database.sh --file backup.sql.gz --validate-only

# Restore to new database
./restore-database.sh --new-database --target-database ridehailing_restored

# Skip confirmation prompts (dangerous!)
./restore-database.sh --no-confirm
```

### check-backup-health.sh
Monitor backup status and send alerts.

```bash
# Basic health check
./check-backup-health.sh

# With email alerts
./check-backup-health.sh --alert-email ops@example.com

# With Slack alerts
./check-backup-health.sh --slack-webhook https://hooks.slack.com/...

# Custom max backup age (in hours)
./check-backup-health.sh --max-age 24

# Verbose output
./check-backup-health.sh --verbose
```

**Environment Variables:**
```bash
BACKUP_ALERT_EMAIL=ops@example.com
BACKUP_SLACK_WEBHOOK=https://hooks.slack.com/...
```

## PITR (Point-in-Time Recovery) Scripts

### archive-wal.sh
Archive WAL files for point-in-time recovery (called automatically by PostgreSQL).

**PostgreSQL Configuration:**
```ini
archive_command = '/path/to/scripts/archive-wal.sh %p %f'
```

**Environment Variables:**
```bash
PITR_ENABLED=true
PITR_WAL_ARCHIVE_DIR=/var/lib/postgresql/wal_archive
PITR_STORAGE_TYPE=s3
PITR_S3_BUCKET=wal-archive-bucket
PITR_COMPRESS_WAL=true
```

## Seeding Scripts

### seed-database.sql (Light)
Light seed data for development (11 users, 9 rides).

```bash
psql -d ridehailing -f seed-database.sql
# Or: make db-seed
```

### seed-medium.sql (Medium)
Medium seed data for testing (52 users, 200 rides).

```bash
psql -d ridehailing -f seed-medium.sql
```

### seed-heavy.sql (Heavy)
Heavy seed data for load testing (710 users, 5000 rides).

```bash
psql -d ridehailing -f seed-heavy.sql
```

## Pre-commit Hooks

Located in `hooks/` directory.

### validate-migrations.sh
Validate migration files for common issues.

**Checks:**
- File exists and is readable
- File contains SQL statements
- DOWN migrations have corresponding UP migrations
- Dangerous operations without safeguards
- Missing comments
- Transaction statements

### check-migration-naming.sh
Check migration file naming convention.

**Expected Format:**
```
NNNNNN_description.up.sql
NNNNNN_description.down.sql
```

**Example:**
```
000001_create_users_table.up.sql
000001_create_users_table.down.sql
```

## Common Workflows

### Development Setup

```bash
# 1. Start database
docker-compose -f docker-compose.dev.yml up -d

# 2. Run migrations
make migrate-up

# 3. Seed database
make db-seed
```

### Testing Changes

```bash
# 1. Create new migration
make migrate-create NAME=add_feature

# 2. Test migration
./test-migrations.sh

# 3. Seed with test data
psql -f seed-medium.sql
```

### Backup & Restore

```bash
# Daily backup
./backup-database.sh --compress --storage s3

# Check backup health
./check-backup-health.sh

# Test restore
./restore-database.sh --from-remote --storage s3 --validate-only
```

### Disaster Recovery

```bash
# Create restore point before risky operation
psql -c "SELECT pg_create_restore_point('before_migration');"

# If needed, restore to restore point
# (See docs/DISASTER_RECOVERY.md for full procedures)
```

## Automation

### Cron Jobs

See [deploy/cron/database-backup.cron](../deploy/cron/database-backup.cron) for example crontab.

```bash
# Install cron jobs
crontab deploy/cron/database-backup.cron

# Or append to existing
crontab -l | cat - deploy/cron/database-backup.cron | crontab -
```

### Kubernetes CronJobs

```bash
# Deploy backup CronJob
kubectl apply -f deploy/cronjobs/database-backup-cronjob.yaml

# Check status
kubectl get cronjobs
kubectl get jobs | grep database-backup
kubectl logs -l app=database-backup
```

## Troubleshooting

### Script Not Executable

```bash
chmod +x *.sh
chmod +x hooks/*.sh
```

### Missing Dependencies

```bash
# Install PostgreSQL client
sudo apt-get install postgresql-client

# Install migration tool
go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest

# Install AWS CLI
pip install awscli

# Install Google Cloud SDK
# See: https://cloud.google.com/sdk/docs/install
```

### Permission Errors

```bash
# Ensure proper ownership
sudo chown -R postgres:postgres /var/lib/postgresql

# Check file permissions
ls -la backups/
```

## Documentation

- [Database Operations Guide](../docs/DATABASE_OPERATIONS.md) - Complete operations guide
- [PITR Documentation](../docs/DATABASE_PITR.md) - Point-in-time recovery
- [Disaster Recovery Runbook](../docs/DISASTER_RECOVERY.md) - Emergency procedures
- [Tooling Summary](../docs/DATABASE_TOOLING_SUMMARY.md) - Implementation overview

## Environment Variables Reference

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ridehailing

# Backup
BACKUP_DIR=backups
BACKUP_STORAGE_TYPE=s3
BACKUP_S3_BUCKET=ridehailing-backups
BACKUP_S3_PREFIX=database-backups
BACKUP_GPG_RECIPIENT=backup@example.com
BACKUP_ALERT_EMAIL=ops@example.com
BACKUP_SLACK_WEBHOOK=https://hooks.slack.com/...

# PITR
PITR_ENABLED=true
PITR_WAL_ARCHIVE_DIR=/var/lib/postgresql/wal_archive
PITR_STORAGE_TYPE=s3
PITR_S3_BUCKET=ridehailing-wal-archive
PITR_S3_PREFIX=wal-files
PITR_COMPRESS_WAL=true
```

## Support

For issues or questions:
1. Check script help: `./script-name.sh --help`
2. Review documentation in [docs/](../docs/)
3. Check logs for detailed error messages
4. Use `--verbose` flag for debugging

---

**Last Updated**: January 2025
