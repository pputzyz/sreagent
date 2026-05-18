# SREAgent AI Agent 路线图

> 2026-05-18 | 基于 PagerDuty 路线 + 国内 SRE 实践

---

## 核心判断

**当前瓶颈不是 LLM 能力不够，而是喂给它的结构化上下文太薄。**

SREAgent 已有的 incident pipeline + label registry + biz_group + oncall 四件套，正好是构建这个上下文的基础设施。

---

## 七维能力对照表

| # | 能力维度 | SREAgent 现状 | 差距 | 阶段 |
|---|----------|--------------|------|------|
| 1 | 告警→事故对象 | v2 pipeline + incident 状态机 + noise reducer + inhibition | AI 未在 incident 粒度工作 | P1 |
| 2 | 语义治理 | label_registry + JSONLabels + biz_group | service/env/component/owner/runbook/change_id 未强制结构化 | P1 |
| 3 | 历史事故库 | postmortem + chat_history 存在 | 未向量化索引，AI 无法召回相似历史 | P2 |
| 4 | 变更一等公民 | 无 | 最大缺口 — 无 CI/CD 集成 | P3 |
| 5 | 服务-组织模型 | team + schedule + biz_group 分散存在 | 未形成 "告警→服务→团队→值班人" 知识图谱 | P1-P2 |
| 6 | 自动化分阶段 | AI 仅做分析建议 | 无诊断工作流编排、无自动修复 | P2-P3 |
| 7 | 协作本地化 | Lark 完整 + webhook | 缺钉钉/企微/飞书文档深层集成 | 持续 |

---

## Phase 1 — 结构化上下文（让 AI 不再猜）

**目标：** 把 incident 的完整上下文打包送给 LLM，而不是只送一条告警。

### 1.1 强制 Label 语义规范

扩展现有 `label_registry` 为结构化语义字段：

```
必须字段（规则创建时校验）：
- service      → 所属服务名（如 order-service）
- env          → 环境（prod/staging/dev）
- component    → 组件（api/db/cache/queue）
- owner        → 负责团队（对应 team.name）
- severity     → 严重等级（critical/warning/info）
- business_impact → 业务影响描述（自由文本）

推荐字段（自动补全 + 模板引导）：
- runbook_url  → 处理手册链接
- change_id    → 关联变更单号
- cluster      → 集群标识
- instance     → 实例标识
```

**实现方案：**
- `handler/alert_rule.go` 创建规则时校验必须字段
- `label_registry` 新增 `required` 和 `semantic_type` 属性
- 前端 RuleFormModal 根据 semantic_type 渲染不同输入控件

### 1.2 Incident 上下文聚合器

新建 `service/incident_context.go`：

```go
type IncidentContext struct {
    Incident      *model.Incident
    RelatedAlerts []model.AlertEvent        // 同 fingerprint 的历史告警
    RelatedChanges []ChangeEvent            // 时间窗口内的变更（Phase 3 接入）
    SimilarPostMortems []model.PostMortem   // 向量化相似复盘（Phase 2 接入）
    OnCallPerson  *model.User               // 当前值班人
    Team          *model.Team               // 负责团队
    Runbook       string                    // 处理手册内容
    Labels        map[string]string         // 结构化标签
}

func (s *IncidentContextService) BuildContext(ctx context.Context, incidentID uint) (*IncidentContext, error)
```

**数据来源（当前可实现）：**
- RelatedAlerts: `alert_event_repo.ListByFingerprint()` — 已有
- OnCallPerson: `schedule_svc.GetCurrentOnCallForAlert()` — 已有
- Team: `biz_group.FindMatchingGroups()` → team — 已有
- Labels: incident.Labels — 已有
- Runbook: label_registry 中的 runbook_url → HTTP 获取 — 新增

### 1.3 AI Prompt 模板升级

改造 `service/ai.go` 的 `AnalyzeAlert` 方法：

```
当前：单条告警的 labels + annotations → LLM
升级：IncidentContext 全量结构化数据 → LLM

System Prompt 增加约束：
- 你是一个 SRE AI 助手，基于以下结构化上下文分析事故
- 必须引用具体的 label 字段和服务名称
- 必须参考历史复盘（如有）和处理手册（如有）
- 输出格式：根因假设（概率排序）→ 推荐操作 → 涉及组件
- 禁止猜测没有数据支撑的结论
```

---

## Phase 2 — RAG + 诊断工作流

### 2.1 历史事故库向量化

**技术选型：**
- Embedding: OpenAI text-embedding-3-small 或本地 bge-m3
- 向量 DB: pgvector（已有 PostgreSQL）或 Qdrant（独立部署）
- 存储内容: postmortem 摘要 + 根因 + 修复方案 + 时间窗口 + 涉及服务

