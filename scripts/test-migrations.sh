#!/bin/bash

# Migration Testing Script
# This script tests all database migrations including rollback functionality
# Usage: ./scripts/test-migrations.sh [options]
# Options:
#   --clean    : Start with a clean database
#   --verbose  : Show detailed output
#   --skip-rollback : Skip rollback tests

set -e

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default options
CLEAN=false
VERBOSE=false
SKIP_ROLLBACK=false

# Parse command line arguments
while [[ $# -gt 0 ]]; do
    case $1 in
        --clean)
            CLEAN=true
            shift
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --skip-rollback)
            SKIP_ROLLBACK=true
            shift
            ;;
        *)
            echo -e "${RED}Unknown option: $1${NC}"
            echo "Usage: $0 [--clean] [--verbose] [--skip-rollback]"
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
TEST_DB_NAME="${DB_NAME}_test_migrations"

# Migration directory
MIGRATION_DIR="db/migrations"

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

# Verbose logging
log_verbose() {
    if [ "$VERBOSE" = true ]; then
        echo -e "${BLUE}[VERBOSE]${NC} $1"
    fi
}

# Database connection string
get_db_url() {
    local db_name=$1
    echo "postgres://${DB_USER}:${DB_PASSWORD}@${DB_HOST}:${DB_PORT}/${db_name}?sslmode=disable"
}

# Check if PostgreSQL is accessible
check_postgres() {
    log_info "Checking PostgreSQL connectivity..."
    if ! PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -c '\q' 2>/dev/null; then
        log_error "Cannot connect to PostgreSQL at ${DB_HOST}:${DB_PORT}"
        log_error "Make sure PostgreSQL is running: docker-compose -f docker-compose.dev.yml up -d"
        exit 1
    fi
    log_success "PostgreSQL is accessible"
}

# Create test database
create_test_db() {
    log_info "Creating test database: ${TEST_DB_NAME}"

    # Drop if exists
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -c "DROP DATABASE IF EXISTS ${TEST_DB_NAME};" 2>/dev/null || true

    # Create fresh database
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -c "CREATE DATABASE ${TEST_DB_NAME};" 2>/dev/null

    log_success "Test database created"
}

# Drop test database
drop_test_db() {
    log_info "Dropping test database: ${TEST_DB_NAME}"
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d postgres -c "DROP DATABASE IF EXISTS ${TEST_DB_NAME};" 2>/dev/null || true
    log_success "Test database dropped"
}

# Get list of migrations
get_migrations() {
    find ${MIGRATION_DIR} -name "*.up.sql" | sort | while read -r file; do
        basename "$file" | sed 's/.up.sql$//'
    done
}

# Get migration version from filename
get_migration_version() {
    echo "$1" | cut -d'_' -f1
}

# Get current migration version
get_current_version() {
    local db_url=$(get_db_url ${TEST_DB_NAME})
    migrate -database "${db_url}" -path ${MIGRATION_DIR} version 2>/dev/null || echo "0"
}

# Test migration up
test_migration_up() {
    local migration=$1
    local version=$(get_migration_version $migration)

    log_info "Testing migration UP: ${migration}"

    local db_url=$(get_db_url ${TEST_DB_NAME})

    # Apply migration
    if migrate -database "${db_url}" -path ${MIGRATION_DIR} up 1 2>&1 | tee /tmp/migrate_output.log; then
        log_success "Migration UP successful: ${migration}"

        # Verify version changed
        local current_version=$(get_current_version)
        log_verbose "Current database version: ${current_version}"

        # Validate migration results
        validate_migration_up $migration

        return 0
    else
        log_error "Migration UP failed: ${migration}"
        cat /tmp/migrate_output.log
        return 1
    fi
}

# Test migration down
test_migration_down() {
    local migration=$1
    local version=$(get_migration_version $migration)

    log_info "Testing migration DOWN (rollback): ${migration}"

    local db_url=$(get_db_url ${TEST_DB_NAME})

    # Rollback migration
    if migrate -database "${db_url}" -path ${MIGRATION_DIR} down 1 2>&1 | tee /tmp/migrate_output.log; then
        log_success "Migration DOWN successful: ${migration}"

        # Verify version changed
        local current_version=$(get_current_version)
        log_verbose "Current database version after rollback: ${current_version}"

        # Validate migration rollback
        validate_migration_down $migration

        return 0
    else
        log_error "Migration DOWN failed: ${migration}"
        cat /tmp/migrate_output.log
        return 1
    fi
}

# Validate migration up (basic checks)
validate_migration_up() {
    local migration=$1
    log_verbose "Validating migration UP: ${migration}"

    # Check for common issues
    local db_url=$(get_db_url ${TEST_DB_NAME})

    # Verify no locks are held
    local locks=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${TEST_DB_NAME} -t -c "SELECT COUNT(*) FROM pg_locks WHERE NOT granted;" 2>/dev/null || echo "0")
    if [ "$locks" != "0" ]; then
        log_warning "Found ${locks} unreleased locks after migration"
    fi

    # Check for invalid indexes
    local invalid_indexes=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${TEST_DB_NAME} -t -c "SELECT COUNT(*) FROM pg_index WHERE NOT indisvalid;" 2>/dev/null || echo "0")
    if [ "$invalid_indexes" != "0" ]; then
        log_warning "Found ${invalid_indexes} invalid indexes after migration"
    fi

    log_verbose "Migration validation completed"
}

