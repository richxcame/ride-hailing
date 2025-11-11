#!/bin/bash

# Setup script for OpenTelemetry tracing infrastructure
# This script prepares the environment for Tempo and OTel Collector

set -e

echo "🔧 Setting up OpenTelemetry tracing infrastructure..."

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Create necessary directories for Tempo with proper permissions
echo -e "${YELLOW}📁 Creating Tempo data directories...${NC}"
mkdir -p ./data/tempo/wal
mkdir -p ./data/tempo/blocks
mkdir -p ./data/tempo/generator/wal

# Set permissions (for development - more permissive)
chmod -R 777 ./data/tempo

echo -e "${GREEN}✅ Tempo directories created with permissions${NC}"

# Verify configuration files exist
echo -e "${YELLOW}🔍 Verifying configuration files...${NC}"

if [ ! -f ./deploy/tempo.yml ]; then
    echo -e "${RED}❌ Error: deploy/tempo.yml not found${NC}"
    exit 1
fi

if [ ! -f ./deploy/otel-collector.yml ]; then
    echo -e "${RED}❌ Error: deploy/otel-collector.yml not found${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Configuration files verified${NC}"

# Check if Docker is running
echo -e "${YELLOW}🐳 Checking Docker status...${NC}"
if ! docker info > /dev/null 2>&1; then
    echo -e "${RED}❌ Error: Docker is not running${NC}"
    exit 1
fi

echo -e "${GREEN}✅ Docker is running${NC}"

# Stop existing containers if any
echo -e "${YELLOW}🛑 Stopping existing tracing containers...${NC}"
docker-compose stop tempo otel-collector 2>/dev/null || true
docker-compose rm -f tempo otel-collector 2>/dev/null || true

echo -e "${GREEN}✅ Cleanup complete${NC}"

# Pull latest images
echo -e "${YELLOW}📥 Pulling latest images...${NC}"
docker-compose pull tempo otel-collector

echo -e "${GREEN}✅ Images pulled${NC}"

echo ""
echo -e "${GREEN}🎉 Setup complete!${NC}"
echo ""
echo "Next steps:"
echo "1. Start the services: docker-compose up -d"
echo "2. Check logs: docker-compose logs -f tempo otel-collector"
echo "3. Access Grafana: http://localhost:3000"
echo ""
echo "For more information, see docs/tracing-setup.md"
