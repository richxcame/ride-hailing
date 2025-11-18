#!/bin/bash

# Database Restore Script with Validation
# Supports restore from local and remote storage (S3, GCS, Azure Blob)
# Usage: ./scripts/restore-database.sh [options]
# Options:
#   --file FILE       : Local backup file to restore
#   --from-remote     : Download and restore from remote storage
#   --storage TYPE    : Remote storage type (s3|gcs|azure)
#   --latest          : Use latest backup (default)
#   --timestamp TS    : Specific backup timestamp (YYYYMMDD_HHMMSS)
#   --database NAME   : Target database name [default: from env]
#   --new-database    : Create new database instead of overwriting
#   --validate-only   : Only validate backup without restoring
#   --skip-validation : Skip pre-restore validation
#   --no-confirm      : Skip confirmation prompts (dangerous!)
#   --verbose         : Show detailed output

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
BACKUP_FILE=""
FROM_REMOTE=false
STORAGE_TYPE=""
USE_LATEST=true
TIMESTAMP=""
NEW_DATABASE=false
VALIDATE_ONLY=false
SKIP_VALIDATION=false
NO_CONFIRM=false
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --file)
            BACKUP_FILE="$2"
            shift 2
            ;;
        --from-remote)
            FROM_REMOTE=true
            shift
            ;;
        --storage)
            STORAGE_TYPE="$2"
            shift 2
            ;;
        --latest)
            USE_LATEST=true
            shift
            ;;
        --timestamp)
            TIMESTAMP="$2"
            USE_LATEST=false
            shift 2
            ;;
        --database)
            DB_NAME="$2"
            shift 2
            ;;
        --new-database)
            NEW_DATABASE=true
            shift
            ;;
        --validate-only)
            VALIDATE_ONLY=true
            shift
            ;;
        --skip-validation)
            SKIP_VALIDATION=true
            shift
            ;;
        --no-confirm)
            NO_CONFIRM=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Usage: $0 [--file FILE] [--from-remote] [--storage TYPE] [--latest] [--timestamp TS] [--database NAME] [--new-database] [--validate-only] [--skip-validation] [--no-confirm] [--verbose]"
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

# Remote storage configuration
STORAGE_TYPE=${STORAGE_TYPE:-${BACKUP_STORAGE_TYPE:-}}
S3_BUCKET=${BACKUP_S3_BUCKET:-}
S3_PREFIX=${BACKUP_S3_PREFIX:-database-backups}
GCS_BUCKET=${BACKUP_GCS_BUCKET:-}
GCS_PREFIX=${BACKUP_GCS_PREFIX:-database-backups}
AZURE_CONTAINER=${BACKUP_AZURE_CONTAINER:-}
AZURE_PREFIX=${BACKUP_AZURE_PREFIX:-database-backups}

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

    # Check psql
    if ! command -v psql &> /dev/null; then
        log_error "psql not found. Install PostgreSQL client tools."
        exit 1
    fi

    # Check remote storage tools if needed
    if [ "$FROM_REMOTE" = true ]; then
        if [ "$STORAGE_TYPE" = "s3" ]; then
            if ! command -v aws &> /dev/null; then
                log_error "AWS CLI not found. Install aws-cli."
                exit 1
            fi
        elif [ "$STORAGE_TYPE" = "gcs" ]; then
            if ! command -v gsutil &> /dev/null; then
                log_error "gsutil not found. Install Google Cloud SDK."
                exit 1
            fi
        elif [ "$STORAGE_TYPE" = "azure" ]; then
            if ! command -v az &> /dev/null; then
                log_error "Azure CLI not found. Install azure-cli."
                exit 1
            fi
        fi
    fi

    log_success "Prerequisites check passed"
}

# Check database connectivity
check_database() {
    log_info "Checking database connectivity..."

    if ! PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -c '\q' 2>/dev/null; then
        log_error "Cannot connect to PostgreSQL at ${DB_HOST}:${DB_PORT}"
        exit 1
    fi

    log_success "Database server is accessible"
}

# List available local backups
list_local_backups() {
    log_info "Available local backups:"

    if [ ! -d "${BACKUP_DIR}" ]; then
        log_warning "Backup directory not found: ${BACKUP_DIR}"
        return 1
    fi

    local count=0
    find "${BACKUP_DIR}" -name "backup_${DB_NAME}_*.sql*" -type f | sort -r | head -10 | while read -r backup; do
        local size=$(du -h "${backup}" | cut -f1)
        local filename=$(basename "${backup}")
        echo "  - ${filename} (${size})"
        count=$((count + 1))
    done

    if [ $count -eq 0 ]; then
        log_warning "No local backups found for database: ${DB_NAME}"
        return 1
    fi

    return 0
}

