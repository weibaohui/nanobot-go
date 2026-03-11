import { test, expect } from '@playwright/test';

/**
 * CronJobs 定时任务管理页面测试
 */

test.test('CronJobs 管理', async ({ page }) => {
  test.beforeEach(async ({ page }) => {
    await page.goto('/cron');
    await page.waitForSelector('.ant-card', { timeout: 10000 });
  });

  test('页面加载成功', async ({ page }) => {
    await expect(page.locator('text=定时任务管理')).toBeVisible();
    await expect(page.locator('button:has-text("新建任务")')).toBeVisible();
    await expect(page.locator('.ant-table')).toBeVisible();
  });

  test('读取 CronJob 列表', async ({ page }) => {
    await expect(page.locator('th:has-text("ID")')).toBeVisible();
    await expect(page.locator('th:has-text("名称")')).toBeVisible();
    await expect(page.locator('th:has-text("来源渠道")')).toBeVisible();
    await expect(page.locator('th:has-text("Cron 表达式")')).toBeVisible();
    await expect(page.locator('th:has-text("模型配置")')).toBeVisible();
    await expect(page.locator('th:has-text("状态")')).toBeVisible();
  });
});
