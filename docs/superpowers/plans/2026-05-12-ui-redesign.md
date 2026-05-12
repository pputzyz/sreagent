# UI 风格重设计 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Upgrade the visual language — larger rail icons, refined animation system, per-app color differentiation, mascot integration, and card polish.

**Architecture:** CSS-first changes in `global.css` for tokens and animations, component-level changes in layout files for icon sizing and mascot placement. No new backend dependencies.

**Tech Stack:** Vue 3 + Naive UI + CSS Custom Properties + @vicons/ionicons5

---

## File Structure

| File | Change Type | Responsibility |
|------|-------------|----------------|
| `web/src/styles/global.css` | Modify | New animation keyframes, per-app background utilities, card radius tweak |
| `web/src/layouts/AppRail.vue` | Modify | Icon size 20→24px, refined active/hover states |
| `web/src/layouts/AppShell.vue` | Modify | Page transition animation upgrade |
| `web/src/layouts/AppSidebar.vue` | Modify | Per-app accent color injection via CSS variable |
| `web/src/pages/Login.vue` | Modify | Mascot integration in login page |
| `web/src/i18n/zh-CN.ts` | Modify | Empty state text additions |
| `web/src/i18n/en.ts` | Modify | Empty state text additions |

---

### Task 1: global.css — Animation System + Per-App Backgrounds

**Files:**
- Modify: `web/src/styles/global.css`

- [ ] **Step 1: Add animation keyframes**

Append after existing keyframes in global.css:

```css
/* --- Page transition --- */
@keyframes sre-page-enter {
  from { opacity: 0; transform: translateY(8px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* --- Card stagger --- */
@keyframes sre-card-enter {
  from { opacity: 0; transform: translateY(12px); }
  to   { opacity: 1; transform: translateY(0); }
}

/* --- List row stagger --- */
@keyframes sre-row-enter {
  from { opacity: 0; transform: translateX(-6px); }
  to   { opacity: 1; transform: translateX(0); }
}
```

- [ ] **Step 2: Add animation utility classes**

```css
/* Page-level transition */
.page-enter-active {
  animation: sre-page-enter 300ms var(--sre-ease-out) both;
}

/* Card stagger — apply to card containers */
.stagger-card > * {
  animation: sre-card-enter 400ms var(--sre-ease-out) both;
}
.stagger-card > *:nth-child(1) { animation-delay: 0ms; }
.stagger-card > *:nth-child(2) { animation-delay: 50ms; }
.stagger-card > *:nth-child(3) { animation-delay: 100ms; }
.stagger-card > *:nth-child(4) { animation-delay: 150ms; }
.stagger-card > *:nth-child(5) { animation-delay: 200ms; }
.stagger-card > *:nth-child(6) { animation-delay: 250ms; }

/* List row stagger */
.stagger-row > * {
  animation: sre-row-enter 300ms var(--sre-ease-out) both;
}
.stagger-row > *:nth-child(1)  { animation-delay: 0ms; }
.stagger-row > *:nth-child(2)  { animation-delay: 30ms; }
.stagger-row > *:nth-child(3)  { animation-delay: 60ms; }
.stagger-row > *:nth-child(4)  { animation-delay: 90ms; }
.stagger-row > *:nth-child(5)  { animation-delay: 120ms; }
.stagger-row > *:nth-child(6)  { animation-delay: 150ms; }
.stagger-row > *:nth-child(7)  { animation-delay: 180ms; }
.stagger-row > *:nth-child(8)  { animation-delay: 210ms; }
.stagger-row > *:nth-child(9)  { animation-delay: 240ms; }
.stagger-row > *:nth-child(10) { animation-delay: 270ms; }
```

- [ ] **Step 3: Add per-app background tint utilities**

