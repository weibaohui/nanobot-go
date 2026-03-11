import { test, expect } from '@playwright/test';

test.test('调试测试', () => {
  test('捕获网络日志和错误', async ({ page }) => {
    // 捕获控制台日志
    page.on('console', msg => {
      console.log(`CONSOLE: ${msg.type()}: ${msg.text()}`);
    });

    // 捕获页面错误
    page.on('pageerror', error => {
      console.log(`PAGE ERROR: ${error.message}`);
    });

    // 捕获网络请求
    page.on('request', request => {
      console.log(`REQUEST: ${request.method()} ${request.url()}`);
      console.log(`  Headers: ${JSON.stringify(request.headers())}`);
    });

    // 捕获网络响应
    page.on('response', response => {
      console.log(`RESPONSE: ${response.status()} ${response.url()}`);
    });

    await page.goto('/agents');
    await page.waitForSelector('.ant-card', { timeout: 10000 });

    // 点击新建按钮
    await page.click('button:has-text("新建 Agent")');
    await page.waitForSelector('.ant-modal', { state: 'visible' });

    // 填写表单
    await page.fill('input[placeholder="Agent 名称"]', 'Debug-Agent');
    await page.fill('textarea[placeholder="Agent 描述"]', '调试测试');

    // 点击提交
    await page.click('.ant-modal-footer .ant-btn-primary');

    // 等待一段时间看结果
    await page.waitForTimeout(3000);

    // 截图
    await page.screenshot({ path: 'debug-screenshot.png' });

    console.log('Test completed');
  });
});
