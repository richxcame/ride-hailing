#!/bin/bash

# Pre-commit hook to check migration file naming convention
# Migration files should follow the pattern: NNNNNN_description.up.sql / NNNNNN_description.down.sql

set -e

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

ERRORS=0

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
    ERRORS=$((ERRORS + 1))
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check if migration files were passed
if [ $# -eq 0 ]; then
    exit 0
fi

echo "Checking migration naming convention..."

for file in "$@"; do
    # Only check files in db/migrations directory
    if [[ ! "$file" =~ ^db/migrations/ ]]; then
        continue
    fi

    filename=$(basename "$file")

    # Check naming pattern: 000000_name.up.sql or 000000_name.down.sql
    if [[ ! "$filename" =~ ^[0-9]{6}_[a-z_]+\.(up|down)\.sql$ ]]; then
        log_error "Invalid migration filename: $filename"
        echo "  Expected format: NNNNNN_description.up.sql or NNNNNN_description.down.sql"
        echo "  Example: 000001_create_users_table.up.sql"
        continue
    fi

    # Extract version number
    version=$(echo "$filename" | cut -d'_' -f1)

    # Check if both UP and DOWN migrations exist
    up_file="db/migrations/${version}_*.up.sql"
    down_file="db/migrations/${version}_*.down.sql"

    # Check for matching pair
    if [[ "$filename" =~ \.up\.sql$ ]]; then
        corresponding_down=$(echo "$filename" | sed 's/\.up\.sql$/.down.sql/')
        if [ ! -f "db/migrations/$corresponding_down" ]; then
            log_error "Missing corresponding DOWN migration for: $filename"
            continue
        fi
    elif [[ "$filename" =~ \.down\.sql$ ]]; then
        corresponding_up=$(echo "$filename" | sed 's/\.down\.sql$/.up.sql/')
        if [ ! -f "db/migrations/$corresponding_up" ]; then
            log_error "Missing corresponding UP migration for: $filename"
            continue
        fi
    fi

    log_success "Migration naming is correct: $filename"
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo -e "${RED}Migration naming check failed with $ERRORS error(s)${NC}"
    echo ""
    echo "Migration Naming Convention:"
    echo "  - Files must be in db/migrations/ directory"
    echo "  - Format: NNNNNN_description.up.sql and NNNNNN_description.down.sql"
    echo "  - Version: 6 digits (e.g., 000001, 000002, ...)"
    echo "  - Description: lowercase with underscores (e.g., create_users_table)"
    echo "  - Each migration must have both UP and DOWN files"
    echo ""
    exit 1
fi

echo ""
echo -e "${GREEN}All migration file names are valid!${NC}"
exit 0