```css
/* Per-app subtle background tints (5%) */
.bg-app-oncall   { background-color: color-mix(in srgb, var(--sre-brand-oncall) 5%, var(--sre-bg-page)); }
.bg-app-alert    { background-color: color-mix(in srgb, var(--sre-brand-alert) 5%, var(--sre-bg-page)); }
.bg-app-platform { background-color: color-mix(in srgb, var(--sre-brand-platform) 5%, var(--sre-bg-page)); }
```

- [ ] **Step 4: Tweak card border-radius from 12px to 10px**

Find the `.surface-card` rule and change `border-radius: var(--sre-radius-lg)` to `border-radius: 10px`. Also update `--sre-radius-lg` in `:root` from `12px` to `10px`.

- [ ] **Step 5: Commit**

```bash
cd web && git add src/styles/global.css && git commit -m "feat(ui): animation system + per-app backgrounds + card radius polish"
```

---

### Task 2: AppRail — Icon Size Upgrade + Refined States

**Files:**
- Modify: `web/src/layouts/AppRail.vue`

- [ ] **Step 1: Enlarge rail icons from 20px to 24px**

In the template, change all three `<n-icon :size="20" />` to `:size="24"`:
- Line with `CallOutline`: `:size="20"` → `:size="24"`
- Line with `NotificationsOutline`: `:size="20"` → `:size="24"`
- Line with `SettingsOutline`: `:size="20"` → `:size="24"`

- [ ] **Step 2: Refine active state with background highlight**

Update `.rail-icon-btn.active` CSS:

```css
.rail-icon-btn.active {
  background: var(--sre-bg-active);
  color: var(--sre-text-primary);
  box-shadow: inset 0 0 0 1.5px var(--sre-border-strong);
}
```

This is already present. Verify it's correct and no changes needed.

- [ ] **Step 3: Add hover scale micro-interaction**

Update `.rail-icon-btn:hover` to include `transform: scale(1.05)`:

```css
.rail-icon-btn:hover {
  background: color-mix(in srgb, var(--sre-bg-hover) 100%, transparent);
  color: var(--sre-text-secondary);
  transform: scale(1.05);
}
```

This is already present. Verify it's correct.

- [ ] **Step 4: Verify colored dot indicators are working**

Check that `.rail-dot` CSS has correct colors:
- `[data-app="oncall"] .rail-dot` → `var(--sre-brand-oncall)` (red)
- `[data-app="alert"] .rail-dot` → `var(--sre-brand-alert)` (blue)
- `[data-app="platform"] .rail-dot` → `var(--sre-brand-platform)` (purple)

Already present. Verify.

- [ ] **Step 5: Commit**

```bash
cd web && git add src/layouts/AppRail.vue && git commit -m "feat(ui): enlarge rail icons to 24px"
```

---

### Task 3: AppShell — Page Transition Animation

**Files:**
- Modify: `web/src/layouts/AppShell.vue`

- [ ] **Step 1: Add transition wrapper around router-view**

Find the `<router-view>` in the template. Wrap it with a Vue `<transition>`:

```html
<router-view v-slot="{ Component, route }">
  <transition name="page" mode="out-in">
    <component :is="Component" :key="route.path" />
  </transition>
</router-view>
```

Note: If `<router-view>` is already wrapped in a transition, update the transition name to `page` and mode to `out-in`.

- [ ] **Step 2: Add transition CSS**

Add scoped CSS in AppShell.vue:

```css
.page-enter-active {
  animation: sre-page-enter 300ms var(--sre-ease-out) both;
}

.page-leave-active {
  animation: sre-page-enter 200ms var(--sre-ease-out) reverse both;
}
```

Note: The `sre-page-enter` keyframe is defined in global.css (Task 1).

- [ ] **Step 3: Commit**

```bash
cd web && git add src/layouts/AppShell.vue && git commit -m "feat(ui): page transition animation (fade + translateY)"
```

---

### Task 4: AppSidebar — Per-App Accent Color

**Files:**
- Modify: `web/src/layouts/AppSidebar.vue`

