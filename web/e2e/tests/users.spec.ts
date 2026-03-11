import { test, expect } from '@playwright/test';

/**
 * Users 用户管理页面 CRUD 测试
 */

test.test('Users 管理', () => {
  const TEST_USERNAME = `e2etest${Date.now()}`;
  const EDITED_DISPLAY_NAME = `E2E-Edited-User-${Date.now()}`;

  test.beforeEach(async ({ page }) => {
    // 每个测试前导航到 Users 页面
    await page.goto('/users');
    await page.waitForSelector('.ant-card', { timeout: 10000 });
  });

  test('页面加载成功', async ({ page }) => {
    // 验证页面标题
    await expect(page.locator('text=用户管理')).toBeVisible();

    // 验证新建按钮存在
    await expect(page.locator('button:has-text("新建用户")')).toBeVisible();

    // 验证表格存在
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('读取 User 列表', async ({ page }) => {
    // 验证表格列头
    await expect(page.locator('th:has-text("ID")')).toBeVisible();
    await expect(page.locator('th:has-text("用户名")')).toBeVisible();
    await expect(page.locator('th:has-text("邮箱")')).toBeVisible();
    await expect(page.locator('th:has-text("显示名称")')).toBeVisible();
    await expect(page.locator('th:has-text("状态")')).toBeVisible();
    await expect(page.locator('th:has-text("创建时间")')).toBeVisible();
    await expect(page.locator('th:has-text("操作")')).toBeVisible();
  });

  test('创建 User', async ({ page }) => {
    // 点击新建按钮
    await page.click('button:has-text("新建用户")');

    // 等待弹窗出现
    await page.waitForSelector('.ant-modal:has-text("新建用户")', { state: 'visible' });

    // 填写表单 - 使用 label 文本定位
    await page.fill('.ant-modal:has-text("新建用户") input#username', TEST_USERNAME);
    await page.fill('.ant-modal:has-text("新建用户") input#password', 'TestPassword123!');
    await page.fill('.ant-modal:has-text("新建用户") input#email', `${TEST_USERNAME}@example.com`);
    await page.fill('.ant-modal:has-text("新建用户") input#display_name', 'E2E Test User');

    // 提交表单
    await page.click('.ant-modal:has-text("新建用户") .ant-btn-primary');

    // 等待成功消息
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证列表中出现新用户
    await expect(page.locator(`text=${TEST_USERNAME}`)).toBeVisible();

    // 清理
    await page.click(`.ant-table-row:has-text("${TEST_USERNAME}") button:has-text("删除")`);
    await page.click('.ant-popover-buttons .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });
  });

  test('编辑 User', async ({ page }) => {
    // 先创建一个新用户
    await page.click('button:has-text("新建用户")');
    await page.waitForSelector('.ant-modal:has-text("新建用户")', { state: 'visible' });
    await page.fill('.ant-modal:has-text("新建用户") input#username', TEST_USERNAME);
    await page.fill('.ant-modal:has-text("新建用户") input#password', 'TestPassword123!');
    await page.click('.ant-modal:has-text("新建用户") .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 点击编辑按钮
    await page.click(`.ant-table-row:has-text("${TEST_USERNAME}") button:has-text("编辑")`);

    // 等待编辑弹窗
    await page.waitForSelector('.ant-modal:has-text("编辑用户")', { state: 'visible' });

    // 修改显示名称
    await page.fill('.ant-modal:has-text("编辑用户") input#display_name', EDITED_DISPLAY_NAME);

    // 提交
    await page.click('.ant-modal:has-text("编辑用户") .ant-btn-primary');

    // 等待成功消息
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证修改成功
    await expect(page.locator(`text=${EDITED_DISPLAY_NAME}`)).toBeVisible();

    // 清理
    await page.click(`.ant-table-row:has-text("${TEST_USERNAME}") button:has-text("删除")`);
    await page.click('.ant-popover-buttons .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });
  });

  test('删除 User', async ({ page }) => {
    // 先创建一个新用户
    await page.click('button:has-text("新建用户")');
    await page.waitForSelector('.ant-modal:has-text("新建用户")', { state: 'visible' });
    await page.fill('.ant-modal:has-text("新建用户") input#username', TEST_USERNAME);
    await page.fill('.ant-modal:has-text("新建用户") input#password', 'TestPassword123!');
    await page.click('.ant-modal:has-text("新建用户") .ant-btn-primary');
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 点击删除按钮
    await page.click(`.ant-table-row:has-text("${TEST_USERNAME}") button:has-text("删除")`);

    // 等待确认弹窗
    await page.waitForSelector('.ant-popover', { state: 'visible' });

    // 确认删除
    await page.click('.ant-popover-buttons .ant-btn-primary');

    // 等待成功消息
    await page.waitForSelector('.ant-message-success', { timeout: 10000 });

    // 验证用户已从列表中消失
    await expect(page.locator(`text=${TEST_USERNAME}`)).not.toBeVisible();
  });
});
