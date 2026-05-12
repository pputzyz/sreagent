# UI 重设计 + 宠物系统 + AI Chat 设计文档

> **日期**: 2026-05-12
> **版本**: v1.0
> **状态**: 已确认

## 概述

三个独立子系统的完整设计方案：
1. **UI 风格重设计** — 图标、动画、布局升级
2. **AI Chat** — 告警分析 + 通用对话 + 宠物对话
3. **宠物系统** — 养成 + AI 对话一步到位

## 子系统 A: UI 风格重设计

### Rail 栏
- 保持 56px 窄条结构
- 图标从 20px 放大到 24px
- 彩色指示器：oncall=红点、alert=蓝点、platform=紫点
- active 状态：背景高亮 + inset border ring
- hover：scale(1.05) 微放大

### 卡片系统
- 圆角：10px（从 12px 微调）
- 阴影：已升级到可见深度（--sre-shadow-lift）
- hover：translateY(-1px) 浮起 + 阴影增强
- 间距层次：compact (16px) / default (20px) / relaxed (24px)

### 动画系统
- 页面切换：fade + translateY(8px)
- 卡片入场：stagger slide-up（已有）
- 列表行入场：stagger 动画
- 图标 hover：scale(1.05)（已完成）
- 菜单项 hover：translateX(2px) 微右移（已完成）

### Mascot
- 保留现有狐狸 mascot（MascotFox.vue）
- 在空状态、登录页中使用
- 不添加新 mascot

### 配色
- per-app 背景色 5%（oncall=红、alert=蓝、platform=紫）
- 侧边栏 accent 跟随当前 app
- 保持 Restrained 策略（accent ≤10%）

## 子系统 B: AI Chat

### 交互形式
- 右下角浮动按钮（AI 图标）
- 点击展开侧边抽屉（右侧，400px 宽）
- 三种模式切换：告警分析 / 通用对话 / 宠物对话

### 三种模式

**告警分析模式**：
- 从告警详情页触发，传入告警上下文
- AI 分析根因、建议 SOP、推荐相关告警
- 显示告警标签、时间线、相关指标

**通用对话模式**：
- 自由问答，类似 ChatGPT
- 支持代码块、markdown 渲染
- 对话历史持久化

**宠物对话模式**：
- 和宠物聊天，宠物有个性化回复
- 宠物人设：活泼、好奇、偶尔犯傻的狐狸
- 对话影响宠物心情

### 后端 API
- `POST /api/v1/ai/chat` — 发送消息
- `GET /api/v1/ai/history` — 获取对话历史
- `DELETE /api/v1/ai/history` — 清空历史
- 复用现有 AI 配置（Platform > AI 配置页面）

### 前端组件
- `AIChatPanel.vue` — 侧边抽屉主体
- `AIChatButton.vue` — 浮动入口按钮
- `AIChatMessage.vue` — 单条消息组件
- `useAIChat.ts` — composable（消息管理、API 调用）

## 子系统 C: 宠物系统

### 宠物模型
- 物种：狐狸（和 mascot 一致）
- 状态：饥饿度 (0-100)、心情 (0-100)、等级 (1-N)、经验值
- 成长：使用系统、完成任务积累经验值
- 衰减：饥饿度随时间增加（每小时 +1）

### UI 组件

**角落常驻**：
- 侧边栏底部或主内容区右下角
- 显示宠物头像 + 迷你状态条
- 点击展开互动面板

**可展开面板**：
- 喂食（减少饥饿度）
- 玩耍（增加心情）
- 对话（跳转 AI Chat 宠物模式）
- 显示详细状态

**独立详情页** (`/pet`)：
- 宠物大图 + 完整状态
- 历史互动记录
- 成长里程碑
- 宠物设置（名字、外观）

### 后端
- `pet` 表：id, user_id, name, species, level, exp, hunger, mood, created_at, updated_at
- `pet_interaction` 表：id, pet_id, type, value, created_at
- API 端点：
  - `GET /api/v1/pet` — 获取宠物信息
  - `POST /api/v1/pet` — 领养宠物
  - `PUT /api/v1/pet` — 更新宠物信息
  - `POST /api/v1/pet/feed` — 喂食
  - `POST /api/v1/pet/play` — 玩耍
  - `GET /api/v1/pet/interactions` — 互动历史

### AI 人设
- 名字：用户可自定义
- 性格：活泼、好奇、偶尔犯傻
- 回复风格：简短、有趣、偶尔卖萌
- 和宠物状态联动：饥饿时抱怨、开心时卖萌

## 实现策略

全部并行，用多个 subagent 同时推进：
- Subagent 1: UI 风格重设计（前端）
- Subagent 2: AI Chat 前端
- Subagent 3: AI Chat 后端
- Subagent 4: 宠物系统前端
- Subagent 5: 宠物系统后端

每个 subagent 完成后：spec review → code quality review → 修复 → 合并

## 依赖关系

- 宠物系统依赖 AI Chat（宠物对话模式）
- AI Chat 不依赖宠物系统
- UI 重设计独立

建议实现顺序：
1. UI 重设计（无依赖）
2. AI Chat 后端 + 前端
3. 宠物系统后端 + 前端（依赖 AI Chat）

## 技术栈

- 前端：Vue 3 + Naive UI + Pinia + TypeScript
- 后端：Go + Gin + MySQL
- AI：复用现有 AI 配置（支持 OpenAI 兼容 API）
- 图标：@vicons/ionicons5
