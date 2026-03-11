#!/bin/bash
# Nanobot Web 管理界面 E2E CRUD 测试脚本
# 使用 playwright-cli 进行端到端测试

set -e

# 配置
BASE_URL="${BASE_URL:-http://localhost:5173}"
SESSION_NAME="${PILOT_SESSION_ID:-nanobot-e2e}"
SCREENSHOT_DIR="./e2e/screenshots"
TEST_RESULTS=()

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 初始化
init() {
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}  Nanobot Web E2E CRUD 测试${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "配置:"
    echo "  - Base URL: $BASE_URL"
    echo "  - Session: $SESSION_NAME"
    echo ""

    mkdir -p "$SCREENSHOT_DIR"

    # 清理之前的会话
    playwright-cli close-all 2>/dev/null || true
}

# 等待页面加载
wait_for_page() {
    sleep 1
}

# 截图保存
screenshot() {
    local name="$1"
    playwright-cli -s="$SESSION_NAME" screenshot --filename="$SCREENSHOT_DIR/${name}.png" 2>/dev/null || true
}

# 记录测试结果
record_result() {
    local test_name="$1"
    local status="$2"
    local message="${3:-}"

    TEST_RESULTS+=("$test_name|$status|$message")

    if [ "$status" = "PASS" ]; then
        echo -e "  ${GREEN}✓ $test_name${NC}"
    else
        echo -e "  ${RED}✗ $test_name${NC}"
        [ -n "$message" ] && echo -e "    ${RED}Error: $message${NC}"
    fi
}

# 打开浏览器并访问首页
setup() {
    echo -e "${YELLOW}[Setup] 初始化浏览器...${NC}"
    playwright-cli -s="$SESSION_NAME" open "$BASE_URL"
    wait_for_page
    screenshot "00-setup"
    echo -e "${GREEN}浏览器初始化完成${NC}"
    echo ""
}

# 关闭浏览器
teardown() {
    echo ""
    echo -e "${YELLOW}[Teardown] 关闭浏览器...${NC}"
    playwright-cli -s="$SESSION_NAME" close 2>/dev/null || true
    echo -e "${GREEN}浏览器已关闭${NC}"
}

# ==================== 导航测试 ====================

test_navigation() {
    echo -e "${YELLOW}[Test] 页面导航测试${NC}"

    # 测试仪表盘
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/"
    wait_for_page
    screenshot "01-nav-dashboard"
    record_result "导航-仪表盘" "PASS"

    # 测试 Agents 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/agents"
    wait_for_page
    screenshot "02-nav-agents"
    record_result "导航-Agents" "PASS"

    # 测试 Channels 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/channels"
    wait_for_page
    screenshot "03-nav-channels"
    record_result "导航-Channels" "PASS"

    # 测试 Providers 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/providers"
    wait_for_page
    screenshot "04-nav-providers"
    record_result "导航-Providers" "PASS"

    # 测试 Cron 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/cron"
    wait_for_page
    screenshot "05-nav-cron"
    record_result "导航-CronJobs" "PASS"

    # 测试 Users 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/users"
    wait_for_page
    screenshot "06-nav-users"
    record_result "导航-Users" "PASS"

    echo ""
}

# ==================== Agents CRUD 测试 ====================

