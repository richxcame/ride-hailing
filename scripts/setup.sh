#!/bin/bash

# Ride Hailing Platform - Setup Script
# This script helps you get started quickly

set -e

echo "🚀 Ride Hailing Platform Setup"
echo "================================"
echo ""

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Check if Docker is installed
if ! command -v docker &> /dev/null; then
    echo -e "${RED}❌ Docker is not installed. Please install Docker first.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Docker found${NC}"

# Check if Docker Compose is installed
if ! command -v docker-compose &> /dev/null; then
    echo -e "${RED}❌ Docker Compose is not installed. Please install Docker Compose first.${NC}"
    exit 1
fi

echo -e "${GREEN}✓ Docker Compose found${NC}"
echo ""

# Create .env file if it doesn't exist
if [ ! -f .env ]; then
    echo -e "${YELLOW}⚙️  Creating .env file...${NC}"
    cp .env.example .env
    echo -e "${GREEN}✓ .env file created${NC}"
else
    echo -e "${GREEN}✓ .env file already exists${NC}"
fi
echo ""

# Start Docker containers
echo -e "${YELLOW}🐳 Starting Docker containers...${NC}"
docker-compose up -d

echo ""
echo -e "${YELLOW}⏳ Waiting for services to be ready (30 seconds)...${NC}"
sleep 30

echo ""
echo -e "${YELLOW}🏥 Checking service health...${NC}"

# Check Auth Service
if curl -s http://localhost:8081/healthz | grep -q "healthy"; then
    echo -e "${GREEN}✓ Auth Service is healthy (Port 8081)${NC}"
else
    echo -e "${RED}❌ Auth Service is not responding${NC}"
fi

# Check Rides Service
if curl -s http://localhost:8082/healthz | grep -q "healthy"; then
    echo -e "${GREEN}✓ Rides Service is healthy (Port 8082)${NC}"
else
    echo -e "${RED}❌ Rides Service is not responding${NC}"
fi

# Check Geo Service
if curl -s http://localhost:8083/healthz | grep -q "healthy"; then
    echo -e "${GREEN}✓ Geo Service is healthy (Port 8083)${NC}"
else
    echo -e "${RED}❌ Geo Service is not responding${NC}"
fi

echo ""
echo -e "${YELLOW}📊 Running database migrations...${NC}"

# Check if migrate is installed
if command -v migrate &> /dev/null; then
    migrate -path db/migrations \
      -database "postgresql://postgres:postgres@localhost:5432/ridehailing?sslmode=disable" \
      up
    echo -e "${GREEN}✓ Migrations completed${NC}"
else
    echo -e "${YELLOW}⚠️  'migrate' tool not found. Install it with:${NC}"
    echo "   go install -tags 'postgres' github.com/golang-migrate/migrate/v4/cmd/migrate@latest"
    echo ""
    echo "   Or run migrations manually using the command above."
fi

echo ""
echo -e "${GREEN}✅ Setup Complete!${NC}"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📌 Service URLs:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "   Auth Service:       http://localhost:8081"
echo "   Rides Service:      http://localhost:8082"
echo "   Geo Service:        http://localhost:8083"
echo "   Prometheus:         http://localhost:9090"
echo "   Grafana:            http://localhost:3000 (admin/admin)"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "📚 Next Steps:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "   1. Read QUICKSTART.md for API testing examples"
echo "   2. Check docs/API.md for full API documentation"
echo "   3. View logs: docker-compose logs -f"
echo "   4. Stop services: docker-compose down"
echo ""
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "🧪 Quick Test:"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo "   Register a user:"
echo "   curl -X POST http://localhost:8081/api/v1/auth/register \\"
echo '     -H "Content-Type: application/json" \'
echo '     -d '"'"'{"email":"test@example.com","password":"test123","phone_number":"+1234567890","first_name":"John","last_name":"Doe","role":"rider"}'"'"
echo ""
