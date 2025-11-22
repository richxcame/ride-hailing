#!/bin/bash

# Pre-commit hook to validate migration files
# This hook checks migration files for common issues before commit

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

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

# Check if migration files were passed
if [ $# -eq 0 ]; then
    echo "No migration files to validate"
    exit 0
fi

echo "Validating migration files..."

for file in "$@"; do
    # Skip non-SQL files
    if [[ ! "$file" =~ \.sql$ ]]; then
        continue
    fi

    echo "Checking: $file"

    # Check 1: File exists and is readable
    if [ ! -f "$file" ] || [ ! -r "$file" ]; then
        log_error "File not found or not readable: $file"
        continue
    fi

    # Check 2: File is not empty
    if [ ! -s "$file" ]; then
        log_error "Migration file is empty: $file"
        continue
    fi

    # Check 3: File has proper SQL syntax (basic check)
    if ! grep -q -i -E '(CREATE|ALTER|DROP|INSERT|UPDATE|DELETE|BEGIN|COMMIT)' "$file"; then
        log_warning "File may not contain valid SQL statements: $file"
    fi

    # Check 4: DOWN migration should reverse UP migration
    if [[ "$file" =~ \.down\.sql$ ]]; then
        # Check if corresponding .up.sql exists
        up_file="${file%.down.sql}.up.sql"
        if [ ! -f "$up_file" ]; then
            log_error "Missing corresponding UP migration for: $file"
        fi

        # DOWN migration should contain DROP/ALTER for reversing
        if ! grep -q -i -E '(DROP|ALTER|DELETE)' "$file"; then
            log_warning "DOWN migration should contain DROP/ALTER statements: $file"
        fi
    fi

    # Check 5: Dangerous operations without safeguards
    if grep -q -i 'DROP TABLE' "$file" && ! grep -q -i 'IF EXISTS' "$file"; then
        log_warning "DROP TABLE without IF EXISTS detected in: $file"
    fi

    if grep -q -i 'DROP DATABASE' "$file"; then
        log_error "DROP DATABASE is not allowed in migrations: $file"
    fi

    # Check 6: Check for common SQL anti-patterns
    if grep -q -i 'SELECT \*' "$file"; then
        log_warning "SELECT * found in: $file (prefer explicit column names)"
    fi

    # Check 7: Check for missing semicolons
    if ! grep -q ';' "$file"; then
        log_warning "No semicolons found in: $file (SQL statements should end with ;)"
    fi

    # Check 8: Check for transaction statements in UP migrations
    if [[ "$file" =~ \.up\.sql$ ]]; then
        if ! grep -q -i 'BEGIN' "$file" && ! grep -q -i 'START TRANSACTION' "$file"; then
            log_warning "UP migration missing transaction BEGIN: $file"
        fi
        if ! grep -q -i 'COMMIT' "$file"; then
            log_warning "UP migration missing COMMIT: $file"
        fi
    fi

    # Check 9: Check for CASCADE without careful consideration
    if grep -q -i 'CASCADE' "$file"; then
        log_warning "CASCADE found in: $file (ensure this is intentional)"
    fi

    # Check 10: Check for missing comments/documentation
    if ! grep -q -E '^--' "$file"; then
        log_warning "No comments found in: $file (migrations should be documented)"
    fi

    log_success "Basic validation passed for: $file"
done

if [ $ERRORS -gt 0 ]; then
    echo ""
    echo -e "${RED}Migration validation failed with $ERRORS error(s)${NC}"
    exit 1
fi

echo ""
echo -e "${GREEN}All migration files validated successfully!${NC}"
exit 0
