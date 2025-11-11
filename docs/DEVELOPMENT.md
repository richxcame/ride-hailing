# Development Guide

This guide explains how to efficiently develop the Ride Hailing platform without running the entire Docker Compose stack.

## Problem

With 13 microservices, running `docker-compose up` becomes too heavy for active development:

-   Long startup times
-   High resource consumption (CPU, memory, disk I/O)
-   Slow feedback loop when making changes
-   Difficult to debug individual services

## Solution: Hybrid Development Setup

Run **only infrastructure** in Docker, and **run services natively** with `go run`.

---

## Quick Start

### 1. Start Minimal Infrastructure

```bash
# Start only Postgres + Redis (recommended for most development)
make dev-infra

# Or start with full observability stack (Prometheus, Grafana, Tempo)
make dev-infra-full
```

This starts:

-   **Minimal** (`dev-infra`): PostgreSQL + Redis (~200MB RAM)
-   **Full** (`dev-infra-full`): + Prometheus + Grafana + Tempo + OTEL Collector (~500MB RAM)

### 2. Run Database Migrations

```bash
make migrate-up
```

### 3. Optional: Seed the Database

```bash
make db-seed
```

### 4. Run the Service(s) You're Working On

```bash
# Run a single service
make run-auth
make run-rides
make run-geo
# ... etc

# Or run multiple services in separate terminals
```

### 5. Stop Infrastructure When Done

```bash
make dev-stop
```

---

## Development Workflows

### Scenario 1: Working on a Single Service (e.g., Auth)

```bash
# Terminal 1: Start infrastructure
make dev-infra

# Terminal 1 or 2: Run migrations
make migrate-up

# Terminal 2: Run the auth service
make run-auth

# Make code changes, restart service (Ctrl+C and run again)
# Hot reload is fast since it's just one Go process
```

**Benefits:**

-   Fast startup (~2 seconds for Go service)
-   Quick iteration cycle
-   Easy to attach debugger
-   Full console output for debugging

---

### Scenario 2: Working on Multiple Interacting Services

Example: Working on the Rides service which calls Promos service.

```bash
# Terminal 1: Start infrastructure
make dev-infra

# Terminal 2: Run rides service
make run-rides

# Terminal 3: Run promos service (if rides depends on it)
make run-promos

```

---

### Scenario 3: Full Stack Development with Observability

When you need to test distributed tracing, metrics, or monitoring:

```bash
# Start infrastructure + observability stack
make dev-infra-full

# Run migrations
make migrate-up

# Run services you need
make run-auth
make run-rides
# ... etc

# Access monitoring:
# - Grafana: http://localhost:3000 (admin/admin)
# - Prometheus: http://localhost:9090
```

**Benefits:**

-   Full observability stack available
-   Test OpenTelemetry tracing
-   Monitor metrics in real-time
-   Debug performance issues

---

### Scenario 4: Testing with Kong API Gateway

If you need to test API Gateway behavior:

```bash
# Start infrastructure with gateway profile
docker-compose -f docker-compose.dev.yml --profile gateway up -d

# Run your services natively
make run-auth
make run-rides
# ... etc

# Access Kong:
# - Gateway: http://localhost:8000
# - Admin API: http://localhost:8001
# - Manager UI: http://localhost:8002
```

---

## Environment Configuration

### Default Environment Variables

Services use these defaults (localhost-friendly):

```bash
# Database
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=ridehailing
DB_SSLMODE=disable

# Redis
REDIS_HOST=localhost
REDIS_PORT=6379
REDIS_PASSWORD=

# JWT
JWT_SECRET=your-super-secret-jwt-key-change-in-production

# Service Communication
PROMOS_SERVICE_URL=http://localhost:8089
ML_ETA_SERVICE_URL=http://localhost:8093

# OpenTelemetry (optional)
OTEL_ENABLED=false  # Set to true if using observability stack
OTEL_EXPORTER_OTLP_ENDPOINT=localhost:4317
```

### Creating a .env File

Create a [.env](.env) file in the root directory:

```bash
cp .env.example .env
# Edit .env with your local settings
```

