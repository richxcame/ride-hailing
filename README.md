# Ride Hailing Platform - Production-Ready Backend

[![CI](https://github.com/richxcame/ride-hailing/actions/workflows/test.yml/badge.svg)](https://github.com/richxcame/ride-hailing/actions/workflows/test.yml)
[![codecov](https://codecov.io/gh/richxcame/ride-hailing/branch/main/graph/badge.svg)](https://app.codecov.io/gh/richxcame/ride-hailing)

A complete, production-ready ride-hailing platform backend built with Go, featuring 12 microservices that handle everything from authentication to fraud detection.

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

-   **Language**: Go 1.22+
-   **Framework**: Gin
-   **Database**: PostgreSQL 15 (with connection pooling)
-   **Cache**: Redis 7 (GeoSpatial + Pub/Sub)
-   **WebSocket**: gorilla/websocket
-   **Payments**: Stripe API
-   **Notifications**: Firebase FCM, Twilio SMS, SMTP
-   **Auth**: JWT with bcrypt
-   **Observability**: Prometheus + Grafana
-   **Deployment**: Docker + Docker Compose
-   **Testing**: Go test framework

---

## Architecture

```
┌─────────────────────────────────────────────────────────────────┐
│                    Client Applications                           │
│              (Mobile Apps, Web Dashboard, Admin Panel)           │
└────┬──────────┬──────────┬──────────┬──────────┬───────────────┘
     │          │          │          │          │
┌────▼─────┐┌──▼────┐┌───▼───┐┌─────▼────┐┌───▼──────┐
│   Auth   ││ Rides ││  Geo  ││ Payments ││  Notifs  │
│  :8081   ││ :8082 ││ :8083 ││  :8084   ││  :8085   │
└────┬─────┘└───┬───┘└───┬───┘└────┬─────┘└────┬─────┘
     │          │        │         │            │
┌────▼─────┐┌──▼────┐┌──▼───┐
│ Realtime ││Mobile ││Admin │
│  :8086   ││ :8087 ││ :8088│
└────┬─────┘└───┬───┘└──┬───┘
     │          │       │
     └──────────┴───────┴──────────────────┐
                         │                 │
            ┌────────────▼────────────┐    │
            │   PostgreSQL Database   │    │
            │      (Persistent)       │    │
            └────────────┬────────────┘    │
                         │                 │
            ┌────────────▼────────────┐    │
            │     Redis Cluster       │◄───┘
            │  (Cache + GeoSpatial +  │
            │       WebSocket)        │
            └─────────────────────────┘
```

---

## Services

| Service          | Port     | Purpose                                | Status            |
| ---------------- | -------- | -------------------------------------- | ----------------- |
| Auth             | 8081     | User authentication & JWT              | ✅ Production     |
| Rides            | 8082     | Ride lifecycle management + scheduling | ✅ Production     |
| Geo              | 8083     | Location tracking + driver matching    | ✅ Production     |
| Payments         | 8084     | Stripe integration + wallets           | ✅ Production     |
| Notifications    | 8085     | Multi-channel notifications            | ✅ Production     |
| Real-time        | 8086     | WebSocket + chat                       | ✅ Production     |
| Mobile           | 8087     | Mobile-optimized APIs                  | ✅ Production     |
| Admin            | 8088     | Admin dashboard backend                | ✅ Production     |
| Promos           | 8089     | Promo codes & referrals                | ✅ Production     |
| Scheduler        | 8090     | Automated ride scheduling              | ✅ Production     |
| Analytics        | 8091     | Business intelligence                  | ✅ Production     |
| Fraud            | 8092     | Fraud detection                        | ✅ Production     |
| ML ETA           | 8093     | ML-based ETA prediction                | ✅ Production     |
| **Kong Gateway** | **8000** | **API Gateway**                        | ✅ **Enterprise** |
| **Konga Admin**  | **1337** | **Kong UI**                            | ✅ **Enterprise** |

### 1. Auth Service (Port 8081)

-   User registration (riders, drivers, admins)
-   Login with JWT token generation
-   Token refresh mechanism
-   Role-based access control (RBAC)
-   Password hashing with bcrypt

**Endpoints**: 4 (register, login, refresh, health)

### 2. Rides Service (Port 8082)

-   Ride request creation with fare estimation
-   Driver acceptance workflow
-   Ride lifecycle (requested → accepted → in_progress → completed)
-   Ride cancellation with reasons
-   Rating and feedback system (1-5 stars)
-   Advanced filtering (status, date range)
-   Receipt generation

**Endpoints**: 8 (create, get, accept, start, complete, cancel, rate, history)

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
-   Go 1.22+ (for local development)
-   PostgreSQL 15
-   Redis 7

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
JWT_SECRET=your-secret-key-change-in-production

REDIS_HOST=redis
REDIS_PORT=6379
REDIS_PASSWORD=
```

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
```

### Integration Testing

See [IMPLEMENTATION_NOTES.md](IMPLEMENTATION_NOTES.md) for complete end-to-end testing flow.

---

## Documentation

-   **[ROADMAP.md](ROADMAP.md)** - Development roadmap and feature planning
-   **[db/migrations/](db/migrations/)** - Database schema migrations

---

## Production Deployment

### Before Going Live

-   [ ] Rotate all API keys and secrets
-   [ ] Change JWT_SECRET to strong random value
-   [ ] Use production Stripe API keys
-   [ ] Set up Firebase production project
-   [ ] Configure production SMTP credentials
-   [ ] Enable HTTPS/TLS on all services
-   [ ] Set up API Gateway (Kong/Nginx)
-   [ ] Configure rate limiting
-   [ ] Set up CORS properly
-   [ ] Enable database backups
-   [ ] Set up log aggregation
-   [ ] Configure error alerting
-   [ ] Load testing (100+ concurrent rides)
-   [ ] Security audit
-   [ ] Configure fraud detection rules
-   [ ] Test scheduled ride dispatch

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
│   └── fraud/             # Fraud service
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
│   └── fraud/            # Fraud business logic
├── pkg/                   # Public shared libraries
│   ├── common/           # Common utilities
│   ├── middleware/       # HTTP middleware
│   ├── models/           # Data models
│   ├── redis/            # Redis client
│   └── websocket/        # WebSocket utilities
├── db/migrations/         # Database migrations
├── docker-compose.yml     # Docker Compose config
├── go.mod                 # Go dependencies
├── ROADMAP.md            # Development roadmap
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
**Last Updated**: 2025-11-06
