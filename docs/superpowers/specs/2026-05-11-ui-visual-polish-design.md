# Visual Polish Layer — Implementation Spec

> **Date:** 2026-05-11
> **Target:** SREAgent v3.1 — Three-Column Layout
> **Style Direction:** 可爱俏皮灵动大方 (cute, playful, lively, generous)
> **Constraint:** Professional SaaS feel — polish, not gimmick

## Design System Context

The current system uses CSS custom properties (`--sre-*`) defined in `web/src/styles/global.css`:

- **Brand:** Teal-600 (`#0d9488`) primary + Amber-500 (`#f59e0b`) accent
- **Motion tokens:** `--sre-duration-fast` (150ms), `--sre-duration-base` (220ms), `--sre-duration-slow` (340ms)
- **Easing tokens:** `--sre-ease-out`, `--sre-ease-spring` (with overshoot), `--sre-ease-bounce`
- **Radii:** Soft, warm range from `--sre-radius-xs` (4px) to `--sre-radius-pill` (9999px)
- **Dark theme:** Via `body.dark-theme` class, overrides surfaces, borders, text, shadows
- **Reduced motion:** Already respected via `@media (prefers-reduced-motion: reduce)` block

### Existing Animations

The codebase already has: `sre-pulse-dot`, `sre-pulse-ring`, `sre-fade-in`, `sre-shimmer`, `sre-slide-up`, `sre-scale-in`, `sre-bounce-in`, `sre-count-in`, `sre-glow-pulse`, `sre-block-in`, `sre-stagger-up`, `sre-spin`, and stagger helpers. These are reusable and should be leveraged where possible.

### Layout Structure

```
AppShell.vue (flex column, 100vh)
  +-- topbar (48px, flex row)
  +-- app-body (flex row, flex:1)
       +-- AppRail.vue (48px wide, icon rail)
       |    +-- .rail-top (top icons)
       |    +-- .rail-spacer
       |    +-- .rail-bottom (settings icon)
       +-- AppSidebar.vue (220px / 64px collapsed, menu + user area)
       |    +-- .sidebar-nav (n-menu, scrollable)
       |    +-- .sidebar-spacer
       |    +-- .sidebar-bottom (user avatar, collapse btn, version)
       +-- main.main (flex:1, content)
            +-- .main-header (title + actions)
            +-- .main-content (router-view, scrollable)
```

---

## 1. Click Ripple Effect

**Goal:** Subtle teal ripple on every interactive element click, using CSS only (no JS event listeners).

### Approach

A global `.sre-ripple` mixin applied via a shared utility class. The ripple is created with a `::after` pseudo-element that scales outward from the click point using `transform: scale()`.

Since CSS cannot detect click coordinates natively, we use a **Vue directive** (`v-ripple`) that sets `--ripple-x` and `--ripple-y` CSS custom properties on the element via a `mousedown` listener, then triggers the CSS animation via a class toggle.

### New File: `web/src/directives/ripple.ts`

```ts
// Vue directive: v-ripple
// On mousedown, sets --ripple-x, --ripple-y (relative to element bounds)
// Adds a short-lived <span class="sre-ripple-wave"> child
// Auto-removes after animation ends (~600ms)
```

### CSS Addition to `global.css`

```css
/* === Ripple Effect === */
.sre-ripple {
  position: relative;
  overflow: hidden;
}

.sre-ripple-wave {
  position: absolute;
  border-radius: 50%;
  background: var(--sre-primary-soft);
  transform: scale(0);
  animation: sre-ripple-expand 600ms var(--sre-ease-out) forwards;
  pointer-events: none;
  width: 100px;
  height: 100px;
  margin-left: -50px;
  margin-top: -50px;
  left: var(--ripple-x, 50%);
  top: var(--ripple-y, 50%);
  opacity: 0.4;
}

@keyframes sre-ripple-expand {
  to {
    transform: scale(4);
    opacity: 0;
  }
}
```

### Application Points

