#!/bin/bash

# Restart Single Service Script
# Usage: ./scripts/restart-service.sh <service-name>

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Check if service name is provided
if [ -z "$1" ]; then
    echo -e "${RED}Error: Service name not provided${NC}"
    echo "Usage: $0 <service-name>"
    echo "Example: $0 admin"
    exit 1
fi

SERVICE=$1
PID_DIR="./.pids"
LOG_DIR="./logs"
PID_FILE="$PID_DIR/$SERVICE.pid"
LOG_FILE="$LOG_DIR/$SERVICE.log"

# Check if service directory exists
if [ ! -d "cmd/$SERVICE" ]; then
    echo -e "${RED}✗ Service '$SERVICE' not found (cmd/$SERVICE does not exist)${NC}"
    exit 1
fi

# Create directories
mkdir -p "$PID_DIR"
mkdir -p "$LOG_DIR"

echo -e "${YELLOW}=== Restarting $SERVICE service ===${NC}"
echo ""

# Step 1: Stop the service
echo -e "${YELLOW}[1/4] Stopping $SERVICE...${NC}"
./scripts/stop-service.sh "$SERVICE" 2>/dev/null || true

# Step 2: Rebuild if requested (optional, check for --build flag)
if [[ "$2" == "--build" ]]; then
    echo -e "${YELLOW}[2/4] Building $SERVICE...${NC}"
    go build -o bin/$SERVICE ./cmd/$SERVICE
    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ Build successful${NC}"
    else
        echo -e "${RED}✗ Build failed${NC}"
        exit 1
    fi
else
    echo -e "${YELLOW}[2/4] Skipping build (use --build flag to rebuild)${NC}"
fi

echo -e "${YELLOW}[3/4] Checking port availability...${NC}"

# Step 3: Ensure port is free
# Map service names to ports
PORT=""
case "$SERVICE" in
    auth) PORT="8081" ;;
    rides) PORT="8082" ;;
    geo) PORT="8083" ;;
    payments) PORT="8084" ;;
    notifications) PORT="8085" ;;
    realtime) PORT="8086" ;;
    mobile) PORT="8087" ;;
    admin) PORT="8088" ;;
    promos) PORT="8089" ;;
    scheduler) PORT="8090" ;;
    analytics) PORT="8091" ;;
    fraud) PORT="8092" ;;
    ml-eta) PORT="8093" ;;
esac

if [ -n "$PORT" ]; then
    if lsof -ti:$PORT > /dev/null 2>&1; then
        echo -e "${YELLOW}Port $PORT is in use, killing process...${NC}"
        lsof -ti:$PORT | xargs kill -9 2>/dev/null || true
        sleep 1
    fi
fi

# Step 4: Start the service
echo -e "${YELLOW}[4/4] Starting $SERVICE...${NC}"
nohup go run ./cmd/$SERVICE > "$LOG_FILE" 2>&1 &
pid=$!
echo $pid > "$PID_FILE"

# Wait a bit and check if it's still running
sleep 2
if ps -p "$pid" > /dev/null 2>&1; then
    echo -e "${GREEN}✓ $SERVICE restarted successfully (PID: $pid)${NC}"
    echo ""
    echo -e "${BLUE}Tail logs: tail -f $LOG_FILE${NC}"
    echo -e "${BLUE}Stop service: make stop-$SERVICE${NC}"
    echo ""
else
    echo -e "${RED}✗ $SERVICE failed to start${NC}"
    echo -e "${YELLOW}Check logs: cat $LOG_FILE${NC}"
    rm -f "$PID_FILE"
    exit 1
fi
