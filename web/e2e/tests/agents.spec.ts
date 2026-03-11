import { test, expect } from '@playwright/test';

/**
 * Agents 管理页面 CRUD 测试
 */

test.test('Agents 管理', () => {
  const TEST_AGENT_NAME = `E2E-Test-Agent-${Date.now()}`;
  const EDITED_NAME = `E2E-Edited-Agent-${Date.now()}`;

  test.beforeEach(async ({ page }) => {
    // 每个测试前导航到 Agents 页面
    await page.goto('/agents');
    await page.waitForSelector('.ant-card', { timeout: 10000 });
  });

  test('页面加载成功', async ({ page }) => {
    // 验证页面标题
    await expect(page.locator('text=Agent 管理')).toBeVisible();

    // 验证新建按钮存在
    await expect(page.locator('button:has-text("新建 Agent")')).toBeVisible();

    // 验证表格存在
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('创建 Agent', async ({ page }) => {
    // 点击新建按钮
    await page.click('button:has-text("新建 Agent")');

    // 等待弹窗出现
    await page.waitForSelector('.ant-modal', { state: 'visible' });

    // 填写表单
    await page.fill('input[placeholder="Agent 名称"]', TEST_AGENT_NAME);
    await page.fill('textarea[placeholder="Agent 描述"]', '这是一个 E2E 测试创建的 Agent');

    // 选择模型选择模式（自动）
    await page.click('.ant-select-selector');
    await page.click('.ant-select-item:has-text("自动选择")');

    // 提交表单
    await page.click('.ant-modal-footer .ant-btn-primary');

    // 等待成功消息
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证列表中出现新 Agent
    await expect(page.locator(`text=${TEST_AGENT_NAME}`)).toBeVisible();
  });

  test('读取 Agent 列表', async ({ page }) => {
    // 验证表格列头
    await expect(page.locator('th:has-text("ID")')).toBeVisible();
    await expect(page.locator('th:has-text("名称")')).toBeVisible();
    await expect(page.locator('th:has-text("描述")')).toBeVisible();
    await expect(page.locator('th:has-text("模型配置")')).toBeVisible();
    await expect(page.locator('th:has-text("状态")')).toBeVisible();
    await expect(page.locator('th:has-text("操作")')).toBeVisible();

    // 验证操作按钮存在
    const firstRow = page.locator('.ant-table-row').first();
    await expect(firstRow.locator('button:has-text("编辑")')).toBeVisible();
    await expect(firstRow.locator('button:has-text("删除")')).toBeVisible();
  });

  test('编辑 Agent', async ({ page }) => {
    // 先创建一个新 Agent
    await page.click('button:has-text("新建 Agent")');
    await page.waitForSelector('.ant-modal', { state: 'visible' });
    await page.fill('input[placeholder="Agent 名称"]', TEST_AGENT_NAME);
    await page.click('.ant-modal-footer .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 点击编辑按钮
    await page.click(`.ant-table-row:has-text("${TEST_AGENT_NAME}") button:has-text("编辑")`);

    // 等待弹窗
    await page.waitForSelector('.ant-modal:has-text("编辑 Agent")', { state: 'visible' });

    // 修改名称
    await page.fill('input[placeholder="Agent 名称"]', EDITED_NAME);

    // 提交
    await page.click('.ant-modal-footer .ant-btn-primary');

    // 等待成功消息
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证修改成功
    await expect(page.locator(`text=${EDITED_NAME}`)).toBeVisible();

    // 清理：删除测试数据
    await page.click(`.ant-table-row:has-text("${EDITED_NAME}") button:has-text("删除")`);
    await page.click('.ant-popover-buttons .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });
  });

  test('删除 Agent', async ({ page }) => {
    // 先创建一个新 Agent
    await page.click('button:has-text("新建 Agent")');
    await page.waitForSelector('.ant-modal', { state: 'visible' });
    await page.fill('input[placeholder="Agent 名称"]', TEST_AGENT_NAME);
    await page.click('.ant-modal-footer .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 点击删除按钮
    await page.click(`.ant-table-row:has-text("${TEST_AGENT_NAME}") button:has-text("删除")`);

    // 等待确认弹窗
    await page.waitForSelector('.ant-popover', { state: 'visible' });

    // 确认删除
    await page.click('.ant-popover-buttons .ant-btn-primary');

    // 等待成功消息
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证 Agent 已从列表中消失
    await expect(page.locator(`text=${TEST_AGENT_NAME}`)).not.toBeVisible();
  });

  test('设置默认 Agent', async ({ page }) => {
    // 先创建一个新 Agent
    await page.click('button:has-text("新建 Agent")');
    await page.waitForSelector('.ant-modal', { state: 'visible' });
    await page.fill('input[placeholder="Agent 名称"]', TEST_AGENT_NAME);

    // 设置为默认
    await page.click('.ant-switch');

    await page.click('.ant-modal-footer .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证默认标签
    await expect(page.locator(`.ant-table-row:has-text("${TEST_AGENT_NAME}") .ant-tag:has-text("默认")`)).toBeVisible();

    // 清理
    await page.click(`.ant-table-row:has-text("${TEST_AGENT_NAME}") button:has-text("删除")`);
    await page.click('.ant-popover-buttons .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });
  });
});
