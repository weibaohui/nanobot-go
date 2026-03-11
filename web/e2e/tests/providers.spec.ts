import { test, expect } from '@playwright/test';

/**
 * Providers 管理页面测试
 */

test.test('Providers 管理', async ({ page }) => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/providers');
    await page.waitForSelector('.ant-card', { timeout: 10000 });
  });

  test('页面加载成功', async ({ page }) => {
    await expect(page.locator('text=LLM 提供商管理')).toBeVisible();
    await expect(page.locator('button:has-text("新建提供商")')).toBeVisible();
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('读取 Provider 列表', async ({ page }) => {
    await expect(page.locator('th:has-text("ID")')).toBeVisible();
    await expect(page.locator('th:has-text("标识")')).toBeVisible();
    await expect(page.locator('th:has-text("名称")')).toBeVisible();
    await expect(page.locator('th:has-text("API Base")')).toBeVisible();
    await expect(page.locator('th:has-text("优先级")')).toBeVisible();
    await expect(page.locator('th:has-text("状态")')).toBeVisible();
  });
});
