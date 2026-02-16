#!/bin/bash

# Stop Single Service Script
# Usage: ./scripts/stop-service.sh <service-name>

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
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
PID_FILE="$PID_DIR/$SERVICE.pid"

echo -e "${YELLOW}Stopping $SERVICE service...${NC}"

# Stop service using PID file
if [ -f "$PID_FILE" ]; then
    pid=$(cat "$PID_FILE")

    if ps -p "$pid" > /dev/null 2>&1; then
        echo -e "${YELLOW}  Killing PID $pid...${NC}"
        kill "$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null

        # Wait for process to stop
        sleep 1

        if ! ps -p "$pid" > /dev/null 2>&1; then
            echo -e "${GREEN}✓ $SERVICE stopped${NC}"
        else
            echo -e "${RED}✗ Failed to stop $SERVICE${NC}"
            exit 1
        fi
    else
        echo -e "${YELLOW}⊘ $SERVICE was not running (stale PID file)${NC}"
    fi

    rm -f "$PID_FILE"
else
    echo -e "${YELLOW}⊘ No PID file found for $SERVICE${NC}"
fi

# Also check for any running "go run ./cmd/$SERVICE" processes
found=0
while IFS= read -r line; do
    if [ -n "$line" ]; then
        pid=$(echo "$line" | awk '{print $2}')
        echo -e "${YELLOW}  Found orphaned process (PID: $pid), stopping...${NC}"
        kill "$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
        found=1
    fi
done < <(ps aux | grep "go run ./cmd/$SERVICE" | grep -v grep)

if [ $found -eq 0 ]; then
    echo -e "${GREEN}✓ $SERVICE service stopped${NC}"
else
    echo -e "${GREEN}✓ $SERVICE service and orphaned processes stopped${NC}"
fi
