import { defineConfig, devices } from '@playwright/test';

/**
 * Nanobot Web E2E 测试配置
 * @see https://playwright.dev/docs/test-configuration
 */
export default defineConfig({
  testDir: './tests',

  /* 每个测试超时时间 */
  timeout: 30 * 1000,

  /* 完全并行运行测试 */
  fullyParallel: true,

  /* 如果之前测试失败，停止构建 */
  forbidOnly: !!process.env.CI,

  /* 重试次数 */
  retries: process.env.CI ? 2 : 0,

  /* 并行 workers 数 */
  workers: process.env.CI ? 1 : undefined,

  /* 报告器配置 */
  reporter: [
    ['html', { open: 'never' }],
    ['list'],
  ],

  /* 共享的测试配置 */
  use: {
    /* 基础 URL */
    baseURL: process.env.BASE_URL || 'http://localhost:5173',

    /* 收集所有跟踪信息 */
    trace: 'on-first-retry',

    /* 失败时截图 */
    screenshot: 'only-on-failure',

    /* 录制视频 */
    video: 'on-first-retry',

    /* 视口大小 */
    viewport: { width: 1280, height: 720 },

    /* 动作超时 */
    actionTimeout: 10 * 1000,

    /* 导航超时 */
    navigationTimeout: 10 * 1000,
  },

  /* 不同浏览器的项目配置 */
  projects: [
    {
      name: 'chromium',
      use: { ...devices['Desktop Chrome'] },
    },
    // 可以添加更多浏览器
    // {
    //   name: 'firefox',
    //   use: { ...devices['Desktop Firefox'] },
    // },
    // {
    //   name: 'webkit',
    //   use: { ...devices['Desktop Safari'] },
    // },
  ],

  /* 测试前启动开发服务器（可选） */
  // webServer: {
  //   command: 'cd .. && npm run dev',
  //   url: 'http://localhost:5173',
  //   reuseExistingServer: !process.env.CI,
  // },
});
