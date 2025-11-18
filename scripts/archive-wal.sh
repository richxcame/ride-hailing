#!/bin/bash

# WAL Archive Script for PostgreSQL PITR
# This script is called by PostgreSQL's archive_command
# Usage: archive_command = '/path/to/archive-wal.sh %p %f'
#   %p = full path of WAL file
#   %f = WAL filename only

set -e

# Arguments from PostgreSQL
WAL_PATH="$1"      # Full path: /var/lib/postgresql/15/main/pg_wal/000000010000000000000042
WAL_FILE="$2"      # Filename only: 000000010000000000000042

# Load environment variables
if [ -f /etc/postgresql/wal-archive.env ]; then
    source /etc/postgresql/wal-archive.env
fi

# Configuration
PITR_ENABLED=${PITR_ENABLED:-true}
PITR_WAL_ARCHIVE_DIR=${PITR_WAL_ARCHIVE_DIR:-/var/lib/postgresql/wal_archive}
PITR_STORAGE_TYPE=${PITR_STORAGE_TYPE:-}
PITR_COMPRESS_WAL=${PITR_COMPRESS_WAL:-true}
PITR_COMPRESSION_LEVEL=${PITR_COMPRESSION_LEVEL:-6}

# S3 Configuration
PITR_S3_BUCKET=${PITR_S3_BUCKET:-}
PITR_S3_PREFIX=${PITR_S3_PREFIX:-wal-files}

# GCS Configuration
PITR_GCS_BUCKET=${PITR_GCS_BUCKET:-}
PITR_GCS_PREFIX=${PITR_GCS_PREFIX:-wal-files}

# Azure Configuration
PITR_AZURE_CONTAINER=${PITR_AZURE_CONTAINER:-}
PITR_AZURE_PREFIX=${PITR_AZURE_PREFIX:-wal-files}

# Logging
LOG_FILE=${PITR_LOG_FILE:-/var/log/postgresql/wal-archive.log}

# Ensure log directory exists
mkdir -p "$(dirname $LOG_FILE)"

# Logging function
log() {
    echo "[$(date +'%Y-%m-%d %H:%M:%S')] $1" >> "$LOG_FILE"
}

# Error handling
error_exit() {
    log "ERROR: $1"
    exit 1
}

# Check if PITR is enabled
if [ "$PITR_ENABLED" != "true" ]; then
    log "PITR disabled, skipping WAL archive for ${WAL_FILE}"
    exit 0
fi

# Validate inputs
if [ -z "$WAL_PATH" ] || [ -z "$WAL_FILE" ]; then
    error_exit "Missing required arguments: WAL_PATH or WAL_FILE"
fi

if [ ! -f "$WAL_PATH" ]; then
    error_exit "WAL file not found: ${WAL_PATH}"
fi

# Create archive directory if it doesn't exist
if [ ! -d "$PITR_WAL_ARCHIVE_DIR" ]; then
    mkdir -p "$PITR_WAL_ARCHIVE_DIR" || error_exit "Failed to create archive directory"
    chmod 700 "$PITR_WAL_ARCHIVE_DIR"
fi

# Archive to local directory
LOCAL_ARCHIVE_PATH="${PITR_WAL_ARCHIVE_DIR}/${WAL_FILE}"

# Copy to local archive
if [ "$PITR_COMPRESS_WAL" = "true" ]; then
    # Compress and copy
    gzip -c -${PITR_COMPRESSION_LEVEL} "$WAL_PATH" > "${LOCAL_ARCHIVE_PATH}.gz" || error_exit "Failed to compress and copy WAL file"
    log "Compressed and archived WAL file: ${WAL_FILE}.gz"
else
    # Just copy
    cp "$WAL_PATH" "$LOCAL_ARCHIVE_PATH" || error_exit "Failed to copy WAL file to local archive"
    log "Archived WAL file: ${WAL_FILE}"
fi