# List available remote backups (S3)
list_s3_backups() {
    log_info "Available S3 backups:"

    aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/" --recursive | \
        grep "backup_${DB_NAME}_" | \
        tail -10 | \
        awk '{printf "  - %s (%s %s)\n", $4, $3, $2}'
}

# List available remote backups (GCS)
list_gcs_backups() {
    log_info "Available GCS backups:"

    gsutil ls "gs://${GCS_BUCKET}/${GCS_PREFIX}/backup_${DB_NAME}_*" | \
        tail -10 | \
        xargs -I {} gsutil ls -l {} | \
        awk '{printf "  - %s (%s)\n", $3, $1}'
}

# List available remote backups (Azure)
list_azure_backups() {
    log_info "Available Azure backups:"

    az storage blob list \
        --container-name "${AZURE_CONTAINER}" \
        --prefix "${AZURE_PREFIX}/backup_${DB_NAME}_" \
        --query "[].{name:name, size:properties.contentLength}" \
        --output table
}

# Find latest local backup
find_latest_local_backup() {
    find "${BACKUP_DIR}" -name "backup_${DB_NAME}_*.sql*" -type f | sort -r | head -1
}

# Find backup by timestamp
find_backup_by_timestamp() {
    local ts=$1
    find "${BACKUP_DIR}" -name "backup_${DB_NAME}_${ts}*" -type f | head -1
}

# Download from S3
download_from_s3() {
    local remote_file=$1
    local local_file="${BACKUP_DIR}/$(basename ${remote_file})"

    log_info "Downloading from S3: ${remote_file}"

    aws s3 cp "s3://${S3_BUCKET}/${S3_PREFIX}/${remote_file}" "${local_file}"

    echo "${local_file}"
}

# Download from GCS
download_from_gcs() {
    local remote_file=$1
    local local_file="${BACKUP_DIR}/$(basename ${remote_file})"

    log_info "Downloading from GCS: ${remote_file}"

    gsutil cp "gs://${GCS_BUCKET}/${GCS_PREFIX}/${remote_file}" "${local_file}"

    echo "${local_file}"
}

# Download from Azure
download_from_azure() {
    local remote_file=$1
    local local_file="${BACKUP_DIR}/$(basename ${remote_file})"

    log_info "Downloading from Azure: ${remote_file}"

    az storage blob download \
        --container-name "${AZURE_CONTAINER}" \
        --name "${AZURE_PREFIX}/${remote_file}" \
        --file "${local_file}"

    echo "${local_file}"
}

# Determine backup file to restore
determine_backup_file() {
    if [ -n "$BACKUP_FILE" ]; then
        # Use specified file
        if [ ! -f "$BACKUP_FILE" ]; then
            log_error "Backup file not found: ${BACKUP_FILE}"
            exit 1
        fi
        echo "$BACKUP_FILE"
        return
    fi

    if [ "$FROM_REMOTE" = true ]; then
        # List and download from remote
        case "$STORAGE_TYPE" in
            s3)
                list_s3_backups
                local remote_file=$(aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/" --recursive | grep "backup_${DB_NAME}_" | sort -r | head -1 | awk '{print $4}')
                download_from_s3 "$(basename ${remote_file})"
                ;;
            gcs)
                list_gcs_backups
                local remote_file=$(gsutil ls "gs://${GCS_BUCKET}/${GCS_PREFIX}/backup_${DB_NAME}_*" | sort -r | head -1)
                download_from_gcs "$(basename ${remote_file})"
                ;;
            azure)
                list_azure_backups
                local remote_file=$(az storage blob list --container-name "${AZURE_CONTAINER}" --prefix "${AZURE_PREFIX}/backup_${DB_NAME}_" --query "[0].name" -o tsv)
                download_from_azure "$(basename ${remote_file})"
                ;;
            *)
                log_error "Unknown storage type: ${STORAGE_TYPE}"
                exit 1
                ;;
        esac
    else
        # Use local backup
        list_local_backups

        if [ -n "$TIMESTAMP" ]; then
            local backup=$(find_backup_by_timestamp "$TIMESTAMP")
            if [ -z "$backup" ]; then
                log_error "No backup found for timestamp: ${TIMESTAMP}"
                exit 1
            fi
            echo "$backup"
        else
            local backup=$(find_latest_local_backup)
            if [ -z "$backup" ]; then
                log_error "No backups found for database: ${DB_NAME}"
                exit 1
            fi
            echo "$backup"
        fi
    fi
}

