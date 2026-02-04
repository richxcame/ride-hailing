# Disaster Recovery Runbook

Critical procedures for database disaster recovery scenarios.

## Recovery Objectives

| Metric | Target | Acceptable |
|--------|--------|-----------|
| **RTO** (Recovery Time Objective) | 4 hours | 8 hours |
| **RPO** (Recovery Point Objective) | 1 hour | 4 hours |
| **Data Loss** | < 1 hour | < 4 hours |

---

## Scenario 1: Complete Database Loss

**Symptoms**: Database unresponsive, cannot connect, server destroyed.

### Option A: Promote Replica (15-30 min)

```bash
# Promote read replica
psql -c "SELECT pg_promote();"

# Point services to new primary
export DB_HOST=replica.ridehailing.com
kubectl rollout restart deployment --all

./scripts/verify-database.sh
```

### Option B: Restore from Backup (1-2 hours)

```bash
# Download and restore latest backup
./scripts/restore-database.sh --from-remote --storage s3 --latest

# Or manually:
aws s3 cp s3://ridehailing-backups/database-backups/latest.sql.gz .
gunzip latest.sql.gz
psql -d ridehailing -f latest.sql

# Update DNS/config, restart services
kubectl rollout restart deployment --all
./scripts/verify-database.sh
```

---

## Scenario 2: Data Corruption

**Symptoms**: Corrupted data returned, consistency check failures, index errors.

### Option A: Reindex (for index corruption)

```sql
REINDEX INDEX idx_rides_status;
REINDEX TABLE rides;
REINDEX DATABASE ridehailing;  -- during maintenance window
```

### Option B: Point-in-Time Recovery (for data corruption)

```bash
./scripts/pitr-restore.sh \
    --timestamp "2024-01-15 14:00:00" \
    --target-database ridehailing_recovery

# Export good data, restore to production
pg_dump -t affected_table ridehailing_recovery > good_data.sql
psql -d ridehailing -f good_data.sql
```

---

## Scenario 3: Accidental Data Deletion

**Symptoms**: Important records deleted, users report missing data.

1. **Block further writes immediately**:
   ```sql
   REVOKE DELETE ON ALL TABLES IN SCHEMA public FROM app_user;
   ```

2. **Restore deleted records via PITR**:
   ```bash
   psql -c "SELECT pg_create_restore_point('before_recovery');"

   ./scripts/pitr-restore.sh \
       --timestamp "2024-01-15 13:55:00" \
       --target-database ridehailing_recovery

   # Export and re-import deleted records
   psql -d ridehailing_recovery -c "\COPY (
       SELECT * FROM rides WHERE id IN ('uuid1', 'uuid2', 'uuid3')
   ) TO '/tmp/deleted_rides.csv' CSV HEADER;"

   psql -d ridehailing -c "\COPY rides FROM '/tmp/deleted_rides.csv' CSV HEADER;"
   ```

---

## Scenario 4: Bad Migration Deployed

**Symptoms**: Application errors after deployment, schema incompatibility.

### Option A: Rollback Migration (fastest)

```bash
make migrate-version
make migrate-down

# Or rollback multiple:
migrate -path db/migrations -database "$DATABASE_URL" down 2

# Redeploy previous app version
kubectl rollout undo deployment/rides-service
```

### Option B: Restore Database + Revert Code

```bash
./scripts/restore-database.sh --timestamp "2024-01-15 14:00:00"
git revert <bad-commit>
git push origin main
./deploy.sh
```

---

## Scenario 5: Ransomware Attack

**Symptoms**: Database files encrypted, ransom note found, data inaccessible.

1. **Isolate immediately**:
   ```bash
   sudo ifconfig eth0 down
   sudo systemctl stop postgresql
   sudo iptables -A INPUT -p tcp --dport 5432 -j DROP
   ```

2. **Alert security team** -- notify IT security, contact law enforcement, document evidence. Do NOT pay ransom.

3. **Restore from clean backup on a fresh server**:
   ```bash
   ./scripts/restore-database.sh \
       --from-remote --storage s3 \
       --timestamp "2024-01-14 02:00:00"  # Known good backup before attack

   clamscan -r /backup/location  # Scan for malware
   ```

4. **Harden**: rotate all passwords, review access controls, enable audit logging.

---

## Scenario 6: Cloud Provider Outage

**Symptoms**: Cannot reach cloud services, multi-zone/regional failure.

1. Check cloud provider status page, test from multiple locations.
2. **Switch DNS to DR region**:
   ```bash
   aws route53 change-resource-record-sets --hosted-zone-id Z123 --change-batch file://dr-dns.json
   ```
3. **Restore to alternative provider** (if cross-cloud backup exists):
   ```bash
   ./scripts/restore-database.sh --from-remote --storage gcs --latest
   ```

---

## Scenario 7: Hardware Failure

**Symptoms**: Disk failure alerts, I/O errors, database crashes.

1. **Check hardware**:
   ```bash
   sudo smartctl -a /dev/sda
   sudo mdadm --detail /dev/md0
   dmesg | grep -i error
   ```
2. **Promote replica** if available: `SELECT pg_promote();`
3. Otherwise follow Scenario 1 (Complete Database Loss) procedure.

---

## Recovery Validation Checklist

After any recovery:

- [ ] Database accessible, all tables present
- [ ] Record counts match expected, recent data present
- [ ] Foreign key constraints valid, indexes rebuilt
- [ ] Application health checks and smoke tests passing
- [ ] Replica lag normal, backup system operational
- [ ] All services running, performance acceptable
- [ ] Incident report completed
