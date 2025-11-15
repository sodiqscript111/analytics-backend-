#!/bin/bash

# Install vegeta if not already installed:
# go install github.com/tsenart/vegeta/v12@latest

echo "🎯 Vegeta Stress Test for Analytics Backend"
echo "═══════════════════════════════════════════════"

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
TARGET_URL="http://localhost:8080/event"
DURATION="30s"

# Create sample payload generator
cat > payload.txt << 'EOF'
POST http://localhost:8080/event
Content-Type: application/json

{
  "user_id": "user_{{rand 1 100}}",
  "action": "{{oneOf "click" "view" "scroll" "hover" "submit"}}",
  "element": "{{oneOf "button" "link" "image" "form" "video"}}",
  "duration": {{rand 0 10}}.{{rand 0 99}},
  "timestamp": "{{now}}"
}
EOF

# Test scenarios
declare -a RATES=("100" "500" "1000" "2000")

for RATE in "${RATES[@]}"
do
    echo ""
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"
    echo -e "${GREEN}Testing at ${RATE} requests/second${NC}"
    echo -e "${BLUE}━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━${NC}"

    # Generate events on the fly
    cat << EOF | vegeta attack -rate=${RATE} -duration=${DURATION} | tee results_${RATE}rps.bin | vegeta report
POST ${TARGET_URL}
Content-Type: application/json

{
  "user_id": "user_1",
  "action": "click",
  "element": "button",
  "duration": 2.5,
  "timestamp": "$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
}
EOF

    echo ""
    echo -e "${YELLOW}Detailed latency report:${NC}"
    vegeta report -type='hist[0,10ms,50ms,100ms,200ms,500ms,1s,2s]' results_${RATE}rps.bin

    echo ""
    echo "Waiting 5 seconds before next test..."
    sleep 5
done

# Cleanup
rm -f payload.txt

echo ""
echo -e "${GREEN}✨ All tests completed!${NC}"
echo ""
echo "Results saved to: results_*rps.bin"
echo "To view a report: vegeta report results_100rps.bin"
echo "To plot results: vegeta plot results_100rps.bin > plot.html"
