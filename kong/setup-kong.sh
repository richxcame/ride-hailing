#!/bin/bash

# Kong API Gateway Setup Script
# This script configures Kong with all microservices, rate limiting, and authentication

set -e

KONG_ADMIN_URL="${KONG_ADMIN_URL:-http://localhost:8001}"

echo "🚀 Setting up Kong API Gateway..."
echo "Kong Admin URL: $KONG_ADMIN_URL"
echo ""

# Wait for Kong to be ready
echo "⏳ Waiting for Kong to be ready..."
until curl -s "${KONG_ADMIN_URL}/status" > /dev/null 2>&1; do
  echo "   Waiting for Kong..."
  sleep 2
done
echo "✅ Kong is ready!"
echo ""

# Function to create service
create_service() {
  local service_name=$1
  local service_url=$2

  echo "📦 Creating service: $service_name"
  curl -i -X POST "${KONG_ADMIN_URL}/services" \
    --data "name=${service_name}" \
    --data "url=${service_url}" \
    --silent --show-error || echo "   Service might already exist"
}

# Function to create route
create_route() {
  local service_name=$1
  local route_path=$2
  local route_name=$3

  echo "🛣️  Creating route: $route_name for $service_name"
  curl -i -X POST "${KONG_ADMIN_URL}/services/${service_name}/routes" \
    --data "name=${route_name}" \
    --data "paths[]=${route_path}" \
    --data "strip_path=false" \
    --silent --show-error || echo "   Route might already exist"
}

# Function to add rate limiting plugin
add_rate_limiting() {
  local service_name=$1
  local rate_limit=${2:-1000}  # Default 1000 requests
  local window=${3:-minute}     # Default per minute

  echo "⏱️  Adding rate limiting to $service_name: $rate_limit/$window"
  curl -i -X POST "${KONG_ADMIN_URL}/services/${service_name}/plugins" \
    --data "name=rate-limiting" \
    --data "config.minute=${rate_limit}" \
    --data "config.policy=local" \
    --silent --show-error || echo "   Plugin might already exist"
}

# Function to add JWT plugin
add_jwt_auth() {
  local service_name=$1

  echo "🔐 Adding JWT authentication to $service_name"
  curl -i -X POST "${KONG_ADMIN_URL}/services/${service_name}/plugins" \
    --data "name=jwt" \
    --silent --show-error || echo "   Plugin might already exist"
}

# Function to add CORS plugin
add_cors() {
  local service_name=$1

  echo "🌐 Adding CORS to $service_name"
  curl -i -X POST "${KONG_ADMIN_URL}/services/${service_name}/plugins" \
    --data "name=cors" \
    --data "config.origins=*" \
    --data "config.methods=GET,HEAD,PUT,PATCH,POST,DELETE,OPTIONS" \
    --data "config.headers=Accept,Authorization,Content-Type,Origin,X-Requested-With" \
    --data "config.exposed_headers=X-RateLimit-Limit,X-RateLimit-Remaining" \
    --data "config.credentials=true" \
    --data "config.max_age=3600" \
    --silent --show-error || echo "   Plugin might already exist"
}

# Function to add request transformer
add_request_transformer() {
  local service_name=$1

  echo "🔄 Adding request transformer to $service_name"
  curl -i -X POST "${KONG_ADMIN_URL}/services/${service_name}/plugins" \
    --data "name=request-transformer" \
    --data "config.add.headers[]=X-Gateway-Version:3.0" \
    --silent --show-error || echo "   Plugin might already exist"
}

# Function to add prometheus plugin
add_prometheus() {
  echo "📊 Adding Prometheus metrics plugin (global)"
  curl -i -X POST "${KONG_ADMIN_URL}/plugins" \
    --data "name=prometheus" \
    --silent --show-error || echo "   Plugin might already exist"
}

echo "═══════════════════════════════════════════════════════════"
echo "Setting up Services and Routes"
echo "═══════════════════════════════════════════════════════════"
echo ""

# 1. Auth Service
create_service "auth-service" "http://auth-service:8080"
create_route "auth-service" "/api/v1/auth" "auth-route"
add_rate_limiting "auth-service" 100 "minute"
add_cors "auth-service"
add_request_transformer "auth-service"

echo ""

# 2. Rides Service
create_service "rides-service" "http://rides-service:8080"
create_route "rides-service" "/api/v1/rides" "rides-route"
add_rate_limiting "rides-service" 1000 "minute"
add_jwt_auth "rides-service"
add_cors "rides-service"
add_request_transformer "rides-service"

echo ""

# 3. Geo Service
create_service "geo-service" "http://geo-service:8080"
create_route "geo-service" "/api/v1/geo" "geo-route"
add_rate_limiting "geo-service" 2000 "minute"  # Higher limit for location updates
add_jwt_auth "geo-service"
add_cors "geo-service"
add_request_transformer "geo-service"

echo ""