| Element | Selector | Notes |
|---------|----------|-------|
| Rail icon buttons | `.rail-icon-btn` | Already has `overflow: hidden` potential |
| Sidebar menu items | Naive UI `.n-menu-item` | Apply via global selector override |
| Topbar buttons | `.topbar-btn` | Already styled |
| All `<button>` elements | Global | Broad application |
| Naive UI `NButton` | `.n-button` | Global override |

### Theme Compatibility

- **Light:** `var(--sre-primary-soft)` = `rgba(13, 148, 136, 0.10)` — subtle teal wash
- **Dark:** `var(--sre-primary-soft)` = `rgba(13, 148, 136, 0.15)` — slightly stronger for visibility
- No additional dark-mode overrides needed; the CSS variable handles it.

### Reduced Motion

The existing `@media (prefers-reduced-motion: reduce)` block already zeroes all animation durations. The ripple span will appear and vanish instantly — effectively invisible. No extra handling needed.

---

## 2. Menu Hover Animations

**Goal:** Sidebar menu items feel alive — subtle entrance, animated active indicator, gentle hover feedback.

### 2a. Hover: Gentle Scale + Background

```css
/* Applied globally to Naive UI menu items inside .app-sidebar */
.app-sidebar .n-menu-item {
  transition:
    transform var(--sre-duration-fast) var(--sre-ease-out),
    background var(--sre-duration-base) var(--sre-ease-out);
  border-radius: var(--sre-radius-md);
}

.app-sidebar .n-menu-item:hover {
  transform: translateX(2px);
  background: var(--sre-bg-hover);
}

.app-sidebar .n-menu-item:active {
  transform: translateX(2px) scale(0.98);
  transition-duration: 60ms;
}
```

**Rationale:** `translateX(2px)` gives a "nudge right" on hover — a classic sidebar micro-interaction. The `scale(0.98)` on active provides tactile click feedback.

### 2b. Active Indicator: Animated Width Expansion

Naive UI's `NMenu` uses `::before` or a border for the active indicator. We override it:

```css
.app-sidebar .n-menu-item-content--selected::before {
  content: "";
  position: absolute;
  left: 0;
  top: 6px;
  bottom: 6px;
  width: 3px;
  background: var(--sre-primary);
  border-radius: 0 var(--sre-radius-pill) var(--sre-radius-pill) 0;
  transform-origin: center;
  transition:
    height var(--sre-duration-base) var(--sre-ease-spring),
    opacity var(--sre-duration-fast) var(--sre-ease-out);
  animation: sre-active-bar-in 280ms var(--sre-ease-spring) both;
}

@keyframes sre-active-bar-in {
  from {
    opacity: 0;
    transform: scaleY(0);
  }
  to {
    opacity: 1;
    transform: scaleY(1);
  }
}
```

### 2c. Group Label Entrance

```css
.app-sidebar .n-menu-item-group-title {
  font-size: var(--sre-fs-xs);
  font-weight: var(--sre-fw-semibold);
  letter-spacing: 0.06em;
  text-transform: uppercase;
  color: var(--sre-text-tertiary);
  padding: var(--sre-space-3) var(--sre-space-3) var(--sre-space-1);
  animation: sre-fade-in var(--sre-duration-slow) var(--sre-ease-out) both;
}
```

### Reduced Motion

All transforms and animations will be suppressed by the existing global `prefers-reduced-motion` block.

---

## 3. Rail Icon Animations

**Goal:** App rail icons feel bouncy and responsive.

### 3a. Click Bounce

```css
.rail-icon-btn:active {
  transform: scale(0.85);
  transition: transform 80ms var(--sre-ease-bounce);
}

/* Restore with spring overshoot */
.rail-icon-btn:not(:active) {
  transition:
    background var(--sre-duration-base) var(--sre-ease-out),
    color var(--sre-duration-base) var(--sre-ease-out),
    transform var(--sre-duration-slow) var(--sre-ease-spring);
}
```

