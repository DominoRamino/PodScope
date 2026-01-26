#!/bin/bash
# PodScope Stress Test Script
# Generates a variety of HTTP/HTTPS requests to test capture capabilities

set -e

# Configuration
CONCURRENCY=${CONCURRENCY:-10}      # Number of parallel requests
TOTAL_REQUESTS=${TOTAL_REQUESTS:-100}  # Total requests to make
DELAY_MS=${DELAY_MS:-50}            # Delay between batches (ms)

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m' # No Color

echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║           PodScope Stress Test                               ║${NC}"
echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
echo -e "${CYAN}║  Concurrency: ${YELLOW}$CONCURRENCY${CYAN} | Total Requests: ${YELLOW}$TOTAL_REQUESTS${CYAN}              ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Free APIs - No signup required
ENDPOINTS=(
    # httpbin.org - Echo service (HTTPS)
    "GET|https://httpbin.org/get"
    "POST|https://httpbin.org/post|{\"test\":\"data\",\"timestamp\":\"TIME\"}"
    "PUT|https://httpbin.org/put|{\"update\":\"value\"}"
    "DELETE|https://httpbin.org/delete"
    "PATCH|https://httpbin.org/patch|{\"partial\":\"update\"}"
    "GET|https://httpbin.org/headers"
    "GET|https://httpbin.org/ip"
    "GET|https://httpbin.org/user-agent"
    "GET|https://httpbin.org/uuid"
    "GET|https://httpbin.org/bytes/1024"
    "GET|https://httpbin.org/bytes/4096"
    "GET|https://httpbin.org/stream/5"
    "GET|https://httpbin.org/delay/1"
    "GET|https://httpbin.org/status/200"
    "GET|https://httpbin.org/status/201"
    "GET|https://httpbin.org/status/204"
    "GET|https://httpbin.org/status/301"
    "GET|https://httpbin.org/status/400"
    "GET|https://httpbin.org/status/404"
    "GET|https://httpbin.org/status/500"
    "GET|https://httpbin.org/response-headers?X-Custom=PodScope"
    "GET|https://httpbin.org/gzip"
    "GET|https://httpbin.org/deflate"
    "GET|https://httpbin.org/encoding/utf8"
    "GET|https://httpbin.org/html"
    "GET|https://httpbin.org/json"
    "GET|https://httpbin.org/xml"
    "GET|https://httpbin.org/robots.txt"

    # JSONPlaceholder - Fake REST API (HTTPS)
    "GET|https://jsonplaceholder.typicode.com/posts"
    "GET|https://jsonplaceholder.typicode.com/posts/1"
    "GET|https://jsonplaceholder.typicode.com/posts/1/comments"
    "GET|https://jsonplaceholder.typicode.com/comments?postId=1"
    "GET|https://jsonplaceholder.typicode.com/users"
    "GET|https://jsonplaceholder.typicode.com/users/1"
    "GET|https://jsonplaceholder.typicode.com/albums"
    "GET|https://jsonplaceholder.typicode.com/photos?albumId=1"
    "GET|https://jsonplaceholder.typicode.com/todos"
    "GET|https://jsonplaceholder.typicode.com/todos/1"
    "POST|https://jsonplaceholder.typicode.com/posts|{\"title\":\"test\",\"body\":\"content\",\"userId\":1}"
    "PUT|https://jsonplaceholder.typicode.com/posts/1|{\"id\":1,\"title\":\"updated\",\"body\":\"new content\",\"userId\":1}"
    "PATCH|https://jsonplaceholder.typicode.com/posts/1|{\"title\":\"patched\"}"
    "DELETE|https://jsonplaceholder.typicode.com/posts/1"

    # Reqres.in - User API (HTTPS)
    "GET|https://reqres.in/api/users?page=1"
    "GET|https://reqres.in/api/users?page=2"
    "GET|https://reqres.in/api/users/1"
    "GET|https://reqres.in/api/users/2"
    "GET|https://reqres.in/api/unknown"
    "GET|https://reqres.in/api/unknown/2"
    "POST|https://reqres.in/api/users|{\"name\":\"morpheus\",\"job\":\"leader\"}"
    "PUT|https://reqres.in/api/users/2|{\"name\":\"morpheus\",\"job\":\"zion resident\"}"
    "PATCH|https://reqres.in/api/users/2|{\"name\":\"morpheus\",\"job\":\"updated\"}"
    "DELETE|https://reqres.in/api/users/2"
    "POST|https://reqres.in/api/register|{\"email\":\"eve.holt@reqres.in\",\"password\":\"pistol\"}"
    "POST|https://reqres.in/api/login|{\"email\":\"eve.holt@reqres.in\",\"password\":\"cityslicka\"}"

    # Fun APIs (HTTPS)
    "GET|https://dog.ceo/api/breeds/list/all"
    "GET|https://dog.ceo/api/breeds/image/random"
    "GET|https://dog.ceo/api/breed/hound/images/random"
    "GET|https://catfact.ninja/fact"
    "GET|https://catfact.ninja/facts?limit=5"
    "GET|https://catfact.ninja/breeds?limit=5"

    # World Time API (HTTPS)
    "GET|https://worldtimeapi.org/api/ip"
    "GET|https://worldtimeapi.org/api/timezone"
    "GET|https://worldtimeapi.org/api/timezone/America/New_York"
    "GET|https://worldtimeapi.org/api/timezone/Europe/London"

    # Other useful APIs
    "GET|https://api.publicapis.org/entries?category=animals"
    "GET|https://api.publicapis.org/random"
    "GET|https://api.ipify.org?format=json"
    "GET|https://api.genderize.io?name=peter"
    "GET|https://api.nationalize.io?name=michael"
    "GET|https://api.agify.io?name=bella"
)

