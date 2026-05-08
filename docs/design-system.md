# SREAgent Design System

> Version 2.5+ · FlashCat-aligned Operational Refinement aesthetic
> Last updated: 2026-05-08

## 设计哲学

SREAgent v2.x 走的是 **refined minimalism**：拒绝 AI slop（紫色渐变 / 立体阴影 / 卡通插画 / 圆角狂魔），向 Linear、Splunk Observability、FlashCat 学密度与克制。每个像素都要为「告警工程师在凌晨三点能否快速读懂」服务——所以我们偏爱左色条 stripe 而不是大色块、单点 dot 而不是色块 tag、4px 网格而不是任意间距、tabular nums 而不是浮动数字。绿色主调（#18a058）保留，所有情绪靠 severity stripe + status dot 透出。

## Typography

### Font stack

- **Display / Sans**: Geist（通过 `web/index.html` 的 Google Fonts 引入）
- **Mono / Numbers**: JetBrains Mono（同样 Google Fonts 引入；fallback 到 SF Mono / Consolas）
- **CJK fallback**: PingFang SC / Hiragino Sans GB
- **拒绝**：Inter / Roboto / Arial / system defaults / 苹方以外的中文字体

### Tokens

| Token | Value | Usage |
|-------|-------|-------|
| `--sre-font-display` | Geist + CJK fallback | 大标题、品牌区 |
| `--sre-font-sans` | Geist + 完整 fallback | 通用 UI 文字 |
| `--sre-font-mono` | SF Mono / JetBrains Mono / Consolas | 数字、代码、labels、IDs |

### Type scale

| Token | Value | Usage |
|-------|-------|-------|
| `--sre-fs-2xs` | 10px | micro labels |
| `--sre-fs-xs`  | 11px | eyebrow / dense meta |
| `--sre-fs-sm`  | 12px | secondary text |
| `--sre-fs-md`  | 13px | body small |
| `--sre-fs-base`| 14px | body |
| `--sre-fs-lg`  | 16px | section title |
| `--sre-fs-xl`  | 18px | sub-hero |
| `--sre-fs-2xl` | 22px | page title |
| `--sre-fs-3xl` | 28px | hero |
| `--sre-fs-4xl` | 34px | display |

Weights: `--sre-fw-regular: 400` / `--sre-fw-medium: 500` / `--sre-fw-semibold: 600` / `--sre-fw-bold: 700`.
Line-heights: `--sre-lh-tight: 1.2` / `--sre-lh-snug: 1.35` / `--sre-lh-normal: 1.5` / `--sre-lh-loose: 1.7`.

### Tabular nums

所有数字（counts / timestamps / durations / IDs / latencies）必须 `font-variant-numeric: tabular-nums`，统一用 `.tnum` / `[data-nums]` / `.num` utility class，避免数字滚动时左右晃动。

## Color

### Brand & severity

| Role | Token | Value | Usage |
|------|-------|-------|-------|
| Primary | `--sre-primary` | `#18a058` | CTA / active / success state |
| Critical | `--sre-critical` | `#ef4444` | 4px stripe / dot / chip |
| Warning | `--sre-warning` | `#f59e0b` | warning stripe / dot |
| Info | `--sre-info` | `#3b82f6` | info stripe / dot |
| Success | `--sre-success` | `#10b981` | resolved state |

每个 semantic color 都有对应 `-soft` 变体（dark mode 14% 透明、light mode 10% 透明），用于背景填色。

### Surfaces

| Token | Dark | Light |
|-------|------|-------|
| `--sre-bg-base` | `#07090d` | `#eef1f5` |
| `--sre-bg-page` | `#0b0e14` | `#f3f5f8` |
| `--sre-bg-card` | `#121722` | `#ffffff` |
| `--sre-bg-elevated` | `#192030` | `#ffffff` |
| `--sre-bg-sunken` | `#05070a` | `#eff2f6` |
| `--sre-bg-hover` | `rgba(255,255,255,0.045)` | `rgba(15,23,42,0.04)` |
| `--sre-bg-active` | `rgba(255,255,255,0.08)` | `rgba(15,23,42,0.08)` |

### Text

