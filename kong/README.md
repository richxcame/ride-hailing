# Kong API Gateway - Configuration Guide

This directory contains the Kong API Gateway configuration for the Ride Hailing platform.

## Overview

Kong serves as the centralized API Gateway for all 12 microservices, providing:

-   **Unified API Entry Point** - Single endpoint for all services
-   **Rate Limiting** - Protect services from abuse
-   **Authentication** - JWT validation at gateway level
-   **CORS Handling** - Proper cross-origin request handling
-   **Monitoring** - Prometheus metrics integration
-   **Request Transformation** - Add headers and modify requests
-   **Load Balancing** - Distribute traffic across service instances

## Architecture

```
┌─────────────────────────────────────────────────────────┐
│                   Client Applications                    │
│         (Mobile Apps, Web Dashboard, Admin Panel)        │
└────────────────────────┬────────────────────────────────┘
                         │
                         ▼
              ┌──────────────────────┐
              │   Kong API Gateway   │
              │    (Port 8000)       │
              │                      │
              │  • Rate Limiting     │
              │  • JWT Auth          │
              │  • CORS              │
              │  • Metrics           │
              └──────────┬───────────┘
                         │
         ┌───────────────┼───────────────┐
         │               │               │
    ┌────▼────┐    ┌────▼────┐    ┌────▼────┐
    │  Auth   │    │  Rides  │    │   Geo   │
    │  :8080  │    │  :8080  │    │  :8080  │
    └─────────┘    └─────────┘    └─────────┘
         │               │               │
         └───────────────┴───────────────┘
                         │
              ┌──────────▼───────────┐
              │   Internal Services  │
              │  (8 more services)   │
              └──────────────────────┘
```

## Quick Start

### 1. Start Kong

Kong is automatically started with docker-compose:

```bash
docker-compose up -d kong kong-database konga
```

### 2. Configure Kong

Run the setup script to configure all services:

```bash
./kong/setup-kong.sh
```

Or run it from Docker:

```bash
docker-compose exec kong sh -c "cd /tmp && curl -O http://host.docker.internal:8000/setup-kong.sh && chmod +x setup-kong.sh && ./setup-kong.sh"
```

### 3. Verify Setup

Check Kong status:

```bash
curl http://localhost:8001/status
```

List all configured services:

```bash
curl http://localhost:8001/services
```

List all routes:

```bash
curl http://localhost:8001/routes
```

## Access Points

| Service            | URL                           | Purpose                       |
| ------------------ | ----------------------------- | ----------------------------- |
| Kong Proxy         | http://localhost:8000         | Main API gateway endpoint     |
| Kong Admin API     | http://localhost:8001         | Kong management API           |
| Konga UI           | http://localhost:1337         | Web-based Kong administration |
| Prometheus Metrics | http://localhost:8000/metrics | Gateway metrics               |

## Service Configuration

All 12 microservices are configured with the following plugins:

### Rate Limiting

Each service has custom rate limits based on expected load:

| Service        | Rate Limit | Reason                    |
| -------------- | ---------- | ------------------------- |
| Auth           | 100/min    | Login attempts, security  |
| Rides          | 1000/min   | High volume ride requests |
| Geo            | 2000/min   | Frequent location updates |
| Payments       | 500/min    | Payment processing        |
| Notifications  | 500/min    | Message delivery          |
| Real-time (WS) | 100/min    | WebSocket connections     |
| Mobile         | 1000/min   | Mobile app requests       |
| Admin          | 200/min    | Admin operations          |
| Promos         | 500/min    | Promo code validation     |
| Scheduler      | 200/min    | Scheduled rides           |
| Analytics      | 300/min    | Report generation         |
| Fraud          | 500/min    | Fraud checks              |

### Authentication

-   **Auth Service**: No JWT required (handles login)
-   **All Other Services**: JWT authentication enforced

### CORS

All services have CORS enabled with:

-   Allowed Origins: `*` (configure for production)
-   Allowed Methods: `GET, POST, PUT, PATCH, DELETE, OPTIONS`
-   Allowed Headers: `Accept, Authorization, Content-Type, Origin, X-Requested-With`
-   Exposed Headers: `X-RateLimit-Limit, X-RateLimit-Remaining`
-   Credentials: Enabled
-   Max Age: 3600 seconds

## Routes

All services are accessible through Kong at `http://localhost:8000`:

| Service       | Route                     | Backend                             |
| ------------- | ------------------------- | ----------------------------------- |
| Auth          | `/api/v1/auth/*`          | `http://auth-service:8080`          |
| Rides         | `/api/v1/rides/*`         | `http://rides-service:8080`         |
| Geo           | `/api/v1/geo/*`           | `http://geo-service:8080`           |
| Payments      | `/api/v1/payments/*`      | `http://payments-service:8080`      |
| Wallet        | `/api/v1/wallet/*`        | `http://payments-service:8080`      |
| Notifications | `/api/v1/notifications/*` | `http://notifications-service:8080` |
| Real-time     | `/ws`                     | `http://realtime-service:8080`      |
| Mobile        | `/api/v1/mobile/*`        | `http://mobile-service:8080`        |
| Admin         | `/api/v1/admin/*`         | `http://admin-service:8080`         |
| Promos        | `/api/v1/promos/*`        | `http://promos-service:8080`        |
| Scheduler     | `/api/v1/scheduler/*`     | `http://scheduler-service:8080`     |
| Analytics     | `/api/v1/analytics/*`     | `http://analytics-service:8080`     |
| Fraud         | `/api/v1/fraud/*`         | `http://fraud-service:8080`         |

## Usage Examples

### Without Kong (Direct Service Access)

```bash
# Before - Access services directly
curl http://localhost:8081/api/v1/auth/healthz
curl http://localhost:8082/api/v1/rides
```

### With Kong (Through Gateway)

```bash
# After - All requests go through Kong
curl http://localhost:8000/api/v1/auth/healthz
curl http://localhost:8000/api/v1/rides \
  -H "Authorization: Bearer YOUR_JWT_TOKEN"
```

### Example: Login through Kong

```bash
# Login
TOKEN=$(curl -X POST http://localhost:8000/api/v1/auth/login \
  -H "Content-Type: application/json" \
  -d '{
    "email": "rider@test.com",
    "password": "password123"
  }' | jq -r '.token')

# Use token for authenticated requests
curl http://localhost:8000/api/v1/rides \
  -H "Authorization: Bearer $TOKEN"
```

### Example: Check Rate Limits

```bash
# Make a request and check rate limit headers
curl -I http://localhost:8000/api/v1/auth/healthz

# Response headers include:
# X-RateLimit-Limit-Minute: 100
# X-RateLimit-Remaining-Minute: 99
```

## Konga Administration

### First Time Setup

1. Access Konga at http://localhost:1337
2. Create an admin account
3. Add Kong connection:
    - Name: `Ride Hailing Kong`
    - Kong Admin URL: `http://kong:8001`

### Managing Services

Through Konga UI, you can:

-   View all services and routes
-   Configure plugins
-   Monitor traffic
-   Adjust rate limits
-   Manage consumers
-   View logs

## Plugins

### 1. Rate Limiting

Protects services from abuse and ensures fair usage.

**Configuration:**

```json
{
	"name": "rate-limiting",
	"config": {
		"minute": 1000,
		"policy": "local"
	}
}
```

### 2. JWT Authentication

Validates JWT tokens for protected endpoints.

**Configuration:**

```json
{
	"name": "jwt"
}
```

**JWT Header Format:**

```
Authorization: Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...
```

### 3. CORS

Handles cross-origin requests for web clients.

**Configuration:**

```json
{
	"name": "cors",
	"config": {
		"origins": ["*"],
		"methods": ["GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"],
		"credentials": true
	}
}
```

### 4. Request Transformer

Adds custom headers to all requests.

**Configuration:**

```json
{
	"name": "request-transformer",
	"config": {
		"add": {
			"headers": ["X-Gateway-Version:3.0"]
		}
	}
}
```

### 5. Prometheus

Exports metrics for monitoring.

**Metrics Endpoint:** http://localhost:8000/metrics

**Available Metrics:**

-   `kong_http_requests_total` - Total HTTP requests
-   `kong_latency` - Request latency
-   `kong_bandwidth` - Bandwidth usage

## Advanced Configuration

### Custom Rate Limits

Add custom rate limits for specific consumers:

```bash
# Create a consumer
curl -X POST http://localhost:8001/consumers \
  --data "username=premium-user"

# Add custom rate limit
curl -X POST http://localhost:8001/consumers/premium-user/plugins \
  --data "name=rate-limiting" \
  --data "config.minute=5000"
```

### Load Balancing

Add multiple upstream targets for a service:

```bash
# Create upstream
curl -X POST http://localhost:8001/upstreams \
  --data "name=auth-upstream"

# Add targets
curl -X POST http://localhost:8001/upstreams/auth-upstream/targets \
  --data "target=auth-service-1:8080"
curl -X POST http://localhost:8001/upstreams/auth-upstream/targets \
  --data "target=auth-service-2:8080"

# Update service to use upstream
curl -X PATCH http://localhost:8001/services/auth-service \
  --data "host=auth-upstream"
```

