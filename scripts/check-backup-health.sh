#!/bin/bash

# Backup Health Check Script
# Monitors backup status and sends alerts if backups are failing or missing
# Usage: ./scripts/check-backup-health.sh [options]
# Options:
#   --alert-email EMAIL  : Send alerts to this email
#   --slack-webhook URL  : Send alerts to Slack
#   --max-age HOURS      : Maximum backup age in hours [default: 26]
#   --storage TYPE       : Check remote storage (s3|gcs|azure)
#   --verbose            : Show detailed output

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
ALERT_EMAIL=""
SLACK_WEBHOOK=""
MAX_BACKUP_AGE_HOURS=26  # Daily backup should not be older than 26 hours
STORAGE_TYPE=""
VERBOSE=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --alert-email)
            ALERT_EMAIL="$2"
            shift 2
            ;;
        --slack-webhook)
            SLACK_WEBHOOK="$2"
            shift 2
            ;;
        --max-age)
            MAX_BACKUP_AGE_HOURS="$2"
            shift 2
            ;;
        --storage)
            STORAGE_TYPE="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            exit 1
            ;;
    esac
done

# Load environment variables
if [ -f .env ]; then
    export $(cat .env | grep -v '^#' | xargs)
fi

# Configuration
DB_NAME=${DB_NAME:-ridehailing}
BACKUP_DIR=${BACKUP_DIR:-backups}
STORAGE_TYPE=${STORAGE_TYPE:-${BACKUP_STORAGE_TYPE:-}}
S3_BUCKET=${BACKUP_S3_BUCKET:-}
S3_PREFIX=${BACKUP_S3_PREFIX:-database-backups}
ALERT_EMAIL=${ALERT_EMAIL:-${BACKUP_ALERT_EMAIL:-}}
SLACK_WEBHOOK=${SLACK_WEBHOOK:-${BACKUP_SLACK_WEBHOOK:-}}

# Health check results
HEALTH_STATUS="OK"
HEALTH_MESSAGES=()

# Logging functions
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
    HEALTH_STATUS="WARNING"
    HEALTH_MESSAGES+=("WARNING: $1")
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    HEALTH_STATUS="ERROR"
    HEALTH_MESSAGES+=("ERROR: $1")
}

log_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# Check local backups
check_local_backups() {
    log_info "Checking local backups in ${BACKUP_DIR}"

    if [ ! -d "${BACKUP_DIR}" ]; then
        log_error "Backup directory not found: ${BACKUP_DIR}"
        return 1
    fi

    # Find latest backup
    local latest_backup=$(find "${BACKUP_DIR}" -name "backup_${DB_NAME}_*.sql*" -type f | sort -r | head -1)

    if [ -z "$latest_backup" ]; then
        log_error "No backups found for database: ${DB_NAME}"
        return 1
    fi

    log_verbose "Latest backup: ${latest_backup}"

    # Check backup age
    local backup_time=$(stat -f%m "${latest_backup}" 2>/dev/null || stat -c%Y "${latest_backup}" 2>/dev/null)
    local current_time=$(date +%s)
    local age_seconds=$((current_time - backup_time))
    local age_hours=$((age_seconds / 3600))

    log_info "Latest backup age: ${age_hours} hours"

    if [ $age_hours -gt $MAX_BACKUP_AGE_HOURS ]; then
        log_error "Latest backup is too old: ${age_hours} hours (max: ${MAX_BACKUP_AGE_HOURS})"
        return 1
    fi

    # Check backup size
    local backup_size=$(stat -f%z "${latest_backup}" 2>/dev/null || stat -c%s "${latest_backup}" 2>/dev/null)
    local backup_size_mb=$((backup_size / 1024 / 1024))

    log_info "Latest backup size: ${backup_size_mb} MB"

    if [ $backup_size -eq 0 ]; then
        log_error "Backup file is empty"
        return 1
    fi

    if [ $backup_size_mb -lt 1 ]; then
        log_warning "Backup size is suspiciously small: ${backup_size_mb} MB"
    fi

    # Check number of backups
    local backup_count=$(find "${BACKUP_DIR}" -name "backup_${DB_NAME}_*.sql*" -type f | wc -l)
    log_info "Total local backups: ${backup_count}"

    if [ $backup_count -lt 2 ]; then
        log_warning "Only ${backup_count} backup(s) found. Recommended: at least 2"
    fi

    log_success "Local backup health check passed"
    return 0
}