# Upload to remote storage
if [ -n "$PITR_STORAGE_TYPE" ]; then
    case "$PITR_STORAGE_TYPE" in
        s3)
            if [ -z "$PITR_S3_BUCKET" ]; then
                error_exit "PITR_S3_BUCKET not configured"
            fi

            # Determine file to upload
            if [ "$PITR_COMPRESS_WAL" = "true" ]; then
                UPLOAD_FILE="${LOCAL_ARCHIVE_PATH}.gz"
                REMOTE_FILE="${WAL_FILE}.gz"
            else
                UPLOAD_FILE="$LOCAL_ARCHIVE_PATH"
                REMOTE_FILE="$WAL_FILE"
            fi

            # Upload to S3
            if command -v aws &> /dev/null; then
                aws s3 cp "$UPLOAD_FILE" "s3://${PITR_S3_BUCKET}/${PITR_S3_PREFIX}/${REMOTE_FILE}" \
                    --storage-class STANDARD_IA \
                    --metadata "source=postgresql,archived=$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
                    || error_exit "Failed to upload WAL file to S3"
                log "Uploaded to S3: s3://${PITR_S3_BUCKET}/${PITR_S3_PREFIX}/${REMOTE_FILE}"
            else
                log "WARNING: aws command not found, skipping S3 upload"
            fi
            ;;

        gcs)
            if [ -z "$PITR_GCS_BUCKET" ]; then
                error_exit "PITR_GCS_BUCKET not configured"
            fi

            # Determine file to upload
            if [ "$PITR_COMPRESS_WAL" = "true" ]; then
                UPLOAD_FILE="${LOCAL_ARCHIVE_PATH}.gz"
                REMOTE_FILE="${WAL_FILE}.gz"
            else
                UPLOAD_FILE="$LOCAL_ARCHIVE_PATH"
                REMOTE_FILE="$WAL_FILE"
            fi

            # Upload to GCS
            if command -v gsutil &> /dev/null; then
                gsutil -h "x-goog-meta-source:postgresql" \
                       -h "x-goog-meta-archived:$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
                       cp "$UPLOAD_FILE" "gs://${PITR_GCS_BUCKET}/${PITR_GCS_PREFIX}/${REMOTE_FILE}" \
                       || error_exit "Failed to upload WAL file to GCS"
                log "Uploaded to GCS: gs://${PITR_GCS_BUCKET}/${PITR_GCS_PREFIX}/${REMOTE_FILE}"
            else
                log "WARNING: gsutil command not found, skipping GCS upload"
            fi
            ;;

        azure)
            if [ -z "$PITR_AZURE_CONTAINER" ]; then
                error_exit "PITR_AZURE_CONTAINER not configured"
            fi

            # Determine file to upload
            if [ "$PITR_COMPRESS_WAL" = "true" ]; then
                UPLOAD_FILE="${LOCAL_ARCHIVE_PATH}.gz"
                REMOTE_FILE="${WAL_FILE}.gz"
            else
                UPLOAD_FILE="$LOCAL_ARCHIVE_PATH"
                REMOTE_FILE="$WAL_FILE"
            fi

            # Upload to Azure
            if command -v az &> /dev/null; then
                az storage blob upload \
                    --container-name "$PITR_AZURE_CONTAINER" \
                    --name "${PITR_AZURE_PREFIX}/${REMOTE_FILE}" \
                    --file "$UPLOAD_FILE" \
                    --metadata source=postgresql archived="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
                    || error_exit "Failed to upload WAL file to Azure"
                log "Uploaded to Azure: ${PITR_AZURE_CONTAINER}/${PITR_AZURE_PREFIX}/${REMOTE_FILE}"
            else
                log "WARNING: az command not found, skipping Azure upload"
            fi
            ;;

        *)
            log "WARNING: Unknown storage type: ${PITR_STORAGE_TYPE}"
            ;;
    esac
fi

# Successful archival
log "Successfully archived ${WAL_FILE}"
exit 0
