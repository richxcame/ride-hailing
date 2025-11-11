# Quick Start - Development Mode

**Skip the heavy Docker Compose setup!** Run only what you need.

---

## 🚀 First Time Setup (5 minutes)

```bash
# 1. Start infrastructure (Postgres + Redis)
make dev-infra

# 2. Run migrations
make migrate-up

# 3. Seed database (optional)
make db-seed

# 4. Run the service you're working on
make run-auth  # or run-rides, run-geo, etc.
```

**That's it!** You're ready to develop.

---

## 📋 Daily Development Workflow

### Option 1: Single Service Development

```bash
# Start infrastructure (if not running)
make dev-infra

# Run your service
make run-auth  # Fast startup (~2 seconds)

# Make changes, press Ctrl+C, run again
# No Docker rebuild needed!
```

### Option 2: Multiple Services

```bash
# Start infrastructure
make dev-infra

# Terminal 1
make run-auth

# Terminal 2
make run-rides

# Terminal 3
make run-payments
```

### Option 3: With Observability (Prometheus, Grafana, Tracing)

```bash
# Start full stack
make dev-infra-full

# Run services
make run-auth
make run-rides
# ...

# Visit Grafana: http://localhost:3000 (admin/admin)
```

---

## 🛑 Stop Infrastructure

```bash
make dev-stop
```

---

## 🎯 Common Commands

| Command               | What it does                                       |
| --------------------- | -------------------------------------------------- |
| `make dev-infra`      | Start Postgres + Redis                             |
| `make dev-infra-full` | Start + Observability (Grafana, Prometheus, Tempo) |
| `make dev-stop`       | Stop all infrastructure                            |
| `make migrate-up`     | Run database migrations                            |
| `make db-seed`        | Seed database                                      |
| `make run-auth`       | Run auth service natively                          |
| `make run-rides`      | Run rides service natively                         |
| `make test`           | Run all tests                                      |
| `make lint`           | Run linter                                         |
| `make fmt`            | Format code                                        |

---

## 🔌 Available Services

Run any service with `make run-<service>`:

| Service       | Command                  | Port |
| ------------- | ------------------------ | ---- |
| Auth          | `make run-auth`          | 8081 |
| Rides         | `make run-rides`         | 8082 |
| Geo           | `make run-geo`           | 8083 |
| Payments      | `make run-payments`      | 8084 |
| Notifications | `make run-notifications` | 8085 |
| Realtime      | `make run-realtime`      | 8086 |
| Mobile        | `make run-mobile`        | 8087 |
| Admin         | `make run-admin`         | 8088 |
| Promos        | `make run-promos`        | 8089 |
| Scheduler     | `make run-scheduler`     | 8090 |
| Analytics     | `make run-analytics`     | 8091 |
| Fraud         | `make run-fraud`         | 8092 |
| ML ETA        | `make run-ml-eta`        | 8093 |

---

## 💡 Pro Tips

### Use tmux/tmuxinator for Multiple Services

```bash
# Install tmuxinator
gem install tmuxinator

# Start all services in tmux session
tmuxinator start ridehailing

# Detach: Ctrl+B, then D
# Reattach: tmuxinator start ridehailing
```

### Hot Reload with Air

```bash
# Install Air
go install github.com/air-verse/air@latest

# Run with hot reload
cd cmd/auth
air
```

---

## 🔧 Troubleshooting

### Port Already in Use

```bash
# Find what's using port 5432
lsof -i :5432

# Kill it
kill -9 <PID>
```

### Can't Connect to Database

```bash
# Check if Postgres is running
docker ps | grep postgres

# Check logs
docker-compose -f docker-compose.dev.yml logs postgres

# Restart infrastructure
make dev-stop && make dev-infra
```

### Service Dependencies

Most services are **independent**! Exception:

-   Rides Service → can call Promos Service (optional)
-   Rides Service → can call ML ETA Service (optional)

Just run both if needed:

```bash
# Terminal 1
make run-promos

# Terminal 2
make run-rides
```

---

## 📊 Infrastructure Only

What runs in Docker:

-   ✅ PostgreSQL (port 5432)
-   ✅ Redis (port 6379)
-   ✅ Prometheus (port 9090) - if using `dev-infra-full`
-   ✅ Grafana (port 3000) - if using `dev-infra-full`
-   ✅ Tempo (port 3200) - if using `dev-infra-full`
-   ✅ OTEL Collector (ports 4317, 4318) - if using `dev-infra-full`

What runs natively:

-   ✅ All Go microservices (auth, rides, geo, payments, etc.)

---

## 🎬 Full Production Test

Need to test everything together like production?

```bash
# Use original docker-compose
make docker-up

# This starts all 13 services + infrastructure
```

**When to use:**

-   Integration testing
-   Performance testing
-   Production-like validation
-   Testing Kong API Gateway
-   Testing Istio Service Mesh

---

## 📖 More Information

-   [docs/DEVELOPMENT.md](docs/DEVELOPMENT.md) - Full development guide
-   [README.md](README.md) - Architecture and features
-   [docs/API.md](docs/API.md) - API documentation
-   [Makefile](Makefile) - All available commands

---

## ⚡ Performance Comparison

| Setup           | Memory  | Startup Time | Iteration Speed        |
| --------------- | ------- | ------------ | ---------------------- |
| **Full Docker** | ~4-6 GB | 2-3 minutes  | Slow (rebuild image)   |
| **Hybrid Dev**  | ~500 MB | ~5 seconds   | Fast (instant restart) |

**Verdict:** Hybrid development is **10-50x faster** for daily work!

---

**Happy coding!** 🎉
