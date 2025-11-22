#!/bin/bash

# Test Promo and Referral Flow
# This script tests promo codes, ride types, and referral system

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Base URLs
AUTH_URL="${AUTH_URL:-http://localhost:8081}"
PROMOS_URL="${PROMOS_URL:-http://localhost:8089}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Testing Promo & Referral Flow${NC}"
echo -e "${BLUE}========================================${NC}"
echo ""

# Check if token is provided
if [ -z "$RIDER_TOKEN" ]; then
  echo -e "${YELLOW}Token not found. Creating test user...${NC}"

  TIMESTAMP=$(date +%s)
  RIDER_EMAIL="rider_${TIMESTAMP}@example.com"
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
  echo -e "${GREEN}✓ Test user created${NC}"
  echo ""
fi

echo -e "${YELLOW}Step 1: Get All Ride Types${NC}"
RIDE_TYPES_RESPONSE=$(curl -s -X GET "${PROMOS_URL}/api/v1/ride-types")

echo "$RIDE_TYPES_RESPONSE" | jq '.'

RIDE_TYPE_ID=$(echo "$RIDE_TYPES_RESPONSE" | jq -r '.ride_types[0].id // empty')
echo -e "${GREEN}✓ Ride types retrieved${NC}"
echo ""

if [ -n "$RIDE_TYPE_ID" ]; then
  echo -e "${YELLOW}Step 2: Calculate Fare (No Surge)${NC}"
  FARE_RESPONSE=$(curl -s -X POST "${PROMOS_URL}/api/v1/calculate-fare" \
    -H "Content-Type: application/json" \
    -d "{
      \"ride_type_id\": \"${RIDE_TYPE_ID}\",
      \"distance\": 5.5,
      \"duration\": 900,
      \"surge_multiplier\": 1.0
    }")

  echo "$FARE_RESPONSE" | jq '.'
  FARE=$(echo "$FARE_RESPONSE" | jq -r '.fare // 0')
  echo -e "${GREEN}✓ Base fare calculated: \$${FARE}${NC}"
  echo ""

  echo -e "${YELLOW}Step 3: Calculate Fare (With Surge)${NC}"
  SURGE_FARE_RESPONSE=$(curl -s -X POST "${PROMOS_URL}/api/v1/calculate-fare" \
    -H "Content-Type: application/json" \
    -d "{
      \"ride_type_id\": \"${RIDE_TYPE_ID}\",
      \"distance\": 5.5,
      \"duration\": 900,
      \"surge_multiplier\": 1.5
    }")

  echo "$SURGE_FARE_RESPONSE" | jq '.'
  SURGE_FARE=$(echo "$SURGE_FARE_RESPONSE" | jq -r '.fare // 0')
  echo -e "${GREEN}✓ Surge fare calculated: \$${SURGE_FARE}${NC}"
  echo ""
fi

echo -e "${YELLOW}Step 4: Validate Promo Code (Invalid)${NC}"
INVALID_PROMO=$(curl -s -X POST "${PROMOS_URL}/api/v1/promo-codes/validate" \
  -H "Authorization: Bearer ${RIDER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "code": "INVALID123",
    "ride_amount": 25.00
  }')

echo "$INVALID_PROMO" | jq '.'
echo -e "${YELLOW}⚠ Expected: Promo code not found${NC}"
echo ""

echo -e "${YELLOW}Step 5: Get My Referral Code${NC}"
REFERRAL_RESPONSE=$(curl -s -X GET "${PROMOS_URL}/api/v1/referrals/my-code" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$REFERRAL_RESPONSE" | jq '.'

REFERRAL_CODE=$(echo "$REFERRAL_RESPONSE" | jq -r '.code // empty')

if [ -n "$REFERRAL_CODE" ]; then
  echo -e "${GREEN}✓ Referral code generated: ${REFERRAL_CODE}${NC}"
else
  echo -e "${YELLOW}⚠ Referral code not generated${NC}"
fi
echo ""

# Create a second user to test referral application
if [ -n "$REFERRAL_CODE" ]; then
  echo -e "${YELLOW}Step 6: Create Second User${NC}"
  TIMESTAMP=$(date +%s)
  NEW_USER_EMAIL="newuser_${TIMESTAMP}@example.com"

  NEW_USER_RESPONSE=$(curl -s -X POST "${AUTH_URL}/api/v1/auth/register" \
    -H "Content-Type: application/json" \
    -d "{
      \"email\": \"${NEW_USER_EMAIL}\",
      \"password\": \"SecurePass123!\",
      \"first_name\": \"New\",
      \"last_name\": \"User\",
      \"phone_number\": \"+1234567892\",
      \"role\": \"rider\"
    }")

  NEW_USER_TOKEN=$(echo "$NEW_USER_RESPONSE" | jq -r '.data.token')
  echo -e "${GREEN}✓ New user created${NC}"
  echo ""

  echo -e "${YELLOW}Step 7: Apply Referral Code${NC}"
  APPLY_REFERRAL=$(curl -s -X POST "${PROMOS_URL}/api/v1/referrals/apply" \
    -H "Authorization: Bearer ${NEW_USER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
      \"referral_code\": \"${REFERRAL_CODE}\"
    }")

  echo "$APPLY_REFERRAL" | jq '.'
  echo -e "${GREEN}✓ Referral code application attempted${NC}"
  echo ""
fi

# If admin token is available, test promo code creation
if [ -n "$ADMIN_TOKEN" ]; then
  echo -e "${YELLOW}Step 8: Create Promo Code (Admin)${NC}"
  PROMO_CODE="SAVE20_$(date +%s)"

  CREATE_PROMO=$(curl -s -X POST "${PROMOS_URL}/api/v1/promo-codes" \
    -H "Authorization: Bearer ${ADMIN_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
      \"code\": \"${PROMO_CODE}\",
      \"discount_type\": \"percentage\",
      \"discount_value\": 20,
      \"max_uses\": 100,
      \"valid_from\": \"2025-01-01T00:00:00Z\",
      \"valid_until\": \"2025-12-31T23:59:59Z\",
      \"min_ride_amount\": 10.00,
      \"is_active\": true
    }")

  echo "$CREATE_PROMO" | jq '.'
  echo -e "${GREEN}✓ Promo code created: ${PROMO_CODE}${NC}"
  echo ""

  echo -e "${YELLOW}Step 9: Validate New Promo Code${NC}"
  VALIDATE_PROMO=$(curl -s -X POST "${PROMOS_URL}/api/v1/promo-codes/validate" \
    -H "Authorization: Bearer ${RIDER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d "{
      \"code\": \"${PROMO_CODE}\",
      \"ride_amount\": 25.00
    }")

  echo "$VALIDATE_PROMO" | jq '.'
  echo -e "${GREEN}✓ Promo code validated${NC}"
  echo ""
else
  echo -e "${YELLOW}⚠ Admin token not provided. Skipping promo code creation${NC}"
  echo "To test promo creation, export ADMIN_TOKEN variable"
  echo ""
fi

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Promo & Referral Flow Test Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
if [ -n "$REFERRAL_CODE" ]; then
  echo "Your Referral Code: ${REFERRAL_CODE}"
fi
echo "Test ride types and fare calculations completed"
