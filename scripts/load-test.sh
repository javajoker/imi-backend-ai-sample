#!/bin/bash

# IP Marketplace Load Testing Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Default configuration
API_URL=${API_URL:-http://localhost:8080}
CONCURRENT_USERS=${CONCURRENT_USERS:-10}
DURATION=${DURATION:-60}
RAMP_UP=${RAMP_UP:-10}

echo "🚀 IP Marketplace Load Testing"
echo "=============================="

show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help              Show this help message"
    echo "  -u, --url URL           API base URL (default: $API_URL)"
    echo "  -c, --concurrent N      Concurrent users (default: $CONCURRENT_USERS)"
    echo "  -d, --duration N        Test duration in seconds (default: $DURATION)"
    echo "  -r, --ramp-up N         Ramp-up time in seconds (default: $RAMP_UP)"
    echo "  --smoke                 Run smoke test (light load)"
    echo "  --stress                Run stress test (heavy load)"
    echo "  --endpoints             Test specific endpoints only"
    echo ""
    echo "Examples:"
    echo "  $0 --smoke                          # Light load test"
    echo "  $0 --stress                         # Heavy load test"
    echo "  $0 -c 50 -d 300                     # 50 users for 5 minutes"
    echo "  $0 --endpoints                      # Test critical endpoints"
}

