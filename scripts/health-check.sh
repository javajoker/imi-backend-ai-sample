#!/bin/bash

# IP Marketplace Health Check Script

set -e

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Load environment variables
if [ -f "${PROJECT_ROOT}/.env" ]; then
    source "${PROJECT_ROOT}/.env"
fi

# Configuration
API_URL=${API_URL:-http://localhost:8080}
DB_HOST=${DB_HOST:-localhost}
DB_PORT=${DB_PORT:-5432}
DB_USER=${DB_USER:-postgres}
DB_NAME=${DB_NAME:-ip_marketplace}
REDIS_HOST=${REDIS_HOST:-localhost}
REDIS_PORT=${REDIS_PORT:-6379}
TIMEOUT=${HEALTH_CHECK_TIMEOUT:-10}

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0

echo "🏥 IP Marketplace Health Check"
echo "================================"
echo ""

# Function to print status
print_status() {
    local service="$1"
    local status="$2"
    local details="$3"
    
    TOTAL_CHECKS=$((TOTAL_CHECKS + 1))
    
    if [ "$status" = "PASS" ]; then
        echo -e "${GREEN}✅ $service: PASS${NC} $details"
        PASSED_CHECKS=$((PASSED_CHECKS + 1))
    elif [ "$status" = "WARN" ]; then
        echo -e "${YELLOW}⚠️  $service: WARN${NC} $details"
    else
        echo -e "${RED}❌ $service: FAIL${NC} $details"
        FAILED_CHECKS=$((FAILED_CHECKS + 1))
    fi
}

# Check API Health
check_api_health() {
    echo -e "${BLUE}🌐 Checking API Health...${NC}"
    
    # Basic health endpoint
    if response=$(curl -s -w "%{http_code}" --max-time $TIMEOUT "$API_URL/health" 2>/dev/null); then
        http_code="${response: -3}"
        body="${response%???}"
        
        if [ "$http_code" = "200" ]; then
            print_status "API Health Endpoint" "PASS" "($http_code)"
            
            # Check response format
            if echo "$body" | grep -q '"status".*"healthy"'; then
                print_status "API Response Format" "PASS" "(Valid JSON)"
            else
                print_status "API Response Format" "WARN" "(Unexpected format)"
            fi
        else
            print_status "API Health Endpoint" "FAIL" "HTTP $http_code"
        fi
    else
        print_status "API Health Endpoint" "FAIL" "(Connection failed)"
    fi
    
    # Check API version endpoint
    if response=$(curl -s --max-time $TIMEOUT "$API_URL/v1/ip-assets?limit=1" 2>/dev/null); then
        if echo "$response" | grep -q '"success"'; then
            print_status "API Endpoints" "PASS" "(Responding)"
        else
            print_status "API Endpoints" "WARN" "(Unexpected response)"
        fi
    else
        print_status "API Endpoints" "FAIL" "(Not responding)"
    fi
}

# Check Database Health
check_database_health() {
    echo -e "${BLUE}🗄️  Checking Database Health...${NC}"
    
    # Check connection
    if PGPASSWORD="$DB_PASSWORD" pg_isready -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" &>/dev/null; then
        print_status "Database Connection" "PASS" "($DB_HOST:$DB_PORT)"
        
        # Check database size
        db_size=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT pg_size_pretty(pg_database_size('$DB_NAME'));" 2>/dev/null | xargs)
        if [ -n "$db_size" ]; then
            print_status "Database Size" "PASS" "($db_size)"
        fi
        
        # Check critical tables
        tables=("users" "ip_assets" "products" "transactions")
        for table in "${tables[@]}"; do
            count=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT COUNT(*) FROM $table;" 2>/dev/null | xargs)
            if [ -n "$count" ] && [ "$count" -ge 0 ]; then
                print_status "Table: $table" "PASS" "($count rows)"
            else
                print_status "Table: $table" "FAIL" "(Error or missing)"
            fi
        done
        
        # Check active connections
        active_connections=$(PGPASSWORD="$DB_PASSWORD" psql -h "$DB_HOST" -p "$DB_PORT" -U "$DB_USER" -d "$DB_NAME" -t -c "SELECT count(*) FROM pg_stat_activity WHERE datname = '$DB_NAME';" 2>/dev/null | xargs)
        if [ -n "$active_connections" ]; then
            if [ "$active_connections" -lt 50 ]; then
                print_status "Database Connections" "PASS" "($active_connections active)"
            else
                print_status "Database Connections" "WARN" "($active_connections active - high)"
            fi
        fi
        
    else
        print_status "Database Connection" "FAIL" "($DB_HOST:$DB_PORT)"
    fi
}

# Check Redis Health
check_redis_health() {
    echo -e "${BLUE}🔴 Checking Redis Health...${NC}"
    
    if command -v redis-cli &> /dev/null; then
        # Check Redis connection
        if redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" ping &>/dev/null; then
            print_status "Redis Connection" "PASS" "($REDIS_HOST:$REDIS_PORT)"
            
            # Check Redis memory usage
            memory_used=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" info memory 2>/dev/null | grep "used_memory_human" | cut -d: -f2 | tr -d '\r')
            if [ -n "$memory_used" ]; then
                print_status "Redis Memory Usage" "PASS" "($memory_used)"
            fi
            
            # Check Redis keyspace
            keyspace_info=$(redis-cli -h "$REDIS_HOST" -p "$REDIS_PORT" info keyspace 2>/dev/null | grep "^db" | head -1)
            if [ -n "$keyspace_info" ]; then
                print_status "Redis Keyspace" "PASS" "(Has data)"
            else
                print_status "Redis Keyspace" "WARN" "(No data)"
            fi
            
        else
            print_status "Redis Connection" "FAIL" "($REDIS_HOST:$REDIS_PORT)"
        fi
    else
        print_status "Redis CLI" "WARN" "(redis-cli not installed)"
    fi
}

# Check File System Health
check_filesystem_health() {
    echo -e "${BLUE}💾 Checking File System Health...${NC}"
    
    # Check uploads directory
    uploads_dir="$PROJECT_ROOT/uploads"
    if [ -d "$uploads_dir" ]; then
        print_status "Uploads Directory" "PASS" "(Exists)"
        
        # Check permissions
        if [ -w "$uploads_dir" ]; then
            print_status "Upload Permissions" "PASS" "(Writable)"
        else
            print_status "Upload Permissions" "FAIL" "(Not writable)"
        fi
        
        # Check disk usage
        disk_usage=$(df -h "$uploads_dir" | awk 'NR==2 {print $5}' | sed 's/%//')
        if [ -n "$disk_usage" ]; then
            if [ "$disk_usage" -lt 80 ]; then
                print_status "Disk Usage" "PASS" "(${disk_usage}% used)"
            elif [ "$disk_usage" -lt 90 ]; then
                print_status "Disk Usage" "WARN" "(${disk_usage}% used - high)"
            else
                print_status "Disk Usage" "FAIL" "(${disk_usage}% used - critical)"
            fi
        fi
    else
        print_status "Uploads Directory" "FAIL" "(Missing)"
    fi
    
    # Check logs directory
    logs_dir="$PROJECT_ROOT/logs"
    if [ -d "$logs_dir" ]; then
        print_status "Logs Directory" "PASS" "(Exists)"
    else
        print_status "Logs Directory" "WARN" "(Missing - will create if needed)"
    fi
}

# Check External Services
check_external_services() {
    echo -e "${BLUE}🌍 Checking External Services...${NC}"
    
    # Check AWS S3 (if configured)
    if [ -n "$AWS_S3_BUCKET" ] && command -v aws &> /dev/null; then
        if aws s3 ls "s3://$AWS_S3_BUCKET" &>/dev/null; then
            print_status "AWS S3 Access" "PASS" "($AWS_S3_BUCKET)"
        else
            print_status "AWS S3 Access" "FAIL" "($AWS_S3_BUCKET)"
        fi
    else
        print_status "AWS S3 Config" "WARN" "(Not configured or AWS CLI missing)"
    fi
    
    # Check SMTP (basic connectivity)
    if [ -n "$SMTP_HOST" ] && [ -n "$SMTP_PORT" ]; then
        if timeout 5 bash -c "</dev/tcp/$SMTP_HOST/$SMTP_PORT" &>/dev/null; then
            print_status "SMTP Connectivity" "PASS" "($SMTP_HOST:$SMTP_PORT)"
        else
            print_status "SMTP Connectivity" "FAIL" "($SMTP_HOST:$SMTP_PORT)"
        fi
    else
        print_status "SMTP Config" "WARN" "(Not configured)"
    fi
    
    # Check Stripe (if configured)
    if [ -n "$STRIPE_SECRET_KEY" ] && command -v curl &> /dev/null; then
        if curl -s --max-time 5 -H "Authorization: Bearer $STRIPE_SECRET_KEY" "https://api.stripe.com/v1/balance" | grep -q '"object": "balance"'; then
            print_status "Stripe API" "PASS" "(Connected)"
        else
            print_status "Stripe API" "FAIL" "(Connection or auth failed)"
        fi
    else
        print_status "Stripe Config" "WARN" "(Not configured)"
    fi
}

# Check System Resources
check_system_resources() {
    echo -e "${BLUE}💻 Checking System Resources...${NC}"
    
    # Check memory usage
    if command -v free &> /dev/null; then
        memory_usage=$(free | awk 'NR==2{printf "%.0f", $3*100/$2}')
        if [ "$memory_usage" -lt 80 ]; then
            print_status "Memory Usage" "PASS" "(${memory_usage}%)"
        elif [ "$memory_usage" -lt 90 ]; then
            print_status "Memory Usage" "WARN" "(${memory_usage}% - high)"
        else
            print_status "Memory Usage" "FAIL" "(${memory_usage}% - critical)"
        fi
    fi
    
    # Check CPU load
    if command -v uptime &> /dev/null; then
        load_avg=$(uptime | awk -F'load average:' '{print $2}' | awk '{print $1}' | sed 's/,$//')
        cpu_cores=$(nproc 2>/dev/null || echo "1")
        load_percent=$(echo "$load_avg * 100 / $cpu_cores" | bc 2>/dev/null || echo "0")
        
        if [ "${load_percent%.*}" -lt 70 ]; then
            print_status "CPU Load" "PASS" "(${load_avg} avg)"
        elif [ "${load_percent%.*}" -lt 90 ]; then
            print_status "CPU Load" "WARN" "(${load_avg} avg - high)"
        else
            print_status "CPU Load" "FAIL" "(${load_avg} avg - critical)"
        fi
    fi
}

# Main execution
main() {
    check_api_health
    echo ""
    
    check_database_health
    echo ""
    
    check_redis_health
    echo ""
    
    check_filesystem_health
    echo ""
    
    check_external_services
    echo ""
    
    check_system_resources
    echo ""
    
    # Summary
    echo "================================"
    echo -e "${BLUE}📊 Health Check Summary${NC}"
    echo "================================"
    echo "Total Checks: $TOTAL_CHECKS"
    echo -e "Passed: ${GREEN}$PASSED_CHECKS${NC}"
    echo -e "Failed: ${RED}$FAILED_CHECKS${NC}"
    
    if [ $FAILED_CHECKS -eq 0 ]; then
        echo ""
        echo -e "${GREEN}🎉 All critical checks passed!${NC}"
        exit 0
    else
        echo ""
        echo -e "${RED}⚠️  $FAILED_CHECKS checks failed. Please review the issues above.${NC}"
        exit 1
    fi
}

# Show usage
show_usage() {
    echo "Usage: $0 [OPTIONS]"
    echo ""
    echo "Options:"
    echo "  -h, --help     Show this help message"
    echo "  --api-only     Check API health only"
    echo "  --db-only      Check database health only"
    echo "  --quick        Run quick checks only"
    echo ""
    echo "Environment variables:"
    echo "  API_URL, DB_HOST, DB_PORT, DB_USER, DB_NAME, DB_PASSWORD"
    echo "  REDIS_HOST, REDIS_PORT, HEALTH_CHECK_TIMEOUT"
}

# Parse arguments
API_ONLY=false
DB_ONLY=false
QUICK=false

while [[ $# -gt 0 ]]; do
    case $1 in
        -h|--help)
            show_usage
            exit 0
            ;;
        --api-only)
            API_ONLY=true
            shift
            ;;
        --db-only)
            DB_ONLY=true
            shift
            ;;
        --quick)
            QUICK=true
            shift
            ;;
        *)
            echo "Unknown option: $1"
            show_usage
            exit 1
            ;;
    esac
done

# Execute based on options
if [ "$API_ONLY" = true ]; then
    check_api_health
elif [ "$DB_ONLY" = true ]; then
    check_database_health
elif [ "$QUICK" = true ]; then
    check_api_health
    echo ""
    check_database_health
else
    main
fi
