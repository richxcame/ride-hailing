#!/bin/bash

# Development Environment Health Check Script
# Checks if all required infrastructure is running

set -e

GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo "🔍 Checking Development Environment..."
echo ""

# Check if Docker is running
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Docker is not running${NC}"
    echo "   Start Docker Desktop and try again"
    exit 1
fi
echo -e "${GREEN}✓${NC} Docker is running"

# Check if PostgreSQL container is running
if docker ps | grep -q "ridehailing-postgres-dev"; then
    echo -e "${GREEN}✓${NC} PostgreSQL container is running"

    # Check if PostgreSQL is healthy
    if docker exec ridehailing-postgres-dev pg_isready -U postgres > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} PostgreSQL is healthy"
    else
        echo -e "${YELLOW}⚠${NC}  PostgreSQL container is running but not ready yet"
        echo "   Wait a few seconds and try again"
    fi
else
    echo -e "${YELLOW}⚠${NC}  PostgreSQL container is not running"
    echo "   Run: make dev-infra"
fi

# Check if Redis container is running
if docker ps | grep -q "ridehailing-redis-dev"; then
    echo -e "${GREEN}✓${NC} Redis container is running"

    # Check if Redis is healthy
    if docker exec ridehailing-redis-dev redis-cli ping > /dev/null 2>&1; then
        echo -e "${GREEN}✓${NC} Redis is healthy"
    else
        echo -e "${YELLOW}⚠${NC}  Redis container is running but not ready yet"
        echo "   Wait a few seconds and try again"
    fi
else
    echo -e "${YELLOW}⚠${NC}  Redis container is not running"
    echo "   Run: make dev-infra"
fi

# Check if Postgres is accessible on localhost
if nc -z localhost 5432 2>/dev/null; then
    echo -e "${GREEN}✓${NC} PostgreSQL is accessible on localhost:5432"
else
    echo -e "${RED}❌${NC} Cannot connect to PostgreSQL on localhost:5432"
    echo "   Check if port 5432 is exposed in docker-compose.dev.yml"
fi

# Check if Redis is accessible on localhost
if nc -z localhost 6379 2>/dev/null; then
    echo -e "${GREEN}✓${NC} Redis is accessible on localhost:6379"
else
    echo -e "${RED}❌${NC} Cannot connect to Redis on localhost:6379"
    echo "   Check if port 6379 is exposed in docker-compose.dev.yml"
fi

# Check if Go is installed
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo -e "${GREEN}✓${NC} Go is installed ($GO_VERSION)"
else
    echo -e "${RED}❌${NC} Go is not installed"
    echo "   Install Go: https://golang.org/doc/install"
fi

# Check if migrate tool is installed
if command -v migrate &> /dev/null; then
    echo -e "${GREEN}✓${NC} migrate tool is installed"
else
    echo -e "${YELLOW}⚠${NC}  migrate tool is not installed"
    echo "   Install: go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
fi

# Check if .env file exists
if [ -f .env ]; then
    echo -e "${GREEN}✓${NC} .env file exists"
else
    echo -e "${YELLOW}⚠${NC}  .env file not found"
    echo "   Create: cp .env.example .env"
fi

echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Check optional observability stack
if docker ps | grep -q "ridehailing-grafana-dev"; then
    echo ""
    echo "📊 Observability Stack:"
    echo -e "${GREEN}✓${NC} Grafana: http://localhost:3000 (admin/admin)"
    echo -e "${GREEN}✓${NC} Prometheus: http://localhost:9090"
    echo -e "${GREEN}✓${NC} Tempo: http://localhost:3200"
fi

# Summary
echo ""
POSTGRES_RUNNING=$(docker ps | grep -q "ridehailing-postgres-dev" && echo "yes" || echo "no")
REDIS_RUNNING=$(docker ps | grep -q "ridehailing-redis-dev" && echo "yes" || echo "no")

if [ "$POSTGRES_RUNNING" = "yes" ] && [ "$REDIS_RUNNING" = "yes" ]; then
    echo -e "${GREEN}🎉 Development environment is ready!${NC}"
    echo ""
    echo "Next steps:"
    echo "  1. Run migrations: make migrate-up"
    echo "  2. Start a service: make run-auth"
    echo "  3. Visit: http://localhost:8081/healthz"
else
    echo -e "${YELLOW}⚠ Development environment is not fully ready${NC}"
    echo ""
    echo "To start infrastructure:"
    echo "  make dev-infra          # Start Postgres + Redis"
    echo "  make dev-infra-full     # Start + Observability"
fi

echo ""
