# Disaster Recovery Runbook

**Critical procedures for database disaster recovery scenarios**

## Emergency Contacts

| Role | Name | Contact | Availability |
|------|------|---------|--------------|
| Database Admin | TBD | TBD | 24/7 |
| DevOps Lead | TBD | TBD | 24/7 |
| CTO | TBD | TBD | On-call |
| Cloud Provider Support | AWS/GCP | See support portal | 24/7 |

---

## Recovery Objectives

| Metric | Target | Acceptable |
|--------|--------|-----------|
| **RTO** (Recovery Time Objective) | 4 hours | 8 hours |
| **RPO** (Recovery Point Objective) | 1 hour | 4 hours |
| **Data Loss** | < 1 hour | < 4 hours |

---

## Table of Contents

1. [Complete Database Loss](#scenario-1-complete-database-loss)
2. [Data Corruption](#scenario-2-data-corruption)
3. [Accidental Data Deletion](#scenario-3-accidental-data-deletion)
4. [Bad Migration Deployed](#scenario-4-bad-migration-deployed)
5. [Ransomware Attack](#scenario-5-ransomware-attack)
6. [Cloud Provider Outage](#scenario-6-cloud-provider-outage)
7. [Hardware Failure](#scenario-7-hardware-failure)

---

## Scenario 1: Complete Database Loss

### Symptoms
- Database server unresponsive
- Cannot connect to PostgreSQL
- Physical server/instance destroyed

### Immediate Actions (0-15 minutes)

1. **Verify the incident**
   ```bash
   # Check if PostgreSQL is running
   systemctl status postgresql

   # Try database connection
   psql -h DB_HOST -U postgres -d ridehailing

   # Check server status
   ping DB_HOST
   ssh admin@DB_HOST
   ```

2. **Activate incident response**
   - Alert all stakeholders
   - Start incident log
   - Notify cloud provider if applicable

3. **Switch to read-only mode** (if possible)
   - Update load balancer to reject writes
   - Display maintenance page

### Recovery Steps (15 minutes - 4 hours)

#### Option A: Restore from Replica (Fastest - 15-30 min)

```bash
# 1. Promote read replica to primary
# On replica server:
SELECT pg_promote();

# 2. Update application configuration
# Point all services to new primary
export DB_HOST=replica.ridehailing.com

# 3. Restart services
kubectl rollout restart deployment/rides-service
kubectl rollout restart deployment/payments-service
# ... all services

# 4. Verify functionality
./scripts/verify-database.sh
```

#### Option B: Restore from Backup (1-2 hours)

```bash
# 1. Provision new database server
# Via cloud provider console or terraform

# 2. Install PostgreSQL
sudo apt-get update
sudo apt-get install postgresql-15

# 3. Download latest backup
./scripts/restore-database.sh --from-remote --storage s3 --latest

# Or manual download:
aws s3 cp s3://ridehailing-backups/database-backups/latest.sql.gz .
gunzip latest.sql.gz

# 4. Restore database
psql -d ridehailing -f latest.sql

# 5. Apply WAL files for PITR (if available)
# See PITR documentation

# 6. Update DNS/configuration
# Point applications to new database

# 7. Restart all services
kubectl rollout restart deployment --all

# 8. Verify
./scripts/verify-database.sh
```

### Verification

```bash
# Check database is accessible
psql -d ridehailing -c "SELECT COUNT(*) FROM users;"

# Check recent data
psql -d ridehailing -c "SELECT MAX(created_at) FROM rides;"

# Run smoke tests
./scripts/smoke-test.sh

# Check application health
kubectl get pods
curl https://api.ridehailing.com/health
```

### Post-Recovery (4-8 hours)

1. **Root cause analysis**
   - Review logs
   - Document timeline
   - Identify prevention measures

2. **Restore redundancy**
   - Set up new replica
   - Verify backup systems
   - Test failover procedures

3. **Communication**
   - Update stakeholders
   - Prepare incident report
   - Plan improvements

---

## Scenario 2: Data Corruption

### Symptoms
- Database accepts connections but returns corrupted data
- Consistency check failures
- Index corruption errors

### Immediate Actions

1. **Identify corruption scope**
   ```sql
   -- Check table integrity
   SELECT pg_catalog.pg_check_visible_in_rel(oid, oid)
   FROM pg_class WHERE relkind = 'r';

   -- Check indexes
   SELECT * FROM pg_stat_user_indexes
   WHERE idx_scan = 0 AND indexrelname NOT LIKE 'pg_%';
   ```

2. **Stop writes to affected tables**
   ```sql
   -- Make table read-only
   REVOKE INSERT, UPDATE, DELETE ON affected_table FROM PUBLIC;
   ```

### Recovery Steps

#### Option A: Reindex (for index corruption)

```sql
-- Reindex specific index
REINDEX INDEX idx_rides_status;

-- Reindex entire table
REINDEX TABLE rides;

-- Reindex entire database (during maintenance)
REINDEX DATABASE ridehailing;
```

#### Option B: Point-in-Time Recovery

```bash
# Restore to just before corruption occurred
./scripts/pitr-restore.sh \
    --timestamp "2024-01-15 14:00:00" \
    --target-database ridehailing_recovery

# Export good data
pg_dump -t affected_table ridehailing_recovery > good_data.sql

# Restore to production
psql -d ridehailing -f good_data.sql
```

### Verification

```sql
-- Run consistency checks
VACUUM ANALYZE;

-- Check data integrity
SELECT COUNT(*) FROM affected_table;
SELECT MAX(id), MIN(id) FROM affected_table;

-- Verify foreign keys
SELECT * FROM affected_table WHERE foreign_key_id NOT IN (SELECT id FROM parent_table);
```

---

## Scenario 3: Accidental Data Deletion

### Symptoms
- Important records deleted
- User reports missing data
- Audit logs show DELETE operations

### Immediate Actions (0-5 minutes)

1. **Stop further changes**
   ```sql
   -- Block write operations
   REVOKE DELETE ON ALL TABLES IN SCHEMA public FROM app_user;

   -- Or stop application
   kubectl scale deployment/rides-service --replicas=0
   ```

2. **Identify deletion details**
   ```sql
   -- Check audit logs
   SELECT * FROM audit_logs
   WHERE action = 'DELETE'
   AND timestamp > NOW() - INTERVAL '1 hour'
   ORDER BY timestamp DESC;

   -- Check PostgreSQL logs
   tail -n 1000 /var/log/postgresql/postgresql-*.log | grep DELETE
   ```

### Recovery Steps (5-60 minutes)

#### Option A: Restore from PITR (Best - preserves all data)

```bash
# 1. Create restore point NOW (before more changes)
psql -c "SELECT pg_create_restore_point('before_recovery');"

# 2. Restore to just before deletion
./scripts/pitr-restore.sh \
    --timestamp "2024-01-15 13:55:00" \
    --target-database ridehailing_recovery

# 3. Export deleted records
psql -d ridehailing_recovery -c "\COPY (
    SELECT * FROM rides
    WHERE id IN ('uuid1', 'uuid2', 'uuid3')
) TO '/tmp/deleted_rides.csv' CSV HEADER;"

# 4. Re-import to production
psql -d ridehailing -c "\COPY rides FROM '/tmp/deleted_rides.csv' CSV HEADER;"

# 5. Verify
psql -d ridehailing -c "SELECT COUNT(*) FROM rides WHERE id IN ('uuid1', 'uuid2', 'uuid3');"
```

#### Option B: Restore from Transaction Log (if within same transaction)

```sql
-- If deletion not yet committed, rollback
ROLLBACK;

-- Check if in transaction
SELECT * FROM pg_stat_activity WHERE state = 'idle in transaction';
```

### Verification

```sql
-- Verify record count
SELECT COUNT(*) FROM rides;

-- Check affected records
SELECT * FROM rides WHERE id IN (<deleted_ids>);

-- Verify relationships
SELECT r.*, u.email FROM rides r
JOIN users u ON r.rider_id = u.id
WHERE r.id IN (<deleted_ids>);
```

---

## Scenario 4: Bad Migration Deployed

### Symptoms
- Application errors after deployment
- Schema incompatibility
- Data loss in specific tables

### Immediate Actions (0-10 minutes)

1. **Create restore point**
   ```sql
   SELECT pg_create_restore_point('before_rollback');
   ```

2. **Stop deployments**
   ```bash
   # Stop rolling update
   kubectl rollout pause deployment/rides-service

   # Scale down new pods
   kubectl scale deployment/rides-service --replicas=0
   ```

### Recovery Steps (10-30 minutes)

#### Option A: Rollback Migration (Fastest)

```bash
# 1. Check current migration version
make migrate-version

# 2. Rollback bad migration
make migrate-down

# Or rollback multiple migrations
migrate -path db/migrations -database "$DATABASE_URL" down 2

# 3. Verify schema
psql -d ridehailing -c "\dt"
psql -d ridehailing -c "\d table_name"

# 4. Redeploy old application version
kubectl rollout undo deployment/rides-service
```

#### Option B: Restore Database + Revert Code

```bash
# 1. Restore from backup before migration
./scripts/restore-database.sh \
    --timestamp "2024-01-15 14:00:00"

# 2. Revert application code
git revert <bad-commit>
git push origin main

# 3. Redeploy
./deploy.sh

# 4. Verify
./scripts/verify-database.sh
```

### Verification

```sql
-- Check migration version
SELECT version, dirty FROM schema_migrations;

-- Verify table structure
\d+ rides
\d+ users

-- Run application tests
go test ./internal/rides/...
```

---

## Scenario 5: Ransomware Attack

### Symptoms
- Database files encrypted
- Ransom note in file system
- Cannot access data
- Unusual file modifications

### Immediate Actions (0-15 minutes)

1. **Isolate infected systems**
   ```bash
   # Disconnect network
   sudo ifconfig eth0 down

   # Stop PostgreSQL
   sudo systemctl stop postgresql

   # Block access at firewall level
   sudo iptables -A INPUT -p tcp --dport 5432 -j DROP
   ```

2. **Alert security team**
   - Notify IT security
   - Contact law enforcement
   - Document evidence
   - Do NOT pay ransom

3. **Assess damage**
   ```bash
   # Check file modifications
   find /var/lib/postgresql -type f -mtime -1

   # Check for encryption
   file /var/lib/postgresql/15/main/*
   ```

### Recovery Steps (1-6 hours)

1. **Provision clean environment**
   ```bash
   # New, isolated server
   # Fresh OS installation
   # Update all packages
   sudo apt-get update && sudo apt-get upgrade
   ```

2. **Restore from clean backup**
   ```bash
   # Use backup from BEFORE attack (verify date)
   ./scripts/restore-database.sh \
       --from-remote \
       --storage s3 \
       --timestamp "2024-01-14 02:00:00"  # Known good backup

   # Scan backup for malware
   clamscan -r /backup/location
   ```

3. **Security hardening**
   ```bash
   # Update passwords
   psql -c "ALTER USER postgres WITH PASSWORD 'new-secure-password';"

   # Review access controls
   psql -c "\du"

   # Enable audit logging
   # Update firewall rules
   # Enable 2FA for all access
   ```

4. **Restore operations gradually**
   - Start with read-only mode
   - Verify data integrity
   - Enable writes carefully
   - Monitor for suspicious activity

### Post-Incident

1. **Security review**
   - How did attack occur?
   - What vulnerabilities exist?
   - Implement additional security controls

2. **Backup verification**
   - Ensure clean backups preserved
   - Test restore procedures
   - Increase backup frequency

---

## Scenario 6: Cloud Provider Outage

### Symptoms
- Cannot reach cloud provider services
- Multi-zone failure
- Regional outage

### Immediate Actions

1. **Verify outage scope**
   - Check cloud provider status page
   - Test from multiple locations
   - Identify affected regions/zones

2. **Activate DR site** (if configured)
   ```bash
   # Switch DNS to DR region
   aws route53 change-resource-record-sets --hosted-zone-id Z123 --change-batch file://dr-dns.json

   # Or manually update DNS
   # Point api.ridehailing.com to DR IP
   ```

3. **Restore from backup to alternative provider**
   ```bash
   # If cross-cloud backup exists
   ./scripts/restore-database.sh \
       --from-remote \
       --storage gcs \  # Alternative provider
       --latest
   ```

### Recovery Steps

Follow "Complete Database Loss" procedure but use alternative cloud provider or region.

---

## Scenario 7: Hardware Failure

### Symptoms
- Disk failure alerts
- I/O errors in logs
- Database crashes
- Unreadable data blocks

### Immediate Actions

1. **Check hardware status**
   ```bash
   # Check disk health
   sudo smartctl -a /dev/sda

   # Check RAID status
   sudo mdadm --detail /dev/md0

   # Check system logs
   dmesg | grep -i error
   sudo journalctl -xe | grep -i error
   ```

2. **Switch to replica if available**
   ```sql
   # On replica
   SELECT pg_promote();
   ```

3. **If no replica, restore from backup**
   - Follow "Complete Database Loss" procedure

---

## Prevention Measures

### Daily

- [x] Automated backups running
- [x] Backup health checks passing
- [x] WAL archiving active
- [x] Replica lag < 1 second
- [x] Monitoring alerts configured

### Weekly

- [x] Test restore procedure
- [x] Review backup sizes
- [x] Check disk space
- [x] Review access logs

### Monthly

- [x] Full DR drill
- [x] Update documentation
- [x] Review incident response
- [x] Test failover procedures
- [x] Verify off-site backups

### Quarterly

- [x] Update emergency contacts
- [x] Review recovery objectives
- [x] Security audit
- [x] Disaster recovery training

---

## Recovery Validation Checklist

After any recovery:

- [ ] Database accessible
- [ ] All tables present
- [ ] Record counts match expected
- [ ] Recent data present (check timestamps)
- [ ] Foreign key constraints valid
- [ ] Indexes rebuilt
- [ ] Materialized views refreshed
- [ ] Application health checks passing
- [ ] Replica lag normal
- [ ] Backup system operational
- [ ] Monitoring alerts configured
- [ ] All services running
- [ ] Smoke tests passing
- [ ] User acceptance testing
- [ ] Performance acceptable
- [ ] Documentation updated
- [ ] Incident report completed

---

## Appendix: Quick Command Reference

```bash
# Emergency Read-Only Mode
psql -c "ALTER DATABASE ridehailing SET default_transaction_read_only = on;"

# Emergency Restore
./scripts/restore-database.sh --from-remote --storage s3 --latest

# Promote Replica
psql -c "SELECT pg_promote();"

# Create Restore Point
psql -c "SELECT pg_create_restore_point('emergency_$(date +%Y%m%d_%H%M%S)');"

# Kill All Connections
psql -c "SELECT pg_terminate_backend(pid) FROM pg_stat_activity WHERE datname='ridehailing' AND pid <> pg_backend_pid();"

# Backup NOW
./scripts/backup-database.sh --compress --storage s3

# Check Replication Lag
psql -c "SELECT NOW() - pg_last_xact_replay_timestamp() AS replication_lag;"
```

---

## Contact Information

### Cloud Provider Support

- **AWS**: https://console.aws.amazon.com/support
- **GCP**: https://console.cloud.google.com/support

### Database Support

- **PostgreSQL Community**: https://www.postgresql.org/support/
- **Enterprise Support**: [Your support contract]

---

**Last Updated**: 2024-01-15
**Next Review**: 2024-04-15
**Document Owner**: Database Team
