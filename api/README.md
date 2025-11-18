# API Collections Documentation

This directory contains API testing tools for the Ride Hailing platform, including Postman collections and shell scripts for common workflows.

## Table of Contents

- [Postman Collection](#postman-collection)
- [Shell Scripts](#shell-scripts)
- [Quick Start](#quick-start)
- [Authentication](#authentication)
- [Common Workflows](#common-workflows)
- [Troubleshooting](#troubleshooting)

## Directory Structure

```
api/
├── postman/
│   ├── ride-hailing.postman_collection.json  # Complete API collection
│   └── environment.json                       # Environment variables
├── scripts/
│   ├── test-auth-flow.sh                     # Authentication testing
│   ├── test-ride-flow.sh                     # Complete ride flow
│   ├── test-payment-flow.sh                  # Payment and wallet testing
│   └── test-promo-flow.sh                    # Promos and referrals
└── README.md                                  # This file
```

## Postman Collection

### Importing the Collection

1. Open Postman
2. Click **Import** in the top left
3. Select `api/postman/ride-hailing.postman_collection.json`
4. Import the environment file: `api/postman/environment.json`
5. Select "Ride Hailing - Local" environment from the dropdown

### Collection Features

- **90+ API endpoints** across 13 microservices
- **Automatic token management** - Tokens are saved automatically after login/register
- **Pre-configured variables** for all service URLs
- **Test assertions** for critical endpoints
- **Environment support** for local, staging, and production

### Services Included

| Service | Base URL | Port | Description |
|---------|----------|------|-------------|
| Auth | http://localhost:8081 | 8081 | Authentication & user management |
| Rides | http://localhost:8082 | 8082 | Ride requests & lifecycle |
| Payments | http://localhost:8083 | 8083 | Wallet & payment processing |
| Geo | http://localhost:8084 | 8084 | Geolocation & routing |
| Notifications | http://localhost:8085 | 8085 | Push, SMS, email notifications |
| Real-time | http://localhost:8086 | 8086 | WebSocket connections |
| Mobile | http://localhost:8087 | 8087 | Mobile gateway |
| Admin | http://localhost:8088 | 8088 | Admin operations |
| Promos | http://localhost:8089 | 8089 | Promo codes & referrals |
| Scheduler | http://localhost:8090 | 8090 | Scheduled rides |
| Analytics | http://localhost:8091 | 8091 | Metrics & reporting |
| Fraud | http://localhost:8092 | 8092 | Fraud detection |
| ML ETA | http://localhost:8093 | 8093 | ML-based ETA prediction |

### Key Endpoints

#### Authentication
- `POST /api/v1/auth/register` - Register new user
- `POST /api/v1/auth/login` - Login
- `GET /api/v1/auth/profile` - Get profile
- `PUT /api/v1/auth/profile` - Update profile

#### Rides
- `POST /api/v1/rides` - Request ride
- `GET /api/v1/rides/:id` - Get ride details
- `POST /api/v1/driver/rides/:id/accept` - Accept ride (driver)
- `POST /api/v1/driver/rides/:id/start` - Start ride (driver)
- `POST /api/v1/driver/rides/:id/complete` - Complete ride (driver)
- `POST /api/v1/rides/:id/rate` - Rate ride (rider)

#### Payments
- `GET /api/v1/wallet` - Get wallet balance
- `POST /api/v1/wallet/topup` - Add funds
- `POST /api/v1/payments/process` - Process payment
- `POST /api/v1/payments/:id/refund` - Request refund

#### Promos
- `GET /api/v1/ride-types` - Get available ride types
- `POST /api/v1/calculate-fare` - Calculate fare
- `POST /api/v1/promo-codes/validate` - Validate promo code
- `GET /api/v1/referrals/my-code` - Get referral code

## Shell Scripts

### Prerequisites

The shell scripts require:
- `curl` - For making HTTP requests
- `jq` - For JSON parsing (install: `brew install jq` on macOS)
- Running microservices (see [Quick Start](#quick-start))

### Available Scripts

#### 1. test-auth-flow.sh

Tests the complete authentication flow including registration, login, and profile management.

**Usage:**
```bash
cd api/scripts
chmod +x test-auth-flow.sh
./test-auth-flow.sh
```

**What it tests:**
- Register rider
- Register driver
- Login
- Get profile
- Update profile
- Retrieve updated profile

**Output:**
- Saves `RIDER_TOKEN` and `DRIVER_TOKEN` for use in other scripts
- Displays all API responses in formatted JSON

#### 2. test-ride-flow.sh

Simulates a complete ride flow from request to completion and rating.

**Usage:**
```bash
# With existing tokens
export RIDER_TOKEN="your-rider-token"
export DRIVER_TOKEN="your-driver-token"
./test-ride-flow.sh

# Without tokens (creates new users)
./test-ride-flow.sh
```

**What it tests:**
- Get ride types
- Check surge pricing
- Request ride (rider)
- Get available rides (driver)
- Accept ride (driver)
- Get ride details
- Start ride (driver)
- Complete ride (driver)
- Rate ride (rider)
- View ride history

**Output:**
- Creates a complete ride and returns `RIDE_ID`
- Shows full ride lifecycle

#### 3. test-payment-flow.sh

Tests wallet management and payment processing.

**Usage:**
```bash
export RIDER_TOKEN="your-rider-token"
./test-payment-flow.sh
```

**What it tests:**
- Get wallet balance
- Top up wallet
- View updated balance
- Get transaction history
- Process payment
- Request refund
- Final balance check

**Output:**
- Shows balance changes throughout the flow
- Lists all transactions

#### 4. test-promo-flow.sh

Tests promo codes, ride types, and referral system.

**Usage:**
```bash
export RIDER_TOKEN="your-rider-token"
export ADMIN_TOKEN="your-admin-token"  # Optional for promo creation
./test-promo-flow.sh
```

**What it tests:**
- Get all ride types
- Calculate fare (with/without surge)
- Validate promo codes
- Generate referral code
- Apply referral code (creates second user)
- Create promo code (if admin token provided)

**Output:**
- Displays ride types and fare calculations
- Shows referral code generation
- Tests promo code validation

### Environment Variables

All scripts support the following environment variables:

```bash
# Service URLs
export AUTH_URL="http://localhost:8081"
export RIDES_URL="http://localhost:8082"
export PAYMENTS_URL="http://localhost:8083"
export PROMOS_URL="http://localhost:8089"

# Authentication tokens
export RIDER_TOKEN="your-rider-jwt-token"
export DRIVER_TOKEN="your-driver-jwt-token"
export ADMIN_TOKEN="your-admin-jwt-token"
```

## Quick Start

### 1. Start All Services

```bash
# Using docker-compose
docker-compose up -d

# Or using Make
make docker-up
```

### 2. Run Database Migrations

```bash
make migrate-up
```

### 3. Seed Test Data (Optional)

```bash
make db-seed
```

### 4. Test with Postman

1. Import collection: `api/postman/ride-hailing.postman_collection.json`
2. Import environment: `api/postman/environment.json`
3. Run "Register Rider" request
4. Token will be automatically saved
5. Test other endpoints

### 5. Test with Shell Scripts

```bash
cd api/scripts
chmod +x *.sh

# Run authentication flow
./test-auth-flow.sh

# Export tokens from output
export RIDER_TOKEN="eyJhbGc..."
export DRIVER_TOKEN="eyJhbGc..."

# Run complete ride flow
./test-ride-flow.sh

# Test payments
./test-payment-flow.sh

# Test promos
./test-promo-flow.sh
```

## Authentication

### Token Management

The system uses JWT tokens for authentication.

**Token Lifetime:** 15 hours

**Getting a Token:**
```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@example.com",
    "password": "SecurePass123!"
  }'
```

**Using a Token:**
```bash
curl -X GET http://localhost:8082/api/v1/rides \
  -H "Authorization: Bearer YOUR_TOKEN_HERE"
```

### Roles

The system supports three user roles:

- **rider** - Can request rides, make payments, rate drivers
- **driver** - Can accept/complete rides, receive payments
- **admin** - Can manage users, view analytics, create promos

### RBAC (Role-Based Access Control)

Certain endpoints require specific roles:

| Endpoint | Allowed Roles |
|----------|---------------|
| `POST /api/v1/rides` | rider, driver |
| `POST /api/v1/driver/rides/:id/accept` | driver |
| `POST /api/v1/admin/*` | admin |
| `POST /api/v1/promo-codes` | admin |

## Common Workflows

### Complete Ride Flow (Step-by-Step)

```bash
# 1. Register and login as rider
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"rider@test.com","password":"Pass123!","first_name":"John","last_name":"Doe","phone_number":"+1234567890","role":"rider"}'

# Save the token from response
RIDER_TOKEN="eyJhbGc..."

# 2. Request a ride
curl -X POST http://localhost:8082/api/v1/rides \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "pickup_lat": 37.7749,
    "pickup_lon": -122.4194,
    "pickup_address": "123 Market St, SF",
    "dropoff_lat": 37.7849,
    "dropoff_lon": -122.4094,
    "dropoff_address": "456 Mission St, SF",
    "ride_type_id": "RIDE_TYPE_UUID"
  }'

# Save ride_id from response
RIDE_ID="abc-123-def"

# 3. Register as driver and accept ride
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{"email":"driver@test.com","password":"Pass123!","first_name":"Jane","last_name":"Smith","phone_number":"+1234567891","role":"driver"}'

DRIVER_TOKEN="eyJhbGc..."

curl -X POST http://localhost:8082/api/v1/driver/rides/$RIDE_ID/accept \
  -H "Authorization: Bearer $DRIVER_TOKEN"

# 4. Start the ride
curl -X POST http://localhost:8082/api/v1/driver/rides/$RIDE_ID/start \
  -H "Authorization: Bearer $DRIVER_TOKEN"

# 5. Complete the ride
curl -X POST http://localhost:8082/api/v1/driver/rides/$RIDE_ID/complete \
  -H "Authorization: Bearer $DRIVER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"actual_distance": 5.2}'

# 6. Rate the ride (as rider)
curl -X POST http://localhost:8082/api/v1/rides/$RIDE_ID/rate \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"rating": 5, "comment": "Great ride!"}'
```

### Payment Flow

```bash
# 1. Get wallet balance
curl -X GET http://localhost:8083/api/v1/wallet \
  -H "Authorization: Bearer $RIDER_TOKEN"

# 2. Top up wallet
curl -X POST http://localhost:8083/api/v1/wallet/topup \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"amount": 50.00, "stripe_payment_method": "pm_card_visa"}'

# 3. Process payment
curl -X POST http://localhost:8083/api/v1/payments/process \
  -H "Authorization: Bearer $RIDER_TOKEN" \
  -H "Content-Type: application/json" \
  -d '{"ride_id": "$RIDE_ID", "amount": 25.50, "payment_method": "wallet"}'
```

## Troubleshooting

### Common Issues

#### 1. "Connection refused" errors

**Problem:** Services are not running

**Solution:**
```bash
# Check service status
docker-compose ps

# Start services
docker-compose up -d

# Check logs
docker-compose logs -f auth-service
```

#### 2. "Unauthorized" errors

**Problem:** Missing or expired token

**Solution:**
```bash
# Login again to get new token
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{"email":"your@email.com","password":"yourpassword"}'
```

#### 3. "Invalid ride ID" errors

**Problem:** Using wrong ride ID or ride doesn't exist

**Solution:**
```bash
# Get your rides
curl -X GET http://localhost:8082/api/v1/rides \
  -H "Authorization: Bearer $RIDER_TOKEN"

# Use the ID from the response
```

#### 4. Database connection errors

**Problem:** Database is not running or migrations not applied

**Solution:**
```bash
# Check database
docker-compose ps postgres

# Run migrations
make migrate-up
```

#### 5. "jq command not found"

**Problem:** jq is not installed (for shell scripts)

**Solution:**
```bash
# macOS
brew install jq

# Ubuntu/Debian
sudo apt-get install jq

# Fedora/RHEL
sudo yum install jq
```

### Service Health Checks

Check if services are healthy:

```bash
# Auth service
curl http://localhost:8081/healthz

# Rides service
curl http://localhost:8082/health/ready

# Payments service
curl http://localhost:8083/health/ready
```

### Viewing Logs

```bash
# All services
docker-compose logs -f

# Specific service
docker-compose logs -f rides-service

# Last 100 lines
docker-compose logs --tail=100 auth-service
```

### Port Conflicts

If ports are already in use:

1. Edit `docker-compose.yml`
2. Change port mappings (e.g., `8081:8080` → `9081:8080`)
3. Update environment variables in scripts
4. Restart services

## API Response Format

All API responses follow a consistent format:

### Success Response (200, 201)
```json
{
  "success": true,
  "data": {
    "id": "uuid",
    "field": "value"
  },
  "message": "Operation successful"
}
```

### Error Response (4xx, 5xx)
```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "Human readable error message"
  }
}
```

### Paginated Response
```json
{
  "success": true,
  "data": [...],
  "meta": {
    "total": 100,
    "limit": 20,
    "offset": 0
  }
}
```

## Rate Limiting

The API implements rate limiting:

- **Default limit:** 120 requests per minute
- **Burst allowance:** 40 requests
- **Headers returned:**
  - `X-RateLimit-Limit` - Total requests allowed
  - `X-RateLimit-Remaining` - Remaining requests
  - `X-RateLimit-Reset` - Time when limit resets

When rate limited:
- **Status:** 429 Too Many Requests
- **Header:** `Retry-After` - Seconds to wait before retrying

## Additional Resources

- [Backend Architecture](../docs/ARCHITECTURE.md)
- [Development Guide](../CONTRIBUTING.md)
- [API Documentation](../docs/API.md)
- [Deployment Guide](../docs/DEPLOYMENT.md)

## Support

For issues or questions:

1. Check [Troubleshooting](#troubleshooting) section
2. Review service logs
3. Check GitHub Issues
4. Contact the development team

## Contributing

To add new endpoints to the collection:

1. Update `api/postman/ride-hailing.postman_collection.json`
2. Add corresponding shell script examples
3. Update this README with usage instructions
4. Test all flows end-to-end
5. Submit a pull request

---

**Last Updated:** 2025-01-18
**Version:** 1.0.0
