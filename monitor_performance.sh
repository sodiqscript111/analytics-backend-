#!/bin/bash

# Monitor Redis and Database during stress test

echo "📊 Real-time Performance Monitor"
echo "════════════════════════════════════════════════════════"
echo "Press Ctrl+C to stop monitoring"
echo ""

# Configuration
REDIS_HOST="localhost"
REDIS_PORT="6379"
STREAM_NAME="events"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Function to get Redis stream length
get_stream_length() {
    redis-cli -h $REDIS_HOST -p $REDIS_PORT XLEN $STREAM_NAME 2>/dev/null || echo "0"
}

# Function to get Redis memory usage
get_redis_memory() {
    redis-cli -h $REDIS_HOST -p $REDIS_PORT INFO memory | grep "used_memory_human" | cut -d: -f2 | tr -d '\r\n'
}

# Function to get consumer group info
get_consumer_info() {
    redis-cli -h $REDIS_HOST -p $REDIS_PORT XINFO GROUPS $STREAM_NAME 2>/dev/null | grep "pending" | head -1 | awk '{print $2}'
}

# Initialize tracking
PREV_LENGTH=0
START_TIME=$(date +%s)

# Header
printf "${BLUE}%-20s %-15s %-15s %-15s %-15s${NC}\n" "Time" "Stream Length" "Pending Msgs" "Redis Memory" "Events/sec"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"

# Monitor loop
while true; do
    CURRENT_TIME=$(date +"%H:%M:%S")
    STREAM_LENGTH=$(get_stream_length)
    PENDING_MSGS=$(get_consumer_info)
    REDIS_MEM=$(get_redis_memory)

    # Calculate events per second
    ELAPSED=$(($(date +%s) - START_TIME))
    if [ $ELAPSED -gt 0 ]; then
        EVENTS_PER_SEC=$((($STREAM_LENGTH - $PREV_LENGTH) / 1))
    else
        EVENTS_PER_SEC=0
    fi

    # Color coding based on stream length
    if [ $STREAM_LENGTH -gt 10000 ]; then
        COLOR=$RED
    elif [ $STREAM_LENGTH -gt 5000 ]; then
        COLOR=$YELLOW
    else
        COLOR=$GREEN
    fi

    printf "${COLOR}%-20s %-15s %-15s %-15s %-15s${NC}\n" \
        "$CURRENT_TIME" \
        "$STREAM_LENGTH" \
        "${PENDING_MSGS:-0}" \
        "$REDIS_MEM" \
        "$EVENTS_PER_SEC"

    PREV_LENGTH=$STREAM_LENGTH
    START_TIME=$(date +%s)

    sleep 1
done