# Check S3 backups
check_s3_backups() {
    log_info "Checking S3 backups: s3://${S3_BUCKET}/${S3_PREFIX}/"

    if ! command -v aws &> /dev/null; then
        log_warning "AWS CLI not installed. Skipping S3 check."
        return 0
    fi

    # List recent backups
    local latest_backup=$(aws s3 ls "s3://${S3_BUCKET}/${S3_PREFIX}/" --recursive | grep "backup_${DB_NAME}_" | sort -r | head -1)

    if [ -z "$latest_backup" ]; then
        log_error "No S3 backups found for database: ${DB_NAME}"
        return 1
    fi

    log_verbose "Latest S3 backup: ${latest_backup}"

    # Parse backup date
    local backup_date=$(echo "$latest_backup" | awk '{print $1, $2}')
    local backup_timestamp=$(date -d "$backup_date" +%s 2>/dev/null || date -j -f "%Y-%m-%d %H:%M:%S" "$backup_date" +%s 2>/dev/null)
    local current_time=$(date +%s)
    local age_seconds=$((current_time - backup_timestamp))
    local age_hours=$((age_seconds / 3600))

    log_info "Latest S3 backup age: ${age_hours} hours"

    if [ $age_hours -gt $MAX_BACKUP_AGE_HOURS ]; then
        log_error "Latest S3 backup is too old: ${age_hours} hours"
        return 1
    fi

    # Check backup size
    local backup_size=$(echo "$latest_backup" | awk '{print $3}')
    local backup_size_mb=$((backup_size / 1024 / 1024))

    log_info "Latest S3 backup size: ${backup_size_mb} MB"

    if [ $backup_size_mb -lt 1 ]; then
        log_warning "S3 backup size is suspiciously small: ${backup_size_mb} MB"
    fi

    log_success "S3 backup health check passed"
    return 0
}

# Check GCS backups
check_gcs_backups() {
    log_info "Checking GCS backups: gs://${GCS_BUCKET}/${GCS_PREFIX}/"

    if ! command -v gsutil &> /dev/null; then
        log_warning "gsutil not installed. Skipping GCS check."
        return 0
    fi

    # List recent backups
    local latest_backup=$(gsutil ls -l "gs://${GCS_BUCKET}/${GCS_PREFIX}/backup_${DB_NAME}_*" | sort -r | head -1)

    if [ -z "$latest_backup" ]; then
        log_error "No GCS backups found for database: ${DB_NAME}"
        return 1
    fi

    log_verbose "Latest GCS backup: ${latest_backup}"
    log_success "GCS backup health check passed"
    return 0
}

# Send email alert
send_email_alert() {
    if [ -z "$ALERT_EMAIL" ]; then
        return
    fi

    local subject="Database Backup Health Alert - ${HEALTH_STATUS}"
    local body="Database: ${DB_NAME}\nStatus: ${HEALTH_STATUS}\n\n"

    for message in "${HEALTH_MESSAGES[@]}"; do
        body="${body}${message}\n"
    done

    echo -e "$body" | mail -s "$subject" "$ALERT_EMAIL"
    log_info "Email alert sent to ${ALERT_EMAIL}"
}

# Send Slack alert
send_slack_alert() {
    if [ -z "$SLACK_WEBHOOK" ]; then
        return
    fi

    local color="good"
    if [ "$HEALTH_STATUS" = "WARNING" ]; then
        color="warning"
    elif [ "$HEALTH_STATUS" = "ERROR" ]; then
        color="danger"
    fi

    local message_text=""
    for message in "${HEALTH_MESSAGES[@]}"; do
        message_text="${message_text}${message}\n"
    done

    local payload=$(cat <<EOF
{
    "attachments": [{
        "color": "${color}",
        "title": "Database Backup Health Check",
        "fields": [
            {
                "title": "Database",
                "value": "${DB_NAME}",
                "short": true
            },
            {
                "title": "Status",
                "value": "${HEALTH_STATUS}",
                "short": true
            },
            {
                "title": "Messages",
                "value": "${message_text}",
                "short": false
            }
        ],
        "footer": "Backup Health Monitor",
        "ts": $(date +%s)
    }]
}
EOF
)

    curl -X POST -H 'Content-type: application/json' --data "$payload" "$SLACK_WEBHOOK"
    log_info "Slack alert sent"
}

# Send Prometheus metrics
send_metrics() {
    # TODO: Push metrics to Prometheus Pushgateway
    # Example: backup_age_hours, backup_size_bytes, backup_health_status
    log_verbose "Metrics: backup_health_status=${HEALTH_STATUS}"
}

# Generate health report
generate_report() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Backup Health Report${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo "Database: ${DB_NAME}"
    echo "Check Time: $(date)"
    echo "Status: ${HEALTH_STATUS}"
    echo ""

    if [ ${#HEALTH_MESSAGES[@]} -gt 0 ]; then
        echo "Messages:"
        for message in "${HEALTH_MESSAGES[@]}"; do
            echo "  - ${message}"
        done
    else
        echo "All checks passed!"
    fi

    echo ""
    echo -e "${BLUE}========================================${NC}"
}

# Main execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Database Backup Health Check${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    # Check local backups
    check_local_backups || true

    # Check remote backups
    if [ "$STORAGE_TYPE" = "s3" ]; then
        check_s3_backups || true
    elif [ "$STORAGE_TYPE" = "gcs" ]; then
        check_gcs_backups || true
    fi

    # Send alerts if needed
    if [ "$HEALTH_STATUS" != "OK" ]; then
        send_email_alert
        send_slack_alert
    fi

    # Send metrics
    send_metrics

    # Generate report
    generate_report

    # Exit with appropriate status
    if [ "$HEALTH_STATUS" = "ERROR" ]; then
        exit 1
    elif [ "$HEALTH_STATUS" = "WARNING" ]; then
        exit 0  # Don't fail on warnings
    else
        log_success "All backup health checks passed!"
        exit 0
    fi
}

# Run main
main
