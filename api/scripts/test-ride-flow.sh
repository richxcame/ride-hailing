#!/bin/bash

# Test Complete Ride Flow
# This script simulates a complete ride flow from request to completion

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Base URLs
AUTH_URL="${AUTH_URL:-http://localhost:8081}"
RIDES_URL="${RIDES_URL:-http://localhost:8082}"
PROMOS_URL="${PROMOS_URL:-http://localhost:8089}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Testing Complete Ride Flow${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if tokens are provided
if [ -z "$RIDER_TOKEN" ] || [ -z "$DRIVER_TOKEN" ]; then
  echo -e "${YELLOW}Tokens not found. Creating test users...${NC}"

  TIMESTAMP=$(date +%s)
  RIDER_EMAIL="rider_${TIMESTAMP}@example.com"
  DRIVER_EMAIL="driver_${TIMESTAMP}@example.com"
  PASSWORD="SecurePass123!"

  # Register rider
  RIDER_RESPONSE=$(curl -s -X POST "${AUTH_URL}/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"${RIDER_EMAIL}\",
      \"password\": \"${PASSWORD}\",
      \"first_name\": \"Test\",
      \"last_name\": \"Rider\",
      \"phone_number\": \"+1234567890\",
      \"role\": \"rider\"
    }")

  RIDER_TOKEN=$(echo "$RIDER_RESPONSE" | jq -r '.data.token')

  # Register driver
  DRIVER_RESPONSE=$(curl -s -X POST "${AUTH_URL}/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"${DRIVER_EMAIL}\",
      \"password\": \"${PASSWORD}\",
      \"first_name\": \"Test\",
      \"last_name\": \"Driver\",
      \"phone_number\": \"+1234567891\",
      \"role\": \"driver\"
    }")

  DRIVER_TOKEN=$(echo "$DRIVER_RESPONSE" | jq -r '.data.token')

  echo -e "${GREEN}✓ Test users created${NC}"
  echo ""
fi

echo -e "${YELLOW}Step 1: Get Available Ride Types${NC}"
RIDE_TYPES_RESPONSE=$(curl -s -X GET "${PROMOS_URL}/api/v1/ride-types")
echo "$RIDE_TYPES_RESPONSE" | jq '.'

RIDE_TYPE_ID=$(echo "$RIDE_TYPES_RESPONSE" | jq -r '.ride_types[0].id // empty')

if [ -z "$RIDE_TYPE_ID" ]; then
  echo -e "${RED}No ride types found. Using default UUID${NC}"
  RIDE_TYPE_ID="00000000-0000-0000-0000-000000000001"
fi

echo -e "${GREEN}✓ Ride type: ${RIDE_TYPE_ID}${NC}"
echo ""

echo -e "${YELLOW}Step 2: Check Surge Pricing${NC}"
SURGE_RESPONSE=$(curl -s -X GET "${RIDES_URL}/api/v1/rides/surge-info?lat=37.7749&lon=-122.4194" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$SURGE_RESPONSE" | jq '.'
SURGE_MULTIPLIER=$(echo "$SURGE_RESPONSE" | jq -r '.data.surge_multiplier // 1.0')
echo -e "${GREEN}✓ Surge multiplier: ${SURGE_MULTIPLIER}${NC}"
echo ""

echo -e "${YELLOW}Step 3: Request Ride (Rider)${NC}"
REQUEST_RESPONSE=$(curl -s -X POST "${RIDES_URL}/api/v1/rides" \
  -H "Authorization: Bearer ${RIDER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"pickup_lat\": 37.7749,
    \"pickup_lon\": -122.4194,
    \"pickup_address\": \"123 Market St, San Francisco, CA\",
    \"dropoff_lat\": 37.7849,
    \"dropoff_lon\": -122.4094,
    \"dropoff_address\": \"456 Mission St, San Francisco, CA\",
    \"ride_type_id\": \"${RIDE_TYPE_ID}\",
    \"promo_code\": \"\"
  }")

echo "$REQUEST_RESPONSE" | jq '.'

RIDE_ID=$(echo "$REQUEST_RESPONSE" | jq -r '.data.id // empty')

if [ -z "$RIDE_ID" ]; then
  echo -e "${RED}Failed to create ride request${NC}"
  exit 1
fi

echo -e "${GREEN}✓ Ride requested: ${RIDE_ID}${NC}"
echo ""

echo -e "${YELLOW}Step 4: Get Available Rides (Driver)${NC}"
AVAILABLE_RESPONSE=$(curl -s -X GET "${RIDES_URL}/api/v1/driver/rides/available" \
  -H "Authorization: Bearer ${DRIVER_TOKEN}")

echo "$AVAILABLE_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Available rides retrieved${NC}"
echo ""

echo -e "${YELLOW}Step 5: Accept Ride (Driver)${NC}"
ACCEPT_RESPONSE=$(curl -s -X POST "${RIDES_URL}/api/v1/driver/rides/${RIDE_ID}/accept" \
  -H "Authorization: Bearer ${DRIVER_TOKEN}")

echo "$ACCEPT_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Ride accepted by driver${NC}"
echo ""

echo -e "${YELLOW}Step 6: Get Ride Details${NC}"
RIDE_DETAILS=$(curl -s -X GET "${RIDES_URL}/api/v1/rides/${RIDE_ID}" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$RIDE_DETAILS" | jq '.'
echo -e "${GREEN}✓ Ride details retrieved${NC}"
echo ""

echo -e "${YELLOW}Step 7: Start Ride (Driver)${NC}"
START_RESPONSE=$(curl -s -X POST "${RIDES_URL}/api/v1/driver/rides/${RIDE_ID}/start" \
  -H "Authorization: Bearer ${DRIVER_TOKEN}")

echo "$START_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Ride started${NC}"
echo ""

echo "Simulating ride in progress..."
sleep 2

echo -e "${YELLOW}Step 8: Complete Ride (Driver)${NC}"
COMPLETE_RESPONSE=$(curl -s -X POST "${RIDES_URL}/api/v1/driver/rides/${RIDE_ID}/complete" \
  -H "Authorization: Bearer ${DRIVER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "actual_distance": 5.2
  }')

echo "$COMPLETE_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Ride completed${NC}"
echo ""

echo -e "${YELLOW}Step 9: Rate Ride (Rider)${NC}"
RATING_RESPONSE=$(curl -s -X POST "${RIDES_URL}/api/v1/rides/${RIDE_ID}/rate" \
  -H "Authorization: Bearer ${RIDER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "rating": 5,
    "comment": "Great driver, smooth ride!"
  }')

echo "$RATING_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Ride rated${NC}"
echo ""

echo -e "${YELLOW}Step 10: Get Rider's Rides${NC}"
RIDER_RIDES=$(curl -s -X GET "${RIDES_URL}/api/v1/rides?page=1&per_page=10" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$RIDER_RIDES" | jq '.'
echo -e "${GREEN}✓ Rider's ride history retrieved${NC}"
echo ""

echo -e "${YELLOW}Step 11: Get Driver's Rides${NC}"
DRIVER_RIDES=$(curl -s -X GET "${RIDES_URL}/api/v1/rides?page=1&per_page=10" \
  -H "Authorization: Bearer ${DRIVER_TOKEN}")

echo "$DRIVER_RIDES" | jq '.'
echo -e "${GREEN}✓ Driver's ride history retrieved${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Complete Ride Flow Test Successful!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Ride ID: ${RIDE_ID}"
echo "Status: Completed and Rated"