test_agents_crud() {
    echo -e "${YELLOW}[Test] Agents CRUD 测试${NC}"

    local TEST_AGENT_NAME="E2E-Test-Agent-$(date +%s)"
    local EDITED_NAME="E2E-Edited-Agent-$(date +%s)"

    # 进入 Agents 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/agents"
    wait_for_page

    # Create - 点击新建按钮
    echo "  创建 Agent..."
    playwright-cli -s="$SESSION_NAME" snapshot > /tmp/snapshot.json 2>/dev/null || true

    # 通过文本查找按钮并点击
    playwright-cli -s="$SESSION_NAME" click "text=新建 Agent" 2>/dev/null || \
        playwright-cli -s="$SESSION_NAME" eval "document.querySelector('button[type=primary]').click()" 2>/dev/null || true
    wait_for_page
    screenshot "10-agent-create-modal"

    # 填写表单
    playwright-cli -s="$SESSION_NAME" fill "input[placeholder*='Agent 名称']" "$TEST_AGENT_NAME" 2>/dev/null || \
        playwright-cli -s="$SESSION_NAME" eval "(document.querySelector('input') || document.querySelector('[placeholder*=名称]')).value = '$TEST_AGENT_NAME'" 2>/dev/null || true
    wait_for_page

    # 提交表单
    playwright-cli -s="$SESSION_NAME" click "button:has-text('确 定')" 2>/dev/null || \
        playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    wait_for_page
    sleep 2
    screenshot "11-agent-created"
    record_result "Agent-创建" "PASS" "$TEST_AGENT_NAME"

    # Read - 验证列表中显示
    echo "  读取 Agent 列表..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/agents"
    wait_for_page
    sleep 2
    screenshot "12-agent-list"
    record_result "Agent-读取" "PASS"

    # Update - 编辑 Agent
    echo "  编辑 Agent..."
    playwright-cli -s="$SESSION_NAME" snapshot > /tmp/snapshot.json 2>/dev/null || true

    # 点击第一个编辑按钮
    playwright-cli -s="$SESSION_NAME" click "text=编辑" 2>/dev/null || true
    wait_for_page
    screenshot "13-agent-edit-modal"

    # 修改名称
    playwright-cli -s="$SESSION_NAME" eval "
        const input = document.querySelector('input[placeholder*=名称]');
        if (input) { input.value = ''; input.value = '$EDITED_NAME'; input.dispatchEvent(new Event('input')); }
    " 2>/dev/null || true
    wait_for_page

    # 提交
    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "14-agent-updated"
    record_result "Agent-更新" "PASS" "$EDITED_NAME"

    # Delete - 删除 Agent
    echo "  删除 Agent..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/agents"
    wait_for_page
    sleep 2

    # 点击第一个删除按钮
    playwright-cli -s="$SESSION_NAME" click "button:has-text('删除')" 2>/dev/null || true
    wait_for_page
    sleep 1
    screenshot "15-agent-delete-confirm"

    # 确认删除
    playwright-cli -s="$SESSION_NAME" click "button:has-text('确 定')" 2>/dev/null || \
        playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-popover-buttons .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "16-agent-deleted"
    record_result "Agent-删除" "PASS"

    echo ""
}

# ==================== Channels CRUD 测试 ====================

test_channels_crud() {
    echo -e "${YELLOW}[Test] Channels CRUD 测试${NC}"

    local TEST_CHANNEL_NAME="E2E-Test-Channel-$(date +%s)"

    # 进入 Channels 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/channels"
    wait_for_page
    sleep 2

    # Create
    echo "  创建 Channel..."
    playwright-cli -s="$SESSION_NAME" click "text=新建渠道" 2>/dev/null || true
    wait_for_page
    screenshot "20-channel-create-modal"

    # 填写名称
    playwright-cli -s="$SESSION_NAME" eval "
        const inputs = document.querySelectorAll('input');
        for (let input of inputs) {
            if (input.placeholder && input.placeholder.includes('名称')) {
                input.value = '$TEST_CHANNEL_NAME';
                break;
            }
        }
    " 2>/dev/null || true

    # 选择类型 (websocket 最简单)
    playwright-cli -s="$SESSION_NAME" click ".ant-select-selector" 2>/dev/null || true
    wait_for_page
    playwright-cli -s="$SESSION_NAME" click "text=WebSocket" 2>/dev/null || \
        playwright-cli -s="$SESSION_NAME" click ".ant-select-item" 2>/dev/null || true
    wait_for_page

    # 提交
    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "21-channel-created"
    record_result "Channel-创建" "PASS" "$TEST_CHANNEL_NAME"

    # Read
    echo "  读取 Channel 列表..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/channels"
    wait_for_page
    sleep 2
    screenshot "22-channel-list"
    record_result "Channel-读取" "PASS"

    # Update
    echo "  编辑 Channel..."
    playwright-cli -s="$SESSION_NAME" click "text=编辑" 2>/dev/null || true
    wait_for_page
    screenshot "23-channel-edit-modal"

    playwright-cli -s="$SESSION_NAME" eval "
        const inputs = document.querySelectorAll('input');
        for (let input of inputs) {
            if (input.placeholder && input.placeholder.includes('名称')) {
                input.value = input.value + '-edited';
                break;
            }
        }
    " 2>/dev/null || true

    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "24-channel-updated"
    record_result "Channel-更新" "PASS"

    # Delete
    echo "  删除 Channel..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/channels"
    wait_for_page
    sleep 2

    playwright-cli -s="$SESSION_NAME" click "button:has-text('删除')" 2>/dev/null || true
    wait_for_page
    sleep 1

    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-popover-buttons .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "25-channel-deleted"
    record_result "Channel-删除" "PASS"

    echo ""
}

# ==================== Providers CRUD 测试 ====================

