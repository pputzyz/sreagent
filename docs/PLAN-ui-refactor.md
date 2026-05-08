# UI 全面重构计划（FlashCat 对齐）

> 目标：将所有页面对齐 FlashCat 设计语言（简洁、紧凑、视觉层次清晰、交互直觉）。
> 起点：v2.1.0 已完成数据查询、侧栏、故障列表、协作空间列表
> 方法：分 5 个 Phase，每个 Phase 内多 agent 并行，完成后打 tag 推送

---

## 设计语言（贯穿所有 Phase）

### 视觉原则
1. **配色**：critical=#ef4444 / warning=#f59e0b / info=#3b82f6 / success=#10b981 / 主色见 CSS token
2. **圆点 + 文字** 取代大色块 tag（严重程度、状态等）
3. **左侧色条** 取代背景填色（4px 视觉分隔，不喧宾夺主）
4. **卡片 hover** 微抬升（translateY -2px + shadow 加深 + border 主色）
5. **Group label** uppercase 11px 600，颜色 var(--sre-text-tertiary)
6. **空状态** 大图标 + 友好文案 + 主操作 CTA
7. **筛选栏** 紧凑横排：分段控件 + 下拉 + 搜索框

### 交互原则
1. 行点击进入详情，**不留独立"查看"按钮**
2. 操作按钮聚合到 **省略号下拉**（编辑/删除/导出等）
3. 创建/编辑用 **modal 弹窗**，配置类用 **drawer 侧滑**
4. 表格优先 `size="small"`，密度优于留白
5. 危险操作（删除/合并）必须 `n-popconfirm` 二次确认

---

## Phase 1：核心日常页（v2.2.0）

最高频使用，第一感知最强。

| 页面 | 文件 | 关键改造点 |
|------|------|---------|
| 仪表盘 | `dashboard/Index.vue` | 删除冗余 GlowCard / AuroraBackground 视觉特效，改为 FlashCat 卡片网格（4 张关键指标 + 趋势图 + Top 列表） |
| 故障看板 | `dashboard/IncidentDashboard.vue` | 同上，统一视觉到 Phase 1 卡片标准 |
| 告警规则 | `alerts/rules/Index.vue` | 紧凑筛选栏，规则状态用圆点+文字，分类用 group/segmented，启用开关替代 tag |
| 活跃告警 | `alerts/events/Index.vue` | 仿 incidents 列表风格：左侧色条+三行结构，去除大表格 |
| 设置首页 | `settings/Index.vue` | 设置 tab 改为左侧导航 + 右侧内容区（FlashCat 风格） |

---

## Phase 2：详情页（v2.3.0）

详情页是用户处理实际工作的核心场所。

| 页面 | 文件 | 关键改造点 |
|------|------|---------|
| 故障详情 | `incidents/Detail.vue` | 三栏布局（左操作+中Tab+右元数据），状态时间线竖向，复盘 Tab 提升视觉 |
| 协作空间详情 | `channels/Detail.vue` | Tab 风格统一，配置 Tab 用 collapse 折叠组 |
| 告警 v2 详情 | `alerts-v2/Detail.vue` | Header banner（严重程度色） + 关联事件流水线 |
| 告警事件详情 | `alerts/events/Detail.vue` | 同 alerts-v2 风格 |
| 集成中心 | `integrations/Index.vue` | 卡片化（不是表格），shared/exclusive 视觉区分 |

---

## Phase 3：配置类页面（v2.4.0）

| 页面 | 文件 | 关键改造点 |
|------|------|---------|
| 通知中心 | `notification/Index.vue` + 5 子页 | Tab 改 sub-route，每个子页用统一卡片列表 |
| 屏蔽规则 | `alerts/mute/Index.vue` | 卡片列表代替表格，命中预览右侧抽屉 |
| 抑制规则 | `alerts/inhibition/Index.vue` | 同上 |
| 排班 | `schedule/Index.vue` | 日历视图保留，侧栏简化 |
| 数据源 | `datasources/Index.vue` | 卡片列表（按类型分组），健康检查状态点 |
| 历史告警 | `alerts/history/Index.vue` | 紧凑表格 + 时间筛选 |

---

## Phase 4：设置子页（v2.5.0）

| 页面 | 文件 | 关键改造点 |
|------|------|---------|
| AI 配置 | `settings/AIConfig.vue` | 表单分组卡片，连通性测试按钮内联 |
| 飞书机器人 | `settings/LarkBotConfig.vue` | 同上 |
| SMTP | `settings/SMTPConfig.vue` | 同上 |
| OIDC | `settings/OIDCConfig.vue` | 同上 |
| 用户管理 | `settings/UserManagement.vue` | 紧凑表格 + 角色 segmented 筛选 |
| 团队 | `settings/TeamManagement.vue` | 卡片列表（成员头像组） |
| 业务组 | `settings/BizGroupManagement.vue` | 树形 + 右侧详情 |
| 虚拟用户 | `settings/VirtualUsers.vue` | 紧凑表格 |
| 审计日志 | `settings/AuditLog.vue` | 时间线视图 + 操作类型筛选 |
| 登录页 | `Login.vue` | 简化背景，左 illustration 右表单 |

---

## Phase 5：体验打磨（v2.6.0）

跨页面的统一性优化。

1. **设计 tokens 整合**：`web/src/styles/tokens.css` 整理出完整变量表（间距/圆角/阴影/字号），文档化
2. **暗色模式**：所有页面 dark mode 走查，修复对比度问题
3. **响应式**：移动端 / 平板布局走查（Sider 自动收起、Modal 全屏）
4. **空状态库**：`components/common/EmptyState.vue` 统一化（不同场景预设图标+文案）
5. **表格密度模式**：用户偏好 compact / comfortable 切换
6. **键盘导航**：列表上下键、Enter 进详情、Esc 关 Modal/Drawer
7. **加载骨架屏**：高频页面用 skeleton 替代 spinner
8. **Toast 反馈风格**：统一 message API 调用模式（按钮二次反馈）

---

## 执行节奏

- Phase 1 → tag `v2.2.0`
- Phase 2 → tag `v2.3.0`
- Phase 3 → tag `v2.4.0`
- Phase 4 → tag `v2.5.0`
- Phase 5 → tag `v2.6.0`（整体收尾发 v2.6.0 stable）

每个 Phase 内：
1. 多 agent 并行重构（每个 agent 一个或一组页面）
2. 集成提交：`vue-tsc ✅` + `go build ✅`（如改动后端）
3. 更新 `CHANGELOG.md` + `docs/PLAN-status.md`
4. 单一 commit + tag + push 推 GitHub 触发 CI

---

## 当前状态（截至 2026-05-08）

| Phase | 状态 |
|-------|------|
| **v2.1.0 已完成** | 数据查询、侧栏、故障列表、协作空间列表 |
| **Phase 1（v2.2.0）已完成** | dashboard / settings / alert-rules / alert-events |
| **Phase 2（v2.3.0）已完成** | incident-detail / channel-detail / alerts-v2 (×2) / alert-event-detail / integrations |
| **Phase 3（v2.4.0）已完成** | notification (×6) / mute / inhibition / history / datasources / schedule |
| **Phase 4（v2.5.0）已完成** | settings 4 config + 3 org + bizgroup + audit-log + Login |
| Phase 5 (v2.6.0)  | ⬜ 未开始 |