| Token | Dark | Light |
|-------|------|-------|
| `--sre-text-primary` | 92% white | 92% slate-900 |
| `--sre-text-secondary` | 60% white | 56% slate-900 |
| `--sre-text-tertiary` | 38% white | 38% slate-900 |
| `--sre-text-muted` | 24% white | 22% slate-900 |
| `--sre-text-inverse` | `#0b0e14` | `#ffffff` |

WCAG AA 验证（light theme）：text-primary `rgba(15,23,42,0.92)` on `#ffffff` ≈ 14.5:1 ✓; text-secondary 56% ≈ 8.6:1 ✓; text-tertiary 38% ≈ 5.8:1 ✓ AA。

### Borders

`--sre-border` (subtle hairline) / `--sre-border-strong` (hover state) / `--sre-border-focus` = primary。
**重要**: 不要在 component 内部写死 `rgba(255,255,255,0.x)` 边框，用 `--sre-border-strong`，否则浅色模式下白底白边不可见。

## Spacing

4px 网格：`--sre-space-1` (4px) → `--sre-space-2` (8) → `-3` (12) → `-4` (16) → `-5` (20) → `-6` (24) → `-8` (32) → `-10` (40) → `-12` (48) → `-16` (64)。

## Radius

| Token | Value | Usage |
|-------|-------|-------|
| `--sre-radius-xs` | 4px | hairline pills |
| `--sre-radius-sm` | 6px | buttons / chips |
| `--sre-radius-md` | 10px | row cards |
| `--sre-radius-lg` | 14px | surface cards |
| `--sre-radius-xl` | 20px | hero panels |
| `--sre-radius-2xl` | 28px | rare; only for marketing-style |
| `--sre-radius-pill` | 9999px | tag / scrollbar |

> 圆角 > 16px 留给装饰性表面，**列表/表格/输入框 ≤ 10px**，否则会显得卡通。

## Component patterns

### Row card with severity stripe (`.sre-row-card[data-severity]`)

```html
<div class="sre-row-card" data-severity="critical">
  <div class="grow">…</div>
  <div>…</div>
</div>
```

4px 左色条按 severity 着色（critical / warning / info / success；缺省 = tertiary 灰）。`data-dim="true"` 可让整行淡化 0.6（已确认/已静默场景）。Hover 时背景切到 `--sre-bg-hover`、边框升到 `--sre-border-strong`。

### Status / severity dot (`.sre-dot[data-severity]`)

```html
<span class="sre-dot" data-severity="critical" data-pulse></span>
```

8px 圆点，可选 `data-pulse` 启用 1.6s ease 脉动。替代 NTag 用作紧凑 inline severity。

### Eyebrow label (`.sre-label-eyebrow`)

```html
<span class="sre-label-eyebrow">Active Alerts</span>
```

11px / 600 / uppercase / tracking 1px / `--sre-text-tertiary`。用于小标题、KPI label、分组标题。

### Stagger reveal (`.sre-stagger`)

容器加此 class，子元素首次加载错峰浮现（每个 40ms 延迟，最多到第 8 个，第 9 起统一 320ms）。也提供 `.stagger-grid`（55ms 间隔，到第 12 个）和 `.stagger-list`（用 `--sre-stagger-i` inline 设置 index）。

### Hover lift (`.sre-lift`)

hover 时 `translateY(-1px)` + 边框升到 `--sre-border-strong`。轻量 KPI / dashboard 卡片标配。

### Meta divider (`.sre-meta-divider`)

```html
<span>by alice</span>
<span class="sre-meta-divider"></span>
<span>2m ago</span>
```

3px 圆点分隔符，元数据排列时替代竖线 `|`。

### Tabular nums (`.tnum` / `[data-nums]` / `.num`)

```html
<span class="tnum">{{ count }}</span>
```

强制 `font-variant-numeric: tabular-nums + tnum 1`。所有数字必须包一层。

### Empty state component

```vue
<EmptyState icon="…" title="…" description="…">
  <NButton>Create rule</NButton>
</EmptyState>
```

替代 NEmpty。位于 `web/src/components/common/EmptyState.vue`。

### Loading skeleton

```vue
<LoadingSkeleton :rows="6" variant="row" />
```