# Validate backup file
validate_backup() {
    local backup_file=$1

    log_info "Validating backup file: ${backup_file}"

    # Check if file exists
    if [ ! -f "$backup_file" ]; then
        log_error "Backup file not found: ${backup_file}"
        exit 1
    fi

    # Check file size
    local size=$(stat -f%z "${backup_file}" 2>/dev/null || stat -c%s "${backup_file}" 2>/dev/null)
    if [ "$size" -eq 0 ]; then
        log_error "Backup file is empty"
        exit 1
    fi

    log_verbose "Backup file size: $(du -h ${backup_file} | cut -f1)"

    # Determine file type
    local is_encrypted=false
    local is_compressed=false

    if [[ "$backup_file" == *.gpg ]]; then
        is_encrypted=true
        log_info "Backup is encrypted (GPG)"
    fi

    if [[ "$backup_file" == *.gz ]] || [[ "$backup_file" == *.gz.gpg ]]; then
        is_compressed=true
        log_info "Backup is compressed (gzip)"
    fi

    # Validate content (if not encrypted)
    if [ "$is_encrypted" = false ]; then
        local test_file="$backup_file"

        # Decompress temporarily if needed
        if [ "$is_compressed" = true ]; then
            log_verbose "Decompressing for validation..."
            gunzip -c "$backup_file" > /tmp/restore_validate.sql
            test_file="/tmp/restore_validate.sql"
        fi

        # Check if it's valid SQL
        if grep -q "PostgreSQL database dump" "$test_file" 2>/dev/null; then
            log_success "Backup file is valid PostgreSQL dump"
        else
            log_warning "Backup file may not be a valid PostgreSQL dump"
        fi

        # Cleanup
        if [ -f /tmp/restore_validate.sql ]; then
            rm /tmp/restore_validate.sql
        fi
    else
        log_warning "Cannot validate encrypted backup. Will attempt restore anyway."
    fi
}

# Prepare backup file for restore
prepare_backup() {
    local backup_file=$1

    # Decrypt if needed
    if [[ "$backup_file" == *.gpg ]]; then
        log_info "Decrypting backup..."
        local decrypted_file="${backup_file%.gpg}"
        gpg --decrypt "${backup_file}" > "${decrypted_file}"
        backup_file="${decrypted_file}"
    fi

    # Decompress if needed
    if [[ "$backup_file" == *.gz ]]; then
        log_info "Decompressing backup..."
        gunzip -k "$backup_file"
        backup_file="${backup_file%.gz}"
    fi

    echo "$backup_file"
}

# Get current database info
get_database_info() {
    local exists=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -t -c "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}';" 2>/dev/null | xargs)

    if [ "$exists" = "1" ]; then
        local size=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -t -c "SELECT pg_size_pretty(pg_database_size('${DB_NAME}'));" 2>/dev/null | xargs)
        local tables=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';" 2>/dev/null | xargs)

        echo "exists=true size=${size} tables=${tables}"
    else
        echo "exists=false"
    fi
}

# Confirm restore operation
confirm_restore() {
    if [ "$NO_CONFIRM" = true ]; then
        return 0
    fi

    local db_info=$(get_database_info)

    echo ""
    echo -e "${YELLOW}========================================${NC}"
    echo -e "${YELLOW}WARNING: Database Restore Operation${NC}"
    echo -e "${YELLOW}========================================${NC}"

    if [[ "$db_info" == *"exists=true"* ]]; then
        local size=$(echo "$db_info" | grep -o 'size=[^ ]*' | cut -d= -f2)
        local tables=$(echo "$db_info" | grep -o 'tables=[^ ]*' | cut -d= -f2)

        echo -e "${RED}This will DESTROY the existing database!${NC}"
        echo ""
        echo "Current database: ${DB_NAME}"
        echo "Current size: ${size}"
        echo "Current tables: ${tables}"
    else
        echo "Database ${DB_NAME} does not exist. It will be created."
    fi

    echo ""
    echo "Backup file: ${BACKUP_FILE}"
    echo ""
    echo -e "${YELLOW}This operation cannot be undone!${NC}"
    echo ""
    read -p "Are you sure you want to continue? (yes/no): " -r
    echo

    if [[ ! $REPLY =~ ^[Yy][Ee][Ss]$ ]]; then
        log_info "Restore cancelled by user"
        exit 0
    fi
}

# Create database backup before restore
backup_before_restore() {
    local db_info=$(get_database_info)

    if [[ "$db_info" == *"exists=true"* ]]; then
        log_info "Creating safety backup before restore..."

        local safety_backup="${BACKUP_DIR}/safety_backup_${DB_NAME}_$(date +%Y%m%d_%H%M%S).sql"

        PGPASSWORD=${DB_PASSWORD} pg_dump \
            -h ${DB_HOST} \
            -p ${DB_PORT} \
            -U ${DB_USER} \
            -d ${DB_NAME} \
            --format=plain \
            --no-owner \
            --no-acl \
            > "${safety_backup}"

        gzip -f "${safety_backup}"

        log_success "Safety backup created: ${safety_backup}.gz"
    fi
}

