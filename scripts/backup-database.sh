#!/bin/bash

# Database Backup Script with Remote Storage Support
# Supports local backups and remote storage (S3, GCS, Azure Blob)
# Usage: ./scripts/backup-database.sh [options]
# Options:
#   --local-only      : Only create local backup (no remote upload)
#   --remote-only     : Only upload to remote (requires existing backup)
#   --storage TYPE    : Remote storage type (s3|gcs|azure) [default: from env]
#   --compress        : Compress backup with gzip
#   --encrypt         : Encrypt backup (requires GPG_RECIPIENT env var)
#   --retention DAYS  : Set retention period in days [default: 30]
#   --database NAME   : Database name to backup [default: from env]
#   --verbose         : Show detailed output

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
LOCAL_ONLY=false
REMOTE_ONLY=false
COMPRESS=true
ENCRYPT=false
VERBOSE=false
RETENTION_DAYS=30
STORAGE_TYPE=""

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --local-only)
            LOCAL_ONLY=true
            shift
            ;;
        --remote-only)
            REMOTE_ONLY=true
            shift
            ;;
        --storage)
            STORAGE_TYPE="$2"
            shift 2
            ;;
        --compress)
            COMPRESS=true
            shift
            ;;
        --no-compress)
            COMPRESS=false
            shift
            ;;
        --encrypt)
            ENCRYPT=true
            shift
            ;;
        --retention)
            RETENTION_DAYS="$2"
            shift 2
            ;;
        --database)
            DB_NAME="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Usage: $0 [--local-only] [--remote-only] [--storage TYPE] [--compress] [--encrypt] [--retention DAYS] [--database NAME] [--verbose]"
            exit 1
            ;;
    esac
done

# Load environment variables if .env exists
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Database connection details
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_PASSWORD=${DB_PASSWORD:-postgres}
DB_NAME=${DB_NAME:-ridehailing}

# Backup configuration
BACKUP_DIR=${BACKUP_DIR:-backups}
TIMESTAMP=$(date +%Y%m%d_%H%M%S)
BACKUP_NAME="backup_${DB_NAME}_${TIMESTAMP}"
BACKUP_FILE="${BACKUP_DIR}/${BACKUP_NAME}.sql"

# Remote storage configuration
STORAGE_TYPE=${STORAGE_TYPE:-${BACKUP_STORAGE_TYPE:-}}
S3_BUCKET=${BACKUP_S3_BUCKET:-}
S3_PREFIX=${BACKUP_S3_PREFIX:-database-backups}
GCS_BUCKET=${BACKUP_GCS_BUCKET:-}
GCS_PREFIX=${BACKUP_GCS_PREFIX:-database-backups}
AZURE_CONTAINER=${BACKUP_AZURE_CONTAINER:-}
AZURE_PREFIX=${BACKUP_AZURE_PREFIX:-database-backups}

# Encryption configuration
GPG_RECIPIENT=${BACKUP_GPG_RECIPIENT:-}

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

log_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# Check prerequisites
check_prerequisites() {
    log_info "Checking prerequisites..."

    # Check pg_dump
    if ! command -v pg_dump &> /dev/null; then
        log_error "pg_dump not found. Install PostgreSQL client tools."
        exit 1
    fi

    # Check compression tool
    if [ "$COMPRESS" = true ] && ! command -v gzip &> /dev/null; then
        log_error "gzip not found. Install gzip or use --no-compress"
        exit 1
    fi

    # Check encryption tool
    if [ "$ENCRYPT" = true ]; then
        if ! command -v gpg &> /dev/null; then
            log_error "gpg not found. Install GnuPG or disable encryption."
            exit 1
        fi
        if [ -z "$GPG_RECIPIENT" ]; then
            log_error "GPG_RECIPIENT environment variable not set"
            exit 1
        fi
    fi

    # Check remote storage tools
    if [ "$STORAGE_TYPE" = "s3" ] && [ "$LOCAL_ONLY" = false ]; then
        if ! command -v aws &> /dev/null; then
            log_error "AWS CLI not found. Install aws-cli or use --local-only"
            exit 1
        fi
        if [ -z "$S3_BUCKET" ]; then
            log_error "BACKUP_S3_BUCKET environment variable not set"
            exit 1
        fi
    elif [ "$STORAGE_TYPE" = "gcs" ] && [ "$LOCAL_ONLY" = false ]; then
        if ! command -v gsutil &> /dev/null; then
            log_error "gsutil not found. Install Google Cloud SDK or use --local-only"
            exit 1
        fi
        if [ -z "$GCS_BUCKET" ]; then
            log_error "BACKUP_GCS_BUCKET environment variable not set"
            exit 1
        fi
    elif [ "$STORAGE_TYPE" = "azure" ] && [ "$LOCAL_ONLY" = false ]; then
        if ! command -v az &> /dev/null; then
            log_error "Azure CLI not found. Install azure-cli or use --local-only"
            exit 1
        fi
        if [ -z "$AZURE_CONTAINER" ]; then
            log_error "BACKUP_AZURE_CONTAINER environment variable not set"
            exit 1
        fi
    fi

    log_success "Prerequisites check passed"
}

