#!/bin/bash

# Test Payment Flow
# This script tests wallet management and payment processing

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Base URLs
AUTH_URL="${AUTH_URL:-http://localhost:8081}"
PAYMENTS_URL="${PAYMENTS_URL:-http://localhost:8083}"

echo -e "${BLUE}========================================${NC}"
echo -e "${BLUE}Testing Payment Flow${NC}"
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

echo -e "${YELLOW}Step 1: Get Wallet Balance${NC}"
WALLET_RESPONSE=$(curl -s -X GET "${PAYMENTS_URL}/api/v1/wallet" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$WALLET_RESPONSE" | jq '.'

CURRENT_BALANCE=$(echo "$WALLET_RESPONSE" | jq -r '.data.balance // 0')
echo -e "${GREEN}✓ Current balance: \$${CURRENT_BALANCE}${NC}"
echo ""

echo -e "${YELLOW}Step 2: Top Up Wallet${NC}"
TOPUP_RESPONSE=$(curl -s -X POST "${PAYMENTS_URL}/api/v1/wallet/topup" \
  -H "Authorization: Bearer ${RIDER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 50.00,
    "stripe_payment_method": "pm_card_visa"
  }')

echo "$TOPUP_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Wallet topped up${NC}"
echo ""

echo -e "${YELLOW}Step 3: Get Updated Wallet Balance${NC}"
UPDATED_WALLET=$(curl -s -X GET "${PAYMENTS_URL}/api/v1/wallet" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$UPDATED_WALLET" | jq '.'

NEW_BALANCE=$(echo "$UPDATED_WALLET" | jq -r '.data.balance // 0')
echo -e "${GREEN}✓ New balance: \$${NEW_BALANCE}${NC}"
echo ""

echo -e "${YELLOW}Step 4: Get Wallet Transactions${NC}"
TRANSACTIONS_RESPONSE=$(curl -s -X GET "${PAYMENTS_URL}/api/v1/wallet/transactions?limit=10&offset=0" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$TRANSACTIONS_RESPONSE" | jq '.'
echo -e "${GREEN}✓ Transaction history retrieved${NC}"
echo ""

# Create a mock ride ID for payment testing
MOCK_RIDE_ID="00000000-0000-0000-0000-000000000001"

echo -e "${YELLOW}Step 5: Process Payment${NC}"
PAYMENT_RESPONSE=$(curl -s -X POST "${PAYMENTS_URL}/api/v1/payments/process" \
  -H "Authorization: Bearer ${RIDER_TOKEN}" \
  -H "Content-Type: application/json" \
  -d "{
    \"ride_id\": \"${MOCK_RIDE_ID}\",
    \"amount\": 25.50,
    \"payment_method\": \"wallet\"
  }")

echo "$PAYMENT_RESPONSE" | jq '.'

PAYMENT_ID=$(echo "$PAYMENT_RESPONSE" | jq -r '.data.id // empty')

if [ -n "$PAYMENT_ID" ]; then
  echo -e "${GREEN}✓ Payment processed: ${PAYMENT_ID}${NC}"
else
  echo -e "${YELLOW}⚠ Payment may have failed (expected if ride doesn't exist)${NC}"
fi
echo ""

if [ -n "$PAYMENT_ID" ]; then
  echo -e "${YELLOW}Step 6: Get Payment Details${NC}"
  PAYMENT_DETAILS=$(curl -s -X GET "${PAYMENTS_URL}/api/v1/payments/${PAYMENT_ID}" \
    -H "Authorization: Bearer ${RIDER_TOKEN}")

  echo "$PAYMENT_DETAILS" | jq '.'
  echo -e "${GREEN}✓ Payment details retrieved${NC}"
  echo ""

  echo -e "${YELLOW}Step 7: Request Refund${NC}"
  REFUND_RESPONSE=$(curl -s -X POST "${PAYMENTS_URL}/api/v1/payments/${PAYMENT_ID}/refund" \
    -H "Authorization: Bearer ${RIDER_TOKEN}" \
    -H "Content-Type: application/json" \
    -d '{
      "reason": "Ride cancelled by driver"
    }')

  echo "$REFUND_RESPONSE" | jq '.'
  echo -e "${GREEN}✓ Refund processed${NC}"
  echo ""
fi

echo -e "${YELLOW}Step 8: Get Final Wallet Balance${NC}"
FINAL_WALLET=$(curl -s -X GET "${PAYMENTS_URL}/api/v1/wallet" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$FINAL_WALLET" | jq '.'

FINAL_BALANCE=$(echo "$FINAL_WALLET" | jq -r '.data.balance // 0')
echo -e "${GREEN}✓ Final balance: \$${FINAL_BALANCE}${NC}"
echo ""

echo -e "${YELLOW}Step 9: Get Final Transaction History${NC}"
FINAL_TRANSACTIONS=$(curl -s -X GET "${PAYMENTS_URL}/api/v1/wallet/transactions?limit=20&offset=0" \
  -H "Authorization: Bearer ${RIDER_TOKEN}")

echo "$FINAL_TRANSACTIONS" | jq '.'
echo -e "${GREEN}✓ Final transaction history retrieved${NC}"
echo ""

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Payment Flow Test Complete!${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "Initial Balance: \$${CURRENT_BALANCE}"
echo "After Top-up: \$${NEW_BALANCE}"
echo "Final Balance: \$${FINAL_BALANCE}"