- [ ] **Step 1: Add data-app attribute to sidebar root**

In the template, add `:data-app="activeApp"` to the sidebar root element. The `activeApp` prop should already be available from the parent (AppShell passes it).

If `activeApp` is not a prop, add it:

```typescript
defineProps<{
  // ... existing props
  activeApp?: string
}>()
```

- [ ] **Step 2: Add CSS variable injection per app**

Add scoped CSS:

```css
.app-sidebar[data-app="oncall"] {
  --sidebar-accent: var(--sre-brand-oncall);
  --sidebar-accent-soft: color-mix(in srgb, var(--sre-brand-oncall) 8%, transparent);
}
.app-sidebar[data-app="alert"] {
  --sidebar-accent: var(--sre-brand-alert);
  --sidebar-accent-soft: color-mix(in srgb, var(--sre-brand-alert) 8%, transparent);
}
.app-sidebar[data-app="platform"] {
  --sidebar-accent: var(--sre-brand-platform);
  --sidebar-accent-soft: color-mix(in srgb, var(--sre-brand-platform) 8%, transparent);
}
```

- [ ] **Step 3: Apply accent to selected menu item**

Update the selected menu item indicator to use `--sidebar-accent`:

```css
.app-sidebar .n-menu-item-content--selected::before {
  background: var(--sidebar-accent, var(--sre-primary));
}
```

- [ ] **Step 4: Commit**

```bash
cd web && git add src/layouts/AppSidebar.vue && git commit -m "feat(ui): per-app accent color in sidebar"
```

---

### Task 5: Login.vue — Mascot Integration

**Files:**
- Modify: `web/src/pages/Login.vue`

- [ ] **Step 1: Import MascotFox component**

Add import at the top of `<script setup>`:

```typescript
import MascotFox from '@/components/common/MascotFox.vue'
```

- [ ] **Step 2: Add mascot to login page**

Place the mascot near the brand/logo area, below the SREAgent title:

```html
<div class="login-brand">
  <!-- existing logo/title -->
  <MascotFox class="login-mascot" />
</div>
```

- [ ] **Step 3: Style the mascot position**

Add scoped CSS:

```css
.login-mascot {
  width: 48px;
  height: 48px;
  margin-top: 12px;
}
```

- [ ] **Step 4: Commit**

```bash
cd web && git add src/pages/Login.vue && git commit -m "feat(ui): add fox mascot to login page"
```

---

### Task 6: Empty States — Mascot + i18n

**Files:**
- Modify: `web/src/i18n/zh-CN.ts`
- Modify: `web/src/i18n/en.ts`

- [ ] **Step 1: Add empty state i18n keys**

In `zh-CN.ts`, add under a new `empty` section:

```typescript
empty: {
  noData: '暂无数据',
  noResults: '未找到匹配结果',
  noAlerts: '暂无告警，一切正常',
  noIncidents: '暂无故障',
  petDefault: '你的狐狸宠物在这里等你',
}
```

In `en.ts`:

```typescript
empty: {
  noData: 'No data yet',
  noResults: 'No matching results found',
  noAlerts: 'No alerts — all clear',
  noIncidents: 'No incidents',
  petDefault: 'Your fox pet is waiting for you',
}
```

- [ ] **Step 2: Commit**

```bash
cd web && git add src/i18n/zh-CN.ts src/i18n/en.ts && git commit -m "feat(ui): add empty state i18n keys"
```

---

## Verification

1. `cd web && node_modules/.bin/vue-tsc --noEmit` — passes
2. `cd web && npx vite build` — passes
3. Browser check:
   - Rail icons are 24px, colored indicators visible on active
   - Page transitions animate (fade + slide)
   - Cards have stagger entrance animation
   - Sidebar accent color changes when switching oncall/alert/platform
   - Login page shows fox mascot
   - Card border-radius is 10px (slightly tighter)