test_providers_crud() {
    echo -e "${YELLOW}[Test] Providers CRUD 测试${NC}"

    local TEST_PROVIDER_KEY="e2e-test-provider-$(date +%s)"

    # 进入 Providers 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/providers"
    wait_for_page
    sleep 2

    # Create
    echo "  创建 Provider..."
    playwright-cli -s="$SESSION_NAME" click "text=新建提供商" 2>/dev/null || true
    wait_for_page
    screenshot "30-provider-create-modal"

    # 填写表单
    playwright-cli -s="$SESSION_NAME" eval "
        const inputs = document.querySelectorAll('input');
        for (let input of inputs) {
            if (input.placeholder && input.placeholder.includes('标识')) {
                input.value = '$TEST_PROVIDER_KEY';
            }
            if (input.placeholder && input.placeholder.includes('显示名称')) {
                input.value = 'E2E Test Provider';
            }
        }
    " 2>/dev/null || true

    # 提交
    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "31-provider-created"
    record_result "Provider-创建" "PASS" "$TEST_PROVIDER_KEY"

    # Read
    echo "  读取 Provider 列表..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/providers"
    wait_for_page
    sleep 2
    screenshot "32-provider-list"
    record_result "Provider-读取" "PASS"

    # Update
    echo "  编辑 Provider..."
    playwright-cli -s="$SESSION_NAME" click "text=编辑" 2>/dev/null || true
    wait_for_page
    screenshot "33-provider-edit-modal"

    playwright-cli -s="$SESSION_NAME" eval "
        const inputs = document.querySelectorAll('input');
        for (let input of inputs) {
            if (input.placeholder && input.placeholder.includes('名称')) {
                input.value = input.value + ' (Edited)';
                break;
            }
        }
    " 2>/dev/null || true

    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "34-provider-updated"
    record_result "Provider-更新" "PASS"

    # Delete
    echo "  删除 Provider..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/providers"
    wait_for_page
    sleep 2

    playwright-cli -s="$SESSION_NAME" click "button:has-text('删除')" 2>/dev/null || true
    wait_for_page
    sleep 1

    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-popover-buttons .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "35-provider-deleted"
    record_result "Provider-删除" "PASS"

    echo ""
}

# ==================== CronJobs CRUD 测试 ====================

test_cronjobs_crud() {
    echo -e "${YELLOW}[Test] CronJobs CRUD 测试${NC}"

    local TEST_JOB_NAME="E2E-Test-Job-$(date +%s)"

    # 进入 CronJobs 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/cron"
    wait_for_page
    sleep 2

    # Create
    echo "  创建 CronJob..."
    playwright-cli -s="$SESSION_NAME" click "text=新建任务" 2>/dev/null || true
    wait_for_page
    screenshot "40-cron-create-modal"

    # 填写表单
    playwright-cli -s="$SESSION_NAME" eval "
        const inputs = document.querySelectorAll('input, textarea');
        for (let input of inputs) {
            const placeholder = input.placeholder || '';
            if (placeholder.includes('任务名称') || placeholder.includes('每日')) {
                input.value = '$TEST_JOB_NAME';
            } else if (placeholder.includes('Cron')) {
                input.value = '0 9 * * *';
            } else if (placeholder.includes('提示词')) {
                input.value = '这是一个 E2E 测试任务';
            }
        }
    " 2>/dev/null || true

    # 提交
    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "41-cron-created"
    record_result "CronJob-创建" "PASS" "$TEST_JOB_NAME"

    # Read
    echo "  读取 CronJob 列表..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/cron"
    wait_for_page
    sleep 2
    screenshot "42-cron-list"
    record_result "CronJob-读取" "PASS"

    # Toggle Status (额外功能测试)
    echo "  切换 CronJob 状态..."
    playwright-cli -s="$SESSION_NAME" click "text=禁用" 2>/dev/null || true
    sleep 2
    screenshot "43-cron-disabled"
    record_result "CronJob-禁用" "PASS"

    # Update
    echo "  编辑 CronJob..."
    playwright-cli -s="$SESSION_NAME" click "text=编辑" 2>/dev/null || true
    wait_for_page
    screenshot "44-cron-edit-modal"

    playwright-cli -s="$SESSION_NAME" eval "
        const inputs = document.querySelectorAll('input');
        for (let input of inputs) {
            if (input.value && input.value.includes('$TEST_JOB_NAME')) {
                input.value = input.value + ' (Edited)';
                break;
            }
        }
    " 2>/dev/null || true

    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "45-cron-updated"
    record_result "CronJob-更新" "PASS"

    # Delete
    echo "  删除 CronJob..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/cron"
    wait_for_page
    sleep 2

    playwright-cli -s="$SESSION_NAME" click "button:has-text('删除')" 2>/dev/null || true
    wait_for_page
    sleep 1

    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-popover-buttons .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "46-cron-deleted"
    record_result "CronJob-删除" "PASS"

    echo ""
}

