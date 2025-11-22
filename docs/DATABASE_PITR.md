# Point-in-Time Recovery (PITR) Guide

## Overview

Point-in-Time Recovery (PITR) allows you to restore your PostgreSQL database to any specific point in time within your retention window. This is achieved through continuous archiving of Write-Ahead Log (WAL) files.

## Table of Contents

- [How PITR Works](#how-pitr-works)
- [Setup and Configuration](#setup-and-configuration)
- [WAL Archiving](#wal-archiving)
- [Performing PITR](#performing-pitr)
- [Recovery Scenarios](#recovery-scenarios)
- [Best Practices](#best-practices)
- [Troubleshooting](#troubleshooting)

---

## How PITR Works

### Components

1. **Base Backup**: Full database backup taken with `pg_basebackup`
2. **WAL Files**: Continuous stream of database changes
3. **Recovery Target**: Specific timestamp, transaction ID, or named restore point
4. **Archive**: Remote storage (S3/GCS) containing base backups and WAL files

### Timeline

```
Base Backup          WAL Files Archived            Current
     |                    |                           |
     v                    v                           v
[========]===============================================>
         \________________/\___________________________/
         Archived Changes    Live Transactions

You can restore to ANY point in this timeline!
```

### Recovery Process

1. Restore base backup
2. Apply archived WAL files up to recovery target
3. Stop at specified point in time
4. Database is ready with exact state at that moment

---

## Setup and Configuration

### 1. PostgreSQL Configuration

Edit `postgresql.conf`:

```ini
# Enable WAL archiving
wal_level = replica
archive_mode = on
archive_command = '/path/to/scripts/archive-wal.sh %p %f'
archive_timeout = 300  # Archive every 5 minutes

# WAL settings
max_wal_size = 2GB
min_wal_size = 1GB
wal_keep_size = 512MB

# Checkpoint settings (for better recovery performance)
checkpoint_timeout = 15min
checkpoint_completion_target = 0.9
```

### 2. Environment Variables

Add to your `.env` file:

```bash
# PITR Configuration
PITR_ENABLED=true
PITR_WAL_ARCHIVE_DIR=/var/lib/postgresql/wal_archive
PITR_RETENTION_DAYS=7

# Remote WAL Archive
PITR_STORAGE_TYPE=s3
PITR_S3_BUCKET=ridehailing-wal-archive
PITR_S3_PREFIX=wal-files/production

# Or for GCS
# PITR_STORAGE_TYPE=gcs
# PITR_GCS_BUCKET=ridehailing-wal-archive
# PITR_GCS_PREFIX=wal-files/production
```

### 3. Create Archive Directory

```bash
# Create local WAL archive directory
sudo mkdir -p /var/lib/postgresql/wal_archive
sudo chown postgres:postgres /var/lib/postgresql/wal_archive
sudo chmod 700 /var/lib/postgresql/wal_archive
```

### 4. Restart PostgreSQL

```bash
# After configuration changes
sudo systemctl restart postgresql

# Or with Docker
docker-compose restart postgres
```

---

## WAL Archiving

### Archive Script

The `archive-wal.sh` script (created automatically) handles:

- Copying WAL files to archive
- Uploading to remote storage (S3/GCS)
- Compression to save space
- Cleanup of old WAL files

### Verify Archiving

```bash
# Check archive status
psql -U postgres -d ridehailing -c "
SELECT
    archived_count,
    last_archived_wal,
    last_archived_time,
    failed_count,
    last_failed_wal,
    last_failed_time
FROM pg_stat_archiver;
"

# Expected output:
#  archived_count | last_archived_wal  | last_archived_time  | failed_count
# ----------------+-------------------+---------------------+--------------
#            150 | 000000010000000...| 2024-01-15 10:30:00 |            0
```

### Monitor WAL Archive

```bash
# Check local archive
ls -lh /var/lib/postgresql/wal_archive

# Check S3 archive
aws s3 ls s3://ridehailing-wal-archive/wal-files/production/

# Check GCS archive
gsutil ls gs://ridehailing-wal-archive/wal-files/production/
```

---

## Performing PITR

### Option 1: Restore to Specific Timestamp

```bash
./scripts/pitr-restore.sh \
    --timestamp "2024-01-15 09:30:00" \
    --target-database ridehailing_restored
```

### Option 2: Restore to Transaction ID

```bash
# Get transaction ID from logs first
# Then restore to that transaction
./scripts/pitr-restore.sh \
    --transaction-id 12345678 \
    --target-database ridehailing_restored
```

### Option 3: Restore to Named Restore Point

```bash
# Create named restore point
psql -U postgres -d ridehailing -c "SELECT pg_create_restore_point('before_data_migration');"

# Later, restore to this point
./scripts/pitr-restore.sh \
    --restore-point "before_data_migration" \
    --target-database ridehailing_restored
```

### Manual PITR Process

If you need to perform PITR manually:

```bash
# 1. Stop PostgreSQL
sudo systemctl stop postgresql

# 2. Move current data directory
mv /var/lib/postgresql/15/main /var/lib/postgresql/15/main.old

# 3. Restore base backup
pg_basebackup -h localhost -U postgres -D /var/lib/postgresql/15/main -Fp -Xs -P

# 4. Create recovery.signal file
touch /var/lib/postgresql/15/main/recovery.signal

# 5. Configure recovery in postgresql.conf
cat >> /var/lib/postgresql/15/main/postgresql.auto.conf <<EOF
restore_command = 'cp /var/lib/postgresql/wal_archive/%f %p'
recovery_target_time = '2024-01-15 09:30:00'
recovery_target_action = 'promote'
EOF

# 6. Start PostgreSQL (will enter recovery mode)
sudo systemctl start postgresql

# 7. Monitor recovery progress
tail -f /var/log/postgresql/postgresql-15-main.log

# 8. Verify recovery
psql -U postgres -d ridehailing -c "SELECT pg_is_in_recovery();"
# Should return 'f' (false) when recovery is complete
```

---

## Recovery Scenarios

### Scenario 1: Accidental Data Deletion

**Problem**: Accidentally deleted important records at 2:30 PM

**Solution**: Restore to 2:25 PM (5 minutes before deletion)

```bash
./scripts/pitr-restore.sh \
    --timestamp "2024-01-15 14:25:00" \
    --target-database ridehailing_recovered
```

**Then export the deleted data**:

```bash
# Export the data you need
pg_dump -U postgres -d ridehailing_recovered \
    -t users -t rides --data-only > recovered_data.sql

# Import into production
psql -U postgres -d ridehailing -f recovered_data.sql
```

### Scenario 2: Bad Migration Rollback

**Problem**: Migration deployed at 3:00 PM caused data corruption

**Solution**: Restore to just before migration (2:59 PM)

```bash
# Create restore point before migrations (proactive)
psql -U postgres -d ridehailing -c "SELECT pg_create_restore_point('before_migration_v2.5');"

# If migration fails, restore to this point
./scripts/pitr-restore.sh \
    --restore-point "before_migration_v2.5" \
    --target-database ridehailing
```

### Scenario 3: Database Corruption

**Problem**: Database corrupted, need to restore to last known good state

**Solution**: Find last good transaction and restore

```bash
# Check logs for last successful query timestamp
grep "COMMIT" /var/log/postgresql/postgresql-*.log | tail -20

# Restore to that time
./scripts/pitr-restore.sh \
    --timestamp "2024-01-15 16:45:00" \
    --target-database ridehailing
```

### Scenario 4: Ransomware Attack

**Problem**: Database encrypted by ransomware at unknown time

**Solution**: Restore to yesterday and review changes

```bash
# Restore to 24 hours ago
./scripts/pitr-restore.sh \
    --timestamp "$(date -d '24 hours ago' '+%Y-%m-%d %H:%M:%S')" \
    --target-database ridehailing_pre_attack

# Analyze differences
./scripts/compare-databases.sh ridehailing_pre_attack ridehailing
```

---

## Best Practices

### 1. Regular Base Backups

```bash
# Daily base backup (in addition to WAL archiving)
0 2 * * * pg_basebackup -h localhost -U postgres -D /backups/base/$(date +\%Y\%m\%d) -Fp -Xs -P
```

### 2. Create Restore Points Before Major Changes

```sql
-- Before deployment
SELECT pg_create_restore_point('before_release_v2.5.0');

-- Before bulk operations
SELECT pg_create_restore_point('before_user_data_migration');

-- Before maintenance
SELECT pg_create_restore_point('before_index_rebuild');
```

### 3. Test Recovery Monthly

```bash
# Automated monthly restore test
./scripts/test-pitr-restore.sh --timestamp "latest" --verify
```

### 4. Monitor WAL Archive Size

```bash
# Check WAL archive growth
du -sh /var/lib/postgresql/wal_archive

# Ensure you have enough space
df -h /var/lib/postgresql
```

### 5. Set Retention Policy

```bash
# Clean WAL files older than 7 days
find /var/lib/postgresql/wal_archive -name "*.wal" -mtime +7 -delete

# Or use automated script
./scripts/cleanup-wal-archive.sh --retention-days 7
```

### 6. Document Critical Timestamps

Keep a log of important events:

```
2024-01-15 14:00:00 - Release v2.5.0 deployed
2024-01-15 14:05:00 - Restore point: before_schema_migration
2024-01-15 14:30:00 - Migration completed successfully
```

---

## Monitoring and Alerts

### Key Metrics to Monitor

```sql
-- WAL archiving status
SELECT * FROM pg_stat_archiver;

-- Current WAL position
SELECT pg_current_wal_lsn();

-- WAL generation rate
SELECT pg_wal_lsn_diff(pg_current_wal_lsn(), '0/0') / (60 * 60 * 24) AS bytes_per_day;

-- Archive lag (should be near 0)
SELECT
    EXTRACT(EPOCH FROM (now() - last_archived_time)) AS archive_lag_seconds
FROM pg_stat_archiver;
```

### Prometheus Metrics

```yaml
# Add to prometheus.yml
- job_name: 'postgres-wal-archiver'
  static_configs:
    - targets: ['localhost:9187']
  metrics_path: /metrics
```

### Grafana Alerts

- WAL archive failures
- High archive lag (> 5 minutes)
- Low disk space in archive directory
- Restoration test failures

---

## Troubleshooting

### WAL Files Not Being Archived

**Check**:
```bash
# Verify archive_mode is on
psql -U postgres -c "SHOW archive_mode;"

# Check archive_command
psql -U postgres -c "SHOW archive_command;"

# Check for errors
tail -f /var/log/postgresql/postgresql-*.log | grep archive
```

**Fix**:
```bash
# Test archive command manually
su - postgres
/path/to/scripts/archive-wal.sh /var/lib/postgresql/15/main/pg_wal/000000010000000000000001 000000010000000000000001
```

### Recovery Stuck or Slow

**Check**:
```bash
# Monitor recovery progress
psql -U postgres -c "SELECT * FROM pg_stat_recovery_prefetch;"

# Check if waiting for WAL
tail -f /var/log/postgresql/postgresql-*.log | grep "waiting for"
```

**Fix**:
```bash
# Ensure all WAL files are accessible
ls -lh /var/lib/postgresql/wal_archive/

# Check network connectivity to S3/GCS
aws s3 ls s3://ridehailing-wal-archive/wal-files/production/
```

### Cannot Find Required WAL File

**Error**: `could not open file "pg_wal/000000010000000000000042": No such file or directory`

**Cause**: Missing WAL file in archive

**Fix**:
```bash
# Check which WAL files are available
ls /var/lib/postgresql/wal_archive/

# Download missing WAL file from S3
aws s3 cp s3://ridehailing-wal-archive/wal-files/production/000000010000000000000042 \
    /var/lib/postgresql/wal_archive/
```

### Recovery Completed But Data Missing

**Cause**: Restored to wrong point in time

**Fix**:
```bash
# Try different timestamp
./scripts/pitr-restore.sh \
    --timestamp "2024-01-15 15:00:00" \
    --target-database ridehailing_test

# Compare timestamps
psql -U postgres -d ridehailing_test -c "SELECT now();"
```

---

## Recovery Point Objective (RPO)

With PITR properly configured:

- **RPO**: As low as 5 minutes (or less with `archive_timeout`)
- **RTO**: Typically 15-60 minutes depending on database size and WAL file count

---

## Cost Optimization

### WAL Compression

```bash
# Enable WAL compression in archive script
export PITR_COMPRESS_WAL=true
export PITR_COMPRESSION_LEVEL=6  # 1-9, higher = better compression
```

### S3 Storage Classes

```bash
# Use S3 Glacier for older WAL files
aws s3api put-bucket-lifecycle-configuration \
    --bucket ridehailing-wal-archive \
    --lifecycle-configuration file://wal-lifecycle-policy.json
```

Example lifecycle policy:
```json
{
  "Rules": [{
    "Id": "ArchiveOldWAL",
    "Status": "Enabled",
    "Prefix": "wal-files/production/",
    "Transitions": [{
      "Days": 7,
      "StorageClass": "STANDARD_IA"
    }, {
      "Days": 30,
      "StorageClass": "GLACIER"
    }],
    "Expiration": {
      "Days": 90
    }
  }]
}
```

---

## References

- [PostgreSQL PITR Documentation](https://www.postgresql.org/docs/current/continuous-archiving.html)
- [pg_basebackup Documentation](https://www.postgresql.org/docs/current/app-pgbasebackup.html)
- [WAL Archiving Best Practices](https://wiki.postgresql.org/wiki/Point_In_Time_Recovery)

---

## Quick Reference

```bash
# Create restore point
psql -U postgres -d ridehailing -c "SELECT pg_create_restore_point('point_name');"

# Check archiving status
psql -U postgres -c "SELECT * FROM pg_stat_archiver;"

# List restore points
psql -U postgres -c "SELECT * FROM pg_control_checkpoint();"

# Restore to timestamp
./scripts/pitr-restore.sh --timestamp "2024-01-15 14:30:00"

# Restore to restore point
./scripts/pitr-restore.sh --restore-point "before_migration"

# Test restore
./scripts/test-pitr-restore.sh --verify

# Cleanup old WAL files
./scripts/cleanup-wal-archive.sh --retention-days 7
```

---

**Note**: PITR is your safety net for disaster recovery. Test it regularly and ensure it's properly configured before you need it!