# Check database connectivity
check_database() {
    log_info "Checking database connectivity..."

    if ! PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -c '\q' 2>/dev/null; then
        log_error "Cannot connect to database ${DB_NAME} at ${DB_HOST}:${DB_PORT}"
        exit 1
    fi

    log_success "Database is accessible"
}

# Get database size
get_database_size() {
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -t -c "SELECT pg_size_pretty(pg_database_size('${DB_NAME}'));" 2>/dev/null | xargs
}

# Create backup directory
create_backup_dir() {
    if [ ! -d "${BACKUP_DIR}" ]; then
        log_info "Creating backup directory: ${BACKUP_DIR}"
        mkdir -p "${BACKUP_DIR}"
    fi
}

# Perform database backup
perform_backup() {
    if [ "$REMOTE_ONLY" = true ]; then
        log_info "Skipping backup creation (remote-only mode)"
        return
    fi

    local db_size=$(get_database_size)
    log_info "Database size: ${db_size}"
    log_info "Creating backup: ${BACKUP_FILE}"

    # Start timing
    local start_time=$(date +%s)

    # Perform backup with progress
    if [ "$VERBOSE" = true ]; then
        PGPASSWORD=${DB_PASSWORD} pg_dump \
            -h ${DB_HOST} \
            -p ${DB_PORT} \
            -U ${DB_USER} \
            -d ${DB_NAME} \
            --verbose \
            --format=plain \
            --no-owner \
            --no-acl \
            > "${BACKUP_FILE}"
    else
        PGPASSWORD=${DB_PASSWORD} pg_dump \
            -h ${DB_HOST} \
            -p ${DB_PORT} \
            -U ${DB_USER} \
            -d ${DB_NAME} \
            --format=plain \
            --no-owner \
            --no-acl \
            > "${BACKUP_FILE}"
    fi

    # Calculate duration
    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    # Get backup file size
    local backup_size=$(du -h "${BACKUP_FILE}" | cut -f1)

    log_success "Backup created: ${BACKUP_FILE}"
    log_info "Backup size: ${backup_size}"
    log_info "Backup duration: ${duration} seconds"
}

# Compress backup
compress_backup() {
    if [ "$COMPRESS" = false ]; then
        log_verbose "Compression disabled"
        return
    fi

    log_info "Compressing backup..."

    local start_time=$(date +%s)

    gzip -f "${BACKUP_FILE}"
    BACKUP_FILE="${BACKUP_FILE}.gz"

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    local compressed_size=$(du -h "${BACKUP_FILE}" | cut -f1)

    log_success "Backup compressed: ${BACKUP_FILE}"
    log_info "Compressed size: ${compressed_size}"
    log_info "Compression duration: ${duration} seconds"
}

# Encrypt backup
encrypt_backup() {
    if [ "$ENCRYPT" = false ]; then
        log_verbose "Encryption disabled"
        return
    fi

    log_info "Encrypting backup with GPG..."

    local start_time=$(date +%s)

    gpg --encrypt --recipient "${GPG_RECIPIENT}" "${BACKUP_FILE}"
    rm "${BACKUP_FILE}"
    BACKUP_FILE="${BACKUP_FILE}.gpg"

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_success "Backup encrypted: ${BACKUP_FILE}"
    log_info "Encryption duration: ${duration} seconds"
}

