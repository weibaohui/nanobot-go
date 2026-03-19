#!/bin/bash
# API 端到端测试脚本

BASE_URL="http://localhost:8080"

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# 测试计数器
PASSED=0
FAILED=0

# 测试函数
test_endpoint() {
    local name="$1"
    local method="$2"
    local endpoint="$3"
    local data="$4"

    echo -n "${YELLOW}Testing $name...${NC} "

    if [ -n "$data" ]; then
        response=$(curl -s -X "$method" "$BASE_URL$endpoint" \
            -H "Content-Type: application/json" \
            -d "$data")
    else
        response=$(curl -s -X "$method" "$BASE_URL$endpoint")
    fi

    # 检查响应
    if echo "$response" | grep -q '"error"' || [ -z "$response" ]; then
        echo "${RED}FAILED${NC}"
        echo "Response: $response"
        ((FAILED++))
    else
        echo "${GREEN}PASSED${NC}"
        ((PASSED++))
    fi
}

echo "========================================"
echo "API 端到端测试"
echo "========================================"

# Health Check
test_endpoint "Health Check" "GET" "/health"

# Users
test_endpoint "List Users" "GET" "/api/v1/users?user_id=1"

# Agents
test_endpoint "List Agents" "GET" "/api/v1/agents?user_id=1"

# Channels
test_endpoint "List Channels" "GET" "/api/v1/channels?user_id=1"

# Sessions
test_endpoint "List Sessions" "GET" "/api/v1/sessions?user_id=1"

# Providers
test_endpoint "List Providers" "GET" "/api/v1/providers"
test_endpoint "Create Provider" "POST" "/api/v1/providers?user_id=1" \
    '{"provider_key":"test","provider_name":"Test Provider","api_key":"test-key"}'

# Cron Jobs
test_endpoint "List Cron Jobs" "GET" "/api/v1/cron-jobs"
test_endpoint "Create Cron Job" "POST" "/api/v1/cron-jobs?user_id=1" \
    '{"name":"Test Job","channel_id":1,"cron_expression":"0 9 * * *","prompt":"Test prompt"}'

# Conversation Records
test_endpoint "List Conversation Records" "GET" "/api/v1/conversations"

# Stream Memories
test_endpoint "List Stream Memories" "GET" "/api/v1/stream-memories"

# Long-term Memories
test_endpoint "List Long-term Memories" "GET" "/api/v1/long-term-memories"

echo "========================================"
echo "Test Summary:"
echo "========================================"
echo -e "${GREEN}Passed: $PASSED${NC}"
echo -e "${RED}Failed: $FAILED${NC}"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}Some tests failed!${NC}"
    exit 1
fi
