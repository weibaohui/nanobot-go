import { test, expect } from '@playwright/test';

/**
 * Channels 管理页面 CRUD 测试
 */

test.test('Channels 管理', () => {
  const TEST_CHANNEL_NAME = `E2E-Test-Channel-${Date.now()}`;
  const EDITED_NAME = `E2E-Edited-Channel-${Date.now()}`;

  test.beforeEach(async ({ page }) => {
    await page.goto('/channels');
    await page.waitForSelector('.ant-card', { timeout: 10000 });
  });

  test('页面加载成功', async ({ page }) => {
    await expect(page.locator('text=渠道管理')).toBeVisible();
    await expect(page.locator('button:has-text("新建渠道")')).toBeVisible();
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('读取 Channel 列表', async ({ page }) => {
    await expect(page.locator('th:has-text("ID")')).toBeVisible();
    await expect(page.locator('th:has-text("名称")')).toBeVisible();
    await expect(page.locator('th:has-text("类型")')).toBeVisible();
    await expect(page.locator('th:has-text("绑定 Agent")')).toBeVisible();
    await expect(page.locator('th:has-text("状态")')).toBeVisible();
    await expect(page.locator('th:has-text("操作")')).toBeVisible();
  });

  test('创建 WebSocket Channel', async ({ page }) => {
    await page.click('button:has-text("新建渠道")');
    await page.waitForSelector('.ant-modal:has-text("新建渠道")', { state: 'visible' });

    // 填写表单
    await page.fill('.ant-modal:has-text("新建渠道") input#name', TEST_CHANNEL_NAME);

    // 选择类型为 WebSocket
    await page.click('.ant-modal:has-text("新建渠道") .ant-select');
    await page.waitForSelector('.ant-select-dropdown', { state: 'visible' });
    await page.click('.ant-select-item:has-text("WebSocket")');
    await page.waitForTimeout(500);

    // 提交
    await page.click('.ant-modal:has-text("新建渠道") .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证
    await expect(page.locator(`text=${TEST_CHANNEL_NAME}`)).toBeVisible();

    // 清理
    await page.click(`.ant-table-row:has-text("${TEST_CHANNEL_NAME}") button:has-text("删除")`);
    await page.click('.ant-popover-buttons .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });
  });

  test('编辑 Channel', async ({ page }) => {
    // 创建
    await page.click('button:has-text("新建渠道")');
    await page.waitForSelector('.ant-modal:has-text("新建渠道")', { state: 'visible' });
    await page.fill('.ant-modal:has-text("新建渠道") input#name', TEST_CHANNEL_NAME);
    await page.click('.ant-modal:has-text("新建渠道") .ant-select');
    await page.waitForSelector('.ant-select-dropdown', { state: 'visible' });
    await page.click('.ant-select-item:has-text("WebSocket")');
    await page.waitForTimeout(500);
    await page.click('.ant-modal:has-text("新建渠道") .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 编辑
    await page.click(`.ant-table-row:has-text("${TEST_CHANNEL_NAME}") button:has-text("编辑")`);
    await page.waitForSelector('.ant-modal:has-text("编辑渠道")', { state: 'visible' });
    await page.fill('.ant-modal:has-text("编辑渠道") input#name', EDITED_NAME);
    await page.click('.ant-modal:has-text("编辑渠道") .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证
    await expect(page.locator(`text=${EDITED_NAME}`)).toBeVisible();

    // 清理
    await page.click(`.ant-table-row:has-text("${EDITED_NAME}") button:has-text("删除")`);
    await page.click('.ant-popover-buttons .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });
  });

  test('删除 Channel', async ({ page }) => {
    // 创建
    await page.click('button:has-text("新建渠道")');
    await page.waitForSelector('.ant-modal:has-text("新建渠道")', { state: 'visible' });
    await page.fill('.ant-modal:has-text("新建渠道") input#name', TEST_CHANNEL_NAME);
    await page.click('.ant-modal:has-text("新建渠道") .ant-select');
    await page.waitForSelector('.ant-select-dropdown', { state: 'visible' });
    await page.click('.ant-select-item:has-text("WebSocket")');
    await page.waitForTimeout(500);
    await page.click('.ant-modal:has-text("新建渠道") .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 删除
    await page.click(`.ant-table-row:has-text("${TEST_CHANNEL_NAME}") button:has-text("删除")`);
    await page.waitForSelector('.ant-popover', { state: 'visible' });
    await page.click('.ant-popover-buttons .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证
    await expect(page.locator(`text=${TEST_CHANNEL_NAME}`)).not.toBeVisible();
  });
});