# Drop and recreate database
recreate_database() {
    log_info "Recreating database: ${DB_NAME}"

    # Terminate existing connections
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -c "
        SELECT pg_terminate_backend(pid)
        FROM pg_stat_activity
        WHERE datname = '${DB_NAME}' AND pid <> pg_backend_pid();
    " 2>/dev/null || true

    # Drop database
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -c "DROP DATABASE IF EXISTS ${DB_NAME};" 2>/dev/null || true

    # Create database
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -c "CREATE DATABASE ${DB_NAME};"

    log_success "Database recreated"
}

# Perform restore
perform_restore() {
    local backup_file=$1

    log_info "Restoring database from: ${backup_file}"

    local start_time=$(date +%s)

    # Restore database
    if [ "$VERBOSE" = true ]; then
        PGPASSWORD=${DB_PASSWORD} psql \
            -h ${DB_HOST} \
            -p ${DB_PORT} \
            -U ${DB_USER} \
            -d ${DB_NAME} \
            --set ON_ERROR_STOP=on \
            -f "${backup_file}"
    else
        PGPASSWORD=${DB_PASSWORD} psql \
            -h ${DB_HOST} \
            -p ${DB_PORT} \
            -U ${DB_USER} \
            -d ${DB_NAME} \
            --set ON_ERROR_STOP=on \
            -f "${backup_file}" \
            > /dev/null 2>&1
    fi

    local end_time=$(date +%s)
    local duration=$((end_time - start_time))

    log_success "Database restored successfully"
    log_info "Restore duration: ${duration} seconds"
}

# Verify restore
verify_restore() {
    log_info "Verifying restored database..."

    # Check database exists
    local exists=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -t -c "SELECT 1 FROM pg_database WHERE datname='${DB_NAME}';" 2>/dev/null | xargs)

    if [ "$exists" != "1" ]; then
        log_error "Database does not exist after restore"
        exit 1
    fi

    # Get table count
    local tables=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema='public';" 2>/dev/null | xargs)

    # Get database size
    local size=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -t -c "SELECT pg_size_pretty(pg_database_size('${DB_NAME}'));" 2>/dev/null | xargs)

    log_success "Verification completed"
    log_info "Tables: ${tables}"
    log_info "Database size: ${size}"

    # Check for critical tables
    local critical_tables=("users" "drivers" "rides" "payments")
    for table in "${critical_tables[@]}"; do
        local count=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${DB_NAME} -t -c "SELECT COUNT(*) FROM ${table};" 2>/dev/null | xargs)
        log_verbose "Table ${table}: ${count} rows"
    done
}

# Generate restore report
generate_report() {
    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Restore Report${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo "Database: ${DB_NAME}"
    echo "Host: ${DB_HOST}:${DB_PORT}"
    echo "Backup File: ${BACKUP_FILE}"

    local db_info=$(get_database_info)
    if [[ "$db_info" == *"exists=true"* ]]; then
        local size=$(echo "$db_info" | grep -o 'size=[^ ]*' | cut -d= -f2)
        local tables=$(echo "$db_info" | grep -o 'tables=[^ ]*' | cut -d= -f2)
        echo "Current Size: ${size}"
        echo "Tables: ${tables}"
    fi

    echo ""
    echo -e "${GREEN}========================================${NC}"
}

# Main execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Database Restore Tool${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    # Check prerequisites
    check_prerequisites

    # Check database connectivity
    check_database

    # Create backup directory if needed
    mkdir -p "${BACKUP_DIR}"

    # Determine backup file
    BACKUP_FILE=$(determine_backup_file)
    log_info "Selected backup: ${BACKUP_FILE}"

    # Validate backup
    if [ "$SKIP_VALIDATION" = false ]; then
        validate_backup "$BACKUP_FILE"
    fi

    # Exit if validate-only mode
    if [ "$VALIDATE_ONLY" = true ]; then
        log_success "Validation complete. Exiting."
        exit 0
    fi

    # Prepare backup (decrypt/decompress)
    BACKUP_FILE=$(prepare_backup "$BACKUP_FILE")

    # Confirm restore
    confirm_restore

    # Create safety backup
    backup_before_restore

    # Recreate database
    recreate_database

    # Perform restore
    perform_restore "$BACKUP_FILE"

    # Verify restore
    verify_restore

    # Generate report
    generate_report

    log_success "Restore completed successfully!"
}

# Run main
main