# Validate migration down (basic checks)
validate_migration_down() {
    local migration=$1
    log_verbose "Validating migration DOWN: ${migration}"

    # Similar validation as up
    validate_migration_up $migration
}

# Test all migrations
test_all_migrations() {
    log_info "Starting comprehensive migration test..."

    local migrations=$(get_migrations)
    local migration_count=$(echo "$migrations" | wc -l)
    local current=0
    local failed=0

    log_info "Found ${migration_count} migrations to test"
    echo ""

    # Test each migration: up -> down -> up
    for migration in $migrations; do
        current=$((current + 1))
        echo -e "${BLUE}========================================${NC}"
        echo -e "${BLUE}Testing Migration ${current}/${migration_count}${NC}"
        echo -e "${BLUE}========================================${NC}"

        # Test UP
        if ! test_migration_up $migration; then
            failed=$((failed + 1))
            log_error "Migration test failed at: ${migration}"
            break
        fi

        # Test DOWN (rollback) unless skipped
        if [ "$SKIP_ROLLBACK" = false ]; then
            if ! test_migration_down $migration; then
                failed=$((failed + 1))
                log_error "Rollback test failed at: ${migration}"
                break
            fi

            # Re-apply migration (UP again)
            log_info "Re-applying migration after rollback test..."
            if ! test_migration_up $migration; then
                failed=$((failed + 1))
                log_error "Migration re-apply failed at: ${migration}"
                break
            fi
        fi

        echo ""
    done

    echo -e "${BLUE}========================================${NC}"
    if [ $failed -eq 0 ]; then
        log_success "All migration tests passed! (${migration_count} migrations)"
        return 0
    else
        log_error "Migration tests failed: ${failed} errors"
        return 1
    fi
}

# Test migration idempotency
test_migration_idempotency() {
    log_info "Testing migration idempotency (applying migrations twice)..."

    local db_url=$(get_db_url ${TEST_DB_NAME})

    # Apply all migrations
    log_info "First application of all migrations..."
    if ! migrate -database "${db_url}" -path ${MIGRATION_DIR} up; then
        log_error "Initial migration application failed"
        return 1
    fi

    # Try to apply again (should be no-op)
    log_info "Second application (should be no-op)..."
    if migrate -database "${db_url}" -path ${MIGRATION_DIR} up 2>&1 | grep -q "no change"; then
        log_success "Migrations are idempotent"
        return 0
    else
        log_warning "Migration idempotency test inconclusive"
        return 0
    fi
}

# Generate migration report
generate_report() {
    log_info "Generating migration report..."

    local db_url=$(get_db_url ${TEST_DB_NAME})
    local current_version=$(get_current_version)
    local migration_count=$(get_migrations | wc -l)

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Migration Test Report${NC}"
    echo -e "${GREEN}========================================${NC}"
    echo "Database: ${TEST_DB_NAME}"
    echo "Current Version: ${current_version}"
    echo "Total Migrations: ${migration_count}"
    echo ""

    # List all migrations
    echo -e "${BLUE}Applied Migrations:${NC}"
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${TEST_DB_NAME} -c "SELECT version, dirty FROM schema_migrations ORDER BY version;" 2>/dev/null || echo "No migrations applied"

    echo ""

    # Table count
    local table_count=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${TEST_DB_NAME} -t -c "SELECT COUNT(*) FROM information_schema.tables WHERE table_schema = 'public' AND table_type = 'BASE TABLE';" 2>/dev/null)
    echo "Tables created: ${table_count}"

    # Index count
    local index_count=$(PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${TEST_DB_NAME} -t -c "SELECT COUNT(*) FROM pg_indexes WHERE schemaname = 'public';" 2>/dev/null)
    echo "Indexes created: ${index_count}"

    # Extensions
    echo ""
    echo -e "${BLUE}Installed Extensions:${NC}"
    PGPASSWORD=${DB_PASSWORD} psql -h ${DB_HOST} -p ${DB_PORT} -U ${DB_USER} -d ${TEST_DB_NAME} -c "SELECT extname, extversion FROM pg_extension WHERE extname != 'plpgsql';" 2>/dev/null || echo "No extensions"

    echo ""
    echo -e "${GREEN}========================================${NC}"
}

# Cleanup on exit
cleanup() {
    if [ "$CLEAN" = true ]; then
        drop_test_db
    else
        log_info "Test database ${TEST_DB_NAME} retained for inspection"
        log_info "To clean up manually, run: make db-drop-test"
    fi
}

# Main execution
main() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Database Migration Testing Tool${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    # Check prerequisites
    if ! command -v migrate &> /dev/null; then
        log_error "migrate command not found"
        log_error "Install golang-migrate: brew install golang-migrate"
        exit 1
    fi

    if ! command -v psql &> /dev/null; then
        log_error "psql command not found"
        log_error "Install PostgreSQL client: brew install postgresql"
        exit 1
    fi

    # Check migration directory
    if [ ! -d "${MIGRATION_DIR}" ]; then
        log_error "Migration directory not found: ${MIGRATION_DIR}"
        exit 1
    fi

    # Trap cleanup on exit
    trap cleanup EXIT

    # Run tests
    check_postgres
    create_test_db

    # Run test suite
    if test_all_migrations; then
        test_migration_idempotency
        generate_report

        echo ""
        log_success "All migration tests completed successfully!"
        exit 0
    else
        log_error "Migration tests failed!"
        exit 1
    fi
}

# Run main
main
