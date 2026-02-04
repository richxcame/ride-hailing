# Self-Hosted Sentry - Quick Start

## Production Setup

```bash
./scripts/setup-sentry.sh
```

Open http://localhost:9000 and create your admin account.

## Development Setup

```bash
docker-compose -f docker-compose.dev.yml --profile sentry up -d
./scripts/setup-sentry.sh dev
```

Open http://localhost:9000 and create your admin account.

## Configure Your Services

### 1. Create a Project in Sentry UI

1. Click "Create Project" -> select **Go** -> name it `ride-hailing` -> create.
2. Copy the DSN from Settings -> Client Keys (looks like `http://[public-key]@localhost:9000/1`).

### 2. Set Environment Variables

**Production** -- add to `.env` then `docker-compose restart`:
```bash
SENTRY_DSN=http://[your-key]@localhost:9000/1
```

**Development** -- export or add to `.env`, then run services with `make run-*`:
```bash
export SENTRY_DSN=http://[your-key]@localhost:9000/1
```

### 3. Test

```bash
curl http://localhost:8081/test/sentry
```

Check the Sentry dashboard -- the error should appear.

## Full Documentation

See [docs/SELF_HOSTED_SENTRY.md](SELF_HOSTED_SENTRY.md) for the complete guide (troubleshooting, resource tuning, etc.).
