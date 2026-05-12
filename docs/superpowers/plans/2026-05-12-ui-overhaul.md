# SREAgent UI 完整重构计划

> **方向**: 温暖可爱 + 专业可靠。SRE 工具不必冷冰冰，让凌晨值班的工程师感到一丝温暖。

## 设计核心

- **色彩**: 珊瑚粉 `#FF6B6B` 主色 + 天蓝 `#4FACFE` + 薰衣草 `#A855F7` + 琥珀 `#F59E0B`
- **图标**: Lucide Icons 替换 Ionicons，导航图标 32px 圆角方形彩色背景
- **动画**: 丰富但不浮夸，所有操作有反馈，页面切换有过渡
- **宠物**: 可选 8 种类型，更大更显眼，同时出现在侧边栏和右下角
- **AI Chat**: 右下角浮动 "Ask AI" 按钮 + 顶部入口

---

## Task 1: 依赖安装 + 图标系统切换

**文件**: `web/package.json`

- [ ] 安装 `lucide-vue-next`
- [ ] 验证安装成功

---

## Task 2: global.css — 配色 + 动画 + 图标样式重构

**文件**: `web/src/styles/global.css`

- [ ] 新增多色品牌变量（珊瑚粉/天蓝/薰衣草/琥珀）
- [ ] 新增丰富动画 keyframes（bounce/wiggle/float/jelly/pulse-glow）
- [ ] 新增导航图标彩色背景样式
- [ ] 新增浮动按钮样式（右下角 Ask AI）
- [ ] 新增宠物动画样式
- [ ] 更新 surface 样式为更温暖的风格

---

## Task 3: AppRail — 图标系统 + 宠物/AI 入口重构

**文件**: `web/src/layouts/AppRail.vue`

- [ ] Ionicons → Lucide Icons
- [ ] 导航图标加彩色圆角背景
- [ ] 宠物入口放大 + 更显眼
- [ ] AI Chat 入口更明显
- [ ] hover 动画效果

---

## Task 4: AppShell — topbar + 右下角浮动按钮

**文件**: `web/src/layouts/AppShell.vue`

- [ ] topbar 右侧增加 AI 图标入口
- [ ] 右下角新增浮动 "Ask AI" 按钮
- [ ] 右下角新增宠物浮动入口
- [ ] 页面过渡动画增强

---

## Task 5: AppSidebar — 菜单图标 + hover 效果

**文件**: `web/src/layouts/AppSidebar.vue`

- [ ] 菜单项 hover 动画（彩色背景渐入）
- [ ] 选中项动画效果
- [ ] 折叠/展开动画

---

## Task 6: 宠物系统重构

**文件**:
- `web/src/stores/pet.ts` — 新增 petType 字段
- `web/src/components/common/MascotFox.vue` — 重构为多宠物组件
- `web/src/components/pet/PetPanel.vue` — 面板重构
- `web/src/components/pet/PetCorner.vue` — 角落重构
- `web/src/pages/pet/Index.vue` — 详情页重构

- [ ] pet store 新增 petType（fox/cat/owl/panda/tiger/bunny/dragon/penguin）
- [ ] MascotFox 重构为 PetAvatar，支持多类型 SVG
- [ ] PetPanel 图标放大、进度条平滑过渡
- [ ] PetCorner 更显眼、动画更丰富
- [ ] pet/Index 支持切换宠物类型

---

## Task 7: AI Chat 入口增强

**文件**:
- `web/src/components/ai/AIChatButton.vue` — 新增浮动按钮
- `web/src/components/ai/AIChatPanel.vue` — 面板优化

- [ ] 新增浮动 "Ask AI" 按钮组件
- [ ] 按钮带呼吸灯效果
- [ ] 面板空状态更友好

---

## Task 8: Login.vue — 登录页视觉升级

**文件**: `web/src/pages/Login.vue`

- [ ] 彩色渐变背景动画
- [ ] 表单弹跳入场
- [ ] 品牌区增强

---

## Task 9: CommandPalette + i18n

**文件**:
- `web/src/components/common/CommandPalette.vue`
- `web/src/i18n/zh-CN.ts`
- `web/src/i18n/en.ts`

- [ ] CommandPalette 图标更新
- [ ] 新增 i18n keys（宠物类型、AI 入口等）

---

## Task 10: 最终验证 + 打磨

- [ ] vue-tsc --noEmit 通过
- [ ] vite build 通过
- [ ] 检查所有动画在 reduced-motion 下的行为
- [ ] 检查暗色/亮色主题一致性
- [ ] commit + push + tag
