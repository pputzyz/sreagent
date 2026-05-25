# Nightingale V9 功能移植清单

> 基于三代理全面审计结果，逐项移植所有有价值的 Nightingale 功能到 SREAgent。

## 状态说明
- ⬜ 未开始
- 🔄 进行中
- ✅ 完成
- ❌ 不移植（SREAgent 已有更好的替代方案）
- ⏭️ 低优先级（后续版本）

---

## 第一优先级：核心功能缺失

### 1. Recording Rules（录制规则）⬜
**Nightingale 参考：**
- 前端：`C:\project\fe\src\pages\recordingRules\`
- 后端：`C:\project\nightingale\models\recording_rule.go`，`C:\project\nightingale\center\router\router_recording_rule.go`
- API：8 个端点（CRUD + 批量操作 + 按业务组查询）

**功能清单：**
- [ ] 数据库迁移（recording_rules 表）
- [ ] 后端 Model/Repository/Service/Handler
- [ ] 路由注册
- [ ] 前端页面：列表页（筛选、分页、批量操作）
- [ ] 前端页面：新增/编辑表单（PromQL 编辑器、数据源选择、cron 模式、附加标签）
- [ ] 前端功能：克隆、导入/导出 JSON、批量更新、批量删除
- [ ] 前端功能：启用/禁用切换

**SREAgent 适配：**
- 使用现有 biz_group 体系
- PromQLEditor 组件复用
- 路由放在 alerts/ 或新建 recording-rules/ 目录

---

### 2. Metrics Builtin（内置指标目录）⬜
**Nightingale 参考：**
- 前端：`C:\project\fe\src\pages\metricsBuiltin\`
- 后端：`C:\project\nightingale\models\builtin_metrics.go`
- API：8 个端点（CRUD + 类型/采集器元数据 + 筛选器 + PromQL 构建器）

**功能清单：**
- [ ] 数据库迁移（builtin_metrics + metric_filter 表）
- [ ] 后端完整 CRUD
- [ ] 元数据端点（types/collectors）
- [ ] 前端页面：指标列表（类型/采集器/单位筛选、搜索、分页）
- [ ] 前端页面：指标表单抽屉（新增/编辑/克隆）
- [ ] 前端功能：批量导入/导出、批量删除
- [ ] 前端功能：Explorer 抽屉（点击指标名打开查询面板）
- [ ] 前端功能：标签筛选器管理
- [ ] 前端功能：列自定义

---

### 3. Event Pipeline（事件管道）⬜
**Nightingale 参考：**
- 前端：`C:\project\fe\src\pages\eventPipeline\`
- 后端：`C:\project\nightingale\models\event_pipeline.go`
- 功能：DAG 事件处理（relabel、label_enrich、callback、ai_summary 处理器）

**功能清单：**
- [ ] 数据库迁移
- [ ] 后端处理引擎
- [ ] 前端可视化管道编辑器
- [ ] 处理器类型：relabel、label_enrich、callback、ai_summary
- [ ] 与告警引擎集成

---

## 第二优先级：功能增强

### 4. 通知渠道扩展（4 → 20+）⬜
**当前 SREAgent：** 飞书、邮件、Webhook、短信
**Nightingale 额外：** FlashDuty、PagerDuty、DingTalk、WeCom、Telegram、Slack、Discord、Line、Mattermost、Microsoft Teams、Tencent SMS、Aliyun SMS、Voice Call

**功能清单：**
- [ ] 通知渠道模型扩展
- [ ] 各渠道适配器实现
- [ ] 前端渠道配置页面更新
- [ ] 渠道测试功能

---

### 5. Dashboard 增强 ⬜
**当前 SREAgent：** 基础仪表盘
**Nightingale 额外：**
- [ ] 面板编辑器（13+ 面板类型）
- [ ] 22 种数据转换
- [ ] 模板变量
- [ ] 仪表盘链接
- [ ] 注解（annotations）
- [ ] 图表分享

---

### 6. Target/Host 管理 ⬜
**Nightingale 参考：** 设备管理、标签绑定、业务组分配

**功能清单：**
- [ ] 设备发现和注册
- [ ] 标签绑定
- [ ] 业务组分配
- [ ] 设备详情页
- [ ] 批量操作

---

## 第三优先级：Explore 页面完善

### 7. PromQL 变量插值 ⬜
- [ ] `$__from`、`$__to`、`$__interval`、`$__rate_interval` 支持
- [ ] 在查询执行时自动替换

### 8. ViewSelect 后端持久化 ⬜
- [ ] 当前 localStorage → 后端 API 持久化
- [ ] 收藏功能

### 9. Saved Views 后端持久化 ⬜
- [ ] 当前 localStorage → 后端 API 持久化

### 10. PromQL 自动补全连接 API ⬜
- [ ] PromQLEditor 连接 Prometheus/VictoriaMetrics API
- [ ] 指标名和标签名自动补全
- [ ] 标签值自动补全

---

## 第四优先级：缺失前端页面

### 11. Knowledge Base 前端页面 ⬜
- [ ] 知识库列表/编辑页面
- [ ] Markdown 编辑器集成

### 12. Diagnostic Workflows 前端页面 ⬜
- [ ] 工作流设计器
- [ ] 执行历史

### 13. Change Events 前端页面 ⬜
- [ ] 变更事件列表/详情

### 14. Alert Rule Templates 前端页面 ⬜
- [ ] 模板管理页面
- [ ] 模板应用到规则

---

## 第五优先级：高级功能

### 15. ES/ClickHouse/Log Explorer 扩展 ⬜
- [ ] 索引模式
- [ ] 字段侧边栏
- [ ] 日志聚类

### 16. Embedded Products（嵌入产品）⬜
- [ ] iframe 嵌入支持
- [ ] 访问控制

### 17. Job Templates & Tasks（作业模板）⬜
- [ ] 远程执行框架
- [ ] 作业模板管理
- [ ] 执行历史

### 18. Metric Descriptions（指标描述）⬜
- [ ] 为指标名附加描述信息
- [ ] 在查询界面显示

### 19. Batch Operations 扩展 ⬜
- [ ] 跨模块批量操作支持

---

## 移植策略

### 后端适配原则
1. 遵循 SREAgent 分层：handler → service → repository → model
2. 使用现有 biz_group、datasource 体系
3. 错误码使用 SREAgent 标准（10001/10002/10200 等）
4. 迁移文件使用 `000067_xxx` 格式

### 前端适配原则
1. Vue 3 + Naive UI + Pinia（不用 Ant Design）
2. 复用现有 composable（useCrudModal、usePaginatedList）
3. 路由放在对应功能目录下
4. i18n 使用 vue-i18n

### 执行顺序
1. Recording Rules → Metrics Builtin → Event Pipeline（核心功能）
2. 通知渠道 → Dashboard 增强 → Target 管理（功能增强）
3. Explore 完善 → 缺失页面 → 高级功能（体验优化）

---

## 进度追踪

| # | 模块 | 状态 | 版本 | 备注 |
|---|------|------|------|------|
| 1 | Recording Rules | ⬜ | v4.27.0 | |
| 2 | Metrics Builtin | ⬜ | v4.28.0 | |
| 3 | Event Pipeline | ⬜ | v4.29.0 | |
| 4 | 通知渠道扩展 | ⬜ | v4.30.0 | |
| 5 | Dashboard 增强 | ⬜ | v4.31.0 | |
| 6 | Target/Host 管理 | ⬜ | v4.32.0 | |
| 7 | PromQL 变量插值 | ⬜ | - | |
| 8 | ViewSelect 持久化 | ⬜ | - | |
| 9 | Saved Views 持久化 | ⬜ | - | |
| 10 | PromQL 自动补全 | ⬜ | - | |
| 11 | Knowledge Base 页面 | ⬜ | - | |
| 12 | Diagnostic Workflows 页面 | ⬜ | - | |
| 13 | Change Events 页面 | ⬜ | - | |
| 14 | Alert Rule Templates 页面 | ⬜ | - | |
| 15 | Log Explorer 扩展 | ⬜ | - | |
| 16 | Embedded Products | ⬜ | - | |
| 17 | Job Templates | ⬜ | - | |
| 18 | Metric Descriptions | ⬜ | - | |
| 19 | Batch Operations | ⬜ | - | |