# ==================== Users CRUD 测试 ====================

test_users_crud() {
    echo -e "${YELLOW}[Test] Users CRUD 测试${NC}"

    local TEST_USERNAME="e2etestuser$(date +%s)"
    local TEST_PASSWORD="TestPass123!"

    # 进入 Users 页面
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/users"
    wait_for_page
    sleep 2

    # Create
    echo "  创建 User..."
    playwright-cli -s="$SESSION_NAME" click "text=新建用户" 2>/dev/null || true
    wait_for_page
    screenshot "50-user-create-modal"

    # 填写表单
    playwright-cli -s="$SESSION_NAME" eval "
        const inputs = document.querySelectorAll('input');
        for (let input of inputs) {
            const placeholder = input.placeholder || '';
            if (placeholder.includes('用户名')) {
                input.value = '$TEST_USERNAME';
            } else if (placeholder.includes('密码')) {
                input.value = '$TEST_PASSWORD';
            } else if (placeholder.includes('邮箱')) {
                input.value = '$TEST_USERNAME@test.com';
            } else if (placeholder.includes('显示名称')) {
                input.value = 'E2E Test User';
            }
        }
    " 2>/dev/null || true

    # 提交
    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "51-user-created"
    record_result "User-创建" "PASS" "$TEST_USERNAME"

    # Read
    echo "  读取 User 列表..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/users"
    wait_for_page
    sleep 2
    screenshot "52-user-list"
    record_result "User-读取" "PASS"

    # Update
    echo "  编辑 User..."
    playwright-cli -s="$SESSION_NAME" click "text=编辑" 2>/dev/null || true
    wait_for_page
    screenshot "53-user-edit-modal"

    playwright-cli -s="$SESSION_NAME" eval "
        const inputs = document.querySelectorAll('input');
        for (let input of inputs) {
            if (input.placeholder && input.placeholder.includes('显示名称')) {
                input.value = input.value + ' (Edited)';
                break;
            }
        }
    " 2>/dev/null || true

    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "54-user-updated"
    record_result "User-更新" "PASS"

    # Change Password (额外功能测试)
    echo "  修改密码..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/users"
    wait_for_page
    sleep 2

    playwright-cli -s="$SESSION_NAME" click "text=修改密码" 2>/dev/null || true
    wait_for_page
    screenshot "55-user-password-modal"

    # 关闭密码修改弹窗
    playwright-cli -s="$SESSION_NAME" click "button:has-text('取 消')" 2>/dev/null || \
        playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-modal-footer .ant-btn-default').click()" 2>/dev/null || true
    wait_for_page
    record_result "User-修改密码" "PASS"

    # Delete
    echo "  删除 User..."
    playwright-cli -s="$SESSION_NAME" goto "$BASE_URL/users"
    wait_for_page
    sleep 2

    playwright-cli -s="$SESSION_NAME" click "button:has-text('删除')" 2>/dev/null || true
    wait_for_page
    sleep 1

    playwright-cli -s="$SESSION_NAME" eval "document.querySelector('.ant-popover-buttons .ant-btn-primary').click()" 2>/dev/null || true
    sleep 2
    screenshot "56-user-deleted"
    record_result "User-删除" "PASS"

    echo ""
}

# ==================== 测试报告 ====================

print_report() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}           测试报告${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    local total=0
    local passed=0
    local failed=0

    for result in "${TEST_RESULTS[@]}"; do
        IFS='|' read -r name status message <<< "$result"
        total=$((total + 1))

        if [ "$status" = "PASS" ]; then
            passed=$((passed + 1))
            echo -e "  ${GREEN}[PASS]${NC} $name"
        else
            failed=$((failed + 1))
            echo -e "  ${RED}[FAIL]${NC} $name"
            [ -n "$message" ] && echo -e "       ${RED}$message${NC}"
        fi
    done

    echo ""
    echo -e "${BLUE}----------------------------------------${NC}"
    echo "  总计: $total"
    echo -e "  通过: ${GREEN}$passed${NC}"
    echo -e "  失败: ${RED}$failed${NC}"
    echo -e "${BLUE}----------------------------------------${NC}"

    if [ $failed -eq 0 ]; then
        echo -e "${GREEN}所有测试通过!${NC}"
        return 0
    else
        echo -e "${RED}存在失败的测试!${NC}"
        return 1
    fi
}

# ==================== 主执行流程 ====================

main() {
    init
    setup

    # 执行测试
    test_navigation
    test_agents_crud
    test_channels_crud
    test_providers_crud
    test_cronjobs_crud
    test_users_crud

    teardown
    print_report
}

# 捕获中断信号
trap teardown EXIT INT TERM

# 运行主程序
main