Services will automatically load `.env` using [godotenv](https://github.com/joho/godotenv).

---

## Available Make Commands

### Infrastructure

```bash
make dev-infra           # Start Postgres + Redis only
make dev-infra-full      # Start Postgres + Redis + Observability
make dev-stop            # Stop all development infrastructure
```

### Development Workflow

```bash
make dev                 # Start infrastructure + run migrations
make setup               # Initial project setup (install tools, tidy deps)
```

### Database

```bash
make migrate-up          # Run database migrations
make migrate-down        # Rollback last migration
make migrate-create NAME=migration_name  # Create new migration
make db-seed             # Seed database with sample data
make db-reset            # Reset database (rollback, migrate, seed)
```

### Running Services

```bash
make run-auth            # Run auth service
make run-rides           # Run rides service
make run-geo             # Run geo service
make run-payments        # Run payments service
make run-notifications   # Run notifications service
make run-realtime        # Run realtime service
make run-fraud           # Run fraud service
make run-analytics       # Run analytics service
# ... and more (see Makefile)
```

### Building

```bash
make build SERVICE=auth  # Build specific service
make build-all           # Build all services
```

### Testing

```bash
make test                # Run all tests
make test-unit           # Run unit tests only
make test-integration    # Run integration tests
make test-coverage       # Run tests with coverage report
```

### Code Quality

```bash
make lint                # Run golangci-lint
make fmt                 # Format code with gofmt + goimports
make vet                 # Run go vet
make tidy                # Tidy go modules
```

---

## Service Dependencies

Understanding service dependencies helps you know which services to run together:

### Core Services (Independent)

-   **Auth Service** - No dependencies on other services
-   **Geo Service** - No dependencies on other services

### Services with Dependencies

-   **Rides Service** → Promos Service (optional), ML ETA Service (optional)
-   **Payments Service** → No service dependencies (calls Stripe API)
-   **Notifications Service** → No service dependencies (calls FCM, Twilio, SMTP)
-   **Realtime Service** → No service dependencies
-   **Mobile Service** → No service dependencies (aggregates data from DB)
-   **Admin Service** → No service dependencies (aggregates data from DB)
-   **Promos Service** → No service dependencies
-   **Scheduler Service** → No service dependencies
-   **Analytics Service** → No service dependencies
-   **Fraud Service** → No service dependencies
-   **ML ETA Service** → No service dependencies

**Key Insight:** Most services are independent! You can run them individually without dependencies.

---

## Comparison: Full Docker vs Hybrid Development

| Aspect              | Full Docker Compose     | Hybrid Development                  |
| ------------------- | ----------------------- | ----------------------------------- |
| **Startup Time**    | 2-3 minutes             | ~5 seconds per service              |
| **Memory Usage**    | ~4-6 GB                 | ~500 MB (infra) + ~50MB per service |
| **Iteration Speed** | Slow (rebuild image)    | Fast (instant restart)              |
| **Debugging**       | Difficult               | Easy (native debugger)              |
| **Hot Reload**      | Requires volume mounts  | Native Go hot reload                |
| **Resource Usage**  | High CPU/disk I/O       | Low                                 |
| **Best For**        | Production-like testing | Active development                  |

---

## Production Testing

When you need to test the full production-like setup:

```bash
# Use the original docker-compose.yml
make docker-up

# Or start specific services
docker-compose up -d postgres redis auth-service rides-service
```

This is useful for:

-   Integration testing
-   Performance testing
-   Testing Docker networking
-   Testing Kong/Istio configurations
-   Final validation before deployment

---

## Troubleshooting

### Port Already in Use

```bash
# Check what's using port 5432
lsof -i :5432

# Kill the process or use different ports in docker-compose.dev.yml
```

### Cannot Connect to Database

```bash
# Check if Postgres is running
docker-compose -f docker-compose.dev.yml ps

# Check logs
docker-compose -f docker-compose.dev.yml logs postgres

# Wait for health check
docker-compose -f docker-compose.dev.yml ps postgres
```

### Service Can't Connect to Another Service

Make sure both services are running and check the service URL:

```bash
# Example: Rides service calling Promos service
export PROMOS_SERVICE_URL=http://localhost:8089
make run-rides
```

### Database Migrations Fail

```bash
# Check migration version
make migrate-version

# Force to a specific version if needed
make migrate-force VERSION=1

# Re-run migrations
make migrate-up
```

---

## Next Steps

1. Start with `make dev-infra` + `make run-auth` to get familiar
2. Gradually add more services as needed
3. Use `make dev-infra-full` when working on observability
4. Only use full Docker Compose for integration/production testing

---

## Questions?

Check the main [README.md](../README.md) for:

-   API documentation
-   Architecture overview
-   Deployment guides
-   Full feature list
