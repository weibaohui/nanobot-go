#!/bin/bash
# API 性能测试脚本

BASE_URL="http://localhost:8080"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

PASSED=0
FAILED=0
SLOW_COUNT=0
SLOW_THRESHOLD=500

echo "========================================"
echo "API 性能测试"
echo "========================================"
echo ""

# 测试函数
test_api() {
    local name="$1"
    local url="$2"
    local method="${3:-GET}"
    local data="$4"

    local start_time=$(date +%s)

    if [ "$method" = "POST" ] && [ -n "$data" ]; then
        result=$(curl -s -X POST "$BASE_URL$url" -H "Content-Type: application/json" -d "$data" 2>/dev/null)
    else
        result=$(curl -s "$BASE_URL$url" 2>/dev/null)
    fi

    local end_time=$(date +%s)
    local duration=$(( (end_time - start_time) * 1000 ))

    if echo "$result" | grep -q '"error"'; then
        echo -e "${RED}✗${NC} $name - ${RED}ERROR${NC} (${duration}ms)"
        echo "  Response: $result"
        ((FAILED++))
    else
        if [ $duration -gt $SLOW_THRESHOLD ]; then
            echo -e "${YELLOW}⚠${NC} $name - ${YELLOW}SLOW${NC} (${duration}ms)"
            ((SLOW_COUNT++))
        else
            echo -e "${GREEN}✓${NC} $name (${duration}ms)"
        fi
        ((PASSED++))
    fi
}

# 基础接口
echo -e "${BLUE}基础接口${NC}"
test_api "Health" "/health"

# 用户管理
echo -e "\n${BLUE}用户管理${NC}"
test_api "List Users" "/api/v1/users?user_id=1"

# Agent 管理
echo -e "\n${BLUE}Agent 管理${NC}"
test_api "List Agents" "/api/v1/agents?user_id=1"

# 渠道管理
echo -e "\n${BLUE}渠道管理${NC}"
test_api "List Channels" "/api/v1/channels?user_id=1"

# 会话管理
echo -e "\n${BLUE}会话管理${NC}"
test_api "List Sessions" "/api/v1/sessions?user_id=1"

# Provider 管理
echo -e "\n${BLUE}Provider 管理${NC}"
test_api "List Providers" "/api/v1/providers?user_id=1"
test_api "Create Provider" "/api/v1/providers?user_id=1" "POST" '{"provider_key":"perf-test","provider_name":"Perf Test"}'

# 定时任务
echo -e "\n${BLUE}定时任务${NC}"
test_api "List Cron Jobs" "/api/v1/cron-jobs?user_id=1"
test_api "Create Cron Job" "/api/v1/cron-jobs?user_id=1" "POST" '{"name":"Perf Job","channel_id":1,"cron_expression":"0 9 * * *","prompt":"Test"}'

# 对话记录
echo -e "\n${BLUE}对话记录${NC}"
test_api "List Conversations" "/api/v1/conversations?user_id=1"

# 短期记忆
echo -e "\n${BLUE}短期记忆${NC}"
test_api "List Stream Memories" "/api/v1/stream-memories"
test_api "Create Stream Memory" "/api/v1/stream-memories" "POST" '{"trace_id":"perf","session_key":"perf","content":"perf","summary":"perf","event_type":"test"}'

# 长期记忆
echo -e "\n${BLUE}长期记忆${NC}"
test_api "List Long-term Memories" "/api/v1/long-term-memories"
test_api "Create Long-term Memory" "/api/v1/long-term-memories" "POST" '{"date":"2024-03-12","summary":"perf","what_happened":"perf","conclusion":"perf","value":"perf"}'

echo ""
echo "========================================"
echo "测试报告"
echo "========================================"
echo -e "通过: ${GREEN}${PASSED}${NC}"
echo -e "失败: ${RED}${FAILED}${NC}"
echo -e "慢接口: ${YELLOW}${SLOW_COUNT}${NC} (>500ms)"
echo "========================================"

if [ $FAILED -eq 0 ]; then
    echo -e "${GREEN}所有测试通过!${NC}"
    exit 0
else
    echo -e "${RED}部分测试失败!${NC}"
    exit 1
fi
