#!/bin/bash

# Self-Hosted Sentry Setup Script
# This script initializes the self-hosted Sentry instance

set -e

echo "=========================================="
echo "Self-Hosted Sentry Setup"
echo "=========================================="
echo

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Step 1: Generate Secret Key
echo "Step 1: Generating Sentry Secret Key..."
echo

if [ -f .env ] && grep -q "SENTRY_SECRET_KEY=" .env; then
    echo -e "${YELLOW}SENTRY_SECRET_KEY already exists in .env file${NC}"
    echo "Skipping secret key generation..."
else
    echo "Generating new secret key..."
    # Generate a secure random secret key
    SECRET_KEY=$(openssl rand -hex 32)

    if [ -f .env ]; then
        echo "SENTRY_SECRET_KEY=$SECRET_KEY" >> .env
    else
        echo "SENTRY_SECRET_KEY=$SECRET_KEY" > .env
    fi

    echo -e "${GREEN}✓ Secret key generated and saved to .env${NC}"
fi

echo

# Step 2: Start Sentry dependencies
echo "Step 2: Starting Sentry dependencies..."
echo "This may take a few minutes on first run..."
echo

docker-compose up -d sentry-postgres sentry-redis sentry-clickhouse sentry-zookeeper sentry-kafka

echo "Waiting for services to be healthy..."
sleep 10

# Wait for PostgreSQL to be ready
echo "Waiting for PostgreSQL to be ready..."
until docker-compose exec -T sentry-postgres pg_isready -U sentry > /dev/null 2>&1; do
    echo -n "."
    sleep 2
done
echo
echo -e "${GREEN}✓ PostgreSQL is ready${NC}"

# Wait for Redis to be ready
echo "Waiting for Redis to be ready..."
until docker-compose exec -T sentry-redis redis-cli ping > /dev/null 2>&1; do
    echo -n "."
    sleep 2
done
echo
echo -e "${GREEN}✓ Redis is ready${NC}"

echo

# Step 3: Run Sentry database migrations
echo "Step 3: Running Sentry database migrations..."
echo "This will take several minutes on first run..."
echo

docker-compose run --rm sentry upgrade --noinput

echo -e "${GREEN}✓ Database migrations completed${NC}"
echo

# Step 4: Create superuser
echo "Step 4: Creating Sentry superuser..."
echo

echo "Please enter your Sentry administrator credentials:"
read -p "Email address: " SENTRY_ADMIN_EMAIL
read -sp "Password: " SENTRY_ADMIN_PASSWORD
echo

if [ -z "$SENTRY_ADMIN_EMAIL" ] || [ -z "$SENTRY_ADMIN_PASSWORD" ]; then
    echo -e "${RED}✗ Email and password are required${NC}"
    exit 1
fi

# Create superuser
docker-compose run --rm \
    -e SENTRY_ADMIN_EMAIL="$SENTRY_ADMIN_EMAIL" \
    -e SENTRY_ADMIN_PASSWORD="$SENTRY_ADMIN_PASSWORD" \
    sentry createuser \
    --email "$SENTRY_ADMIN_EMAIL" \
    --password "$SENTRY_ADMIN_PASSWORD" \
    --superuser \
    --no-input

echo -e "${GREEN}✓ Superuser created${NC}"
echo

# Step 5: Start all Sentry services
echo "Step 5: Starting all Sentry services..."
echo

docker-compose up -d sentry sentry-worker sentry-cron

echo "Waiting for Sentry web service to be ready..."
sleep 5

echo
echo -e "${GREEN}=========================================="
echo "✓ Sentry Setup Complete!"
echo "==========================================${NC}"
echo
echo "Sentry is now running at: http://localhost:9000"
echo
echo "Login credentials:"
echo "  Email: $SENTRY_ADMIN_EMAIL"
echo "  Password: [the password you entered]"
echo
echo "Next steps:"
echo "  1. Access Sentry at http://localhost:9000"
echo "  2. Create a new project for 'ride-hailing'"
echo "  3. Copy the DSN from project settings"
echo "  4. Update SENTRY_DSN in .env with: http://[public-key]@localhost:9000/[project-id]"
echo
echo "For production deployment, make sure to:"
echo "  - Change default passwords in docker-compose.yml"
echo "  - Set up proper backups for Sentry volumes"
echo "  - Configure email notifications"
echo "  - Enable HTTPS with a reverse proxy"
echo