# Upload to S3
upload_to_s3() {
    log_info "Uploading backup to S3: s3://${S3_BUCKET}/${S3_PREFIX}/"

    local s3_path="s3://${S3_BUCKET}/${S3_PREFIX}/$(basename ${BACKUP_FILE})"

    aws s3 cp "${BACKUP_FILE}" "${s3_path}" \
        --storage-class STANDARD_IA \
        --metadata "database=${DB_NAME},timestamp=${TIMESTAMP},host=${DB_HOST}"

    log_success "Backup uploaded to S3: ${s3_path}"
}

# Upload to GCS
upload_to_gcs() {
    log_info "Uploading backup to GCS: gs://${GCS_BUCKET}/${GCS_PREFIX}/"

    local gcs_path="gs://${GCS_BUCKET}/${GCS_PREFIX}/$(basename ${BACKUP_FILE})"

    gsutil -h "x-goog-meta-database:${DB_NAME}" \
           -h "x-goog-meta-timestamp:${TIMESTAMP}" \
           -h "x-goog-meta-host:${DB_HOST}" \
           cp "${BACKUP_FILE}" "${gcs_path}"

    log_success "Backup uploaded to GCS: ${gcs_path}"
}

# Upload to Azure Blob Storage
upload_to_azure() {
    log_info "Uploading backup to Azure Blob Storage: ${AZURE_CONTAINER}/${AZURE_PREFIX}/"

    local blob_name="${AZURE_PREFIX}/$(basename ${BACKUP_FILE})"

    az storage blob upload \
        --container-name "${AZURE_CONTAINER}" \
        --name "${blob_name}" \
        --file "${BACKUP_FILE}" \
        --metadata database=${DB_NAME} timestamp=${TIMESTAMP} host=${DB_HOST}

    log_success "Backup uploaded to Azure: ${AZURE_CONTAINER}/${blob_name}"
}

# Upload to remote storage
upload_to_remote() {
    if [ "$LOCAL_ONLY" = true ]; then
        log_info "Skipping remote upload (local-only mode)"
        return
    fi

    if [ -z "$STORAGE_TYPE" ]; then
        log_warning "No remote storage type configured. Backup will remain local only."
        return
    fi

    case "$STORAGE_TYPE" in
        s3)
            upload_to_s3
            ;;
        gcs)
            upload_to_gcs
            ;;
        azure)
            upload_to_azure
            ;;
        *)
            log_error "Unknown storage type: ${STORAGE_TYPE}"
            log_error "Supported types: s3, gcs, azure"
            exit 1
            ;;
    esac
}

# Apply retention policy (local backups)
apply_local_retention() {
    log_info "Applying retention policy: keeping backups from last ${RETENTION_DAYS} days"

    local deleted_count=0

    # Find and delete old backups
    find "${BACKUP_DIR}" -name "backup_${DB_NAME}_*.sql*" -type f -mtime +${RETENTION_DAYS} | while read -r old_backup; do
        log_verbose "Deleting old backup: ${old_backup}"
        rm -f "${old_backup}"
        deleted_count=$((deleted_count + 1))
    done

    if [ $deleted_count -gt 0 ]; then
        log_success "Deleted ${deleted_count} old backup(s)"
    else
        log_verbose "No old backups to delete"
    fi
}

# Apply retention policy (S3)
apply_s3_retention() {
    log_info "Applying S3 lifecycle policy for ${RETENTION_DAYS} day retention"

    # Create lifecycle policy
    local policy=$(cat <<EOF
{
    "Rules": [{
        "Id": "DeleteOldBackups",
        "Status": "Enabled",
        "Prefix": "${S3_PREFIX}/",
        "Expiration": {
            "Days": ${RETENTION_DAYS}
        }
    }]
}
EOF
)

    echo "${policy}" > /tmp/s3-lifecycle-policy.json

    aws s3api put-bucket-lifecycle-configuration \
        --bucket "${S3_BUCKET}" \
        --lifecycle-configuration file:///tmp/s3-lifecycle-policy.json

    rm /tmp/s3-lifecycle-policy.json

    log_success "S3 lifecycle policy applied"
}

