# 变更日志 (CHANGELOG)

> 基于 git tag 和 commit 记录整理。格式参考 [Keep a Changelog](https://keepachangelog.com/zh-CN/1.0.0/)

---

## [v4.10.5] — 2026-05-15

### Fixed — 配色系统重构 + 回退过度装饰

**配色重构（遵循 PRODUCT.md "Warm neutrality" 原则）：**
- 背景从 `#FFFBF7`（暖奶油色）改为 `#FAFAF9`（中性暖灰，stone-50）
- 主文本从 `#111827` 改为 `#1C1917`（stone-900，更深沉）
- 次要文本从 `#6b7280` 改为 `#57534E`（stone-600，对比度提升）
- 三级文本从 `#9ca3af` 改为 `#78716C`（stone-500，WCAG AA 达标）
- 静音文本从 `#d1d5db` 改为 `#A8A29E`（stone-400）
- Naive UI 主题覆盖同步更新

**回退违反 PRODUCT.md 的装饰性改动：**
- 移除 main-content 暖色径向渐变背景（anti-reference: decorative overload）
- 移除 dashboard 卡片 ::before 渐变覆盖层（anti-reference: gradient accent lines）
- 移除侧边栏图标浮动动画（anti-reference: animation for animation's sake）
- 保留：卡片 hover 浮起、交错入场动画、页面过渡（符合 "subtle spring easing"）

## [v4.10.4] — 2026-05-15

### Changed — UI 视觉提升 + AI 聊天框全屏

**AI 聊天框：**
- 新增全屏切换按钮（右上角最大化/最小化图标）
- 全屏模式下 drawer 占满整个视口宽度
- i18n 新增 `fullscreen` / `exitFullscreen` 键

**Logo 重新设计：**
- 背景从绿色渐变改为暖橙色系（`#FB923C` → `#F97316` → `#EA580C`）
- 添加 SVG glow 滤镜增强 "S" 字母质感
- 告警圆点改为金黄色 `#FBBF24` + 脉冲光环

**头像库美化：**
- PetAvatar 所有 8 种宠物添加柔和渐变背景圆
- 改进阴影效果：双层 drop-shadow 增强立体感
- UserAvatar SVG 添加 radialGradient 高光 + 增强阴影

**动画与过渡：**
- Dashboard bento 卡片添加交错入场动画（400ms，60ms 间隔）
- 卡片 hover 增加暖橙渐变光泽覆盖层
- 页面过渡改为 scale + opacity 组合（280ms spring easing）
- 主内容区添加微妙暖色径向渐变背景
- 侧边栏活跃图标增加浮动动画（3s 周期）

## [v4.10.3] — 2026-05-15

### Fixed — 字体离线化

- 移除 Google Fonts 外部依赖（Plus Jakarta Sans），改用本地 `@fontsource/inter`
- `index.html` 删除 `fonts.googleapis.com` 和 `fonts.gstatic.com` preconnect/link 标签
- `global.css` 字体栈更新：Inter（本地）+ Segoe UI + 系统字体回退
- `App.vue` Naive UI 主题 fontFamily 同步更新

## [v4.10.0] — 2026-05-14

### Changed — 全平台 UI 视觉重构：暖橙主题 + Bento 布局 + 春季动画

**配色系统重构：**
- 全局主色从 teal `#0d9488` 迁移至暖橙 `#F97316`
- App.vue Naive UI 主题覆盖全部使用暖橙色系
- 新增伴侣色 CSS 变量：`--sre-rose-light`、`--sre-emerald-light`、`--sre-violet-light`、`--sre-mint`
- 侧边栏 accent-soft RGBA 值对齐新品牌色（oncall `#F43F5E`、alert `#3B82F6`、platform `#8B5CF6`）
- mascot-fox.svg 内耳颜色从 `#0d9488` 更新为 `#F97316`
- UserAvatar.vue 调色板更新为新品牌色

**布局重构：**
- Index.vue 和 IncidentDashboard.vue 改为 12 列 CSS Grid bento 布局
- 卡片移除装饰性渐变 `::before` 线条
- 响应式断点：1200px（8+4）、768px（单列）

**动画系统：**
- 新增 `--sre-ease-spring: cubic-bezier(0.34, 1.56, 0.64, 1)` 弹簧缓动
- AppRail 图标 hover 弹跳放大 + active 缩放
- AppSidebar 指示器 scaleY 入场动画 + `transform-origin: top`
- AppShell 页面过渡改为 opacity + transform

**可访问性：**
- Index.vue action-btn/rule-item 添加 `role="button"` + `aria-label`
- tooltip 从硬编码 `#1C1917` 改为 CSS 变量，适配深色模式
- `.sev-seg` transition 从 `flex` 改为 `flex-grow` 避免布局抖动

**其他：**
- AppShell topbar 添加暖色阴影
- AppShell/AppSidebar 使用 display font
- Login.vue mesh blob 更新为暖橙色调
- i18n 补齐 `dashboard.quickActions` 中英文 key
- App.vue border-radius 添加 CSS 变量关联注释

---

## [v4.9.7] — 2026-05-13

### Fixed — 协作空间国际化 + 状态页面 CRUD

**协作空间 (Channels) 国际化：**
- 移除 Detail.vue 中 `|| 'Overview'` / `|| 'Settings'` 硬编码回退
- 修复 `auto_close_minutes` 后硬编码 "min"，改为 i18n `autoCloseMinutesUnit`
- 补齐 `channel.deleteDesc` 中英文 key

**状态页面 (StatusPage) CRUD 管理：**
- 新增"管理服务"入口，支持直接在状态页面增删改查服务
- 管理模态框：服务列表 + 编辑/删除操作
- 创建/编辑表单：名称、状态、描述、图标、链接、排序
- 空状态直接提供"添加服务"按钮
- statusOptions 改为 i18n computed
- 新增 13 个 statusPageModule 中英文 key

## [v4.9.6] — 2026-05-13

### Fixed — 状态页面 + AI 聊天面板 + 国际化清理

**StatusPage：**
- 移除"即将上线"预览文案，改为展示真实服务状态
- 移除 feature cards 营销内容
- 改为"订阅通知"文案

**AIChatPanel：**
- 重构布局：chat-body flex column 结构，消息区自动撑满
- 输入框区域固定在底部，带 border-top 分隔
- textarea autosize 调整为 minRows:2 / maxRows:6

**国际化：**
- 清理 10+ 处不必要的 `|| 'fallback'` 模式
- 修复 2 处硬编码 aria-label（Search / Change language）
- 补齐 statusPageModule 新键（currentStatus / subscribe / noServicesHint）

## [v4.9.5] — 2026-05-13

### Security — 安全加固 + 代码审查 + 品牌重命名

**安全修复：**
- 修复 JWT ParseTokenIgnoreExpiry 未验证签名算法（CRITICAL）
- 修复 OIDC CallbackJSON state 验证可选导致 CSRF 绕过（HIGH）
- 新增全局请求体大小限制 10MB（HIGH）
- 新增 Datasource Endpoint SSRF 防护（MEDIUM）
- 移除 Webhook Secret URL 查询参数支持（MEDIUM）
- 修复 Integration Token 日志泄露（MEDIUM）
- 修复 Lark Bot Config Secret 部分泄露（MEDIUM）

**代码质量：**
- 修复 GetByFingerprint 缺少软删除过滤
- 修复 Goroutine 使用 request context 导致后台任务中断
- 删除死代码：itoa() 包装函数、废弃 fmt.Sprintf
- JWT 签名方法验证添加 fmt 导入

**UI/前端：**
- 品牌重命名：Nexus → SREAgent（7 处）
- 更新 tagline：一站式 DevOps 解决方案平台 · 具身 Agent
- 补齐 en.ts rolesModule 缺失翻译键（14 个）
- 补齐 dashboard.lastSync、profile.selectAvatar 等缺失翻译
- 4 处 window.confirm 替换为 Naive UI useDialog
- AppShell 添加 768px 响应式断点（移动端隐藏侧边栏）

## [v4.9.2] — 2026-05-13

### Fixed — 代码审查修复 + 公开状态页端点

- 移除 global.css 中已删除的 sre-jelly 动画残留引用
- 补全 en.ts statusPageModule + rolesModule 英文翻译
- 补全 zh-CN.ts statusPageModule.invalidEmail 翻译键
- 修复 bizGroup.created 重复键（toast 改为 createSuccess）
- 新增公开端点 `GET /api/v1/status-services`（无需认证）
- 修复 StatusService.GetByID 错误遮蔽（区分 not-found 和 DB 错误）
- 修复 StatusPage.vue 验证消息键 + 移除内联 fallback
- AIChatButton 补充呼吸灯动画效果

## [v4.8.0] — 2026-05-12

### Added — AI Chat + 宠物系统 + UI 动画系统

**AI Chat（AI 对话）：**
- 右下角浮动按钮，点击展开 400px 右侧抽屉
- 三种模式：告警分析 / 通用对话 / 宠物对话
- 多轮对话上下文（自动加载最近 20 条历史）
- 后端：`POST /ai/chat`、`GET /ai/history`、`DELETE /ai/history`
- 新增 `chat_histories` 表（迁移: 000034）
- 组件：AIChatPanel、AIChatMessage、AIChatButton
- Composable：useAIChat

**宠物系统（Pet System）：**
- 狐狸宠物，自动创建（名字"小狐"）
- 喂食（饥饿 -20，经验 +5）、玩耍（心情 +15，经验 +5）
- 升级公式：所需经验 = 等级 × 100
- 互动历史记录
- 角落常驻显示（PetCorner）+ 弹出面板（PetPanel）+ 独立详情页（/pet）
- 后端：`GET/PUT /pet`、`POST /pet/feed`、`POST /pet/play`、`GET /pet/interactions`
- 新增 `pets` + `pet_interactions` 表（迁移: 000035）
- Pinia store：usePetStore

**UI 动画系统：**
- 页面切换动画：fade + translateY(8px)
- 卡片入场 stagger 动画（`.stagger-card`）
- 列表行入场 stagger 动画（`.stagger-row`）
- Per-app 背景色彩染（5%）：`.bg-app-oncall`、`.bg-app-alert`、`.bg-app-platform`
- Rail 图标从 20px 放大到 24px
- 卡片圆角从 12px 微调到 10px

**i18n：**
- 新增 `ai.*` 和 `pet.*` 中英文翻译

### 技术细节

- 迁移: 000034 (chat_histories)、000035 (pets + pet_interactions)
- 新增 composable: useAIChat
- 新增 Pinia store: usePetStore
- 新增 8 个 Vue 组件（ai/ 3 个 + pet/ 3 个 + pages/pet/ 1 个 + composable 1 个）

---

## [v4.7.1] — 2026-05-12

### Changed — UI 视觉深度 + 交互反馈 + 图标优化

**卡片视觉层级：**
- 全局阴影从极淡 (`0 1px 2px rgba(0,0,0,0.05)`) 升级到可见深度
- 新增 `--sre-shadow-lift` 用于 hover 浮起效果
- 所有卡片 hover 时 `translateY(-1px)` 浮起 + 阴影增强
- 涉及：surface-card、surface-clay、surface-glass、content-card、sre-row-card、sre-notify-card、sre-lift

**间距节奏优化：**
- 新增 `--sre-card-pad-compact: 16px`（KPI 卡片、列表行）
- 新增 `--sre-card-pad-relaxed: 24px`（图表卡片、内容区域）
- dashboard KPI 卡片更紧凑，图表区域更舒展
- incidents/alerts 列表行间距用语义化 token

**交互细节：**
- 图标按钮 hover 时 `scale(1.05)` 微放大
- 侧边栏菜单项 hover 时 `translateX(2px)` 微右移 + 文字颜色变深
- 选中菜单项左边框指示器加宽 + 发光效果
- rail 按钮 active 状态加 inset border ring

**侧边栏 hover 展开恢复：**
- 恢复被误删的 hover 展开功能（collapsed 状态下鼠标悬停临时展开）
- 新增 `pinned` 状态管理

**图标更换：**
- On-Call: `FlashOutline`（闪电）→ `CallOutline`（电话）— 更直观
- Alert: `AlertCircleOutline`（感叹号）→ `NotificationsOutline`（铃铛）— 更显眼

**配色：**
- per-app 背景色从 3% 提升到 5%（更易区分 oncall/alert/platform）

---

## [v4.7.0] — 2026-05-12

### Changed — UI 全面重构：去 AI 味 + 无障碍 + 交互优化

**Phase 1 · quieter — 去除 AI slop：**
- Login.vue：删除 mesh blob 动画背景、gradient-text 渐变文字
- global.css：bounce easing (`cubic-bezier(0.34,1.56,0.64,1)`) → 指数减速 (`cubic-bezier(0.16,1,0.3,1)`)
- 删除 `sre-bounce-in`、`sre-glow-pulse`、`sre-rail-active-pulse` 等过度动画

**Phase 2 · typeset + colorize — 字体与主题统一：**
- 字体：Plus Jakarta Sans → Inter（`@fontsource/inter` 本地加载）
- 暗色主题：暖棕 stone → 冷蓝灰 navy（`#0a1018`、`rgba(15,23,42,0.65)`）
- 浅色主题：stone-50 `#fafaf9` → slate-50 `#f8fafc`
- 图表配色：explore、dashboard、QueryResultChart 统一为 slate 色系

**Phase 3 · layout — 列表行简化：**
- incidents/Index.vue：3 行布局 → 2 行（标题行 + 元数据行），操作移入 hover dropdown
- alerts-v2/Index.vue：同上，删除冗余 severity 文字，添加 status pill

**Phase 4 · distill — 侧边栏精简：**
- AppShell.vue：删除 hover-expand 行为（`handleNavEnter/Leave`、`hoverTimeout`、`pinned`）
- AppSidebar.vue：删除 `pinned` prop，简化为纯点击切换

**Phase 5 · harden — 键盘快捷键 + 无障碍：**
- AppShell.vue：添加 skip-to-content 链接、`aria-label`、命令面板动作（主题切换、App 切换）
- 为 icon-only 按钮添加 `aria-label`

**Phase 6 · onboard — 空状态引导：**
- i18n：新增 incidents/alerts/channels 空状态描述文案 + 筛选提示
- 列表空状态：有筛选时显示"无匹配结果"，无筛选时显示功能介绍

**Phase 7 · polish — 最终打磨：**
- 全局 bounce easing → exponential ease-out（9 处修复）
- schedule shift blocks：`border-left: 4px` → 背景色着色
- AuditLog：`border-left: 2px` → `border: 1px`
- AppSidebar width transition easing 修正
- 反模式扫描：12 → 2（剩余 2 个 P3 可接受权衡）

---

## [v4.6.0] — 2026-05-11

### Changed — 设计系统 v5.0：Clean Neutral + Agent Review 修复

**global.css 设计系统重构：**
- 字体从 Plus Jakarta Sans（未加载）改为 Inter（Google Fonts 正确加载）
- 配色从暖色 stone 改为中性灰色（Tailwind gray scale）
- 阴影从 claymorphism 改为 clean Tailwind-style shadows
- 去除所有 clay-shadow、gradient-rainbow、gradient-brand、ease-bounce、hover-nudge 等过时变量
- Dark theme 修复：`--sre-success-soft` 从错误的 teal 改为正确的 green
- 新增 `--sre-shadow-glow` 到 `:root`（之前仅在 dark theme 定义）
- 删除 ~190 行死 CSS（与 AppRail/AppSidebar scoped 样式重复）
- 删除未使用的 `.stagger-list`（无 nth-child 延迟规则）
- 统一 surface 类：`.surface-clay`、`.surface-glass` 改用标准阴影

**Login.vue 安全修复：**
- 修复开放重定向漏洞：`redirect` 参数现在验证以 `/` 开头且不以 `//` 开头
- Mesh blob 性能优化：blur 移到静态父元素，子元素仅动画 transform
- 新增 `prefers-reduced-motion` 媒体查询保护
- `langOptions` 改为 `computed()` 以响应 locale 变化

**Dashboard 类型安全：**
- IncidentDashboard 的 4 个 `ref<any>` 改为完整接口定义
- Index.vue 的 `KpiDef.icon` 从 `any` 改为 `Component`
- Index.vue 的 `setInterval` 移入 `onMounted` 并在 `onUnmounted` 清理
- 修复硬编码英文字符串（"published"、"Triggered"、"Closed"）改用 i18n

**AppShell 安全修复：**
- localStorage JSON.parse 包装在 try/catch 中防止崩溃
- hoverTimeout 在 onUnmounted 时清理

**后端安全修复：**
- SSRF 修复：`datasource.go` 的 `LabelValues` 使用 `url.PathEscape()` 编码用户输入
- Goroutine 泄漏修复：`alert_event.go` 添加 100 容量信号量限制并发 dispatch
- 静默错误修复：`incident.go` 的 11 处 `_ = s.repo.AddTimeline(...)` 改为错误日志记录

---

## [v4.5.0] — 2026-05-11

### Changed — UI 精细化：去掉 AI 风，走精致现代路线

**设计方向调整（基于 ui-ux-pro-max 指导）：**
- Style: Soft UI Evolution — 柔和阴影、改进对比度、现代美学
- 去掉所有"AI 风"元素：彩虹渐变、弹跳动画、彩色圆形图标背景

**AppRail 重写：**
- 去掉彩色渐变圆形图标背景，改为简洁单色图标
- 新增小色点指示器（6px dot）标识当前 active app
- 去掉弹跳/浮动/jelly 动画，改为简单的背景色切换
- 用户头像去掉彩虹渐变光环

**AppSidebar 修复：**
- collapsed 且未 pin 时完全隐藏 nav（opacity: 0 + pointer-events: none）
- 解决了 collapsed 状态下菜单项堆叠的问题
- 菜单项 hover 去掉 translateX 移动，改为简单背景色变化
- 选中指示器从渐变色改为纯色
- border 从 2px 改为 1px

**AppShell 更新：**
- topbar 去掉彩虹渐变底线
- border 从 2px 改为 1px
- 新增 per-app 背景色区分：main-content 根据 data-app 应用微妙色调
  - On-Call: 微妙珊瑚色调
  - Alert: 微妙蓝色调
  - Platform: 微妙紫色调

**登录页精简：**
- 去掉彩虹渐变细线
- 登录按钮改为标准主题色（去掉彩虹渐变）
- 卡片动画从弹跳改为简单 fade + translateY
- 阴影从 claymorphism 改为柔和 shadow

**global.css 清理：**
- 删除过度动画 keyframes：bounce-hover、wiggle、float、jelly、color-breathe
- 删除工具类：.hover-bounce、.hover-wiggle、.hover-float、.jelly
- 页面过渡从 scale + translateY 改为简单 translateY
- 所有 2px border 改为 1px，16px radius 改为 12px
- 暗色主题 gradient mesh 透明度降低（更微妙）

---

## [v4.4.0] — 2026-05-11

## [v4.4.0] — 2026-05-11

### Changed — 视觉重构：Vibrant Clay 色彩丰富的 DevOps 平台

**配色系统重构：**
- 去掉单色 teal 贯穿全部，改为每个功能区独立主色
  - On-Call: 珊瑚橙 `#FF6B6B` → 暖橙 `#FFA07A`
  - Alert: 电光蓝 `#4FACFE` → 青色 `#00F2FE`
  - Platform: 梦幻紫 `#A855F7` → 粉紫 `#D946EF`
- 新增彩虹渐变 `--sre-gradient-rainbow` 用于强调元素
- 新增 Claymorphism 阴影变量（双重阴影 + 厚边框 + 大圆角）

**动画系统：**
- 新增 keyframes：`sre-bounce-hover`、`sre-wiggle`、`sre-float`、`sre-jelly`、`sre-gradient-flow`、`sre-color-breathe`
- 新增工具类：`.hover-bounce`、`.hover-wiggle`、`.hover-float`、`.jelly`、`.gradient-flow`
- 页面过渡：fade + scale(0.97) 组合

**AppRail 重写：**
- 图标包裹在彩色渐变圆形背景中（每个 app 独立颜色）
- hover 时放大 + 阴影增强
- active 时浮动动画 + 彩色光晕
- 用户头像外加彩虹渐变光环

**AppSidebar 更新：**
- 去掉 glassmorphism backdrop-filter，改为 claymorphism
- 新增 `data-app` 属性 + `--sidebar-accent` CSS 变量
- 选中菜单项左侧彩色渐变指示器
- hover 时菜单项微移 + 彩色背景
- app 名称使用对应 app 颜色

**AppShell 更新：**
- topbar 去掉 glassmorphism，改为 claymorphism + 彩虹渐变底线
- main-header 去掉 glassmorphism

**登录页重设计：**
- 背景 blob 颜色从 teal/blue/amber 改为 coral/blue/purple
- 登录卡片去掉 glassmorphism，改为 claymorphism
- 卡片顶部新增彩虹渐变细线（3px）
- 登录按钮改为彩虹渐变背景 + hover 弹跳动画
- 表单入场动画改为弹跳效果
- 卡片入场改为弹跳动画

**品牌更新：**
- tagline 从"一站式可观测性与事件响应平台"改为"一站式 DevOps 平台"

---

## [v4.3.0] — 2026-05-11

### Changed — 视觉系统升级：Glassmorphism + OLED 深色主题

**Dark Theme 全面升级：**
- 背景色从暖石色调（#0c0a09）切换到深邃 OLED 蓝黑色（#050a14）
- 新增 `--sre-gradient-mesh` 多色径向渐变用于页面背景
- 新增 `--sre-shadow-glow` 品牌色辉光阴影
- 文本色从暖白（#fafaf9）切换到冷白（#f1f5f9），层次更分明

**Glassmorphism 全面应用：**
- 所有 surface（glass/clay/card）加入 `backdrop-filter: blur()` 毛玻璃效果
- 卡片、行卡、通知卡、配置区块全部升级为半透明毛玻璃
- hover 时增加微妙的 `box-shadow` 提升层次感

**组件视觉升级：**
- AppRail：毛玻璃背景，active 图标带辉光效果（`box-shadow: 0 0 16px -4px`）
- AppSidebar：毛玻璃背景 + 玻璃边框
- AppShell topbar：毛玻璃背景
- main-content 区域叠加 mesh 渐变背景
- 登录页：完全重设计 — 全屏动画渐变 mesh 背景 + 居中 glass card

**登录页重设计：**
- 去掉 60/40 分栏布局，改为全屏居中 glass card
- 三个动画渐变色球（teal/blue/amber）做 mesh 背景
- card 带 24px blur + 品牌色辉光阴影
- logo + 品牌名移入 card 顶部
- 底部 footer 显示版本号 + 系统状态

---

### Fixed — 头像裂图 + 登录页清理

- 头像图片加载失败时自动 fallback 到首字母显示
- 登录后跳转从 `/dashboard` 修正为 `/oncall/overview`
- 移除登录页暴露的默认凭据（admin/admin123）
- 提示文字改为低调灰色，不再用警告框

---

## [v4.2.0] — 2026-05-11

### Changed — 布局重构：修复 6 个用户反馈问题

**左侧图标栏：**
- 去掉 MascotFox，改为用户头像（NAvatar）+ 点击弹出菜单
- 弹出菜单包含：个人信息、修改密码、退出登录
- 图标加大（20→22px，按钮 36→40px），active 状态加左侧 accent bar
- Tooltip 增强：显示名称+描述

**侧边栏：**
- 去掉底部用户信息区（头像、名字、角色、折叠按钮、版本号）
- 顶部加应用名 + 折叠按钮（`|<`）
- 鼠标移入导航区自动展开，移出 300ms 后自动收起
- 支持「固定」模式（点击折叠按钮切换固定/自动）

**个人信息页：**
- 加 NTabs：基本信息 + 修改密码
- 底部加退出登录按钮（危险区）

**新增组件：**
- `ChangePasswordModal.vue` — 密码修改弹窗

**CSS 增强：**
- 图标栏 active 状态 inset shadow、hover shadow
- 用户弹出菜单完整样式
- nav-zone flex 布局

---

## [v4.1.1] — 2026-05-11

### Fixed — 内网访问页面卡住

- 移除 `index.html` 中 Google Fonts CDN 外链（fonts.googleapis.com）
- 内网环境下 DNS 解析/连接超时会阻塞页面渲染
- 改用系统字体栈 fallback，PingFang SC / Microsoft YaHei 等

---

## [v4.1.0] — 2026-05-11

### Added — UI 视觉美化层

**交互效果：**
- `v-ripple` 全局点击涟漪指令
- 菜单 hover 微位移动画 + 激活条入场动画
- 导轨图标按压缩放 + 弹性回弹 + 激活脉冲
- 页面切换 fade-slide 过渡动画

**视觉细节：**
- hover-reveal 滚动条（5px，平滑过渡）
- CSS 自定义属性扩展（--sre-ripple-color, --sre-hover-nudge-x 等）

**萌宠组件：**
- MascotFox 狐狸萌宠（挥手/发呆/睡觉三状态 + 点击彩蛋）
- 集成至左侧导轨底部

---

## [v4.0.0] — 2026-05-11

### Changed — 应用架构重构：三栏布局 + 三应用分区

**导航重构：**
- 顶部 Tab（Monitor/Incident/System）改为左侧图标栏（On-Call/Alert/Platform）
- 新增三栏布局：图标栏（48px）+ 菜单栏（220px）+ 内容区
- 菜单栏支持多级折叠
- 个人中心从弹窗改为独立页面

**路由重组：**
- 新增 `/oncall/` 前缀：故障响应相关页面
- 新增 `/alert/` 前缀：告警引擎 + 通知管道页面
- 新增 `/platform/` 前缀：平台管理页面
- 所有旧路由添加向后兼容重定向

**新增页面：**
- 状态页面（On-Call）— 即将上线
- 角色权限（Platform）— 即将上线
- 个人中心（Platform）— 独立页面替代弹窗

**文件变更：**
- 新增：`useAppNav.ts`, `AppShell.vue`, `AppRail.vue`, `AppSidebar.vue`
- 删除：`MainLayout.vue`（750行）
- 新增 11 个页面组件（3 stub + 1 profile + 7 wrapper）
- 修改：路由、命令面板、i18n

---

## [v3.1.0] — 2026-05-11

### Changed — UI 重构：Soft Warm 清新温暖设计风格

**设计系统重写 (global.css)：**
- 主色调从 green-500 `#22c55e` 改为 teal-600 `#0d9488`
- 强调色从 indigo-500 `#6366f1` 改为 amber-500 `#f59e0b`
- 默认主题从 Dark 切换为 Light（暖白 `#fafaf9`）
- Dark 模式改为 opt-in，使用 stone 色系暖灰（`#1c1917`/`#292524`/`#44403c`）
- 字体从 Inter 改为 Plus Jakarta Sans（Google Fonts 加载）
- 圆角 lg 从 10px 增大到 12px，xl 从 14px 到 16px，2xl 从 20px 到 24px
- 阴影改为更柔和的低透明度版本（适配浅色背景）

**主题覆盖 (App.vue)：**
- Naive UI `darkOverrides` / `lightOverrides` 全面对齐新色板
- 默认 `isDark` 从 `true` 改为 `false`
- `applyBodyClass` 切换为 `dark-theme` / `light-theme` body class

**硬编码颜色迁移（12 个文件）：**
- `notification/Subscribe.vue` + `Rules.vue`：紫色/绿色硬编码 → CSS 变量
- `dashboard/Index.vue`：ECharts 图表主题从冷灰改为暖石色
- `explore/Index.vue` + `Login.vue`：`#fff`/`#666` → CSS 变量
- `schedule/Index.vue`：box-shadow → `var(--sre-shadow-xs)`
- `CommandPalette.vue`：`#f8f9fa` → `var(--sre-bg-card)`
- `PromQLEditor.vue`：边框/焦点色 → CSS 变量
- `QueryResultChart.vue` + `TimeRangePicker.vue`：灰色文字 → CSS 变量
- `QuickSilenceModal.vue` + `PageHeader.vue`：绿色 rgba → CSS 变量

---

## [v3.0.1] — 2026-05-11

### Fixed

- 修复 `Detail.vue` 中 `severityLabel` 函数的 TypeScript TS7053 类型错误（用 `Record<string, string>` 断言解决索引签名缺失问题）

---

## [v3.0.0] — 2026-05-10

### Changed — 全面 UI 重构：Modern Dark 设计系统 (Linear/Vercel/Raycast 风格)

**设计系统重写 (global.css)：**
- 背景色从 `#070912`/`#090D1A` 改为纯黑 `#09090B`/`#0A0A0B`
- 卡片色从毛玻璃半透明改为实色 `#141416`
- 品牌主色从 emerald `#10b981` 改为 `#22C55E` (green-500)
- 强调色从 violet `#8b5cf6` 改为 `#6366F1` (indigo-500)
- 文字色：主 `#EDEDEF`、次 `#A0A0AB`、三 `#63636E`
- 圆角全面缩小：lg `16px→10px`、md `12px→8px`、sm `8px→6px`、xs `6px→4px`
- 阴影简化为单层，移除 `inset` 高光
- 移除 aurora 渐变、conic 边框、noise 纹理等装饰性 CSS 变量
- 字体改为 Inter + JetBrains Mono（本地安装优先，系统字体回退）

**主题覆盖 (App.vue)：**
- Naive UI `darkOverrides` / `lightOverrides` 全面对齐新色板
- 移除 `AuroraBackground` 组件引用
- 新增 Tooltip 主题覆盖

**布局重构 (MainLayout.vue)：**
- TopBar 高度 56px → 48px
- Sidebar 宽度 232px → 220px
- 内容区 padding 24px → 20px
- 移除所有 `backdrop-filter` / `blur` 毛玻璃效果
- 用户头像简化（移除 gradient box-shadow）

**页面级样式对齐（8 个并行 Agent）：**
- alerts-v2/Index.vue + Detail.vue：圆角值改为设计令牌引用
- channels/Index.vue + DispatchConfig.vue + NoiseConfig.vue：圆角 + 移除 translateY
- incidents/Index.vue + Detail.vue：圆角 + 移除 translateX
- datasources/Index.vue：移除 translateY 悬浮效果
- explore/Index.vue：圆角 12px → 8px
- schedule/Index.vue：圆角 + 移除 translateY
- settings/Index.vue + TeamManagement.vue：移除 translateY 动画
- Login.vue：移除 noise-overlay 和 aurora blur 伪元素
- CommandPalette.vue：移除毛玻璃，改为实色卡片背景

**ECharts 图表主题 (QueryResultChart.vue)：**
- 新增 `chartTheme` 计算属性，通过 CSS 变量自动适配深色/浅色主题
- 硬编码颜色 `#999`/`#666`/`#eee` 改为设计令牌引用

**浅色模式 (global.css body.light-theme)：**
- 背景改为 `#FAFAFA`/`#FFFFFF`，文字 `#18181B`/`#52525B`
- 阴影适配浅色背景

---

## [v2.8.2] — 2026-05-10

### Fixed — 安全修复 + Bug 修复 + 前端 i18n 全面完善

**安全修复：**
- 修复 avatar 切片越界 panic（`auth.go:83`），当 avatar 长度 < 5 时会崩溃
- 默认管理员密码改为从 `SREAGENT_ADMIN_PASSWORD` 环境变量读取，不再硬编码 `admin123`
- 虚拟用户密码改用 `crypto/rand` 生成随机值，替代可预测的 `virtual-{username}-{len}` 格式
- logger 构建错误不再静默忽略，失败时使用 fallback logger
- post_mortem handler `formatStep` 修复死代码，使用 `fmt.Sprintf` 正确格式化
- `handler.go` 新增 `GetCurrentUserIDOK` comma-ok 断言方法，支持 401 响应

**Bug 修复：**
- 修复心跳检查器使用字符串比较 `err.Error() != "record not found"` 代替 `errors.Is(err, gorm.ErrRecordNotFound)`
- 修复 Resolve/Close handler 静默忽略 `ShouldBindJSON` 错误，改为正确返回 400 响应
- V2 模型注册到 AutoMigrate（`model.V2Models()`），确保 Alert/Channel/Incident 等表自动迁移
- AlertRule severity 枚举文档化：critical/warning/info 为主，p0-p4 标记为 Legacy

**前端 i18n — 全面国际化：**
- alerts-v2/Index.vue + Detail.vue：所有硬编码文本替换为 i18n 键（30+ 处）
- Login.vue：品牌标语、功能特性、欢迎语等 8 处替换为 `auth.brand.*` 键
- MainLayout.vue：语言选项、角色文本替换为 i18n 键
- notification/Rules.vue + Subscribe.vue + Media.vue：严重等级、表单标签、占位符等替换为 i18n 键
- settings/AuditLog.vue + OIDCConfig.vue + AIConfig.vue：筛选条件、配置项标签等 37 处替换为 i18n 键
- components/RuleFormModal.vue + QuickSilenceModal.vue + QueryResultChart.vue：表单标签、按钮文本等替换为 i18n 键
- 新增 `severity.*` 顶层 i18n 命名空间（critical/warning/info/p0-p4），供严重等级标签统一使用
- 新增 `notifyRule.repeatDisplay` / `namePlaceholder` / `invalidJson` 等键
- 新增 `subscribe.namePlaceholder` 等键
- 新增 `settings.auditTimeToday` / `oidcIssuerRequired` / `aiSectionProvider` 等 30+ 键

**前端改进：**
- SecurityConfig 组件在设置页导航中注册，用户可访问安全配置
- 抽取 `relTime()` 到 `utils/format.ts`，alerts-v2/Index.vue 和 dashboard-v2/Index.vue 复用
- DispatchConfig.vue NTag `size="tiny"` 修正为 `size="small"`（Naive UI 有效值）
- 严重等级标签颜色统一使用 CSS 变量（`--sre-critical-soft` 等），从全局 `.sev-chip` 类派生
- 路由权限守卫：`/schedule`、`/integrations` 限 admin+team_lead，`/channels` 限 admin+team_lead+member

**文档更新：**
- `docs/api.md` 从 87 个端点扩充至 164 个，覆盖所有 v2 模块

---

## [v2.8.1] — 2026-05-10

### Changed — 仪表盘 UI 升级: 高级 KPI 卡片 + 玻璃-粘土图表容器

**仪表盘重设计 (Dashboard + IncidentDashboard)：**
- KPI 卡片增加图标 (PulseOutline, TimerOutline, CheckmarkCircleOutline 等)，彩色顶部色条去除底部条纹
- 图表容器统一使用 `.surface-clay` 风格，标题 + 操作栏分区清晰
- Top Noisy Rules 增加排名徽章 (#1 主色, #2 强调色, #3 信息色)
- IncidentDashboard KPI 行统一 5 列网格 + hover 交互
- KPI 图标容器按 tone 着色 (critical/红, success/绿, info/蓝)

**MainLayout 视觉打磨：**
- 侧边栏分组标签增加小圆点指示器
- 顶部 Tab 激活态改为发光阴影 (box-shadow glow) 替代 inset ring
- Tab 点击添加 scale 反馈
- 侧边栏用户 Pill 增加 border 过渡
- 折叠按钮按钮态更平滑

**设计系统：**
- 新增 `.text-critical` / `.text-warning` / `.text-success` / `.text-info` 语义颜色工具类

---

## [v2.8.0] — 2026-05-09

### Changed — 顶部 Tab + 侧边栏导航架构重构 (FlashCat 风格)

**交互架构升级：**
- 顶部 Tab 一级模块切换: 监控告警 / 故障管理 / 系统配置
- 左侧二级导航随 Tab 动态切换，每个 Tab 独立菜单项
- Tab 切换自动导航到模块首页（/dashboard / /incident-dashboard / /notification）
- 路由变化自动同步 Tab 激活状态

**视觉打磨：**
- 全局 deepest dark 背景 (#070912)，更专业深邃
- 卡片透明度微调，暗色模式更可见
- 数据表格表头加粗 (font-weight: 600)，圆角统一 12px
- 菜单选中态增强，亮色模式更高对比度
- 新增 `.content-card`、`.surface-glass-interactive`、`.page-container` 语义类
- 新增 `--sre-shadow-card-hover` 悬停发光阴影令牌

**代码清理：**
- MainLayout.vue 去 sidebar 可折叠父菜单，改顶部 Tab 导航
- 移除未使用的 icon 导入 (AlertCircleOutline, ShieldCheckmarkOutline 等)
- 修复 inject 类型标注

**i18n：**
- 新增 `monitorAlert` 翻译键 (zh-CN: 监控告警, en: Monitor & Alert)

---

## [v2.7.0] — 2026-05-09

### Changed — UI 全局重设计: "Vibrant Glass + Clay" 融合风格

**设计系统升级：**
- 色板: 午夜石板 (#090D1A) 替代纯黑，主色翠绿 #10B981 + 紫罗兰强调 #8B5CF6
- 融合 4 种设计风格: Vibrant & Block-based + Glassmorphism + Claymorphism + Motion-Driven
- 玻璃效果 (backdrop-filter blur) 应用于侧边栏、顶栏、模态框
- 软土风格 (Claymorphism) 双阴影卡片，圆角 12-16px
- 弹性动画 (spring physics)，200-340ms 过渡
- 暗色模式: 半透明玻璃背景，柔和午夜底色
- 亮色模式: 暖灰底 #F1F4F9 + 半透白卡片

**侧边栏重构 (FlashCat 风格):**
- 5 个平铺 `type:'group'` → 3 个可折叠父菜单: 监控告警、故障管理、系统配置
- 用户头像/名称从顶栏移至侧边栏底部
- 玻璃材质侧边栏，24px 模糊 + 饱和度增强
- 顶部导航栏简化: 仅保留面包屑、时钟、搜索、语言、主题

**删除死代码:**
- SpotlightCursor (已在 v1.8.1 移除引用，清理残留)
- 废弃的 CSS 动画和注释代码
- 未使用的 page-enter/leave 过渡组件

### Changed — 文件

| 文件 | 变更 |
|------|------|
| `web/src/styles/global.css` | 重写: 新色板、玻璃/土质效果、块布局、弹性动效 |
| `web/src/App.vue` | Naive UI 主题覆盖全面更新对齐新色板 |
| `web/src/layouts/MainLayout.vue` | 3层可折叠菜单 + 玻璃侧边栏 + 用户区移至侧边栏底部 |
| `web/src/i18n/zh-CN.ts` | 新增 menu.alertCenter/incidentMgmt/systemConfig |
| `web/src/i18n/en.ts` | 新增 menu.alertCenter/incidentMgmt/systemConfig |
| `web/package.json` | 版本 2.6.2 → 2.7.0 |
| `CLAUDE.md` | 版本 2.6.2 → 2.7.0 |

---
## [Unreleased]

### Changed — UX Accessibility & Quality Sweep

**CRITICAL - Reduced Motion:**
- `global.css`: 增强 `@media (prefers-reduced-motion)` 查询，显式禁用 `.sre-stagger`、`.sre-lift`、`.sre-row-card`、`.fade-in`、`.slide-up`、`.scale-in`、`.pulse` 等所有动画类，以及 hover lift 变换效果和 conic-border 旋转动画

**CRITICAL - Focus States:**
- `global.css`: 新增 `.sre-row-card`、`.sre-lift`、`.surface-card--interactive`、`.ds-card`、`.sre-notify-card` 的 `:focus-visible` 样式（2px `var(--sre-primary)` 描边）
- `CommandPalette.vue`: `.cp-input` 添加 `:focus-visible` 内描边替代 `outline: none`
- `incidents/Detail.vue`: `.comment-input` 和 `.pm-title-input` 添加 `:focus-visible` 样式

**CRITICAL - Touch Targets:**
- `global.css`: `.sre-icon-btn` 从 24x24 扩大到 36x36（符合 WCAG 2.5.5），添加 hover/focus-visible 过渡
- `global.css`: Naive UI `n-button--tiny-type.n-button--circle-type` 最小尺寸设为 32x32

**CRITICAL - Dark/Light Contrast:**
- `BizGroupManagement.vue`: `.bg-chip` / `.bg-count` / `.bg-role-member` 硬编码 `rgba(255,255,255,*)` 替换为语义化 `var(--sre-bg-hover)` 令牌
- `BizGroupManagement.vue`: `.bg-count` 修复无效 `--sre-text` 变量为 `--sre-text-primary`
- `BizGroupManagement.vue`: `.bg-role-member .sre-dot` 硬编码 `rgba(255,255,255,0.4)` 替换为 `var(--sre-text-tertiary)`
- `BizGroupManagement.vue`: 清理 linter 发现的不一致 fallback 值（`.bg-chip-info` border-color 等）

**HIGH - Loading Feedback:**
- `alerts-v2/Detail.vue`: 初始加载添加 `LoadingSkeleton`，避免空白闪烁

**HIGH - Cursor Pointer:**
- 验证所有可点击元素（`sre-row-card`、`sre-lift`、`surface-card--interactive`、`ds-card`、`stat-card`、`dash-card`）已有 `cursor: pointer`

**MEDIUM - Error States:**
- `datasources/Index.vue`: 3 处 `message.error(err.message)` 添加 fallback 为 `t('common.loadFailed')`

**Global CSS 改进:**
- `.sre-label-eyebrow` 颜色由 `--sre-text-tertiary` 提升为 `--sre-text-secondary`（确保浅色主题 4.5:1 对比度）
- 新增 "Accessibility" 注释段，集中管理焦点样式和触摸目标

## [v2.6.2] - 2026-05-09

### Fixed — P4 全站质量修复完成

**P4 - incidents 模块 (FlashCat 9/10)：**
- `incidents/Index.vue`: 移除 3 处硬编码 CSS fallback 颜色，添加 `font-family`，i18n-ify `durationText()`，2 处 `var(--sre-hairline)` 替换 `1px solid`，添加 i18n 表单 placeholder
- `incidents/Detail.vue`: 2 处 `n-empty` → `EmptyState` 组件，`.incident-id` 添加 monospace 字体，修复硬编码 "by" 使用 i18n，MdEditor language/theme 改为计算属性，初始加载添加 `LoadingSkeleton`，timeline action 映射到 i18n 标签
- 提取 3 个子组件：`SnoozeModal.vue` / `MergeModal.vue` / `ReassignModal.vue`

**P4 - alerts 模块 (FlashCat 9/10)：**
- `events/Index.vue`: i18n-ify severityOptions/timePresetOptions/refreshOptions/NRadioButton/placeholder，修复 color-mix 边框，移除死 CSS
- `events/Detail.vue`: 11 处硬编码 `rgba()` 替换为 `var(--sre-*)` 令牌，`n-spin` → `LoadingSkeleton`，移除非法 `font-feature-settings`
- `mute/Index.vue`: i18n-ify statusText()/dayMap/typeOptions 及全部模板字符串，添加 font-family
- `inhibition/Index.vue`: i18n-ify status 标签/relTime/eyebrow labels/footer 字符串，添加 font-family
- `history/Index.vue`: 全 i18n（11 处），`NEmpty` → `EmptyState`，`NSpin` → `LoadingSkeleton`，4 处 inline style → CSS class

**P4 - 共享组件修复：**
- `LoadingSkeleton.vue`: `rgba()` 替换为 `var(--sre-overlay-subtle)`
- `EmptyState.vue`: 4 处硬编码 `rgba()` 边框替换为 `var(--sre-*-ring/soft)` 令牌
- `global.css`: 新增 `--sre-overlay-subtle` / `--sre-bg-subtle` 令牌（dark + light）

**i18n 新增：**
- `zh-CN.ts` / `en.ts`: 新增 ~54 个 key（incident 14 + alert 20 + mute 12 + inhibition 8）

### Fixed — 全站质量修复（P0-P3）

**P0 - 构建修复：**
- `.gitignore` 添加 `.superpowers/` 排除
- `md-editor-v3` 依赖安装（构建修复）

**P1 - 后端错误处理修复（15 处）：**
- `alert_v2_pipeline.go`: 修复 3 处 `_ =` 静默丢弃 DB 错误
- `schedule.go`: 修复 DeleteEscalationPolicy 中 `steps, _ :=` 和 `_ = stepRepo.Delete` 错误忽略
- `system_setting.go`: `parseBool` 不再静默丢弃解析错误
- `user.go`, `datasource.go`, `team.go`, `alert_event.go`, `message_template.go`, `biz_group.go`, `seed.go`, `integration.go`, `notification.go`: 修复 `existing, _ :=` 模式，区分 `gorm.ErrRecordNotFound` 与真正的 DB 错误

**P2 - i18n 国际化修复：**
- 新增 ~200 个 i18n key（zh-CN + en）
- 修复 6 个严重未国际化文件：`RoutingRules.vue`（完全未翻译）、`QuickSilenceModal.vue`（完全未翻译）、`incidents/Detail.vue`（~35 处）、`explore/Index.vue`（~13 处）、`DispatchConfig.vue`（~11 处）、`IncidentDashboard.vue`（~8 处）
- 修复 `alerts/events/Index.vue`、`channels/Index.vue`、`Login.vue` 中的硬编码字符串

**P3 - UI 设计系统对齐：**
- 字体标准化：22 个文件中的硬编码 `font-family` 替换为 `var(--sre-font-*)` CSS 变量
- 颜色系统：6 个文件中的硬编码颜色表替换为 `var(--sre-*)` 设计令牌
- CSS 去重：将 notification（Rules/Subscribe/Media）和 settings config（AI/Lark/OIDC/SMTP）中 ~280 行重复 CSS 提取到 `global.css` 共享样式

---



## [v2.6.0] - 2026-05-08

### Changed — UI 重构 Phase 5（共享组件 + 设计系统文档 + 浅色审计）

**共享组件抽取：**
- `components/common/EmptyState.vue` 新增（替代散落的 NEmpty）：支持 size sm/md/lg、variant default/success/warning/critical/info、primary/secondary 操作按钮
- `components/common/LoadingSkeleton.vue` 新增（替代 NSpin 大圈圈）：3 种 variant（row / card-grid / kpi），shimmer 动效

**6 个高频页面接入新组件：**
- incidents/Index、channels/Index、alerts/rules/Index、alerts/events/Index、alerts-v2/Index、integrations/Index
- 加载态用 LoadingSkeleton 骨架屏（首次加载，避免空白闪烁）
- 空状态用 EmptyState（统一图标+文案+CTA 按钮）

**浅色主题对比度修复：**
- `global.css` 两处 `.sre-row-card:hover` / `.sre-lift:hover` 边框色：硬编码 rgba(255,255,255,0.x) → var(--sre-border-strong)
- `dashboard/Index.vue` ECharts 浅色模式不可读：新增 isLightTheme 监听 body.classList，chartPalette computed 提供 8 个动态颜色值（tooltipBg/legend/axis/grid/pieCenter）
- `channels/Index.vue` `.card-star:hover` 背景：rgba(0,0,0,0.05) → var(--sre-bg-hover)

**设计系统文档：**
- `docs/design-system.md` 新建（~220 行）
- 内容：设计哲学 / Typography (Geist + 拒绝清单) / Color (主色 + severity + surfaces + text + WCAG) / Spacing / Radius / Component patterns / Page anatomy / Anti-patterns 10 项 / Skill 应用 / Migration notes / File index / Decision log

**v2.x UI 重构总结**（v2.1.0 → v2.6.0）：
- 35+ 文件全站对齐 FlashCat "Operational Refinement" 美学
- 字体 Geist + JetBrains Mono 替代 Inter/system
- 4px severity stripe + sre-dot 圆点 + hairline borders 全站统一
- 自定义 sre-row-card div 列表 取代 NDataTable（除审计外）
- Skills 全程加持：frontend-design / vue 3.5 / web-design-guidelines

vue-tsc ✅

---

## [v2.5.0] - 2026-05-08

### Changed — UI 重构 Phase 4（Settings 子页 + Login）

10 个 settings 子页 + Login 全部对齐 FlashCat 设计语言。

**Platform Config（4 个表单页统一模板）：**
- `AIConfig.vue`：Provider + Behavior 两段卡片，新增 temperature/max_tokens/system_prompt 字段
- `LarkBotConfig.vue`：App Credentials + Defaults 两段
- `SMTPConfig.vue`：Server + Sender + Test Delivery 三段，新增 from_name
- `OIDCConfig.vue`：Provider + Claim Mapping + Behavior 三段，新增 username_claim/email_claim

共享模板：紧凑 header（18px 600 + 描述）+ 状态 banner（success/error tone）+ 段落卡片（hairline 边框）+ 2 列 form-grid + uppercase eyebrow labels + 内联 Test/Save 按钮

**Organization Pages：**
- `UserManagement.vue`：NDataTable → sre-row-card（avatar 圆形 + role chip 颜色映射 + 状态点 + last_login relTime）
- `TeamManagement.vue`：NDataTable → 卡片网格（成员头像组叠层 + member count + incidents count）
- `VirtualUsers.vue`：sre-row-card（type icon tile + type chip mono + truncate notify_target）
- `BizGroupManagement.vue`：300px 自定义递归树 + 右侧详情面板（PARENT/PATH/CHILDREN/CREATED 元信息卡 + MEMBERS 列表）

**Audit：**
- `AuditLog.vue`：NDataTable → 竖向时间线（hairline rail + 8px 圆点按 action tone 着色 + mono 时间戳 + action chip + resource 跳转链接）

**Login 重设计**：
- "Mission Control" 双栏布局（60% brand / 40% form）
- 品牌侧：Geist 800 -2px tracking 56px 标题 + gradient text-clip + 极细噪点 + 一处低饱和极光斑（左下）
- 表单侧：mono uppercase eyebrow labels + stagger 浮现（60ms 增量）+ 主色按钮 hover ring + 警告 banner（默认密码提醒）
- 拒绝紫渐变 / 卡通 illustration / 立体阴影；接受 hairline + 噪点 + 字体层次

修复 6 处 TS 错误：AIConfig/OIDCConfig/SMTPConfig 新增字段类型断言。

vue-tsc ✅

---

## [v2.4.0] - 2026-05-08

### Changed — UI 重构 Phase 3（配置类页面 FlashCat 对齐）

11 个配置类页面统一视觉语言：Geist 字体 / tabular nums / sre-row-card / sre-dot / hairline。

- **通知中心**：
  - `notification/Index.vue`：NTabs → 200px 左导航 + 内容区，URL hash 同步
  - `AlertChannels.vue`：NDataTable → sre-row-card 列表，标签匹配 chips、Webhook URL 复制、Throttle 显示
  - `Media.vue`：紧凑列表，type chip 着色（lark/email/webhook/script），测试发送按钮
  - `Rules.vue`：Match conditions chips → "→" 关联 media 列表
  - `Subscribe.vue`：订阅人头像组 + Match chips + 通知方式
  - `Templates.vue`：type chip + 内容预览 mono + 内置/自定义标记

- **告警引擎补齐**：
  - `mute/Index.vue`：状态分段（生效中/未来/已过期/禁用）+ Match chips + Schedule 摘要 + 命中预览抽屉
  - `inhibition/Index.vue`：Source/Target/Equal 分行展示 + 命中数 tnum
  - `history/Index.vue`：时间分段（7天/30天/90天/自定义）+ 紧凑历史列表（dim 0.85）+ Export CSV

- **集成与数据**：
  - `datasources/Index.vue`：表格 → 卡片网格（顶部 type 色条）+ 健康状态点 + Latency/Version stats + Test 按钮
  - `schedule/Index.vue`：Header + Sidebar 视觉对齐，班次色块（var(--sre-primary-soft) + 主色 marker），当前值班用主色实色

vue-tsc ✅ — 修复 4 处类型问题（DataSourceStatus 类型断言、SelectOption value 不能为 null、NSpin size="tiny" → "small"、duration/relTime null 兼容）

---

## [v2.3.0] - 2026-05-08

### Changed — UI 重构 Phase 2（详情页 FlashCat 对齐）

5 个详情/二级页面同步对齐 Phase 1 设计语言。所有页面：Geist 字体、tabular nums、severity 4px 左色条、sre-dot 圆点、hairline 边框、hover lift。

- **故障详情**（incidents/Detail.vue）：
  - Header 紧凑横条 + sre-row-card 副标题（圆点+severity+状态+空间+持续时间）
  - 操作栏分层：主操作（认领/关闭/重新打开）+ 横排次操作 + 右上 NDropdown 收纳
  - 三栏 Tab：Overview（紧凑 dl 网格 + tabular nums）/ Alerts (sre-row-card) / Timeline (竖向圆点+hairline 连接)/ Post-Mortem (md-editor-v3 dark)
  - 右栏 280px：KEY INFO + TIMELINE BRIEF
  - Snooze/Merge/Reassign 弹窗用 sre-row-card picker 行

- **协作空间详情**（channels/Detail.vue）：
  - Header 24px 700 标题 + 描述·团队副标
  - 4 张 KPI 卡片（Active/Today/MTTA/MTTR）+ 底部 tone 色条
  - 5 Tab：Incidents（sre-row-card）/ Overview / Noise / Dispatch / Settings
  - Settings Tab 含"危险区"删除卡片（红边二次确认）

- **告警 v2 列表 + 详情**（alerts-v2/）：
  - Index：sre-row-card + status segmented + severity/channel 筛选
  - Detail：sre-row-card subtitle + Tabs(Overview/Events) + 右栏 KEY INFO + LABELS

- **告警事件详情**（alerts/events/Detail.vue）：
  - 状态感知操作栏（firing→ack/resolve/close, acked→resolve/close/assign, etc）
  - 三 Tab：Overview（labels mono chips + annotations dl + rule card）/ Timeline（竖向 sre-dot 时间线 + 评论框）/ AI（报告+SOP 推荐）
  - 右栏 280px：Key Info + Responders + Labels + Related

- **集成中心**（integrations/Index.vue）：
  - n-data-table → 卡片网格（auto-fill minmax 320px）
  - Type+Mode 双层 segmented 筛选
  - 卡片：顶部 type 色条 + 状态点 + 标题 + type/mode badges + 描述 + webhook URL chip + 复制按钮 + 底部 alerts count + → 关联空间 + 操作行
  - 共享集成卡片显式"路由规则"按钮 → RoutingRules 抽屉

vue-tsc ✅

---

## [v2.2.0] - 2026-05-08

### Changed — UI 重构 Phase 1（FlashCat 全站对齐）

应用三个 skill：frontend-design (anthropics) / vue 3.5 (antfu) / web-design-guidelines (Vercel)

**字体与设计 tokens：**
- Geist + JetBrains Mono 通过 Google Fonts 全局引入（替代 system fonts，避免 AI slop）
- 新增设计 tokens：`--sre-stripe-w` `--sre-row-pad-y/x` `--sre-card-pad` `--sre-section-gap` `--sre-hairline`
- 新增 utility classes：`.sre-stagger`（错峰浮现）、`.sre-row-card[data-severity]`（4px 左色条卡片）、`.sre-dot[data-severity]`（圆点）、`.sre-meta-divider`、`.sre-stat-value`、`.sre-lift`、`.tnum`、`.sre-label-eyebrow`

**Phase 1 页面（4 个核心页）：**

- **主仪表盘**（dashboard/Index.vue 1158 → 536 行）：
  - 删除 GlowCard / AnimatedNumber / AuroraBackground（AI slop 视觉）
  - 4 张 KPI 卡片（Active / MTTA / MTTR / Resolved Today）+ 底部 3px tone 色条
  - Geist 字体 + 所有数字带 tabular-nums
  - 告警趋势 ECharts 折线图（280px 渐变填充，节制配色）
  - Top 噪音规则自定义列表 + 严重程度环形图
  - sre-stagger 首屏 KPI 错峰浮现

- **设置页**（settings/Index.vue 81 → 332 行）：
  - 顶部 Tabs → 240px 左导航 + 内容区
  - 三组 eyebrow label：PLATFORM / ORGANIZATION / AUDIT
  - 选中态 primary-soft 背景 + 2px 主色左 marker + 主色文字
  - URL hash 同步（#ai / #lark-bot 等）+ hashchange 监听
  - UserManagement v-show 保留以便跨 tab user list 共享

- **告警规则**（alerts/rules/Index.vue 941 → 1288 行）：
  - 抛弃 n-data-table，自定义 sre-row-card div 列表
  - 220px 左侧分类导航（active 态 primary-soft + 2px marker + tnum count）
  - 严重程度作 4px 左色条 + sre-dot 圆点替代 tag
  - 紧凑工具栏（search 240px + 3 selects 160px）
  - 浮现选择栏 + 批量启用/禁用/删除
  - 行 actions：启用 switch + 省略号下拉

- **活跃告警**（alerts/events/Index.vue 817 → 750 行）：
  - 抛弃 n-data-table，自定义 sre-row-card 列表
  - 严重程度 4px 左色条 + sre-dot
  - resolved/closed 行 data-dim 淡化 0.6
  - 4 行结构：headline / context / labels chips / footer 元数据
  - 状态分段：[全部 | Firing | Acked | Resolved]
  - 自动刷新（Off/30s/60s/5min 持久化）+ Export CSV

vue-tsc ✅

---

## [v2.1.0] - 2026-05-08

### Changed — UI 重构（FlashCat 风格）

- **数据查询页（/query）**：
  - 时间范围：横排预设按钮（5m/15m/30m/1h/3h/6h/12h/24h/2d/7d/30d）+ 自定义 datetimerange
  - 自动刷新：5s/10s/30s/1min/5min，倒计时显示
  - Step 选择器（metrics）：自动/15s/30s/1m/5m/15m/1h
  - Limit 选择器：metrics 50-1000，logs 50-5000
  - 查询历史 Popover（localStorage，最近 20 条）+ 清空按钮
  - CSV 导出（前端 Blob）：metrics table 模式 + logs 模式
  - 视觉：三段式卡片布局（工具栏 / 编辑器 / 结果）

- **主侧栏（MainLayout）**：
  - 扁平化：去除"告警管理"父级嵌套，子项铺平
  - 6 个 group 分组：概览 / 故障管理 / 告警引擎 / 集成与数据 / 通知与值班 / 系统
  - Group label 小字号 uppercase，FlashCat 风格
  - 选中态：左侧色条 + 浅色背景 + 主色文字
  - 折叠态隐藏 group label

- **故障列表（/incidents）**：
  - 弃用 n-data-table 改自定义卡片列表
  - 左侧 4px 严重程度色条
  - 三行结构：圆点+严重程度+#ID+标题 / 元数据 / 状态+处理人+时间
  - Hover 背景变化 + 箭头浮现
  - 已关闭行 0.72 opacity 淡化
  - 顶部分段控件：全部/我的，状态/严重程度筛选

- **协作空间列表（/channels）**：
  - 顶部 4px 主色色条
  - 三栏指标（活跃故障 / MTTA / MTTR）
  - 卡片右上角 Star（hover 显隐）
  - 右下角省略号下拉（编辑/删除）
  - Hover translateY(-2px) + 主色边框
  - 视图切换占位（卡片/列表）+ 排序下拉

- **i18n**：menu.* 补齐 6 个 v2 路由的中英键，incident.empty/duration/unassigned 等新增

---

## [v2.0.3] - 2026-05-08

### Fixed — 启动 panic（路由冲突）

- **Gin 路由参数冲突**：`/api/v1/integrations/:token/alerts`（webhook 接收）与 `/api/v1/integrations/:id/routing-rules`（路由规则 CRUD）共享前缀 `/integrations/:X`，但参数名不同，导致 Gin 启动 panic
- **修复**：将路由规则 API 改为扁平路径
  - `GET /api/v1/routing-rules?integration_id=X`（query string）
  - `POST /api/v1/routing-rules`（integration_id 在 body）
  - `PUT/DELETE /api/v1/routing-rules/:id`（不变）
- 前端 `routingRuleApi.listByIntegration` 和 `create` 同步更新

---

## [v2.0.2] - 2026-05-08

### Added — UI 缺口补齐

- **故障详情页 — 暂缓（Snooze）**：操作栏新增"暂缓"按钮，提供 5 个时长预设（15m/30m/1h/2h/4h）+ 自定义截止时间选择
- **故障详情页 — 合并（Merge）**：操作栏新增"合并故障"按钮，支持搜索目标故障并二次确认合并；合并后跳转目标故障
- **故障详情页 — 重新分派（Reassign）**：操作栏新增"重新分派"按钮，展示用户列表并支持实时过滤
- **故障复盘 — Markdown 编辑器**：PostMortem Tab 从纯 textarea 升级为 `md-editor-v3`，支持实时预览、语法高亮、工具栏
- **路由规则 CRUD**：
  - 后端：`RoutingRuleHandler`（List/Create/Update/Delete）+ 路由注册（`GET/POST /integrations/:id/routing-rules`，`PUT/DELETE /routing-rules/:id`）+ main.go wiring
  - 前端：`RoutingRules.vue` — 规则列表 + 优先级上下调整 + 启用开关 + 条件 JSON 编辑 + 目标空间选择
  - 集成中心：共享集成行新增"路由规则"按钮，点击弹出右侧抽屉展示 `RoutingRules.vue`

---

## [v2.0.1] - 2026-05-07

### Added — 告警规则批量操作

- **后端**:
  - `AlertRuleRepository.BatchUpdateStatus(ctx, ids, status)` — 批量更新状态，version 字段自增
  - `AlertRuleRepository.BatchDelete(ctx, ids)` — 批量软删除
  - `AlertRuleService.BatchEnable/BatchDisable/BatchDelete` — 参数校验 + 错误封装
  - `AlertRuleHandler.BatchEnable/BatchDisable/BatchDelete` — Gin handler + 审计日志
  - 路由: `POST /api/v1/alert-rules/batch/enable|disable|delete`（manage 权限）
- **前端**:
  - `alertRuleApi.batchEnable/batchDisable/batchDelete` — API 层
  - `pages/alerts/rules/Index.vue`: columns 添加 `{ type: 'selection' }` 复选列；`v-model:checked-row-keys` 多选；批量工具栏（选中数量显示 + 启用/禁用/删除按钮 + Popconfirm 二次确认）
  - i18n: zh-CN + en `common.selected`、`alert.batchEnabled/Disabled/Deleted/DeleteConfirm` 新增键
- go build ✅ | vue-tsc 无新增错误 ✅

---

## [v2.0.0] - 2026-05-07

### Release — v2.0 正式版

本版本为 SREAgent v2.0 正式发布版，包含 Phase 1-5 全部功能，以及发版收尾工作。

**版本升级路径**：从任意 v1.x 直接升级即可。部署新镜像后 `golang-migrate` 自动执行 000019-000033 共 15 个迁移。

#### 新增功能汇总
- **协作空间**（Channel）：故障聚合、降噪、分派、统计的核心单元
- **故障管理**（Incident）：完整生命周期 + 自动关闭 + 复盘
- **告警 v2**（Alert/AlertEventV2）：去重、关联、事件流水线
- **智能降噪**：聚合规则、风暴预警、抖动检测、排除规则、**快速静默**
- **分派策略**：触发条件、延迟窗口、重复通知、标签增强、升级绑定
- **Webhook 集成**：Standard/AlertManager/Grafana 三格式 + Pipeline + 限流
- **故障复盘**：Markdown 编辑器 + AI 生成初稿 + 发布
- **增强仪表盘**：按协作空间/团队维度的故障统计 + 趋势图

#### 收尾工作
- 版本号更新：CLAUDE.md v2.0.0 / web/package.json 2.0.0
- MODULES.md 更新：34 个模块 + v2 模块清单 + 迁移文件索引
- PLAN-status.md 修正所有遗漏项
- QuickSilenceModal：Incident Detail + Alert Detail 集成快速静默

---

## [v2.4.0-alpha.1] - 2026-05-07

### Added — Phase 5 故障复盘 + 分析增强

- **PostMortem CRUD** (`internal/service/post_mortem.go`):
  - `GetOrCreate`: 按 incident_id 查找或自动创建草稿（含 Markdown 模板预填充）
  - `Update` / `Publish`: 保存内容并可一键发布
  - `List`: 支持按 channel_id（JOIN incidents）和 status 过滤
  - `defaultPostMortemTemplate`: 预填充故障标题/时间/等级
- **AI 故障分析** (`internal/handler/post_mortem.go`):
  - `AIGenerate`: 调用 `AIService.AnalyzeAlertWithContext` → 生成 Markdown 复盘初稿并保存
  - `AISummary`: 返回 `AlertAnalysis` JSON 供前端预览（不保存）
  - `buildPostMortemFromAnalysis`: 将 AI 输出拼装为标准 Markdown 复盘格式
- **API 端点**:
  - `GET/PUT /api/v1/incidents/:id/post-mortem`
  - `POST /api/v1/incidents/:id/post-mortem/publish`
  - `POST /api/v1/incidents/:id/post-mortem/ai-generate`
  - `POST /api/v1/incidents/:id/post-mortem/ai-summary`
  - `GET /api/v1/post-mortems` (全局列表)
- **分析看板增强** (`internal/handler/dashboard.go`):
  - `IncidentStats`: 活跃故障数/今日关闭/紧急/Avg MTTR/复盘统计
  - `ChannelStats`: 按协作空间的故障分布（total/triggered/closed/critical）
  - `TeamStats`: 按团队的故障分布 + Avg MTTR（JOIN channels→teams）
  - `IncidentTrend`: 按日汇总 triggered+closed 趋势
- **前端**:
  - Incident Detail 新增"复盘"Tab：Markdown textarea 编辑器 + 保存/发布/AI 生成按钮 + SparklesOutline 图标
  - `incidentApi` 扩展：getPostMortem/updatePostMortem/publishPostMortem/aiGeneratePostMortem/aiSummaryPostMortem
  - `dashboardV2StatsApi`: incidentStats/channelStats/teamStats/incidentTrend
  - `IncidentDashboard.vue`: 5 张统计卡片 + 趋势柱状图（纯 CSS） + 空间/团队排行表
  - 侧边栏新增"故障看板"菜单（BarChartOutline 图标）
  - i18n: zh-CN + en `postMortem.*` + `dashboardV2.*` 新增键（合并至已有 dashboardV2 节）

---

## [v2.3.0-alpha.1] - 2026-05-07

### Added — Phase 4 告警引擎增强 + Webhook 接入

- **4.3 AlertRule → Channel 关联**: `AlertRule.channel_id` 字段 + 迁移 `000033`；rule_eval 注入 `_channel_id` 标签，AlertV2Pipeline 按规则优先路由到指定 channel
- **4.4-4.6 Webhook 接入** (`internal/service/integration.go`):
  - `IntegrationService.ReceiveAlerts`: 按 token 查找集成，限流检查，格式解析，pipeline 处理，路由到 AlertV2Pipeline
  - `normaliseStandard`: `{alerts:[...]}` 或单对象格式
  - `normaliseAlertManager`: `{alerts:[{status,labels,annotations,startsAt,...}]}`
  - `normaliseGrafana`: `{alerts:[{title,state,labels,...}]}`，state=alerting/ok/normal/no_data
- **4.7 处理管道**: `applyPipeline` — `rewrite_severity`/`rewrite_title`/`rewrite_description`/`drop`；条件匹配复用 `FilterCondition`；模板变量 `{{title}}/{{severity}}/{{labels.xxx}}`
- **4.8 频率限制**: per-integration 令牌桶（in-memory），100/s + 1000/min 双窗口
- **Integration CRUD API**: `GET/POST /api/v1/integrations` + `GET/PUT/DELETE /api/v1/integrations/:id`
- **Webhook 接收端点**: `POST /api/v1/integrations/:token/alerts`（无 JWT，token 鉴权）
- **4.1 NoData**: 引擎已有实现（`NoDataEnabled`/`NoDataDuration`）
- **4.2 规则文件夹**: `AlertRule.category` 已支持，`listCategories` API 已有
- **前端**:
  - `pages/integrations/Index.vue`: CRUD 表格 + Webhook URL/Token 展示与复制 + 创建/编辑弹窗（type/mode/channel/pipeline/label 增强）
  - 侧边栏新增"集成中心"菜单项（GitNetworkOutline 图标）
  - `integrationV2Api` API 层
  - i18n: zh-CN + en `integration.*` / `ruleFolder.*` 键
- **DB 迁移 000033** `alert_rules.channel_id`

---

## [v2.2.0-alpha.1] - 2026-05-07

### Added — Phase 3 分派增强

- **DispatchPolicy 模型** (`internal/model/dispatch.go`):
  - Channel 绑定、多策略优先级、启用开关
  - 触发条件 `match_conditions` (JSON `FilterCondition[]`) + 生效时间段 `active_time_config` (时区/星期/时间段)
  - 延迟窗口 `delay_seconds` (0-3600)
  - 重复通知 `repeat_interval_seconds` + `max_repeats`
  - 通知方式 `notify_mode` (personal_preference | unified) + `unified_media_id`
  - 升级策略绑定 `escalation_policy_id`
  - 标签增强规则 `label_enhancement_rules` (JSON `LabelEnhancementAction[]`)
- **DispatchLog 模型** — 记录每次分派尝试状态
- **DispatchService** (`internal/service/dispatch.go`):
  - `FindMatchingPolicy`: 按优先级匹配第一个满足条件+时间窗口的策略
  - `ApplyLabelEnhancements`: set/extract(regex)/combine(template)/map(lookup)/delete 五种操作
  - `matchConditions` + `isActiveNow`: 复用 `FilterCondition` 匹配逻辑
- **AlertV2Pipeline 集成**: `SetDispatchService` → `process()` 在 upsert 前执行标签增强
- **API**: `GET/POST /api/v1/channels/:id/dispatch-policies` + `GET/PUT/DELETE /api/v1/dispatch-policies/:id`
- **DB 迁移 000031** `dispatch_policies` + **000032** `dispatch_logs`
- **前端 DispatchConfig.vue**: 策略列表 + 优先级上下移动 + 创建/编辑弹窗（全字段覆盖）
- Channel Detail 新增"分派配置" Tab
- i18n: zh-CN + en `channel.dispatch*` 全量键

---

## [v2.1.0-alpha.1] - 2026-05-07

### Added — Phase 2 智能降噪

- **NoiseReducer** (`internal/service/noise_reducer.go`): 降噪核心引擎
  - 排除规则：`matchAllConditions` 支持 eq/ne/contains/not_contains/regex/in/not_in
  - 聚合键计算：统一维度 / 细粒度条件分支，strictMode 控制空值处理
  - 风暴预警：滚动 1 分钟窗口计数，每阈值只触发一次告警
  - 抖动检测：in-memory flapStates，支持 off / notify_only / notify_then_silence 三种模式
- **AlertV2Pipeline 集成**：`SetNoiseReducer` + `process()` 在 upsert 前执行降噪，excluded→drop，silenced→跳过故障创建
- **ExclusionRuleRepository + Service + Handler**：`/api/v1/channels/:id/exclusion-rules` CRUD
- **前端 NoiseConfig.vue**：协作空间详情页新增"降噪配置" Tab，覆盖聚合规则/窗口/风暴预警/抖动检测/排除规则
- **i18n**：zh-CN + en 新增 `channel.noise*` / `channel.flapping*` / `channel.exclusion*` 全量键

---

## [v2.0.0-alpha.1] - 2026-05-07

### Added — Phase 1.1 核心模型重构
- **协作空间 (Channel)**：`model/channel.go` + repository + service + handler + API (`/api/v1/channels`)
  - CRUD + Star/Unstar 收藏 + 列表带收藏标记
  - 降噪配置（聚合规则/抖动检测）、自动关闭配置
- **故障 (Incident)**：`model/incident.go` + repository + service + handler + API (`/api/v1/incidents`)
  - 完整操作：acknowledge / close / reopen / snooze / merge / reassign / escalate / comment
  - 时间线 (IncidentTimeline) 自动记录所有操作
  - 分派人跟踪 (IncidentAssignee)
  - 复盘报告 (PostMortem) 模型
- **告警 v2 (Alert + AlertEventV2)**：`model/alert.go` + repository + service + handler + API (`/api/v1/alerts`)
  - Alert: 按 alert_key 去重的告警序列，关联 Channel + Incident
  - AlertEventV2: 原始事件数据（firing/resolved），按时间戳记录
  - UpsertFromEvent: 核心摄入路径，支持自动去重+合入
- **集成 (Integration) + 路由规则 (RoutingRule)**：模型已定义（repo/service/handler 待 Phase 4 实现）
- **DB 迁移 000019-000030**：
  - 000019: channels
  - 000020: channel_stars
  - 000021: channel_exclusion_rules
  - 000022: incidents
  - 000023: incident_assignees
  - 000024: incident_timelines
  - 000025: post_mortems
  - 000026: alerts
  - 000027: alert_events_v2
  - 000028: integrations
  - 000029: routing_rules
  - 000030: seed default channel

### Fixed
- **Settings 菜单点击无反应**：Naive UI n-menu 当 `:value` 等于点击项 key 时不触发 `@update:value`，改用 ref + 点击前清空解决

---

## [v1.16.23] - 2026-05-06

### Fixed
- **彻底消除 vue-i18n runtime SyntaxError**：移除所有 i18n 消息中的花括号示例文本（PromQL、JSON 示例），改用不含花括号的纯文字描述
  - `{'{'}` 转义在 vue-i18n v11 production JIT 编译模式下仍会触发 `EXPECTED_TOKEN` 错误
  - 涉及 14 条消息（zh-CN 7 条 + en 7 条）：datasource/explore/query placeholder、notifyRule hints、OIDC mapping、Lark bot hint

## [v1.16.22] - 2026-05-06

### Fixed
- `query.promqlPlaceholder` 中 `{instance=~"prod.*"}` 未转义花括号，vue-i18n v11 message-compiler 报 SyntaxError（与 v1.16.18 同类问题）

## [v1.16.21] - 2026-05-06

### Fixed
- 指标查询选择数据源后查询输入框不显示：移除 PromQLEditor 异步组件，改用稳定的 NInput textarea
- `v-if="selectedDsId"` 改为 `v-if="selectedDsId != null"` 显式空值检查

## [v1.16.20] - 2026-05-06

### Changed
- **「数据探索」重命名为「数据查询」(Data Query)**：路由 `/explore` → `/query`，保留旧路由重定向兼容
- **数据查询页面完全重写**，修复长期白屏问题：
  - 根因：`@codemirror/view`、`@codemirror/state`、`@codemirror/commands` 未声明为直接依赖，Rollup 打包解析失败
  - 新增 Tab 切换：「指标查询」(Prometheus/VM/Zabbix) + 「日志查询」(VictoriaLogs)
  - ECharts 改为懒加载 (dynamic import)：加载失败不阻塞页面，自动降级到表格模式
  - PromQLEditor (CodeMirror) 改为 defineAsyncComponent + 5s 超时：加载失败回退到 NInput textarea
  - 数据源按类型分组到对应 Tab，而非混合在一个下拉框
  - 新增 `query.*` i18n key set (中英双语)

### Fixed
- 安装缺失的 `@codemirror/view`、`@codemirror/state`、`@codemirror/commands` 包至 package.json
- 修复 vite build 因 CodeMirror 子包缺失导致的 Rollup resolve 错误

## [v1.16.19] - 2026-04-30

### Changed
- **Explore 页面 UI 重写**：使用 Naive UI 组件替代纯 HTML 元素
  - PromQLEditor（CodeMirror 6 + PromQL 语法高亮）用于指标数据源
  - 日志数据源使用简洁的 textarea + 等宽字体
  - ECharts 时序图表 + DataTable 表格切换
  - 数据源选择器显示类型标签（Prometheus/VM/VLogs/Zabbix）和版本号
  - 自动根据数据源类型切换查询模式（指标/日志）

## [v1.16.18] - 2026-04-30

### Fixed
- **真正的根因修复**：vue-i18n v11 的 message-compiler 对 `{` 字面量比 v9 更严格，i18n 消息中的 PromQL 示例 `{mode="idle"}` 和 JSON 示例 `[{"type":"aggregate"}]` 被错误解析为占位符，导致 `INVALID_TOKEN_IN_PLACEHOLDER` SyntaxError
- 修复 6 处 i18n 消息（zh-CN + en），使用 `{'{'}` / `{'}'}` 转义字面量花括号
- 恢复 esbuild 压缩器（terser 未生效，已移除）

### Changed
- `vite.config.ts`: 恢复 `minify: 'esbuild'`（误判，问题不在压缩器）

## [v1.16.17] - 2026-04-30

### Fixed (未生效)
- 尝试切换到 terser 压缩器，但错误依旧 — 证明问题不在压缩器，在于 i18n 消息内容

### Changed
- `vite.config.ts`: `minify: 'esbuild'` → `minify: 'terser'`（后被 v1.16.18 回滚）

## [v1.16.16] - 2026-04-30

### Fixed
- **DataView Symbol.toStringTag 报错**：lodash（Naive UI 内置）`getRawTag()` 在 ES module strict mode 下尝试覆写只读的 `DataView.prototype[Symbol.toStringTag]`，导致 "Cannot assign to read only property" TypeError
- 新增 `dataview-polyfill.ts`，在 main.ts 最开始执行，将 DataView 的 Symbol.toStringTag 属性设为 writable

## [v1.16.15] - 2026-04-30

### Fixed
- **真正的根因修复**：升级 vue-i18n 9.14.0 → 11.4.0，`@intlify/message-compiler` 新版修复了 esbuild 压缩产生的 `Unterminated closing brace` SyntaxError
- 恢复 esbuild 压缩器（`minify: 'esbuild'`），移除 terser 依赖

### Changed
- vue-i18n 升级到最新版 11.4.0（兼容，typecheck 通过）

## [v1.16.14] - 2026-04-30

### Fixed (未生效)
- 尝试修复：切换到 terser 压缩器避开 `@intlify/message-compiler` esbuild 压缩 bug — 但引入了 "Cannot assign to read only property" 新问题
- Explore 页面简化为纯 HTML 元素，移除 Naive UI 组件依赖

## [v1.16.13] - 2026-04-30

### Debug
- Explore 页面移除 TimeRangePicker/RefreshPicker 依赖，用纯文本替代 — 隔离 DatePicker 是否为白屏根因

## [v1.16.12] - 2026-04-30

### Fixed
- Explore 页面 `row-key` 类型错误 — 单参数函数匹配 `CreateRowKey<any>` 签名
- Explore 页面日志数据添加 `_key` 索引

### Debug
- Explore 页面添加 `onErrorCaptured` 错误边界 + console 诊断日志，定位生产白屏根因

## [v1.16.11] - 2026-04-29

### Changed
- 重写 Explore 页面：移除 ECharts/vue-echarts 依赖，消除生产环境白屏。列 render 函数只返回纯字符串（不再用 `h()` 返回 VNode 数组），所有 Naive UI 组件显式导入 + PascalCase 模板用法

## [v1.16.10] - 2026-04-29

### Fixed
- 修复 Explore 页面生产环境白屏：移除未使用的 `shallowRef` 导入、模板内联 `.map()` 改为 computed `datasourceOptions`、全链路空值防御（`s.labels || {}`、`s.values || []`、`v.value ?? 0`、`Array.isArray` 守卫）

## [v1.16.9] - 2026-04-29

### Added
- P0-P4 严重级别支持：model 常量、前端类型、i18n 标签（P0-紧急/P1-严重/P2-一般/P3-轻微/P4-信息）、表单和过滤器选项
- `/metrics` 端点：Prometheus 暴露格式的 Go 运行时 + 应用指标
- PanelCard 新增 gauge/bar/pie 图表类型（ECharts GaugeChart + BarChart + PieChart）
- Dashboard V2 面板拖拽布局：拖拽标题栏移动面板位置 + 右下角拖拽调整面板尺寸（CSS Grid 24 列）
- Dashboard V2 面板类型扩展按钮：统计值/时序图/仪表盘/柱状图/饼图/表格
- 告警规则模板系统：CRUD + 分类 + "从模板加载"/"保存为模板"（前后端完整实现）
  - Model: `alert_rule_templates` 表（迁移: 000018_alert_rule_templates）
  - API: GET/POST/PUT/DELETE `/api/v1/alert-rule-templates` + `/categories` + `/:id/apply`
  - 前端：创建规则时可从模板加载，编辑时可保存为模板

### Changed
- Alert Detail 页面硬编码颜色全部替换为 CSS 自定义属性（banner、timeline、lifecycle、labels、annotations、responders）
- PanelCard Stat 面板支持阈值颜色：`panel.options.thresholds` 数组 `[{ value, color }]` 自动根据当前值匹配颜色

### Fixed
- PromQLEditor 防御性错误处理：onMounted 和 datasourceId watcher 中 EditorState.create 增加 try-catch

---

## [v1.16.8] - 2026-04-29

### Changed
- Alert Detail 页面硬编码颜色全部替换为 CSS 自定义属性（banner、timeline、lifecycle、labels、annotations、responders）
- PanelCard Stat 面板支持阈值颜色：`panel.options.thresholds` 数组 `[{ value, color }]` 自动根据当前值匹配颜色

## [v1.16.7] - 2026-04-29

### Removed
- 移除可编程告警处理链 (Event Pipeline) 功能：前端页面/路由/菜单/i18n、后端 handler/service/repository/model/engine 全部删除
- 从 onAlertFn 移除 Pipeline 拦截点，简化告警处理流程为: inhibition → mute → bizgroup → group → notify

### Fixed
- 恢复 6 个被误删的 i18n key（addQuery/runQueries/queryLabel/toggleOn/toggleOff/legendFormat），修复 Dashboard V2 查询组件显示原始 key 字符串
- Dashboard V2 列表页完整国际化 + 操作按钮（查看/编辑/删除）
- 补全英文 i18n 缺失的 dashboardV2 段
- Dashboard V2 面板网格渲染：CSS Grid 布局 + PanelCard 组件（支持 timeseries/stat/table 三种面板）
- Dashboard V2 硬编码颜色全部替换为 CSS 自定义属性，适配暗色模式

## [v1.16.4] - 2026-04-29

### Security
- P0-1: Webhook 端点增加共享密钥认证中间件 (`X-Webhook-Secret` header, constant-time compare)
- P0-2: 引入有界 goroutine 池 (`AlertWorkerPool`, 默认 64 并发)，防止告警风暴导致 goroutine 耗尽
- P0-2: `RuleEvaluator.createAlertEvent`/`resolveAlertEvent` 改用 worker pool 替代裸 `go func()`
- P0-2: `AlertEventService.processAlert`/`triggerLarkCardUpdate` 改用 worker pool
- P0-3: 修复优雅关闭顺序 (evaluator → heartbeat → groupMgr → escalation → pool.Wait() → HTTP → Redis)

### Changed
- **数据探索页面重写**: 移除复杂多目标 Grafana 风格 UI，改为简单交互：选数据源→自动匹配查询引擎→输入表达式→执行
- 自动根据数据源类型调整查询占位提示 (PromQL / LogsQL / Zabbix key)
- 查询结果图表自动适配 vector/matrix 类型
- **处理链页面完善**: 100% 国际化覆盖 (40+ i18n key)，列表页增加功能介绍说明，编辑器增加使用指南
- 处理链空状态增加引导文案和新建按钮
- 处理器节点增加 tooltip 功能描述
- 清理 `explore` i18n 中的无用 key (`addQuery`, `runQueries`, `legendFormat`, `toggleOn`, `toggleOff`, `queryLabel`)

### Added
- `internal/middleware/webhook_auth.go` — Webhook 共享密钥认证中间件
- `internal/engine/workerpool.go` — 有界 goroutine 池 (semaphore + WaitGroup)
- `config.Server.WebhookSecret` 配置项
- pipeline i18n keys (zh-CN + en): title/subtitle/create/edit/noData/noDataHint/processors/filters/editorTitle/configureNode/proc*Desc 等 40+ 键
- explore i18n keys: promqlPlaceholder/zabbixPlaceholder/metricName/value/labelsHeader

### Added
- 侧栏新增「处理链」菜单项，Pipeline 页面入口
- i18n：menu.pipelines、explore.toggleOn/Off、common.loadFailed/updateSuccess/createSuccess/confirmDelete/filters/responders 等键值
- i18n：alert.datasourceType/datasourceRequired/selectDatasourceType 键值
- docs/n9e-gap-analysis.md — n9e 功能差距分析 + 三阶段实施路线图

### Fixed
- 修复 QueryRow/QueryPanel/Explore 页硬编码颜色 → CSS 自定义属性
- 修复 A/H 切换按钮未国际化
- 修复 resolveActiveKey/pageTitle 缺失 pipelines/schedule 路由匹配
- 修复 Inhibition 页面使用不存在的 i18n 键（显示原始 key 字符串）
- 修复 Alert Rules 页面缺少 i18n 的 datasourceType 相关键
- 修复路由守卫 role 检查优先使用 Pinia Store 而非 localStorage
- 修复迁移 000006 down.sql 错误删除未创建的索引
- 修复 MODULES.md 指向不存在的 docs/alert-engine.md 和 docs/notification.md

### Removed
- 移除未使用的 mutePreviewApi、heartbeatApi 前端 API 定义
- 移除未使用的 DocumentTextOutline/GridOutline 导入
- 移除未使用的 type Labels (model/base.go)
- 移除未使用的 useScrollReveal.ts、usePromQLCompletion.ts composables
- 移除未使用的 magnetic 指令 + 注册
- 移除未引用的 datasources/Query.vue 页面（路由已重定向到 /explore）

## [v1.16.2] - 2026-04-29

### Changed
- 简化 Explore 页面布局：数据源选择器移至顶栏，移除 QueryRow 内重复选择器
- 数据源切换自动同步到所有查询目标
- 完善 i18n 国际化

## [v1.16.1] - 2026-04-29

### Changed
- 统一数据探索页面（Explore）：合并 PromQL Explore 和 LogExplorer，根据数据源类型自动切换指标/日志模式
- 侧栏新增顶级「探索」菜单，旧路由 `/datasources/query` 和 `/explore/logs` 自动重定向
- 删除独立的 `LogExplorer.vue`

## [v1.16.0] - 2026-04-29

### Added
- 统一数据探索页面（Explore）：根据数据源类型自动切换指标/日志查询模式
- Prometheus/VM 数据源 → PromQL 编辑器 + 时序图表/表格
- VictoriaLogs 数据源 → LogsQL 查询 + 日志条目表格
- 侧栏新增顶级「探索」菜单入口
- 旧路由 `/datasources/query` 和 `/explore/logs` 自动重定向到 `/explore`
- VictoriaLogs 日志查询端点：`POST /api/v1/datasources/:id/log-query`
- 中英文 i18n 支持（所有错误提示和 UI 文本）

### Fixed
- 修复数据查询页白屏：`crypto.randomUUID` 在 HTTP 非安全上下文下不可用，改用 fallback UUID 生成
- 修复登录页 401 错误显示英文：拦截器现在优先使用后端返回的业务错误码进行本地化（如 10102 → "用户名或密码错误"）

### Removed
- 删除独立的 LogExplorer.vue（合并到统一 Explore 页面）

## [v1.15.0] - 2026-04-29

### Added
- 可编程告警处理链（Event Pipeline）：DAG 可视化编辑器 + 5 种处理器
- 处理器：If（条件分支）、Relabel（标签操作）、EventDrop（告警丢弃）、Callback（Webhook 回调）、AISummary（AI 摘要）
- Pipeline CRUD 端点：`/api/v1/event-pipelines`（7 个端点）
- Pipeline 试运行：`POST /api/v1/event-pipelines/tryrun`
- Pipeline 执行记录：`GET /api/v1/event-pipelines/:id/executions`
- 前端 Pipeline 列表页 + DAG 编辑器（原生 SVG + 拖拽连线）
- 前端节点配置面板（右侧抽屉，支持各处理器类型专属配置）
- Pipeline 引擎集成到 onAlertFn（inhibition → mute → bizgroup → **pipeline** → notify）
- 迁移: 000017_event_pipelines

## [v1.14.0] - 2026-04-29

### Added
- 数据源探索页面（Explore）：PromQL 编辑器（CodeMirror 6 + 语法高亮 + 自动补全）
- Range Query 支持：POST /api/v1/datasources/:id/query-range
- 数据源标签代理端点：GET /api/v1/datasources/:id/labels/keys、labels/values、metrics
- ECharts 时间序列图表（dataZoom、tooltip cross 指针、Legend 统计表格）
- 时间范围选择器（相对/绝对时间）+ 自动刷新
- 多查询支持、Legend 格式化、Chart/Table 视图切换
- 仪表盘 V2 系统：Dashboard CRUD 端点（/api/v1/dashboards）
- 变量模板系统：query/custom/textbox/constant 类型，$var 替换
- 仪表盘列表页和查看页（全局时间范围、变量选择器）
- 值格式化工具（bytes/seconds/percent/short/scientific）
- 迁移: 000016_dashboards

### Changed
- /datasources/query 路由指向新的 Explore 页面（替代原生 HTML 查询页）

## [v1.11.0] - 2026-04-27

### Added
- 登录页密码/用户名错误 inline 提示（表单内 alert 替代右上角 message）
- 数据源卡片显示版本号（健康检查成功后持久化 version 字段）
- 数据源状态标签国际化（healthy/unhealthy/unknown 随语言切换）
- 密码复杂度校验（最少 8 位，含大小写字母和数字）
- JWT 超时可配置（系统设置 > 安全配置，预设 1h/4h/8h/24h/7d）
- 数据源查询页面（选择数据源 + 输入 PromQL/LogQL 执行查询）
- 迁移: 000015_datasource_version

### Removed
- 登录页默认账号 admin/admin123 提示

### Changed
- AuthService.Login / RefreshToken 读取 system_settings 中的 jwt_expire_seconds
- handler/auth.go, handler/user.go 密码最小长度约束从 6 提升至 8

## [v1.10.0] - 2026-04-26

### Added
- 测试框架：internal/testutil/ (TestDB, SeedUser, SeedAlertRule, CleanupDB)
- 测试骨架：service/alert_channel_test.go, handler/alert_channel_test.go
- docs/testing.md 测试策略和覆盖目标
- docs/prompts.md AI 提示词模板（新功能/Bug/审查/测试等）
- CLAUDE.md 对话规范（token 节省规则）
- config.example.yaml OIDC 配置段
- GET /schedules/:id/participants 后端 handler + 路由
- GET /schedules/:id/overrides 后端 handler + 路由
- POST /alert-channels/:id/test 后端 handler + 路由

### Fixed
- 修复 3 个前端 API 调用无后端路由的问题（schedule participants/overrides, alert-channel test）

### Removed
- 4 个孤立 Vue 组件（SpotlightCursor, SeverityTag, StatusTag, SkeletonCard）
- 废弃 TS 类型（NotifyChannel, NotifyPolicy v1）
- 无关文档（3th_monitor_readme.md）
- scripts/test-api.sh 中的 v1 通知端点

## [v1.9.10] - 2026-04-26

### Fixed
- label_registry.label_value 从 VARCHAR(512) 扩展到 VARCHAR(2048)，修复 Prometheus 长标签值导致 MySQL Error 1406
- SyncDatasource / RecordFromLabels 添加 >2048 截断安全网
- **迁移**: 000014_label_value_extend

## [v1.9.9] - 2026-04-26

### Added
- Alertmanager 风格 group_wait / group_interval 通知分组
- AlertGroupManager 在引擎回调和 RouteAlert 之间缓冲 firing 事件
- AlertRule 新增 group_wait_seconds / group_interval_seconds 字段
- 前端告警规则表单新增分组等待/间隔配置
- **迁移**: 000013_alert_rule_group_timing

## [v1.9.8] - 2026-04-25

### Added
- CLAUDE.md 与 .opencode/context.md 合并为单一 AI 导航文件
- .gitignore 添加 .claude/ 和 .opencode/ 排除
- Claude Code 全局 settings.json 权限配置

## [v1.6.0] - 2026-04-20

### Added
- 系统级 SMTP 配置（system_settings group=smtp）
- 升级策略 email 分支接入系统 SMTP 真实发送
- JWT 7天宽限续签（POST /auth/refresh）
- 前端 Axios 401 自动刷新 token
- 头像 Go 层大小校验（≤272KB data URL）
- GET /alert-events/export CSV 流式导出
- GET /mute-rules/preview 命中预览
- Lark OpenID → DB User 映射（user.lark_user_id）
- 个人设置新增「飞书账号绑定」tab
- 数据源健康检查返回 latency/version 富结果
- **迁移**: 000008_create_inhibition_rules, 000009_create_label_registry, 000010_alert_rule_datasource_optional, 000011_alert_rule_datasource_type

## [v1.5.0] - 2026-04-15

### Added
- 升级策略 lark_personal 分支接入 Lark Bot API（DM）
- 告警 AutoResolve 时同步 PATCH Lark 卡片
- LarkBotService.SendMessage 优先用 Bot API 回复 chatID
- NotifyChannel Bot API 类型在 TestChannel 支持真发送
- **迁移**: 000006_heartbeat_sla_alert_rules, 000007_sla_escalated_at_alert_events

## [v1.3.1] - 2026-04-10

### Added
- MTTA/MTTR P50/P95 百分位、按严重程度细分
- MTTA/MTTR 每日趋势折线图
- 品牌 logo.svg（sider/login/favicon 统一）
- 个人信息头像扩展为 32 个预设 emoji + 自定义上传

### Fixed
- 顶部栏保存头像后仍显示用户名首字母

## [v1.3.0] - 2026-04-08

### Changed
- 设计系统级视觉翻新：CSS token + Naive UI GlobalThemeOverrides
- 侧栏/顶栏/登录页玻璃态皮肤（dark + light）

## [v1.2.0] - 2026-04-05

### Added
- 告警规则分类 tab
- 仪表盘分析图表（趋势 + Top 规则）
- 操作审计日志
- 表达式实时测试
- **迁移**: 000004_audit_logs, 000005_add_rule_category

## [v1.1.x] - 2026-04-01

### Added
- 告警详情页改版（严重等级横幅 + 生命周期时间线）
- 通知模块合并为单页 Tabs
- **迁移**: 000003_alert_event_lark_message_id

## [v1.0.x] - 2026-03-25

### Added
- OIDC 配置 UI（存 DB）
- K8s 清单
- 多数据源集成
- RBAC 三级权限
- **迁移**: 000001_initial_schema, 000002_system_settings

> Phase 追踪和 QA 修复汇总已移至 [docs/phases.md](docs/phases.md)
