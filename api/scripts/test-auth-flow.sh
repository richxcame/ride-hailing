#!/bin/bash

# Test Authentication Flow
# This script tests user registration, login, and profile management

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Base URL
BASE_URL="${BASE_URL:-http://localhost:8081}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Testing Authentication Flow${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Generate unique email for testing
TIMESTAMP=$(date +%s)
RIDER_EMAIL="rider_${TIMESTAMP}@example.com"
DRIVER_EMAIL="driver_${TIMESTAMP}@example.com"
PASSWORD="SecurePass123!"

echo -e "${YELLOW}Step 1: Register Rider${NC}"
REGISTER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${RIDER_EMAIL}\",
    \"password\": \"${PASSWORD}\",
    \"first_name\": \"Test\",
    \"last_name\": \"Rider\",
    \"phone_number\": \"+1234567890\",
    \"role\": \"rider\"
  }")

echo "$REGISTER_RESPONSE" | jq '.'

# Extract token
RIDER_TOKEN=$(echo "$REGISTER_RESPONSE" | jq -r '.data.token // empty')

if [ -z "$RIDER_TOKEN" ]; then
  echo -e "${RED}Failed to register rider${NC}"
  exit 1
fi

echo -e "${GREEN}✓ Rider registered successfully${NC}"
echo "Token: ${RIDER_TOKEN:0:20}..."
echo ""

echo -e "${YELLOW}Step 2: Register Driver${NC}"
DRIVER_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/auth/register" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${DRIVER_EMAIL}\",
    \"password\": \"${PASSWORD}\",
    \"first_name\": \"Test\",
    \"last_name\": \"Driver\",
    \"phone_number\": \"+1234567891\",
    \"role\": \"driver\"
  }")

echo "$DRIVER_RESPONSE" | jq '.'

DRIVER_TOKEN=$(echo "$DRIVER_RESPONSE" | jq -r '.data.token // empty')

if [ -z "$DRIVER_TOKEN" ]; then
  echo -e "${RED}Failed to register driver${NC}"
  exit 1
fi

echo -e "${GREEN}✓ Driver registered successfully${NC}"
echo "Token: ${DRIVER_TOKEN:0:20}..."
echo ""

echo -e "${YELLOW}Step 3: Login as Rider${NC}"
LOGIN_RESPONSE=$(curl -s -X POST "${BASE_URL}/api/v1/auth/login" \
  -H "Content-Type: application/json" \
  -d "{
    \"email\": \"${RIDER_EMAIL}\",
    \"password\": \"${PASSWORD}\"
  }")

echo "$LOGIN_RESPONSE" | jq '.'

NEW_TOKEN=$(echo "$LOGIN_RESPONSE" | jq -r '.data.token // empty')

if [ -z "$NEW_TOKEN" ]; then
  echo -e "${RED}Failed to login${NC}"
  exit 1
fi

echo -e "${GREEN}✓ Login successful${NC}"
echo ""

echo -e "${YELLOW}Step 4: Get Rider Profile${NC}"
PROFILE_RESPONSE=$(curl -s -X GET "${BASE_URL}/api/v1/auth/profile" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$PROFILE_RESPONSE" | jq '.'

echo -e "${GREEN}✓ Profile retrieved${NC}"
echo ""

echo -e "${YELLOW}Step 5: Update Rider Profile${NC}"
UPDATE_RESPONSE=$(curl -s -X PUT "${BASE_URL}/api/v1/auth/profile" \
  -H "Authorization: Bearer ${RIDER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "first_name": "Updated",
    "last_name": "Rider",
    "phone_number": "+1234567899"
  }')

echo "$UPDATE_RESPONSE" | jq '.'

echo -e "${GREEN}✓ Profile updated${NC}"
echo ""

echo -e "${YELLOW}Step 6: Get Updated Profile${NC}"
UPDATED_PROFILE=$(curl -s -X GET "${BASE_URL}/api/v1/auth/profile" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$UPDATED_PROFILE" | jq '.'

echo -e "${GREEN}✓ Updated profile retrieved${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Authentication Flow Test Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Saved credentials for further testing:"
echo "Rider Email: ${RIDER_EMAIL}"
echo "Rider Token: ${RIDER_TOKEN}"
echo "Driver Email: ${DRIVER_EMAIL}"
echo "Driver Token: ${DRIVER_TOKEN}"
echo ""
echo "Export tokens to use in other scripts:"
echo "export RIDER_TOKEN='${RIDER_TOKEN}'"
echo "export DRIVER_TOKEN='${DRIVER_TOKEN}'"