# Counters
SUCCESS=0
FAILED=0
TOTAL=0

# Function to make a single request
make_request() {
    local spec="$1"
    local method=$(echo "$spec" | cut -d'|' -f1)
    local url=$(echo "$spec" | cut -d'|' -f2)
    local body=$(echo "$spec" | cut -d'|' -f3-)

    # Replace TIME placeholder with current timestamp
    body=$(echo "$body" | sed "s/TIME/$(date +%s)/g")

    local curl_args=(-s -o /dev/null -w "%{http_code}" -X "$method" --connect-timeout 10 --max-time 30)

    if [[ -n "$body" ]]; then
        curl_args+=(-H "Content-Type: application/json" -d "$body")
    fi

    curl_args+=("$url")

    local status_code
    status_code=$(curl "${curl_args[@]}" 2>/dev/null) || status_code="000"

    if [[ "$status_code" =~ ^[23] ]]; then
        echo -e "${GREEN}✓${NC} $method $url -> $status_code"
        return 0
    else
        echo -e "${RED}✗${NC} $method $url -> $status_code"
        return 1
    fi
}

# Function to run a batch of concurrent requests
run_batch() {
    local pids=()
    local results=()

    for ((i=0; i<CONCURRENCY && TOTAL<TOTAL_REQUESTS; i++)); do
        # Pick a random endpoint
        local idx=$((RANDOM % ${#ENDPOINTS[@]}))
        local endpoint="${ENDPOINTS[$idx]}"

        # Run in background
        make_request "$endpoint" &
        pids+=($!)
        ((TOTAL++))
    done

    # Wait for all background jobs and collect results
    for pid in "${pids[@]}"; do
        if wait $pid; then
            ((SUCCESS++))
        else
            ((FAILED++))
        fi
    done
}

# Main execution
echo -e "${YELLOW}Starting stress test...${NC}"
echo ""

START_TIME=$(date +%s)

while ((TOTAL < TOTAL_REQUESTS)); do
    run_batch

    # Small delay between batches to not overwhelm
    if ((DELAY_MS > 0)); then
        sleep $(echo "scale=3; $DELAY_MS/1000" | bc)
    fi

    # Progress indicator
    PERCENT=$((TOTAL * 100 / TOTAL_REQUESTS))
    printf "\r${CYAN}Progress: [%-50s] %d%% (%d/%d)${NC}" \
        "$(printf '#%.0s' $(seq 1 $((PERCENT/2))))" \
        "$PERCENT" "$TOTAL" "$TOTAL_REQUESTS"
done

END_TIME=$(date +%s)
DURATION=$((END_TIME - START_TIME))

echo ""
echo ""
echo -e "${CYAN}╔══════════════════════════════════════════════════════════════╗${NC}"
echo -e "${CYAN}║                      Results Summary                         ║${NC}"
echo -e "${CYAN}╠══════════════════════════════════════════════════════════════╣${NC}"
echo -e "${CYAN}║  ${GREEN}Successful: $SUCCESS${CYAN}                                            ║${NC}"
echo -e "${CYAN}║  ${RED}Failed: $FAILED${CYAN}                                                ║${NC}"
echo -e "${CYAN}║  Total: $TOTAL                                               ║${NC}"
echo -e "${CYAN}║  Duration: ${DURATION}s                                             ║${NC}"
echo -e "${CYAN}║  Rate: $(echo "scale=1; $TOTAL/$DURATION" | bc) req/s                                          ║${NC}"
echo -e "${CYAN}╚══════════════════════════════════════════════════════════════╝${NC}"
