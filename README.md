# Ride Hailing Platform - Production-Ready Backend

[![CI](https://github.com/richxcame/ride-hailing/actions/workflows/test.yml/badge.svg)](https://github.com/richxcame/ride-hailing/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/richxcame/ride-hailing/branch/main/graph/badge.svg)](https://app.codecov.io/gh/richxcame/ride-hailing)

A complete, production-ready ride-hailing platform backend built with Go, featuring 13 microservices that handle everything from authentication to ML-powered predictions.

## Status: Phase 3 Complete 🎉

**13 Microservices** | **90+ API Endpoints** | **Enterprise Ready** | **ML-Powered**

---

## Features

### Core Services

-   ✅ **Authentication Service** - JWT-based auth with role-based access control
-   ✅ **Rides Service** - Complete ride lifecycle management with ratings & scheduled rides
-   ✅ **Geolocation Service** - Redis GeoSpatial driver matching (10km radius)
-   ✅ **Payment Service** - Stripe integration + wallet system with auto payouts
-   ✅ **Notification Service** - Multi-channel (Firebase push, Twilio SMS, Email)
-   ✅ **Real-time Service** - WebSockets with in-app chat (Hub pattern)
-   ✅ **Mobile Service** - Optimized APIs for mobile apps
-   ✅ **Admin Service** - Complete dashboard backend with analytics

### Advanced Services (Phase 2)

-   ✅ **Promos Service** - Promo codes, referral system, ride types
-   ✅ **Scheduler Service** - Automated ride scheduling and notifications
-   ✅ **Analytics Service** - Business intelligence and reporting
-   ✅ **Fraud Service** - Fraud detection and prevention

### Enterprise Services (Phase 3)

-   ✅ **ML ETA Service** - Machine learning-based ETA prediction
-   ✅ **Kong API Gateway** - Centralized API management with rate limiting
-   ✅ **Istio Service Mesh** - mTLS, traffic management, observability

### Key Capabilities

-   🔐 Secure JWT authentication with refresh tokens
-   💰 Real payment processing (Stripe)
-   💳 Wallet system with transaction history
-   📍 Smart driver matching with Redis GeoSpatial
-   🔔 Multi-channel notifications (push, SMS, email)
-   ⚡ Real-time updates via WebSockets
-   💬 In-app chat with 24h message history
-   📊 Admin dashboard with analytics
-   📱 Mobile-optimized APIs
-   🎁 Promo codes & referral system
-   📅 Scheduled rides
-   🚗 Multiple ride types (Economy, Premium, XL)
-   🛡️ Fraud detection & prevention
-   📈 Prometheus metrics + Grafana dashboards
-   🤖 ML-powered ETA prediction (85%+ accuracy)
-   🌐 Kong API Gateway with rate limiting
-   🔒 Istio service mesh with mTLS
-   ☸️ Kubernetes-ready with auto-scaling
-   🔄 Circuit breaking & fault tolerance

---

## Tech Stack

-   **Language**: Go 1.24+
-   **Framework**: Gin
-   **Database**: PostgreSQL 15 with PostGIS (connection pooling + read replicas)
-   **Cache**: Redis 7 (GeoSpatial + Pub/Sub)
-   **WebSocket**: gorilla/websocket
-   **Payments**: Stripe API v83.2.0
-   **Notifications**: Firebase FCM, Twilio SMS, SMTP
-   **Auth**: JWT with bcrypt + refresh tokens
-   **Observability**: Prometheus + Grafana + Structured Logging (Zap)
-   **API Gateway**: Kong with rate limiting & mTLS
-   **Service Mesh**: Istio
-   **Deployment**: Docker + Kubernetes + Istio
-   **Testing**: Go test framework with comprehensive unit & integration tests
-   **Resilience**: Circuit breakers, rate limiting, retry logic

---

## Architecture

```mermaid
%%{init: { "flowchart": { "nodeSpacing": 100, "rankSpacing": 260, "padding": 15 } }}%%
flowchart TB

    %% Client Layer
    subgraph Clients[Client Applications]
        MobileApp[Mobile Apps]
        WebApp[Web Dashboard]
        AdminPanel[Admin Panel]
    end

    %% Gateway Layer
    Kong[Kong API Gateway
    :8000 / :8443]

    %% Core Domain Services
    Auth[Auth Service :8081]
    Rides[Rides Service :8082]
    Geo[Geo Service :8083]
    Payments[Payments Service :8084]
    Notifs[Notifications Service :8085]

    %% Secondary & System Services
    Realtime[Realtime / WebSocket :8086]
    MobileSvc[Mobile API :8087]
    AdminSvc[Admin API :8088]
    Promos[Promos & Referrals :8089]
    Scheduler[Scheduler Workers :8090]
    Analytics[Analytics / BI :8091]
    Fraud[Fraud Detection :8092]
    ML[ML ETA Prediction :8093]

    %% Data Stores
    Postgres[(PostgreSQL
    Persistent Storage)]
    Redis[(Redis Cluster
    Cache + Geo + Pub/Sub + WS State)]

    %% Observability
    Prometheus[(Prometheus
    Metrics Scraper)]
    Grafana[(Grafana
    Dashboards & Monitoring UI)]

    %% Client Access
    Clients --> Kong

    %% Routing from Gateway
    Kong --> Auth
    Kong --> Rides
    Kong --> Geo
    Kong --> Payments
    Kong --> Notifs
    Kong --> MobileSvc
    Kong --> Realtime

    %% Admin internal access
    AdminPanel --> AdminSvc

    %% Service -> Database
    Auth --> Postgres
    Rides --> Postgres
    Payments --> Postgres
    Notifs --> Postgres
    MobileSvc --> Postgres
    AdminSvc --> Postgres
    Promos --> Postgres
    Scheduler --> Postgres
    Analytics --> Postgres
    Fraud --> Postgres
    ML --> Postgres

    %% Service -> Redis
    Geo --> Redis
    Realtime --> Redis
    ML --> Redis

    %% Observability (non-blocking)
    Prometheus --> Auth
    Prometheus --> Rides
    Prometheus --> Geo
    Prometheus --> Payments
    Prometheus --> Notifs
    Prometheus --> Realtime
    Prometheus --> MobileSvc
    Prometheus --> AdminSvc
    Prometheus --> Promos
    Prometheus --> Scheduler
    Prometheus --> Analytics
    Prometheus --> Fraud
    Prometheus --> ML

    %% Dashboards visualize metrics
    Grafana --> Prometheus
```

---

## Services

| Service              | Port        | Purpose                                         | Status          |
| -------------------- | ----------- | ----------------------------------------------- | --------------- |
| **Auth**             | 8081        | User authentication, JWT issuance, roles        | ✅ Production   |
| **Rides**            | 8082        | Ride lifecycle, trip state machine, scheduling  | ✅ Production   |
| **Geo**              | 8083        | Driver locations, geospatial matching, ETA      | ✅ Production   |
| **Payments**         | 8084        | Payments, wallets, Stripe/Tink integration      | ✅ Production   |
| **Notifications**    | 8085        | Push, SMS, email, multi-channel alerts          | ✅ Production   |
| **Realtime**         | 8086        | WebSocket events (rides, driver tracking)       | ✅ Production   |
| **Mobile API**       | 8087        | Mobile-optimized backend gateway                | ✅ Production   |
| **Admin API**        | 8088        | Admin dashboard backend                         | ✅ Production   |
| **Promos**           | 8089        | Promo codes, referral program engine            | ✅ Production   |
| **Scheduler**        | 8090        | Async background jobs (cron workers)            | ✅ Production   |
| **Analytics**        | 8091        | BI, metrics aggregation, reporting              | ✅ Production   |
| **Fraud**            | 8092        | Fraud detection rules + heuristic scoring       | ✅ Production   |
| **ML ETA**           | 8093        | ML-based ETA prediction + model inference       | ✅ Production   |
| **Kong Gateway**     | 8000 / 8443 | API Gateway (Traffic Control, JWT, Rate Limits) | ✅ Production   |
| **Kong Admin API**   | 8001 / 8444 | Internal Admin API (secured; restricted)        | ⚠ Internal Only |
| **Kong Manager GUI** | 8002        | Web UI for Gateway configuration                | 🔐 Protected    |
| **PostgreSQL**       | 5432        | Main persistent relational database             | ✅ Production   |
| **Redis**            | 6379        | Caching, Pub/Sub, Geo, Realtime session states  | ✅ Production   |

### 1. Auth Service (Port 8081)

-   User registration (riders, drivers, admins)
-   Login with JWT token generation
-   Token refresh mechanism
-   Role-based access control (RBAC)
-   Password hashing with bcrypt

**Endpoints**: 5 (register, login, refresh, profile GET/PUT)

### 2. Rides Service (Port 8082)

-   Ride request creation with fare estimation
-   Driver acceptance workflow
-   Ride lifecycle (requested → accepted → in_progress → completed)
-   Ride cancellation with reasons
-   Rating and feedback system (1-5 stars)
-   Scheduled rides support
-   Dynamic surge pricing
-   Advanced filtering (status, date range)
-   Receipt generation

**Endpoints**: 10+ (create, get, accept, start, complete, cancel, rate, history, surge-info, schedule)

### 3. Geo Service (Port 8083)

-   Real-time driver location updates
-   Redis GeoSpatial indexing (GEORADIUS)
-   Find nearby drivers (10km radius, configurable)
-   Driver status tracking (available/busy/offline)
-   Distance calculation (Haversine formula)
-   Automatic geo index maintenance

**Endpoints**: 4 (update location, get nearby, get location, health)

### 4. Payments Service (Port 8084)

-   Stripe Payment Intent integration
-   Wallet system (balance, top-up, transactions)
-   Dual payment methods (wallet or Stripe)
-   Automatic driver payouts (80/20 split)
-   Platform commission (20%)
-   Refunds with cancellation fees (10%)
-   Transaction history
-   Webhook handling

**Endpoints**: 7 (process payment, wallet topup, get wallet, transactions, refund, webhooks, health)

### 5. Notifications Service (Port 8085)

-   Firebase Cloud Messaging (push notifications)
-   Twilio SMS integration
-   SMTP email with HTML templates
-   Multi-channel support (push/SMS/email)
-   Scheduled notifications
-   Bulk notifications (admin only)
-   Background worker (1-minute ticker)
-   Ride event notifications

**Endpoints**: 11 (list, unread count, mark read, send, schedule, ride events, bulk, health)

### 6. Real-time Service (Port 8086)

-   WebSocket server with Hub pattern
-   Real-time driver location streaming
-   Live ride status updates
-   In-app chat (rider-driver)
-   Typing indicators
-   Room-based messaging (ride-specific)
-   Redis-backed chat history (24h TTL)
-   Ping/pong heartbeat (60s)

**Endpoints**: 2 (WebSocket upgrade, internal broadcast API)

### 7. Mobile Service (Port 8087)

-   Ride history with filters (status, date range)
-   Favorite locations (CRUD)
-   Trip receipts with fare breakdown
-   Driver ratings
-   User profile management
-   Pagination support

**Endpoints**: 8 (history, receipt, rate, favorites CRUD, profile)

### 8. Admin Service (Port 8088)

-   Dashboard with aggregated statistics
-   User management (list, view, suspend, activate)
-   Driver approval workflow
-   Ride monitoring (recent rides, stats)
-   Analytics (user stats, ride stats, revenue)
-   Date range filtering
-   All endpoints protected by admin middleware

**Endpoints**: 10 (dashboard, users, drivers, rides, stats, health)

### 9. Promos Service (Port 8089)

-   Promo code creation & management (percentage/fixed discounts)
-   Promo code usage tracking & limits
-   Referral code generation (unique per user)
-   Referral bonus system (for both referrer & referred)
-   Ride type management (Economy, Premium, XL)
-   Usage analytics & reporting

**Endpoints**: 12+ (promo CRUD, apply, validate, referral system, ride types)

### 10. Scheduler Service (Port 8090)

-   Scheduled ride management
-   Automated ride dispatch (30 minutes before scheduled time)
-   Notification system for upcoming rides
-   Background worker for scheduled tasks
-   Ride cleanup & optimization

**Endpoints**: 6 (schedule ride, list scheduled, update, cancel, health)

### 11. Analytics Service (Port 8091)

-   Real-time business metrics & KPIs
-   Revenue tracking & forecasting
-   Driver performance analytics
-   Ride completion & cancellation rates
-   Demand heat maps & patterns
-   Materialized views for performance
-   Custom date range reporting

**Endpoints**: 8 (overview, revenue, drivers, rides, demand, export)

### 12. Fraud Service (Port 8092)

-   Suspicious activity detection
-   Duplicate account prevention
-   Payment fraud monitoring
-   Driver behavior analysis
-   Ride pattern anomaly detection
-   Risk scoring algorithm
-   Automated flagging & alerts

**Endpoints**: 7 (check ride, check payment, check user, reports, health)

### 13. ML ETA Service (Port 8093)

-   Machine learning-based ETA prediction
-   Multi-factor analysis (distance, traffic, weather, time)
-   Historical data learning & pattern recognition
-   Automatic model retraining (daily)
-   Confidence scoring for predictions
-   Batch prediction support
-   Model performance tracking (85%+ accuracy)
-   Feature importance analysis

**Endpoints**: 8 (predict, batch predict, train, model stats, accuracy, tune, analytics)

---

## Quick Start

### Prerequisites

-   Docker and Docker Compose
-   Go 1.24+ (for local development)
-   PostgreSQL 15
-   Redis 7

### Development Mode (Recommended)

For active development, run only infrastructure in Docker and services natively:

```bash
# 1. Start minimal infrastructure (Postgres + Redis)
make dev-infra

# 2. Run migrations
make migrate-up

# 3. Run the service(s) you're working on
make run-auth  # or run-rides, run-geo, etc.

# 4. Check environment health
make dev-check
```

**Benefits:**

-   🚀 Fast startup (~5 seconds vs 2-3 minutes)
-   💾 Low memory (~500MB vs 4-6GB)
-   ⚡ Instant iteration (no Docker rebuild)
-   🐛 Easy debugging (native Go debugger)

**Optional Profiles:**

```bash
# Add observability stack (Prometheus, Grafana, Tempo)
docker-compose -f docker-compose.dev.yml --profile observability up -d

# Add Kong API Gateway
docker-compose -f docker-compose.dev.yml --profile gateway up -d

# Add Sentry error tracking
docker-compose -f docker-compose.dev.yml --profile sentry up -d

# Combine multiple profiles
docker-compose -f docker-compose.dev.yml --profile observability --profile sentry up -d
```

**See [QUICKSTART.md](./docs/QUICKSTART.md) and [DEVELOPMENT.md](./docs/DEVELOPMENT.md) for detailed guides.**

### Running with Docker Compose

1. **Clone the repository**

    ```bash
    git clone https://github.com/richxcame/ride-hailing.git
    cd ride-hailing
    ```

2. **Start all services**

    ```bash
    docker-compose up -d
    ```

3. **Check service health**

    ```bash
    # Check all services
    curl http://localhost:8081/healthz  # Auth
    curl http://localhost:8082/healthz  # Rides
    curl http://localhost:8083/healthz  # Geo
    curl http://localhost:8084/healthz  # Payments
    curl http://localhost:8085/healthz  # Notifications
    curl http://localhost:8086/healthz  # Real-time
    curl http://localhost:8087/healthz  # Mobile
    curl http://localhost:8088/healthz  # Admin
    curl http://localhost:8089/healthz  # Promos
    curl http://localhost:8090/healthz  # Scheduler
    curl http://localhost:8091/healthz  # Analytics
    curl http://localhost:8092/healthz  # Fraud
    ```

4. **View logs**
    ```bash
    docker-compose logs -f
    ```

### Local Development

1. **Install dependencies**

    ```bash
    go mod download
    ```

2. **Build all services**

    ```bash
    go build -o bin/auth ./cmd/auth
    go build -o bin/rides ./cmd/rides
    go build -o bin/geo ./cmd/geo
    go build -o bin/payments ./cmd/payments
    go build -o bin/notifications ./cmd/notifications
    go build -o bin/realtime ./cmd/realtime
    go build -o bin/mobile ./cmd/mobile
    go build -o bin/admin ./cmd/admin
    go build -o bin/promos ./cmd/promos
    go build -o bin/scheduler ./cmd/scheduler
    go build -o bin/analytics ./cmd/analytics
    go build -o bin/fraud ./cmd/fraud
    ```

3. **Run a single service**
    ```bash
    ./bin/auth
    # Or use go run
    go run cmd/auth/main.go
    ```

---

## Environment Variables

### Required for All Services

```bash
DB_HOST=postgres
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ride_hailing
DB_BREAKER_ENABLED=true
DB_BREAKER_FAILURE_THRESHOLD=5
DB_BREAKER_SUCCESS_THRESHOLD=1
DB_BREAKER_TIMEOUT_SECONDS=30
DB_BREAKER_INTERVAL_SECONDS=60
JWT_SECRET=your-secret-key-change-in-production

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=

# Optional: JWT key rotation (defaults shown)
JWT_KEYS_FILE=config/jwt_keys.json
JWT_ROTATION_HOURS=720        # 30 days
JWT_ROTATION_GRACE_HOURS=720  # overlap window
JWT_KEY_REFRESH_MINUTES=5     # how often services reload keys

# Timeout configuration (defaults shown)
HTTP_CLIENT_TIMEOUT=30              # HTTP client timeout in seconds
DB_QUERY_TIMEOUT=10                 # Database query timeout in seconds
REDIS_OPERATION_TIMEOUT=5           # Redis operation timeout in seconds (fallback if read/write not set)
REDIS_READ_TIMEOUT=5                # Redis read timeout in seconds (defaults to REDIS_OPERATION_TIMEOUT if not set)
REDIS_WRITE_TIMEOUT=5               # Redis write timeout in seconds (defaults to REDIS_OPERATION_TIMEOUT if not set)
WS_CONNECTION_TIMEOUT=60            # WebSocket connection timeout in seconds
DEFAULT_REQUEST_TIMEOUT=30          # Default HTTP request timeout in seconds
ROUTE_TIMEOUT_OVERRIDES='{"POST:/api/v1/rides": 60}'  # Optional: JSON map of route-specific timeouts (format: "METHOD:/path": seconds)
```

Each service loads a shared `config/jwt_keys.json` (gitignored). The auth service
rotates keys automatically while other services run in read-only mode, polling
for updates based on `JWT_KEY_REFRESH_MINUTES`. The `JWT_SECRET` value remains as
the fallback legacy key and initial seed. Start the auth service once to seed
`config/jwt_keys.json` before launching the rest of the stack, or provide the
keys file through your secrets manager.

**Timeout Configuration:**

All services now use configurable timeouts for better reliability and resource management:

- **HTTP Client Timeout**: Controls timeout for outbound HTTP requests (e.g., service-to-service calls)
- **Database Query Timeout**: Sets PostgreSQL `statement_timeout` to prevent long-running queries
- **Redis Operation Timeout**: Base timeout for Redis operations (used as fallback)
- **Redis Read Timeout**: Timeout for Redis read operations (defaults to Redis Operation Timeout if not set)
- **Redis Write Timeout**: Timeout for Redis write operations (defaults to Redis Operation Timeout if not set)
- **WebSocket Connection Timeout**: Timeout for WebSocket connection establishment
- **Default Request Timeout**: HTTP request timeout middleware applied to all endpoints (returns 504 Gateway Timeout if exceeded)
- **Route-Specific Timeout Overrides**: Optional per-route timeout overrides using `ROUTE_TIMEOUT_OVERRIDES` environment variable

These timeouts are applied automatically via middleware and client initialization. The request timeout middleware logs timeout events with correlation IDs for debugging.

**Route-Specific Timeout Overrides:**

You can override the default request timeout for specific routes using the `ROUTE_TIMEOUT_OVERRIDES` environment variable. This is useful for routes that require longer processing times (e.g., complex queries, file uploads, or ML model inference).

```bash
# JSON format: {"METHOD:/path": timeout_in_seconds}
ROUTE_TIMEOUT_OVERRIDES='{"POST:/api/v1/rides": 60, "GET:/api/v1/analytics/reports": 120, "POST:/api/v1/rides/batch": 90}'
```

- **Route format**: `"METHOD:/path"` (e.g., `"POST:/api/v1/rides"`, `"GET:/api/v1/users/:id"`)
- **Timeout value**: Integer in seconds (must be > 0)
- **Behavior**: Routes matching the pattern use the specified timeout; all other routes use `DEFAULT_REQUEST_TIMEOUT`
- **Example use cases**:
  - Long-running ML model predictions
  - Complex analytics queries
  - Batch processing endpoints
  - File upload/download operations

Routes with invalid timeout values (≤ 0) are automatically filtered out during configuration loading.

### Rate Limiting (Rides Service)

```bash
# Enable Redis-backed token bucket enforcement
RATE_LIMIT_ENABLED=true
# Default authenticated requests per window
RATE_LIMIT_DEFAULT_LIMIT=120
# Additional burst capacity above the steady limit
RATE_LIMIT_DEFAULT_BURST=40
# Anonymous (IP-based) defaults
RATE_LIMIT_ANON_LIMIT=60
RATE_LIMIT_ANON_BURST=20
# Window size in seconds
RATE_LIMIT_WINDOW_SECONDS=60
# Optional JSON overrides per METHOD:PATH pattern
RATE_LIMIT_ENDPOINTS='{"POST:/api/v1/rides":{"authenticated_limit":30,"anonymous_limit":10,"window_seconds":60}}'
```

### Resilience (Circuit Breakers)

```bash
# Toggle distributed circuit breakers for outbound calls
CB_ENABLED=true
# Consecutive failures required to trip
CB_FAILURE_THRESHOLD=5
# Successful requests required in half-open state
CB_SUCCESS_THRESHOLD=1
# Seconds before attempting to close the breaker
CB_TIMEOUT_SECONDS=30
# Rolling window for failure counts
CB_INTERVAL_SECONDS=60
# Optional JSON overrides per upstream (keys match service names)
CB_SERVICE_OVERRIDES='{"promos-service":{"failure_threshold":3,"timeout_seconds":15}}'
```

Available override keys:

-   `promos-service` (promotions pricing HTTP client)
-   `stripe-api` (payments service calls to Stripe)
-   `firebase-fcm`, `twilio-sms`, `smtp-email` (notifications service channels)
-   `ml-eta-service` (rides service -> ML ETA integration)
-   `database-primary` / `database-replica` (pool creation throttling)

Each breaker surfaces Prometheus metrics (`circuit_breaker_state`, `circuit_breaker_requests_total`, `circuit_breaker_failures_total`, `circuit_breaker_fallbacks_total`) so dashboards can track failure rates and state transitions.

### Payments Service (Port 8084)

```bash
STRIPE_API_KEY=sk_test_51xxxxx...  # Get from Stripe Dashboard
```

### Notifications Service (Port 8085)

```bash
# Firebase (Optional - for push notifications)
FIREBASE_CREDENTIALS_PATH=/path/to/firebase-credentials.json

# Twilio (Optional - for SMS)
TWILIO_ACCOUNT_SID=ACxxxxxxxxx
TWILIO_AUTH_TOKEN=xxxxxxxxx
TWILIO_FROM_NUMBER=+1234567890

# SMTP (Optional - for email)
SMTP_HOST=smtp.gmail.com
SMTP_PORT=587
SMTP_USERNAME=your@email.com
SMTP_PASSWORD=your-app-password
SMTP_FROM_EMAIL=noreply@ridehailing.com
SMTP_FROM_NAME=RideHailing
```

### ML ETA Integration

```bash
ML_ETA_SERVICE_URL=http://ml-eta-service:8080
```

When configured, the rides service fetches machine-learning powered ETAs via `/api/v1/eta/predict` and automatically falls back to distance-based calculations whenever the circuit breaker opens or the ML service is unavailable.

### Secrets Management

The platform now supports centralized secret delivery via a pluggable secrets manager (currently Vault-backed). When `SECRETS_PROVIDER` is set, configuration values for the database, Stripe, notifications, and JWT signing keys are pulled from the manager instead of local environment variables.

```bash
# Enable the secrets manager (currently supports `vault`)
SECRETS_PROVIDER=vault
SECRETS_CACHE_TTL_SECONDS=300        # local cache to limit Vault round-trips
SECRETS_ROTATION_DAYS=90            # rotation reminder window
SECRETS_AUDIT_ENABLED=true

# Vault connection
SECRETS_VAULT_ADDR=https://vault.example.com
SECRETS_VAULT_TOKEN=s.xxxxxx
SECRETS_VAULT_NAMESPACE=platform
SECRETS_VAULT_MOUNT=kv
SECRETS_VAULT_CACERT=/etc/ssl/certs/vault-ca.pem
# Optional client auth
SECRETS_VAULT_CLIENT_CERT=/etc/ssl/certs/vault-client.pem
SECRETS_VAULT_CLIENT_KEY=/etc/ssl/private/vault-client-key.pem

# Secret references ([provider://][mount:]path[@version][#key])
SECRETS_DB_PATH=kv:ride-hailing/database
SECRETS_STRIPE_PATH=kv:ride-hailing/stripe#api_key
SECRETS_TWILIO_PATH=kv:ride-hailing/twilio
SECRETS_SMTP_PATH=kv:ride-hailing/smtp
SECRETS_FIREBASE_PATH=kv:ride-hailing/firebase
SECRETS_JWT_KEYS_PATH=kv:ride-hailing/jwt-keys
```

Other supported providers:

-   **AWS Secrets Manager:** `SECRETS_PROVIDER=aws`, `SECRETS_AWS_REGION=us-east-1`, optional `SECRETS_AWS_ACCESS_KEY_ID`, `SECRETS_AWS_SECRET_ACCESS_KEY`, `SECRETS_AWS_PROFILE`, `SECRETS_AWS_ENDPOINT`.
-   **Google Secret Manager:** `SECRETS_PROVIDER=gcp`, `SECRETS_GCP_PROJECT_ID=ride-hailing-prod`, optional `SECRETS_GCP_CREDENTIALS_JSON` or `SECRETS_GCP_CREDENTIALS_FILE`.
-   **Kubernetes Secrets:** `SECRETS_PROVIDER=kubernetes` with secrets mounted under `SECRETS_K8S_BASE_PATH` (default `/var/run/secrets/ride-hailing`). Each directory/key under that path is treated as a secret payload.

Each referenced secret should store structured values (e.g., the database secret includes `host`, `port`, `username`, `password`, `dbname`, `sslmode`). Firebase secrets can provide either `credentials_json` or a base64 string under `credentials_b64`.

Apply the sample Vault policy in `configs/vault-policy.hcl` to grant the Ride Hailing services least-privilege access and to enable rotation/deletion of old versions:

```bash
vault policy write ride-hailing configs/vault-policy.hcl
```

Audit events for every secret fetch are emitted through the shared zap logger so they land in your central log pipeline.

---

## API Examples

### 1. Register a Rider

```bash
curl -X POST http://localhost:8081/api/v1/auth/register \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@test.com",
    "password": "password123",
    "phone_number": "+1234567890",
    "first_name": "John",
    "last_name": "Doe",
    "role": "rider"
  }'
```

### 2. Login and Get Token

```bash
curl -X POST http://localhost:8081/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@test.com",
    "password": "password123"
  }'

# Save the token!
TOKEN="eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
```

### 3. Top Up Wallet

```bash
curl -X POST http://localhost:8084/api/v1/wallet/topup \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "amount": 50.00,
    "stripe_payment_method": "pm_card_visa"
  }'
```

### 4. Request a Ride

```bash
curl -X POST http://localhost:8082/api/v1/rides \
  -H "Authorization: Bearer $TOKEN" \
  -H "Content-Type: application/json" \
  -d '{
    "pickup_latitude": 40.7128,
    "pickup_longitude": -74.0060,
    "pickup_address": "New York, NY",
    "dropoff_latitude": 40.7589,
    "dropoff_longitude": -73.9851,
    "dropoff_address": "Times Square, NY"
  }'
```

### 5. Connect to WebSocket (Real-time Updates)

```javascript
const ws = new WebSocket('ws://localhost:8086/ws?token=YOUR_TOKEN');

ws.onopen = () => {
	console.log('Connected to real-time service');

	// Join a ride room
	ws.send(
		JSON.stringify({
			type: 'join_ride',
			payload: { ride_id: 'ride-uuid' },
		})
	);
};

ws.onmessage = (event) => {
	const message = JSON.parse(event.data);
	console.log('Received:', message);
};
```

---

## Database Schema

The platform uses PostgreSQL with the following tables:

### Core Tables (Migration 000001)

-   `users` - User accounts (riders, drivers, admins)
-   `drivers` - Driver profiles and vehicle information
-   `rides` - Ride records with full lifecycle
-   `wallets` - User wallet balances
-   `payments` - Payment transaction records
-   `wallet_transactions` - All wallet transactions
-   `notifications` - Notification records

### Location Tables (Migration 000002)

-   `driver_locations` - Driver location history with GPS tracking
-   `favorite_locations` - User's saved addresses

### Promo System (Migration 000003)

-   `promo_codes` - Promotional codes with discount rules
-   `promo_code_uses` - Usage tracking per user/ride
-   `referral_codes` - User referral codes
-   `referrals` - Referral relationships & bonuses
-   `ride_types` - Ride categories (Economy, Premium, XL)

### Scheduling (Migration 000004)

-   Enhanced `rides` table with scheduling columns
-   `scheduled_rides` view for upcoming rides
-   Helper functions for scheduled ride queries

See [db/migrations/](db/migrations/) for complete schema definitions.

---

## Redis Data Structures

-   `drivers:geo:index` - GeoSpatial index for nearby driver search
-   `ride:chat:{rideID}` - Chat history (24h TTL)
-   `driver:location:{driverID}` - Latest driver location cache (5min TTL)

---

## Monitoring

### Prometheus Metrics

All services expose Prometheus metrics at `/metrics`:

-   `http_requests_total` - Request count by service/method/endpoint
-   `http_request_duration_seconds` - Request latency

### Grafana

Access Grafana at: http://localhost:3000

-   Username: admin
-   Password: admin

Pre-configured dashboards:

-   Service health overview
-   Request latency by endpoint
-   Error rates
-   Database connection pool status

---

## Testing

The platform has comprehensive test coverage across all services:

**Unit Tests:**

-   Auth service (JWT validation, RBAC, password hashing)
-   Geo service (Redis GeoSpatial, distance calculations)
-   Notifications service (FCM, Twilio, SMTP mocking)
-   Real-time service (WebSocket hub, message routing)
-   Fraud service (risk scoring, alert generation)
-   ML ETA service (prediction accuracy, feature weights)
-   Analytics service (aggregation queries, metrics)
-   Promos service (discount calculations, referral logic)

**Integration Tests:**

-   Complete ride flow (request → match → pickup → complete → payment)
-   Authentication flow (register → login → refresh token)
-   Payment processing with Stripe webhooks
-   Promo code application and validation
-   Admin operations (user management, driver approval)

### Run Tests

```bash
# Run all tests
go test ./...

# Run with coverage
go test -cover ./...

# Run specific service tests
go test ./internal/auth/... -v
go test ./internal/rides/... -v
go test ./internal/payments/... -v

# Run integration tests
go test ./test/integration/... -v
```

### Test Infrastructure

-   Docker Compose test environment ([docker-compose.test.yml](docker-compose.test.yml))
-   Test helpers and utilities ([test/helpers/](test/helpers/))
-   Mock implementations ([test/mocks/](test/mocks/))
-   Integration test suite ([test/integration/](test/integration/))

---

## Documentation

-   **[TODO.md](TODO.md)** - Development roadmap and improvement plan
-   **[docs/API.md](docs/API.md)** - Complete API documentation for all services
-   **[docs/DEPLOYMENT.md](docs/DEPLOYMENT.md)** - Deployment guides (Docker, K8s, Cloud)
-   **[docs/RETRY_LOGIC.md](docs/RETRY_LOGIC.md)** - Retry logic and exponential backoff guide
-   **[db/migrations/](db/migrations/)** - Database schema migrations (9 migrations)

---

## Production Deployment

### Production-Ready Features ✅

-   ✅ Kong API Gateway configured with rate limiting
-   ✅ Istio service mesh with mTLS
-   ✅ Circuit breakers for external dependencies
-   ✅ Retry logic with exponential backoff and jitter
-   ✅ Rate limiting (Redis-backed token bucket)
-   ✅ Comprehensive logging with correlation IDs
-   ✅ Security headers + input sanitization middleware on every service
-   ✅ JWT signing key rotation with per-token KID headers
-   ✅ Prometheus metrics on all services
-   ✅ Health check endpoints (liveness + readiness)
-   ✅ Database connection pooling + read replicas
-   ✅ Configurable timeouts (HTTP clients, database queries, Redis operations, request middleware)
-   ✅ Kubernetes manifests with HPA
-   ✅ Docker Compose for local/staging

### Before Going Live

-   [ ] Rotate all API keys and secrets (use secrets management)
-   [ ] Change JWT_SECRET to strong random value
-   [ ] Use production Stripe API keys
-   [ ] Set up Firebase production project
-   [ ] Configure production SMTP credentials
-   [ ] Enable HTTPS/TLS on all services
-   [ ] Set up CORS properly for production domains
-   [ ] Enable database backups (automated)
-   [ ] Set up log aggregation (ELK/Loki)
-   [ ] Configure error alerting (Sentry/PagerDuty)
-   [ ] Load testing (target: 1000+ concurrent rides)
-   [ ] Security audit & penetration testing
-   [ ] Configure fraud detection rules
-   [ ] Test scheduled ride dispatch
-   [ ] Set up Grafana dashboards
-   [ ] Configure alerting rules (Prometheus)

---

## Project Structure

```
ride-hailing/
├── cmd/                    # Service entry points
│   ├── auth/              # Auth service
│   ├── rides/             # Rides service
│   ├── geo/               # Geo service
│   ├── payments/          # Payments service
│   ├── notifications/     # Notifications service
│   ├── realtime/          # Real-time service
│   ├── mobile/            # Mobile service
│   ├── admin/             # Admin service
│   ├── promos/            # Promos service
│   ├── scheduler/         # Scheduler service
│   ├── analytics/         # Analytics service
│   ├── fraud/             # Fraud service
│   └── ml_eta/            # ML ETA service
├── internal/              # Private application code
│   ├── auth/             # Auth business logic
│   ├── rides/            # Rides business logic
│   ├── geo/              # Geo business logic
│   ├── payments/         # Payments business logic
│   ├── notifications/    # Notifications business logic
│   ├── realtime/         # Real-time business logic
│   ├── favorites/        # Favorites business logic
│   ├── admin/            # Admin business logic
│   ├── promos/           # Promos business logic
│   ├── analytics/        # Analytics business logic
│   ├── fraud/            # Fraud business logic
│   ├── scheduler/        # Scheduler business logic
│   └── ml_eta/           # ML ETA business logic
├── pkg/                   # Public shared libraries
│   ├── common/           # Common utilities
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── redis/            # Redis client
│   ├── websocket/        # WebSocket utilities
│   ├── resilience/       # Circuit breakers, retries
│   ├── cache/            # Caching utilities
│   └── validation/       # Input validation
├── third_party/           # Third-party packages
│   └── gobreaker/        # Custom circuit breaker
├── test/                  # Test infrastructure
│   ├── integration/      # Integration tests
│   ├── helpers/          # Test utilities
│   └── mocks/            # Mock implementations
├── db/migrations/         # Database migrations (9 migrations)
├── k8s/                   # Kubernetes manifests
├── istio/                 # Istio service mesh config
├── monitoring/            # Prometheus & Grafana config
├── docs/                  # Documentation
├── docker-compose.yml     # Docker Compose config
├── go.mod                 # Go dependencies
├── TODO.md               # Development roadmap
└── README.md             # This file
```

---

## Contributing

1. Fork the repository
2. Create a feature branch (`git checkout -b feature/amazing-feature`)
3. Commit your changes (`git commit -m 'Add amazing feature'`)
4. Push to the branch (`git push origin feature/amazing-feature`)
5. Open a Pull Request

---

## License

This project is licensed under the MIT License.

---

## Support

For questions or issues:

-   Review the [ROADMAP.md](ROADMAP.md) for development plans
-   Check [db/migrations/](db/migrations/) for database schema
-   Examine service code in [cmd/](cmd/) and [internal/](internal/)

---

## Acknowledgments

Built with:

-   [Gin](https://github.com/gin-gonic/gin) - HTTP web framework
-   [PostgreSQL](https://www.postgresql.org/) - Database
-   [Redis](https://redis.io/) - Caching and GeoSpatial
-   [Stripe](https://stripe.com/) - Payment processing
-   [Firebase](https://firebase.google.com/) - Push notifications
-   [Twilio](https://www.twilio.com/) - SMS notifications
-   [gorilla/websocket](https://github.com/gorilla/websocket) - WebSocket implementation

---

**Version**: 3.0.0 (Phase 3 Complete)
**Status**: Enterprise Ready 🚀
**Last Updated**: 2025-11-08
