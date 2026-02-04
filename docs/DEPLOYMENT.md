# Deployment Guide

## Docker Compose

Three compose files are provided:

| File | Purpose |
| --- | --- |
| `docker-compose.yml` | Production |
| `docker-compose.dev.yml` | Local development (hot-reload, debug ports) |
| `docker-compose.test.yml` | CI / integration tests |

### Start all services

```bash
cp .env.example .env   # edit values for your environment
docker-compose up -d --build
```

### Run migrations

```bash
docker exec ridehailing-auth migrate -path /app/db/migrations \
  -database "postgresql://postgres:postgres@postgres:5432/ridehailing?sslmode=disable" up
```

### Useful commands

```bash
docker-compose logs -f              # tail all logs
docker-compose logs -f auth-service # single service
docker-compose down                 # stop everything
```

### Build a single service image

```bash
docker build --build-arg SERVICE_NAME=auth -t ridehailing-auth:latest .

docker run -d --name ridehailing-auth \
  -p 8081:8080 \
  -e DB_HOST=postgres \
  -e DB_PASSWORD=postgres \
  ridehailing-auth:latest
```

## Kubernetes

All manifests live in `k8s/`. Istio configs are in `k8s/istio/`. See `k8s/README.md` for details.

### Quick start

```bash
kubectl apply -f k8s/namespace.yaml
kubectl apply -f k8s/configmap.yaml

# Create secrets (or use k8s/secrets.yaml as a template)
kubectl create secret generic ridehailing-secrets \
  --from-literal=db-password=your-password \
  --from-literal=jwt-secret=your-jwt-secret \
  -n ridehailing

# Infrastructure + all services + ingress
kubectl apply -f k8s/postgres.yaml -f k8s/redis.yaml -f k8s/nats.yaml
kubectl apply -f k8s/auth-service.yaml -f k8s/rides-service.yaml \
  -f k8s/geo-service.yaml -f k8s/realtime-service.yaml \
  -f k8s/payments-service.yaml -f k8s/notifications-service.yaml \
  -f k8s/analytics-service.yaml -f k8s/admin-service.yaml \
  -f k8s/mobile-service.yaml -f k8s/fraud-service.yaml \
  -f k8s/ml-eta-service.yaml -f k8s/promos-service.yaml \
  -f k8s/scheduler-service.yaml
kubectl apply -f k8s/ingress.yaml
```

`k8s/generate-services.sh` can scaffold new service manifests.

### Istio (optional)

```bash
bash k8s/istio/install-istio.sh
kubectl apply -f k8s/istio/gateway.yaml
kubectl apply -f k8s/istio/destination-rules.yaml
kubectl apply -f k8s/istio/security-policies.yaml
```

## Environment Variables

All variables are documented in `.env.example`. Required variables (no usable default):

| Variable | Purpose |
| --- | --- |
| `DB_HOST`, `DB_PORT`, `DB_USER`, `DB_PASSWORD`, `DB_NAME` | PostgreSQL connection |
| `REDIS_HOST`, `REDIS_PORT` | Redis connection |
| `JWT_SECRET` | Auth token signing key |
| `NATS_URL` | Event bus |
| `INTERNAL_API_KEY` | Inter-service auth |
| `STRIPE_API_KEY` | Payment processing |

Service URLs for inter-service communication (set automatically in Docker/K8s networking, needed for bare-metal):

`PROMOS_SERVICE_URL`, `ML_ETA_SERVICE_URL`, `GEO_SERVICE_URL`, `REALTIME_SERVICE_URL`, `NOTIFICATIONS_SERVICE_URL`

Optional integrations (Firebase, Twilio, SMTP, Sentry, OTel, Google Maps, Checkr, Onfido, Pub/Sub) -- see `.env.example`.

## Production Checklist

- [ ] Set `ENVIRONMENT=production`
- [ ] Use strong, unique `JWT_SECRET` and `INTERNAL_API_KEY`
- [ ] Change all default database passwords
- [ ] Enable `DB_SSLMODE=require` (or `verify-full`)
- [ ] Set `REDIS_PASSWORD`
- [ ] Configure `CORS_ORIGINS` to your actual domains
- [ ] Enable rate limiting (`RATE_LIMIT_ENABLED=true`) and tune limits
- [ ] Enable circuit breakers (`CB_ENABLED=true`)
- [ ] Configure Sentry (`SENTRY_DSN`) and lower sample rates for high-traffic
- [ ] Configure OpenTelemetry exporter endpoint
- [ ] Set up secrets management (`SECRETS_PROVIDER=vault` or cloud equivalent)
- [ ] Enable NATS (`NATS_ENABLED=true`) with persistent JetStream storage
- [ ] Review and set timeout values (`DB_QUERY_TIMEOUT`, `HTTP_CLIENT_TIMEOUT`, etc.)
- [ ] Set up monitoring (Prometheus + Grafana); scrape config is in `monitoring/`
- [ ] Configure automated database backups -- see [DATABASE_OPERATIONS.md](DATABASE_OPERATIONS.md)
- [ ] Run `make migrate-up` against the production database before first deploy