替代 NSpin 转圈。位于 `web/src/components/common/LoadingSkeleton.vue`。Variant: `row` / `card` / `kpi`。

## Page anatomy

### List page

```
PageHeader  ──  title 22px 600 + subtitle 13px secondary + 右侧操作按钮
  │
Filter bar  ──  status segmented + filter selects + search input
  │
Selection bar (v-show selectedKeys.length > 0)  ──  bulk actions
  │
Main area
  ├─ List view: <div class="sre-row-card" v-for>...</div>
  └─ Card view: grid auto-fill minmax(280-320px, 1fr) gap 16px
  │
Pagination  ──  NPagination size="small" 靠右
```

### Detail page

```
PageHeader  ──  ← 返回 + title + subtitle + 右上操作下拉
  │
Action bar  ──  横排主操作 + 次操作
  │
Tabs        ──  Overview / Sub-content / Timeline / …
  │
┌── main content ────┬── side panel (280px) ──┐
│                    │  KEY INFO              │
│  Tab content       │  META                  │
│                    │  RELATED               │
└────────────────────┴────────────────────────┘
```

## Anti-patterns（拒绝列表）

- ❌ Inter / Roboto / Arial / system 默认字体
- ❌ 紫色渐变 / 紫渐变白底（dribble 套路）
- ❌ NTag size="large" 大色块
- ❌ NDataTable 表格（除审计 / 调试场景）
- ❌ 卡通 illustration / 立体阴影狂魔
- ❌ AnimatedNumber / GlowCard 类装饰组件
- ❌ backdrop-filter blur 大量使用（GPU 重灾区，仅 modal/palette 用 `.surface-glass-modal`）
- ❌ 圆角 > 16px 软卡片
- ❌ 写死 `rgba(255,255,255,...)` 边框 / 文字（浅色模式破功）
- ❌ ECharts 直接传 `#fff` / `rgba(255,255,255,...)` chart text（必须 theme-aware）

## Skill 应用

设计过程中加载这三个 skill：

- `frontend-design` (anthropics) — distinctive choices, avoid AI slop
- `vue` (antfu) — Vue 3.5 idioms (script setup ts, shallowRef)
- `web-design-guidelines` (vercel-labs) — interface review rules

## Migration notes

从 v1.x 升级到 v2.x：

- 主色保持绿色 `#18a058` 不变
- Geist 字体新引入（替代 Inter / system 默认）
- 共享类如 `sre-row-card` / `sre-dot` / `sre-label-eyebrow` / `sre-meta-divider` 全局可用
- 老页面仍可用 `SeverityTag` / `StatusTag` 组件，新页面优先 `sre-dot`
- 详见 CHANGELOG v2.1.0 → v2.5.0

## File index

设计相关文件：

- `web/src/styles/global.css` — 所有 tokens + utilities + light theme overrides
- `web/index.html` — Geist + JetBrains Mono Google Fonts 引入
- `web/src/components/common/EmptyState.vue` — 共享空状态
- `web/src/components/common/LoadingSkeleton.vue` — 共享骨架屏
- `web/src/components/common/PageHeader.vue` — 标准页面 header
- `web/src/components/common/SeverityTag.vue` (legacy) — 仅老页面用，新页面用 sre-dot
- `web/src/components/common/StatusTag.vue` (legacy) — 同上
- `web/src/App.vue` — `applyBodyClass()` 切换 `body.light-theme`

## Decision log（重要选择）

| 决定 | 时间 | 原因 |
|------|------|------|
| Geist 字体 | 2026-05-08 | 替代 Inter，更具技术感 + distinctive |
| 4px 左色条 | v2.1.0 | 替代背景填色，减少视觉噪音 |
| sre-dot 圆点 | v2.1.0 | 替代 NTag，更紧凑 |
| 自定义 row card | v2.1.0 | 替代 NDataTable，节省密度 + 自由度 |
| md-editor-v3 | v2.0.2 | PostMortem Markdown 编辑器 |
| Theme-aware ECharts | 2026-05-08 | 浅色模式下白字白底问题，chart 需读 `body.light-theme` |
| `--sre-border-strong` for hover | 2026-05-08 | 替代写死 `rgba(255,255,255,0.12)`，浅色模式可见 |