# Apply retention policy (GCS)
apply_gcs_retention() {
    log_info "Applying GCS lifecycle policy for ${RETENTION_DAYS} day retention"

    # Create lifecycle policy
    local policy=$(cat <<EOF
{
    "lifecycle": {
        "rule": [{
            "action": {"type": "Delete"},
            "condition": {
                "age": ${RETENTION_DAYS},
                "matchesPrefix": ["${GCS_PREFIX}/"]
            }
        }]
    }
}
EOF
)

    echo "${policy}" > /tmp/gcs-lifecycle-policy.json

    gsutil lifecycle set /tmp/gcs-lifecycle-policy.json "gs://${GCS_BUCKET}"

    rm /tmp/gcs-lifecycle-policy.json

    log_success "GCS lifecycle policy applied"
}

# Create backup metadata
create_metadata() {
    local metadata_file="${BACKUP_FILE}.meta"

    cat > "${metadata_file}" <<EOF
{
    "database": "${DB_NAME}",
    "host": "${DB_HOST}",
    "port": ${DB_PORT},
    "timestamp": "${TIMESTAMP}",
    "backup_file": "$(basename ${BACKUP_FILE})",
    "compressed": ${COMPRESS},
    "encrypted": ${ENCRYPT},
    "size_bytes": $(stat -f%z "${BACKUP_FILE}" 2>/dev/null || stat -c%s "${BACKUP_FILE}" 2>/dev/null),
    "storage_type": "${STORAGE_TYPE}",
    "retention_days": ${RETENTION_DAYS}
}
EOF

    log_verbose "Metadata created: ${metadata_file}"
}

# Generate backup report
generate_report() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Backup Report${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo "Database: ${DB_NAME}"
    echo "Host: ${DB_HOST}:${DB_PORT}"
    echo "Timestamp: ${TIMESTAMP}"
    echo "Backup File: ${BACKUP_FILE}"
    echo "Compressed: ${COMPRESS}"
    echo "Encrypted: ${ENCRYPT}"
    echo "Storage Type: ${STORAGE_TYPE:-local-only}"
    echo "Retention: ${RETENTION_DAYS} days"

    if [ -f "${BACKUP_FILE}" ]; then
        local size=$(du -h "${BACKUP_FILE}" | cut -f1)
        echo "File Size: ${size}"
    fi

    echo ""
    echo -e "${GREEN}========================================${NC}"
}

# Verify backup integrity
verify_backup() {
    log_info "Verifying backup integrity..."

    if [ "$ENCRYPT" = true ]; then
        log_info "Backup is encrypted. Skipping SQL verification."
        log_info "To verify, decrypt first: gpg -d ${BACKUP_FILE}"
        return
    fi

    local test_file="${BACKUP_FILE}"

    # Decompress if needed for verification
    if [ "$COMPRESS" = true ]; then
        log_verbose "Decompressing for verification..."
        gunzip -c "${BACKUP_FILE}" > /tmp/backup_verify.sql
        test_file="/tmp/backup_verify.sql"
    fi

    # Check if it's valid SQL
    if grep -q "PostgreSQL database dump" "${test_file}"; then
        log_success "Backup integrity verified"
    else
        log_warning "Backup file may be corrupted"
    fi

    # Cleanup
    if [ -f /tmp/backup_verify.sql ]; then
        rm /tmp/backup_verify.sql
    fi
}

# Main execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Database Backup Tool${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    # Check prerequisites
    check_prerequisites

    # Check database (skip if remote-only)
    if [ "$REMOTE_ONLY" = false ]; then
        check_database
    fi

    # Create backup directory
    create_backup_dir

    # Perform backup
    perform_backup

    # Compress backup
    compress_backup

    # Encrypt backup
    encrypt_backup

    # Create metadata
    if [ "$REMOTE_ONLY" = false ]; then
        create_metadata
    fi

    # Verify backup
    if [ "$REMOTE_ONLY" = false ]; then
        verify_backup
    fi

    # Upload to remote storage
    upload_to_remote

    # Apply retention policy
    apply_local_retention

    # Generate report
    generate_report

    log_success "Backup completed successfully!"
}

# Run main
main