# Check dependencies
check_dependencies() {
    local missing_tools=()
    
    if ! command -v curl &> /dev/null; then
        missing_tools+=("curl")
    fi
    
    if ! command -v ab &> /dev/null && ! command -v wrk &> /dev/null; then
        missing_tools+=("apache2-utils (for ab) or wrk")
    fi
    
    if ! command -v jq &> /dev/null; then
        missing_tools+=("jq")
    fi
    
    if [ ${#missing_tools[@]} -gt 0 ]; then
        echo "❌ Missing required tools: ${missing_tools[*]}"
        echo ""
        echo "Install with:"
        echo "  Ubuntu/Debian: sudo apt-get install apache2-utils wrk jq curl"
        echo "  MacOS: brew install wrk jq curl"
        exit 1
    fi
}

# Test basic connectivity
test_connectivity() {
    echo "🔌 Testing API connectivity..."
    
    if ! curl -s --max-time 10 "$API_URL/health" &>/dev/null; then
        echo "❌ Cannot connect to API at $API_URL"
        echo "   Please ensure the server is running and accessible"
        exit 1
    fi
    
    echo "✅ API is accessible"
}

# Create test data
create_test_data() {
    echo "📝 Creating test data..."
    
    # Test user credentials
    cat > /tmp/test_user.json << EOF
{
  "username": "loadtest_user_$(date +%s)",
  "email": "loadtest$(date +%s)@example.com",
  "password": "LoadTest123!",
  "user_type": "buyer"
}
EOF

    # Register test user
    local response=$(curl -s -w "%{http_code}" -X POST \
        -H "Content-Type: application/json" \
        -d @/tmp/test_user.json \
        "$API_URL/v1/auth/register")
    
    local http_code="${response: -3}"
    local body="${response%???}"
    
    if [ "$http_code" = "201" ] || [ "$http_code" = "200" ]; then
        echo "✅ Test user created"
        
        # Extract token
        echo "$body" | jq -r '.data.token' > /tmp/test_token.txt 2>/dev/null || true
    else
        echo "⚠️  Using existing test setup (user may already exist)"
    fi
    
    # Clean up
    rm -f /tmp/test_user.json
}

# Apache Bench load test
run_ab_test() {
    local endpoint="$1"
    local description="$2"
    local concurrent="$3"
    local requests="$4"
    local headers="$5"
    
    echo ""
    echo "🧪 Testing: $description"
    echo "   Endpoint: $endpoint"
    echo "   Concurrent: $concurrent users"
    echo "   Requests: $requests total"
    
    local ab_cmd="ab -c $concurrent -n $requests"
    
    if [ -n "$headers" ]; then
        ab_cmd="$ab_cmd $headers"
    fi
    
    ab_cmd="$ab_cmd -q $API_URL$endpoint"
    
    # Run test and capture output
    local output=$(eval $ab_cmd 2>&1)
    
    # Parse results
    local rps=$(echo "$output" | grep "Requests per second" | awk '{print $4}')
    local mean_time=$(echo "$output" | grep "Time per request.*mean" | head -1 | awk '{print $4}')
    local failed=$(echo "$output" | grep "Failed requests" | awk '{print $3}')
    local p95=$(echo "$output" | grep "95%" | awk '{print $2}')
    
    echo "   📊 Results:"
    echo "      RPS: ${rps:-N/A}"
    echo "      Mean response time: ${mean_time:-N/A}ms"
    echo "      Failed requests: ${failed:-0}"
    echo "      95th percentile: ${p95:-N/A}ms"
}

# WRK load test
run_wrk_test() {
    local endpoint="$1"
    local description="$2"
    local concurrent="$3"
    local duration="$4"
    local headers="$5"
    
    echo ""
    echo "🧪 Testing: $description"
    echo "   Endpoint: $endpoint"
    echo "   Concurrent: $concurrent connections"
    echo "   Duration: ${duration}s"
    
    local wrk_cmd="wrk -c $concurrent -d ${duration}s -t $concurrent"
    
    if [ -n "$headers" ]; then
        wrk_cmd="$wrk_cmd $headers"
    fi
    
    wrk_cmd="$wrk_cmd $API_URL$endpoint"
    
    # Run test
    eval $wrk_cmd
}

# Smoke test (light load)
run_smoke_test() {
    echo "💨 Running smoke test (light load)..."
    
    local endpoints=(
        "/health|Health Check"
        "/v1/ip-assets?limit=5|IP Assets List"
        "/v1/products?limit=5|Products List"
    )
    
    for endpoint_info in "${endpoints[@]}"; do
        IFS='|' read -r endpoint description <<< "$endpoint_info"
        run_ab_test "$endpoint" "$description" 5 50
    done
}

# Stress test (heavy load)
run_stress_test() {
    echo "💪 Running stress test (heavy load)..."
    
    if command -v wrk &> /dev/null; then
        echo "Using WRK for stress testing..."
        
        run_wrk_test "/health" "Health Check" 50 30
        run_wrk_test "/v1/ip-assets?limit=10" "IP Assets List" 30 60
        run_wrk_test "/v1/products?limit=10" "Products List" 30 60
        
    else
        echo "Using Apache Bench for stress testing..."
        
        run_ab_test "/health" "Health Check" 50 1000
        run_ab_test "/v1/ip-assets?limit=10" "IP Assets List" 30 500
        run_ab_test "/v1/products?limit=10" "Products List" 30 500
    fi
}

# Test specific endpoints
run_endpoint_tests() {
    echo "🎯 Testing critical endpoints..."
    
    # Public endpoints (no auth required)
    local public_endpoints=(
        "/health|Health Check"
        "/v1/ip-assets|IP Assets List"
        "/v1/ip-assets/popular|Popular IP Assets"
        "/v1/products|Products List"
        "/v1/products/popular|Popular Products"
        "/verify/ABC123|Verification (will fail but tests endpoint)"
    )
    
    echo ""
    echo "📋 Public Endpoints:"
    for endpoint_info in "${public_endpoints[@]}"; do
        IFS='|' read -r endpoint description <<< "$endpoint_info"
        run_ab_test "$endpoint" "$description" 10 100
    done
    
    # Authenticated endpoints (if token available)
    if [ -f /tmp/test_token.txt ] && [ -s /tmp/test_token.txt ]; then
        local token=$(cat /tmp/test_token.txt)
        local auth_header="-H 'Authorization: Bearer $token'"
        
        echo ""
        echo "🔐 Authenticated Endpoints:"
        
        local auth_endpoints=(
            "/v1/auth/me|User Profile"
            "/v1/licenses/my-licenses|My Licenses"
        )
        
        for endpoint_info in "${auth_endpoints[@]}"; do
            IFS='|' read -r endpoint description <<< "$endpoint_info"
            run_ab_test "$endpoint" "$description" 5 50 "$auth_header"
        done
    fi
}

# Custom load test
run_custom_test() {
    echo "⚙️  Running custom load test..."
    echo "   Users: $CONCURRENT_USERS"
    echo "   Duration: ${DURATION}s"
    echo "   Ramp-up: ${RAMP_UP}s"
    
    if command -v wrk &> /dev/null; then
        # Calculate requests for duration
        local total_requests=$((CONCURRENT_USERS * DURATION))
        
        run_wrk_test "/health" "Health Check" "$CONCURRENT_USERS" "$DURATION"
        run_wrk_test "/v1/ip-assets?limit=20" "IP Assets" "$CONCURRENT_USERS" "$DURATION"
        
    else
        # Use ab with calculated requests
        local total_requests=$((CONCURRENT_USERS * 10))
        
        run_ab_test "/health" "Health Check" "$CONCURRENT_USERS" "$total_requests"
        run_ab_test "/v1/ip-assets?limit=20" "IP Assets" "$CONCURRENT_USERS" "$total_requests"
    fi
}

# Generate report
generate_report() {
    echo ""
    echo "📊 Load Test Report"
    echo "=================="
    echo "API URL: $API_URL"
    echo "Test completed at: $(date)"
    echo ""
    echo "💡 Tips for improving performance:"
    echo "   - Enable Redis caching"
    echo "   - Optimize database queries"
    echo "   - Use CDN for static assets"
    echo "   - Implement connection pooling"
    echo "   - Add rate limiting"
    echo ""
    echo "📈 Monitor these metrics in production:"
    echo "   - Response times"
    echo "   - Error rates"
    echo "   - Database connection pool"
    echo "   - Memory usage"
    echo "   - CPU utilization"
}

# Parse arguments
SMOKE=false
STRESS=false
ENDPOINTS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        -u|--url)
            API_URL="$2"
            shift 2
            ;;
        -c|--concurrent)
            CONCURRENT_USERS="$2"
            shift 2
            ;;
        -d|--duration)
            DURATION="$2"
            shift 2
            ;;
        -r|--ramp-up)
            RAMP_UP="$2"
            shift 2
            ;;
        --smoke)
            SMOKE=true
            shift
            ;;
        --stress)
            STRESS=true
            shift
            ;;
        --endpoints)
            ENDPOINTS=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Main execution
main() {
    check_dependencies
    test_connectivity
    create_test_data
    
    if [ "$SMOKE" = true ]; then
        run_smoke_test
    elif [ "$STRESS" = true ]; then
        run_stress_test
    elif [ "$ENDPOINTS" = true ]; then
        run_endpoint_tests
    else
        run_custom_test
    fi
    
    generate_report
    
    # Cleanup
    rm -f /tmp/test_token.txt
}

# Run main function
main
