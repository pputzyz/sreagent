# SREAgent 自动化 QA 框架

## 架构

```
tests/qa/
├── playwright.config.ts    # Playwright 配置
├── fixtures/
│   ├── auth.ts             # 登录 fixture
│   └── api.ts              # API helper
├── helpers/
│   ├── test-parser.ts      # 解析复测手册 HTML
│   └── page-objects.ts     # 页面对象模式
├── specs/
│   ├── T1-alert-rules.spec.ts
│   ├── T2-alert-events.spec.ts
│   ├── T3-notifications.spec.ts
│   ├── T4-incidents.spec.ts
│   ├── T5-oncall.spec.ts
│   ├── T6-datasources.spec.ts
│   ├── T7-settings.spec.ts
│   ├── T8-ai.spec.ts
│   ├── T9-platform.spec.ts
│   ├── T10-dashboard.spec.ts
│   ├── T11-integrations.spec.ts
│   └── T12-frontend.spec.ts
└── reports/
    └── results.json
```

## 安装

```powershell
cd c:\project\sreagent\tests\qa
npm init -y
npm install -D @playwright/test
npx playwright install chromium
```

## 运行

```powershell
# 运行所有测试
npx playwright test

# 运行单个域
npx playwright test specs/T1-alert-rules.spec.ts

# 运行单个用例
npx playwright test -g "T1-1"

# 生成 HTML 报告
npx playwright test --reporter=html
```

## 测试策略

每个测试用例对应复测手册中的一个 ID（如 T1-1）：
1. 自动登录
2. 导航到目标页面
3. 执行操作步骤
4. 验证预期结果
5. 截图存档

## 优先级

先覆盖 P0 级别的冒烟测试（约 50 个），再逐步扩展到全量 1604 个。