# 4. Payments Service
create_service "payments-service" "http://payments-service:8080"
create_route "payments-service" "/api/v1/payments" "payments-route"
create_route "payments-service" "/api/v1/wallet" "wallet-route"
add_rate_limiting "payments-service" 500 "minute"
add_jwt_auth "payments-service"
add_cors "payments-service"
add_request_transformer "payments-service"

echo ""

# 5. Notifications Service
create_service "notifications-service" "http://notifications-service:8080"
create_route "notifications-service" "/api/v1/notifications" "notifications-route"
add_rate_limiting "notifications-service" 500 "minute"
add_jwt_auth "notifications-service"
add_cors "notifications-service"
add_request_transformer "notifications-service"

echo ""

# 6. Real-time Service (WebSocket)
create_service "realtime-service" "http://realtime-service:8080"
create_route "realtime-service" "/ws" "websocket-route"
add_rate_limiting "realtime-service" 100 "minute"  # Lower limit for WebSocket connections
add_cors "realtime-service"
add_request_transformer "realtime-service"

echo ""

# 7. Mobile Service
create_service "mobile-service" "http://mobile-service:8080"
create_route "mobile-service" "/api/v1/mobile" "mobile-route"
add_rate_limiting "mobile-service" 1000 "minute"
add_jwt_auth "mobile-service"
add_cors "mobile-service"
add_request_transformer "mobile-service"

echo ""

# 8. Admin Service
create_service "admin-service" "http://admin-service:8080"
create_route "admin-service" "/api/v1/admin" "admin-route"
add_rate_limiting "admin-service" 200 "minute"  # Lower limit for admin
add_jwt_auth "admin-service"
add_cors "admin-service"
add_request_transformer "admin-service"

echo ""

# 9. Promos Service
create_service "promos-service" "http://promos-service:8080"
create_route "promos-service" "/api/v1/promos" "promos-route"
add_rate_limiting "promos-service" 500 "minute"
add_jwt_auth "promos-service"
add_cors "promos-service"
add_request_transformer "promos-service"

echo ""

# 10. Scheduler Service
create_service "scheduler-service" "http://scheduler-service:8080"
create_route "scheduler-service" "/api/v1/scheduler" "scheduler-route"
add_rate_limiting "scheduler-service" 200 "minute"
add_jwt_auth "scheduler-service"
add_cors "scheduler-service"
add_request_transformer "scheduler-service"

echo ""

# 11. Analytics Service
create_service "analytics-service" "http://analytics-service:8080"
create_route "analytics-service" "/api/v1/analytics" "analytics-route"
add_rate_limiting "analytics-service" 300 "minute"
add_jwt_auth "analytics-service"
add_cors "analytics-service"
add_request_transformer "analytics-service"

echo ""

# 12. Fraud Service
create_service "fraud-service" "http://fraud-service:8080"
create_route "fraud-service" "/api/v1/fraud" "fraud-route"
add_rate_limiting "fraud-service" 500 "minute"
add_jwt_auth "fraud-service"
add_cors "fraud-service"
add_request_transformer "fraud-service"

echo ""

# 13. ML ETA Service
create_service "ml-eta-service" "http://ml-eta-service:8080"
create_route "ml-eta-service" "/api/v1/eta" "ml-eta-route"
add_rate_limiting "ml-eta-service" 1000 "minute"
add_jwt_auth "ml-eta-service"
add_cors "ml-eta-service"
add_request_transformer "ml-eta-service"

echo ""

# Add global Prometheus metrics
add_prometheus

echo ""
echo "═══════════════════════════════════════════════════════════"
echo "✅ Kong API Gateway setup complete!"
echo "═══════════════════════════════════════════════════════════"
echo ""
echo "📌 Access Points:"
echo "   • Kong Proxy:       http://localhost:8000"
echo "   • Kong Admin API:   http://localhost:8001"
echo "   • Konga Admin UI:   http://localhost:1337"
echo "   • Prometheus:       http://localhost:8000/metrics"
echo ""
echo "📝 Next Steps:"
echo "   1. Access Konga at http://localhost:1337 and create an admin account"
echo "   2. Connect Konga to Kong Admin API: http://kong:8001"
echo "   3. Test API access through Kong: http://localhost:8000/api/v1/auth/healthz"
echo ""
echo "🔐 Rate Limits Configured:"
echo "   • Auth Service:         100/min"
echo "   • Rides Service:        1000/min"
echo "   • Geo Service:          2000/min"
echo "   • Payments Service:     500/min"
echo "   • Notifications:        500/min"
echo "   • Real-time (WS):       100/min"
echo "   • Mobile Service:       1000/min"
echo "   • Admin Service:        200/min"
echo "   • Promos Service:       500/min"
echo "   • Scheduler Service:    200/min"
echo "   • Analytics Service:    300/min"
echo "   • Fraud Service:        500/min"
echo "   • ML ETA Service:       1000/min"
echo ""