### API Versioning

Route different API versions to different services:

```bash
# V1 Route
curl -X POST http://localhost:8001/services/rides-service/routes \
  --data "name=rides-v1" \
  --data "paths[]=/api/v1/rides"

# V2 Route
curl -X POST http://localhost:8001/services/rides-service-v2/routes \
  --data "name=rides-v2" \
  --data "paths[]=/api/v2/rides"
```

## Production Considerations

### 1. Security

**Change default credentials:**

```yaml
# docker-compose.yml
kong-database:
    environment:
        POSTGRES_PASSWORD: ${KONG_DB_PASSWORD} # Use secrets
```

**Enable HTTPS:**

```yaml
kong:
    environment:
        KONG_SSL_CERT: /etc/kong/ssl/cert.pem
        KONG_SSL_CERT_KEY: /etc/kong/ssl/key.pem
```

**Restrict Admin API:**

```yaml
kong:
    environment:
        KONG_ADMIN_LISTEN: '127.0.0.1:8001' # Localhost only
```

### 2. Rate Limiting

**Use Redis for distributed rate limiting:**

```bash
curl -X PATCH http://localhost:8001/plugins/{plugin-id} \
  --data "config.policy=redis" \
  --data "config.redis_host=redis" \
  --data "config.redis_port=6379"
```

### 3. Monitoring

**Set up Prometheus scraping:**

```yaml
# prometheus.yml
scrape_configs:
    - job_name: 'kong'
      static_configs:
          - targets: ['kong:8000']
```

**Create Grafana dashboards:**

-   Import Kong dashboard: https://grafana.com/grafana/dashboards/7424

### 4. High Availability

**Run multiple Kong instances:**

```yaml
# docker-compose.yml
kong-1:
    image: kong:3.9.1
    # ... config

kong-2:
    image: kong:3.9.1
    # ... config
```

**Add load balancer (Nginx):**

```nginx
upstream kong {
  server kong-1:8000;
  server kong-2:8000;
}
```

## Troubleshooting

### Kong Won't Start

```bash
# Check database connection
docker-compose logs kong-database

# Check Kong logs
docker-compose logs kong

# Verify migration
docker-compose logs kong-migration
```

### Service Not Reachable

```bash
# Check service exists
curl http://localhost:8001/services/auth-service

# Check route exists
curl http://localhost:8001/services/auth-service/routes

# Test backend directly
docker-compose exec kong curl http://auth-service:8080/healthz
```

### Rate Limit Issues

```bash
# Check plugin configuration
curl http://localhost:8001/services/auth-service/plugins

# View rate limit counters
curl http://localhost:8001/rate-limiting/consumers/{consumer-id}
```

### JWT Authentication Failing

```bash
# Verify JWT plugin is enabled
curl http://localhost:8001/services/rides-service/plugins

# Check JWT format
echo "YOUR_TOKEN" | base64 -d

# Test without Kong
curl http://localhost:8082/api/v1/rides \
  -H "Authorization: Bearer YOUR_TOKEN"
```

## Monitoring

### Health Checks

```bash
# Kong health
curl http://localhost:8001/status

# All services health
for port in {8081..8092}; do
  echo "Port $port: $(curl -s http://localhost:8000/healthz)"
done
```

### Metrics

```bash
# Get Prometheus metrics
curl http://localhost:8000/metrics

# Filter specific metrics
curl http://localhost:8000/metrics | grep kong_http_requests_total
```

### Logs

```bash
# Kong access logs
docker-compose logs -f kong | grep "proxy_access"

# Kong error logs
docker-compose logs -f kong | grep "proxy_error"
```

## Cleanup

```bash
# Remove all Kong configuration
curl -X DELETE http://localhost:8001/services/auth-service
curl -X DELETE http://localhost:8001/services/rides-service
# ... (repeat for all services)

# Or restart Kong with fresh database
docker-compose down
docker volume rm ride-hailing_kong_data
docker-compose up -d
```

## References

-   [Kong Documentation](https://docs.konghq.com/)
-   [Kong Plugins Hub](https://docs.konghq.com/hub/)
-   [Konga Documentation](https://github.com/pantsel/konga)
-   [Kong Ingress Controller](https://docs.konghq.com/kubernetes-ingress-controller/)

## Next Steps

1. ✅ Kong API Gateway configured
2. ⏭️ Set up Kubernetes deployment
3. ⏭️ Implement service mesh (Istio)
4. ⏭️ Configure auto-scaling
5. ⏭️ Add ML-based features

---

**Last Updated:** 2025-11-06
**Kong Version:** 3.4
**Phase:** 3 - Enterprise Ready
