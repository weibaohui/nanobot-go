#!/bin/bash
# 前端端到端 API 集成测试

set -e

BASE_URL="http://localhost:8080"
FRONTEND_URL="http://localhost:5173"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

PASSED=0
FAILED=0

# 检查 API 响应格式
check_response() {
    local name="$1"
    local response="$2"

    echo -n "${YELLOW}Testing $name...${NC} "

    if echo "$response" | grep -q '"error"' || [ -z "$response" ]; then
        echo -e "${RED}FAILED${NC}"
        echo "Response: $response"
        ((FAILED++))
        return 1
    else
        echo -e "${GREEN}PASSED${NC}"
        ((PASSED++))
        return 0
    fi
}

echo "========================================"
echo "前端-后端集成测试"
echo "========================================"

# 1. 测试 Health Endpoint
echo -e "\n${YELLOW}=== 基础连接测试 ===${NC}"
RESPONSE=$(curl -s "$BASE_URL/health")
check_response "Health Check" "$RESPONSE"

# 2. 测试所有列表 API
echo -e "\n${YELLOW}=== 数据列表 API 测试 ===${NC}"

# Users
curl -s "$BASE_URL/api/v1/users?user_id=1" | grep -q '"items"' && echo -e "${GREEN}Users API OK${NC}" || echo -e "${RED}Users API Failed${NC}"

# Agents
curl -s "$BASE_URL/api/v1/agents?user_id=1" | grep -q '"items"' && echo -e "${GREEN}Agents API OK${NC}" || echo -e "${RED}Agents API Failed${NC}"

# Channels
curl -s "$BASE_URL/api/v1/channels?user_id=1" | grep -q '"items"' && echo -e "${GREEN}Channels API OK${NC}" || echo -e "${RED}Channels API Failed${NC}"

# Sessions
curl -s "$BASE_URL/api/v1/sessions?user_id=1" | grep -q '"items"' && echo -e "${GREEN}Sessions API OK${NC}" || echo -e "${RED}Sessions API Failed${NC}"

# Providers
curl -s "$BASE_URL/api/v1/providers?user_id=1" | grep -q '"items"' && echo -e "${GREEN}Providers API OK${NC}" || echo -e "${RED}Providers API Failed${NC}"

# Cron Jobs
curl -s "$BASE_URL/api/v1/cron-jobs?user_id=1" | grep -q '"items"' && echo -e "${GREEN}CronJobs API OK${NC}" || echo -e "${RED}CronJobs API Failed${NC}"

# Conversation Records
curl -s "$BASE_URL/api/v1/conversations?user_id=1" | grep -q '"items"' && echo -e "${GREEN}Conversations API OK${NC}" || echo -e "${RED}Conversations API Failed${NC}"

# Stream Memories
curl -s "$BASE_URL/api/v1/stream-memories" | grep -q '"items"' && echo -e "${GREEN}Stream Memories API OK${NC}" || echo -e "${RED}Stream Memories API Failed${NC}"

# Long-term Memories
curl -s "$BASE_URL/api/v1/long-term-memories" | grep -q '"items"' && echo -e "${GREEN}Long-term Memories API OK${NC}" || echo -e "${RED}Long-term Memories API Failed${NC}"

# 3. 测试 CRUD 操作
echo -e "\n${YELLOW}=== CRUD 操作测试 ===${NC}"

# Create Provider
PROVIDER_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/providers?user_id=1" \
    -H "Content-Type: application/json" \
    -d '{"provider_key":"test","provider_name":"Test Provider"}')
check_response "Create Provider" "$PROVIDER_RESPONSE"

# Create Cron Job
CRON_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/cron-jobs?user_id=1" \
    -H "Content-Type: application/json" \
    -d '{"name":"Test Job","channel_id":1,"cron_expression":"0 9 * * *","prompt":"Test"}')
check_response "Create Cron Job" "$CRON_RESPONSE"

# Create Stream Memory
STREAM_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/stream-memories" \
    -H "Content-Type: application/json" \
    -d '{"trace_id":"test-trace","session_key":"test-session","content":"Test content","summary":"Test summary","event_type":"test"}')
check_response "Create Stream Memory" "$STREAM_RESPONSE"

# Create Long-term Memory
LONGTERM_RESPONSE=$(curl -s -X POST "$BASE_URL/api/v1/long-term-memories" \
    -H "Content-Type: application/json" \
    -d '{"date":"2024-03-12","summary":"Test summary","what_happened":"Test happened","conclusion":"Test conclusion","value":"Test value"}')
check_response "Create Long-term Memory" "$LONGTERM_RESPONSE"

# 4. 验证前端页面可访问（如果前端正在运行）
echo -e "\n${YELLOW}=== 前端页面测试 ===${NC}"
if curl -s -o /dev/null -w "%{http_code}" "$FRONTEND_URL" | grep -q "200\|404"; then
    echo -e "${GREEN}Frontend is accessible${NC}"
else
    echo -e "${YELLOW}Frontend not running (expected if not started)${NC}"
fi

echo ""
echo "========================================"
echo -e "${GREEN}All backend API tests completed!${NC}"
echo "========================================"