**实现方案：**
- 新增 `service/knowledge_service.go`
- PostMortem 创建/更新时自动生成 embedding 并存入向量 DB
- IncidentContext.BuildContext 时执行相似度检索，取 Top-5 相似复盘

### 2.2 诊断 SOP 编排引擎

新建 `model/diagnostic_workflow.go`：

```go
type DiagnosticWorkflow struct {
    ID          uint
    Name        string             // "Redis 延迟排查" / "Pod OOM 诊断"
    TriggerLabels map[string]string // 匹配条件
    Steps       []DiagnosticStep
}

type DiagnosticStep struct {
    Order       int
    Type        string // "query" / "command" / "check" / "notify"
    DatasourceID *uint  // 查询哪个数据源
    Expression  string // PromQL / SQL / shell command
    Condition   string // "result > 100" → 自动判断是否异常
    AutoAdvance bool   // 是否自动进入下一步
    RequireApproval bool // 是否需要人工确认
}
```

**工作流：**
1. Incident 触发 → 匹配 DiagnosticWorkflow（按 label）
2. AI 生成执行计划（基于 workflow 模板 + incident 上下文）
3. 展示给值班人确认
4. 逐步执行，每步结果反馈给 AI 做下一步决策
5. 全部完成后生成诊断报告

### 2.3 AI 诊断 Agent

```
用户/AI 触发诊断
    ↓
匹配 workflow 模板
    ↓
AI 生成执行计划（带步骤和预期结果）
    ↓
人工确认（可选：设置 auto-approve 级别）
    ↓
执行器逐步执行 → 收集结果
    ↓
AI 分析结果 → 决定下一步 / 结束
    ↓
生成诊断报告 → 存入 postmortem
```

---

## Phase 3 — 有 Guardrails 的自动化闭环

### 3.1 变更事件接入

**集成点：**
- CI/CD webhook → `POST /api/v1/integrations/changes`
- 存入 `change_events` 表
- IncidentContext 自动关联时间窗口内的变更

**ChangeEvent 模型：**
```go
type ChangeEvent struct {
    ID          uint
    Source      string    // "gitlab" / "jenkins" / "argocd" / "manual"
    Type        string    // "deploy" / "config" / "db_migration" / "scaling"
    Service     string    // 涉及服务
    Environment string    // 环境
    CommitSHA   string
    Author      string
    Description string
    Timestamp   time.Time
    RiskLevel   string    // "low" / "medium" / "high"
}
```

### 3.2 自动修复（with Guardrails）

**三级自动化：**

| 级别 | 操作 | 审批要求 | 示例 |
|------|------|----------|------|
| L1 建议 | AI 生成建议，人工执行 | 无 | "建议扩容 order-service 到 5 副本" |
| L2 半自动 | AI 生成操作，人工确认执行 | 每次确认 | 静默规则创建、通知升级触发 |
| L3 全自动 | AI 自动执行，事后通知 | 仅高危操作需确认 | 已知模式的自动扩容、自动静默 |

**实现方案：**
- `model/auto_action.go` — 自动操作定义（action type + guardrails + approval level）
- `service/auto_remediation.go` — 执行引擎
- 每个 action 有 `dry_run` 模式，先模拟再执行
- 所有自动操作写入 `audit_log`

### 3.3 反馈循环

```
自动修复执行
    ↓
监控修复效果（30min 窗口）
    ↓
效果评估：告警是否恢复 / 是否引入新问题
    ↓
更新 SOP 置信度（成功+1 / 失败-1）
    ↓
低置信度 SOP 标记为需人工审查
```

---

## 与现有模块的集成点

| 现有模块 | Phase 1 改动 | Phase 2 改动 | Phase 3 改动 |
|----------|-------------|-------------|-------------|
| label_registry | 新增 required + semantic_type | — | — |
| alert_rule | 创建时校验必须字段 | — | — |
| incident | — | 关联 DiagnosticWorkflow | 关联 ChangeEvent |
| postmortem | — | 向量化索引 | 自动创建 |
| ai_service | 上下文聚合 + prompt 升级 | RAG 检索 + 诊断编排 | 自动修复决策 |
| schedule | 提供值班人上下文 | — | — |
| team | 提供团队上下文 | — | 审批链路 |
| audit_log | — | 记录诊断步骤 | 记录自动操作 |
| integration | — | — | 接收 CI/CD webhook |

---

## 实施建议

**Phase 1（1-2 个月）：** label 规范 + 上下文聚合器 + prompt 升级
- 不需要新技术栈
- 最大收益：AI 分析质量从"猜"提升到"有据可依"

**Phase 2（2-3 个月）：** 向量 DB 接入 + 诊断工作流
- 需要引入 pgvector 或 Qdrant
- 需要前端诊断工作流编辑器

**Phase 3（3-6 个月）：** 变更接入 + 自动修复 + 反馈循环
- 需要 CI/CD 集成开发
- 需要完善的 RBAC 和审批机制
- 安全审计要求最高