The `scale(0.85)` on press + spring overshoot on release creates a satisfying "pop" without any JavaScript.

### 3b. Active State: Gentle Pulse

```css
.rail-icon-btn.active {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  animation: sre-rail-active-pulse 3s ease-in-out infinite;
}

@keyframes sre-rail-active-pulse {
  0%, 100% {
    box-shadow: 0 0 0 0 var(--sre-primary-soft);
  }
  50% {
    box-shadow: 0 0 0 4px var(--sre-primary-soft);
  }
}
```

This is a very subtle glow pulse — not distracting, but enough to indicate "this is the active app."

### 3c. Tooltip Entrance Enhancement

Naive UI's `NTooltip` already has fade. We add a slight upward translate via global CSS override:

```css
.n-tooltip.n-tooltip--show-transition-enter-from {
  opacity: 0;
  transform: translateX(-4px);
}

.n-tooltip.n-tooltip--show-transition-enter-to {
  opacity: 1;
  transform: translateX(0);
}
```

**Note:** Naive UI's tooltip transition class names may vary by version. Verify against the installed version. If global override is unreliable, wrap the tooltip content in a div with a custom transition.

### Reduced Motion

The active pulse and bounce are animation-based and will be suppressed by `prefers-reduced-motion`.

---

## 4. Page Transitions

**Goal:** Route changes feel smooth — fade + slight upward slide on the content area.

### Implementation in `AppShell.vue`

Replace the `<router-view />` with Vue's `<Transition>`:

```vue
<div class="main-content">
  <router-view v-slot="{ Component, route }">
    <Transition name="sre-page" mode="out-in">
      <component :is="Component" :key="route.path" />
    </Transition>
  </router-view>
</div>
```

### CSS (add to `global.css` or scoped in `AppShell.vue`)

```css
/* Page transition — fade + subtle upward slide */
.sre-page-enter-active {
  transition:
    opacity var(--sre-duration-slow) var(--sre-ease-out),
    transform var(--sre-duration-slow) var(--sre-ease-out);
}

.sre-page-leave-active {
  transition:
    opacity var(--sre-duration-fast) var(--sre-ease-out),
    transform var(--sre-duration-fast) var(--sre-ease-out);
}

.sre-page-enter-from {
  opacity: 0;
  transform: translateY(8px);
}

.sre-page-leave-to {
  opacity: 0;
  transform: translateY(-4px);
}
```

**Design decisions:**
- Enter is slower (340ms) with upward slide — draws eye to new content
- Leave is faster (150ms) with slight upward fade — feels like "departing"
- `mode="out-in"` prevents layout jump from two elements coexisting

### Reduced Motion

The existing `prefers-reduced-motion` block handles this — transitions become instant.

---

## 5. Loading Skeleton

**Goal:** Content area shows a pulsing skeleton during route transitions or async data loads.

### Reusable Component: `web/src/components/common/SkeletonPulse.vue`

```vue
<template>
  <div class="sre-skeleton" :style="{ width, height, borderRadius }" />
</template>

<script setup lang="ts">
defineProps<{
  width?: string
  height?: string
  borderRadius?: string
}>()
</script>

<style scoped>
.sre-skeleton {
  background: var(--sre-bg-sunken);
  border-radius: var(--sre-radius-md);
  position: relative;
  overflow: hidden;
}

.sre-skeleton::after {
  content: "";
  position: absolute;
  inset: 0;
  background: linear-gradient(
    90deg,
    transparent 0%,
    var(--sre-bg-elevated) 40%,
    transparent 80%
  );
  background-size: 240px 100%;
  animation: sre-skeleton-shimmer 1.6s ease-in-out infinite;
}

@keyframes sre-skeleton-shimmer {
  from { transform: translateX(-100%); }
  to   { transform: translateX(100%); }
}
</style>
```

**Note:** This is distinct from the existing `.shimmer` class (which uses `background-position`). The skeleton shimmer uses `transform: translateX` for smoother GPU-accelerated animation.

### Skeleton Presets

