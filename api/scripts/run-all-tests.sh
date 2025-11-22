#!/bin/bash

# Run All API Tests
# This script runs all API test flows in sequence

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

# Get script directory
SCRIPT_DIR="$( cd "$( dirname "${BASH_SOURCE[0]}" )" && pwd )"

echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}Running All API Test Flows${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""

# Check for jq
if ! command -v jq &> /dev/null; then
    echo -e "${RED}Error: jq is not installed${NC}"
    echo "Install it with: brew install jq (macOS) or apt-get install jq (Linux)"
    exit 1
fi

# Check if services are running
echo -e "${YELLOW}Checking if services are running...${NC}"
if ! curl -s http://localhost:8081/healthz > /dev/null 2>&1; then
    echo -e "${RED}Error: Auth service is not running${NC}"
    echo "Start services with: docker-compose up -d"
    exit 1
fi
echo -e "${GREEN}✓ Services are running${NC}"
echo ""

# Test 1: Authentication Flow
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test 1: Authentication Flow${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if "$SCRIPT_DIR/test-auth-flow.sh"; then
    echo -e "${GREEN}✓ Authentication flow passed${NC}"
    echo ""
else
    echo -e "${RED}✗ Authentication flow failed${NC}"
    exit 1
fi

# Extract tokens from the last run (you may need to modify this based on your output)
# For now, we'll run the tests independently

# Test 2: Ride Flow
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test 2: Complete Ride Flow${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if "$SCRIPT_DIR/test-ride-flow.sh"; then
    echo -e "${GREEN}✓ Ride flow passed${NC}"
    echo ""
else
    echo -e "${RED}✗ Ride flow failed${NC}"
    exit 1
fi

# Test 3: Payment Flow
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test 3: Payment Flow${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if "$SCRIPT_DIR/test-payment-flow.sh"; then
    echo -e "${GREEN}✓ Payment flow passed${NC}"
    echo ""
else
    echo -e "${RED}✗ Payment flow failed${NC}"
    exit 1
fi

# Test 4: Promo Flow
echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Test 4: Promo & Referral Flow${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

if "$SCRIPT_DIR/test-promo-flow.sh"; then
    echo -e "${GREEN}✓ Promo flow passed${NC}"
    echo ""
else
    echo -e "${RED}✗ Promo flow failed${NC}"
    exit 1
fi

# Summary
echo -e "${CYAN}========================================${NC}"
echo -e "${CYAN}All Tests Summary${NC}"
echo -e "${CYAN}========================================${NC}"
echo ""
echo -e "${GREEN}✓ Authentication Flow - PASSED${NC}"
echo -e "${GREEN}✓ Complete Ride Flow - PASSED${NC}"
echo -e "${GREEN}✓ Payment Flow - PASSED${NC}"
echo -e "${GREEN}✓ Promo & Referral Flow - PASSED${NC}"
echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}All API Tests Passed Successfully! 🎉${NC}"
echo -e "${GREEN}========================================${NC}"