A `SkeletonLayout.vue` component that composes multiple `SkeletonPulse` elements into common page layouts:

```vue
<!-- SkeletonLayout.vue — Props: variant ('list' | 'grid' | 'form' | 'dashboard') -->
<!-- Renders skeleton shapes matching the layout of each page type -->
```

| Variant | Shapes |
|---------|--------|
| `list` | Title bar + N rows of varying width |
| `grid` | 3-column grid of card skeletons |
| `form` | Label + input pairs in 2-col grid |
| `dashboard` | KPI row + chart area + table |

### Integration with Route Transitions

The `<Transition>` in section 4 can use JavaScript hooks to show a skeleton during the `leave` → `enter` gap:

```vue
<router-view v-slot="{ Component, route }">
  <Transition
    name="sre-page"
    mode="out-in"
    @leave="showSkeleton = true"
    @after-enter="showSkeleton = false"
  >
    <SkeletonLayout v-if="showSkeleton" variant="list" />
    <component v-else :is="Component" :key="route.path" />
  </Transition>
</router-view>
```

**Alternative (simpler):** Skip the skeleton during route transitions (they're fast with lazy-loaded components) and only use `SkeletonPulse` within individual pages during API fetches. This is the recommended approach — less coupling.

---

## 6. Scrollbar Styling

**Goal:** Custom scrollbar that matches the warm theme, thin and rounded, with hover-reveal behavior.

### Current State

The existing `global.css` already has scrollbar styling (lines 255-261):
- Width: 6px
- Transparent track
- `rgba(0,0,0,0.12)` thumb with `--sre-radius-pill` border-radius
- Dark theme override at line 215-216

### Enhancement

Replace the current block with an enhanced version that adds **hover-reveal** behavior:

```css
/* === Scrollbar — thin, warm, hover-reveal === */
::-webkit-scrollbar {
  width: 5px;
  height: 5px;
}

::-webkit-scrollbar-track {
  background: transparent;
}

::-webkit-scrollbar-thumb {
  background: transparent;
  border-radius: var(--sre-radius-pill);
  transition: background var(--sre-duration-base) var(--sre-ease-out);
}

/* Reveal on hover of the scrollable container */
*:hover > ::-webkit-scrollbar-thumb,
::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.15);
}

::-webkit-scrollbar-thumb:hover {
  background: rgba(0, 0, 0, 0.28);
}

/* Dark theme */
body.dark-theme *:hover > ::-webkit-scrollbar-thumb {
  background: rgba(255, 255, 255, 0.12);
}

body.dark-theme ::-webkit-scrollbar-thumb:hover {
  background: rgba(255, 255, 255, 0.25);
}
```

**Design decisions:**
- **5px width** (down from 6px) — slightly thinner, more refined
- **Transparent by default** — scrollbar appears only when hovering the container
- **Smooth transition** on the thumb background — fade in/out rather than instant
- **Pill radius** — matches the soft, warm design language

### Scoped Scrollbars

For specific containers (sidebar nav, main content), add `scrollbar-gutter: stable` to prevent layout shift when scrollbar appears:

```css
.sidebar-nav {
  scrollbar-gutter: stable;
}

.main-content {
  scrollbar-gutter: stable;
}
```

---

## 7. Cute Pet/Mascot Elements

**Goal:** A small, subtle mascot that adds personality without undermining professionalism.

### 7a. Mascot Design: Fox

A small fox mascot (32x32px) — chosen over a cat because:
- Foxes are associated with cleverness (fits SRE problem-solving)
- The tail shape works well at small sizes
- Teal/amber color palette maps naturally to fox fur tones

### SVG Asset: `web/src/assets/mascot-fox.svg`

Three states, embedded as a single SVG with CSS class toggles:

| State | Class | Description |
|-------|-------|-------------|
| **Idle** | `.mascot-idle` | Sitting fox, subtle ear twitch animation (loop) |
| **Wave** | `.mascot-wave` | Right paw raised, waving (plays once on first load) |
| **Sleep** | `.mascot-sleep` | Curled up, Zzz particles floating (after 5min inactivity) |

### SVG Structure (conceptual)

```svg
<svg viewBox="0 0 32 32" class="mascot-fox">
  <!-- Body: rounded triangle shapes in warm amber -->
  <!-- Ears: triangular, with inner teal accent -->
  <!-- Tail: curved, gradient from amber to white tip -->
  <!-- Eyes: dots that change with state -->
  <!-- Paw (wave state): raised, separate <g> with animation class -->
  <!-- Zzz (sleep state): small "z" characters, separate <g> -->
</svg>
```

### Component: `web/src/components/common/MascotFox.vue`

```vue
<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

type MascotState = 'wave' | 'idle' | 'sleep'

const state = ref<MascotState>('wave')
let inactivityTimer: ReturnType<typeof setTimeout>

function resetInactivity() {
  clearTimeout(inactivityTimer)
  if (state.value === 'sleep') state.value = 'idle'
  inactivityTimer = setTimeout(() => { state.value = 'sleep' }, 5 * 60 * 1000)
}

onMounted(() => {
  // Play wave once on first load, then switch to idle
  setTimeout(() => { state.value = 'idle' }, 2000)

  // Track user activity
  document.addEventListener('mousemove', resetInactivity)
  document.addEventListener('keydown', resetInactivity)
  resetInactivity()
})

onUnmounted(() => {
  clearTimeout(inactivityTimer)
  document.removeEventListener('mousemove', resetInactivity)
  document.removeEventListener('keydown', resetInactivity)
})
</script>

<template>
  <div class="mascot-container" :class="`mascot-${state}`">
    <MascotFoxSvg />
  </div>
</template>
```

### Placement: Bottom of App Rail

In `AppRail.vue`, add the mascot between `.rail-spacer` and `.rail-bottom`:

```vue
<div class="rail-spacer" />

<div class="rail-mascot">
  <MascotFox />
</div>

<div class="rail-bottom">
  <!-- existing platform icon -->
</div>
```

```css
.rail-mascot {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 8px 0;
}

.mascot-container {
  width: 32px;
  height: 32px;
  cursor: pointer;
  transition: transform var(--sre-duration-base) var(--sre-ease-spring);
}

.mascot-container:hover {
  transform: scale(1.1);
}

/* Idle: subtle ear twitch */
.mascot-idle .mascot-ear {
  animation: mascot-ear-twitch 4s ease-in-out infinite;
}

@keyframes mascot-ear-twitch {
  0%, 92%, 100% { transform: rotate(0deg); }
  95%           { transform: rotate(5deg); }
  97%           { transform: rotate(-3deg); }
}

/* Wave: paw movement */
.mascot-wave .mascot-paw {
  animation: mascot-wave-paw 0.6s ease-in-out 3;
  transform-origin: bottom center;
}

@keyframes mascot-wave-paw {
  0%, 100% { transform: rotate(0deg); }
  25%      { transform: rotate(15deg); }
  75%      { transform: rotate(-10deg); }
}

/* Sleep: gentle breathing + Zzz */
.mascot-sleep .mascot-body {
  animation: mascot-breathe 3s ease-in-out infinite;
}

@keyframes mascot-breathe {
  0%, 100% { transform: scaleY(1); }
  50%      { transform: scaleY(0.97); }
}

.mascot-sleep .mascot-zzz {
  animation: mascot-zzz-float 2s ease-in-out infinite;
  opacity: 0;
}

.mascot-sleep .mascot-zzz:nth-child(1) { animation-delay: 0s; }
.mascot-sleep .mascot-zzz:nth-child(2) { animation-delay: 0.7s; }
.mascot-sleep .mascot-zzz:nth-child(3) { animation-delay: 1.4s; }

@keyframes mascot-zzz-float {
  0%   { opacity: 0; transform: translate(0, 0) scale(0.8); }
  20%  { opacity: 0.6; }
  100% { opacity: 0; transform: translate(8px, -16px) scale(1); }
}
```

### Collapsed State Behavior

When the sidebar is collapsed, the rail remains 48px, so the mascot stays visible. No special handling needed.

### Interaction: Click to Toggle State

Clicking the mascot cycles: wave -> idle -> sleep -> idle. This is a playful easter egg, not a core feature.

### Professional Guardrails

- Mascot is **32x32px** — small, unobtrusive
- Animations are **very subtle** (ear twitch is 5deg rotation, breathing is 3% scale)
- No sound effects
- No tooltip text ("Don't pet the fox during incidents")
- Can be disabled via a `showMascot` localStorage flag for users who prefer no mascot

---

## 8. New CSS Custom Properties

Add these tokens to `global.css` to centralize the polish-layer values:

```css
:root {
  /* === Visual Polish Tokens === */
  --sre-ripple-color: var(--sre-primary-soft);
  --sre-ripple-duration: 600ms;

  --sre-hover-nudge-x: 2px;
  --sre-hover-scale-press: 0.98;
  --sre-hover-scale-bounce: 0.85;

  --sre-page-enter-duration: var(--sre-duration-slow);
  --sre-page-leave-duration: var(--sre-duration-fast);

  --sre-scrollbar-width: 5px;
  --sre-scrollbar-thumb: rgba(0, 0, 0, 0.15);
  --sre-scrollbar-thumb-hover: rgba(0, 0, 0, 0.28);

  --sre-mascot-size: 32px;
}
```

---

## 9. File Change Summary

| File | Change Type | Description |
|------|-------------|-------------|
| `web/src/styles/global.css` | **Modify** | Add ripple, scrollbar, page transition, mascot CSS; update scrollbar section |
| `web/src/directives/ripple.ts` | **New** | Vue directive for click ripple positioning |
| `web/src/layouts/AppShell.vue` | **Modify** | Wrap `<router-view>` in `<Transition>`, add page transition CSS |
| `web/src/layouts/AppRail.vue` | **Modify** | Add mascot slot, import MascotFox component, add click bounce CSS |
| `web/src/layouts/AppSidebar.vue` | **Modify** | Add menu hover animation CSS overrides |
| `web/src/components/common/MascotFox.vue` | **New** | Mascot component with state management |
| `web/src/components/common/SkeletonPulse.vue` | **New** | Reusable skeleton loading component |

---

## 10. Implementation Order

1. **Scrollbar styling** — smallest change, instant visual improvement, zero risk
2. **Click ripple directive + global CSS** — foundational, affects all interactive elements
3. **Menu hover animations** — scoped CSS overrides, no component changes
4. **Rail icon animations** — scoped CSS additions to `AppRail.vue`
5. **Page transitions** — requires `AppShell.vue` template change
6. **Skeleton pulse component** — standalone, usable anywhere
7. **Mascot fox** — most complex (SVG + component + state), do last

Each step is independently deployable and testable.

---

## 11. Risks and Mitigations

| Risk | Mitigation |
|------|------------|
| Naive UI menu internals change between versions | Use `:deep()` selectors sparingly; prefer global overrides in `global.css` |
| Ripple directive adds event listeners to many elements | Use event delegation on `document` instead of per-element listeners |
| Mascot SVG bloats bundle | Keep SVG inline and under 2KB; use simple geometric shapes |
| Page transition `mode="out-in"` adds latency on slow routes | Make leave transition very fast (150ms); skeleton fallback if needed |
| Scrollbar hover-reveal confuses users | Keep thumb width consistent; only opacity changes on hover, not width |

---

## 12. Accessibility Notes

- All animations respect `prefers-reduced-motion` (already handled in global.css)
- Ripple effect has no screen reader impact (visual only)
- Mascot has `aria-hidden="true"` — decorative only
- Page transitions use `mode="out-in"` to avoid confusing screen readers with two active regions
- Scrollbar remains keyboard-accessible regardless of hover-reveal state
