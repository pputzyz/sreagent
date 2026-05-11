# UI Soft Warm Redesign Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Refactor SREAgent's v3.0 "Modern Dark" design system into a warm, fresh, approachable "Soft Warm SaaS" style — light-first with teal primary + amber accent, soft shadows, Plus Jakarta Sans font.

**Architecture:** Update CSS variables in global.css as the single source of truth, then cascade through App.vue Naive UI overrides, MainLayout scoped styles, and 13 page/component files with hardcoded colors. All colors flow from CSS variables; hardcoded hex/rgba values are eliminated.

**Tech Stack:** Vue 3, Naive UI, CSS custom properties, Plus Jakarta Sans

---

## Color Palette Reference

| Token | Old (v3.0) | New (v3.1) | Usage |
|-------|-----------|-----------|-------|
| Brand primary | `#22c55e` green-500 | `#0d9488` teal-600 | Buttons, links, active states |
| Brand accent | `#6366f1` indigo-500 | `#f59e0b` amber-500 | Highlights, badges, secondary CTA |
| BG base | `#09090b` pure black | `#fafaf9` warm white | Page background |
| BG card | `#141416` dark solid | `#ffffff` white | Card surfaces |
| Text primary | `#ededef` light | `#1c1917` stone-900 | Headings, body |
| Text secondary | `#a0a0ab` | `#78716c` stone-500 | Descriptions |
| Text tertiary | `#63636e` | `#a8a29e` stone-400 | Muted, labels |
| Border | `rgba(255,255,255,0.06)` | `rgba(0,0,0,0.06)` | Card borders |

---

### Task 1: Update global.css Design Tokens

**Files:**
- Modify: `web/src/styles/global.css`

- [ ] **Step 1: Update `:root` brand variables (lines 8-28)**

Replace the green/indigo brand system with teal/amber:

```css
:root {
  /* --- Brand (teal-600 + amber-500) --- */
  --sre-brand-50:   #f0fdfa;
  --sre-brand-100:  #ccfbf1;
  --sre-brand-200:  #99f6e4;
  --sre-brand-300:  #5eead4;
  --sre-brand-400:  #2dd4bf;
  --sre-brand-500:  #14b8a6;
  --sre-brand-600:  #0d9488;   /* primary — teal-600 */
  --sre-brand-700:  #0f766e;
  --sre-brand-accent: #f59e0b;  /* amber-500 */

  --sre-primary:        var(--sre-brand-600);
  --sre-primary-hover:  var(--sre-brand-500);
  --sre-primary-soft:   rgba(13, 148, 136, 0.10);
  --sre-primary-ring:   rgba(13, 148, 136, 0.30);

  /* Accent (amber) — used for highlights, badges, secondary CTAs */
  --sre-accent:         #f59e0b;
  --sre-accent-soft:    rgba(245, 158, 11, 0.10);
  --sre-accent-ring:    rgba(245, 158, 11, 0.30);

  /* Gradients */
  --sre-gradient-brand: linear-gradient(135deg, #0d9488 0%, #14b8a6 40%, #f59e0b 100%);
  --sre-gradient-brand-soft: linear-gradient(135deg, rgba(13,148,136,0.08), rgba(245,158,11,0.08));
  --sre-gradient-brand-subtle: linear-gradient(135deg, rgba(13,148,136,0.04), rgba(245,158,11,0.04));
  --sre-gradient-heat:  linear-gradient(135deg, #ef4444 0%, #f59e0b 100%);
```

- [ ] **Step 2: Update semantic colors (lines 37-45)**

```css
  /* --- Semantic --- */
  --sre-critical:       #ef4444;
  --sre-critical-soft:  rgba(239, 68, 68, 0.08);
  --sre-danger:         #ef4444;
  --sre-warning:        #f59e0b;
  --sre-warning-soft:   rgba(245, 158, 11, 0.08);
  --sre-info:           #3b82f6;
  --sre-info-soft:      rgba(59, 130, 246, 0.08);
  --sre-success:        #0d9488;
  --sre-success-soft:   rgba(13, 148, 136, 0.08);
```

- [ ] **Step 3: Update surfaces to light warm (lines 60-76)**

```css
  /* --- Surfaces (warm white, Soft Warm) --- */
  --sre-bg-base:        #fafaf9;
  --sre-bg-page:        #fafaf9;
  --sre-bg-card:        #ffffff;
  --sre-bg-elevated:    #ffffff;
  --sre-bg-sunken:      #f5f5f4;
  --sre-bg-hover:       rgba(0, 0, 0, 0.03);
  --sre-bg-active:      rgba(0, 0, 0, 0.06);
  --sre-bg-subtle:      rgba(0, 0, 0, 0.02);
  --sre-overlay-subtle: rgba(0, 0, 0, 0.02);

  /* Glass surfaces */
  --sre-glass-bg:         #ffffff;
  --sre-glass-border:     rgba(0, 0, 0, 0.06);
  --sre-glass-blur:       0px;
  --sre-glass-saturate:   100%;
```

- [ ] **Step 4: Update borders, text, shadows (lines 78-146)**

```css
  /* --- Borders --- */
  --sre-border:         rgba(0, 0, 0, 0.06);
  --sre-border-strong:  rgba(0, 0, 0, 0.10);
  --sre-border-focus:   var(--sre-primary);

  /* --- Text --- */
  --sre-text-primary:   #1c1917;
  --sre-text-secondary: #78716c;
  --sre-text-tertiary:  #a8a29e;
  --sre-text-muted:     rgba(0, 0, 0, 0.15);
  --sre-text-inverse:   #ffffff;
```

```css
  /* --- Shadows (soft, warm) --- */
  --sre-shadow-xs:  0 1px 2px rgba(0, 0, 0, 0.04);
  --sre-shadow-sm:  0 2px 6px rgba(0, 0, 0, 0.06);
  --sre-shadow-md:  0 8px 24px -6px rgba(0, 0, 0, 0.08);
  --sre-shadow-lg:  0 18px 40px -12px rgba(0, 0, 0, 0.10);
  --sre-shadow-xl:  0 32px 64px -16px rgba(0, 0, 0, 0.12);
  --sre-shadow-glow-brand: 0 0 0 3px var(--sre-primary-ring);
  --sre-shadow-card-hover: 0 0 0 1px rgba(13,148,136,0.10);
  --sre-shadow-ring: 0 0 0 3px var(--sre-primary-ring);
```

- [ ] **Step 5: Update font stack (lines 112-115)**

```css
  /* --- Typography --- */
  --sre-font-display: "Plus Jakarta Sans", -apple-system, BlinkMacSystemFont, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", sans-serif;
  --sre-font-sans: "Plus Jakarta Sans", -apple-system, BlinkMacSystemFont, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Segoe UI",
                   Roboto, "Helvetica Neue", Arial, sans-serif;
  --sre-font-mono: "JetBrains Mono", "Cascadia Code", "SF Mono", "Consolas", "Menlo", ui-monospace, monospace;
```

- [ ] **Step 6: Update radius tokens (lines 102-109)**

```css
  /* --- Radius (soft, warm) --- */
  --sre-radius-xs:   4px;
  --sre-radius-sm:   6px;
  --sre-radius-md:   8px;
  --sre-radius-lg:   12px;
  --sre-radius-xl:   16px;
  --sre-radius-2xl:  24px;
  --sre-radius-pill: 9999px;
```

- [ ] **Step 7: Update severity chip colors for light bg (lines 328-333)**

```css
.sev-chip[data-sev="critical"] { background: var(--sre-critical-soft); color: #dc2626; }
.sev-chip[data-sev="warning"]  { background: var(--sre-warning-soft); color: #d97706; }
.sev-chip[data-sev="info"]     { background: var(--sre-info-soft); color: #2563eb; }
body.light-theme .sev-chip[data-sev="critical"] { color: #dc2626; }
body.light-theme .sev-chip[data-sev="warning"]  { color: #b45309; }
body.light-theme .sev-chip[data-sev="info"]     { color: #1d4ed8; }
```

- [ ] **Step 8: Update scrollbar colors for light bg (lines 261-262)**

```css
::-webkit-scrollbar-thumb { background: rgba(0, 0, 0, 0.10); border-radius: var(--sre-radius-pill); }
::-webkit-scrollbar-thumb:hover { background: rgba(0, 0, 0, 0.20); }
```

- [ ] **Step 9: Remove `body.light-theme` section (lines 175-215)**

Since the default is now light, the `body.light-theme` overrides for surfaces/text/borders are no longer needed. Instead, add a `body.dark-theme` section for dark mode support:

```css
/* ===== Dark theme (opt-in) ===== */
body.dark-theme {
  --sre-bg-base:        #0c0a09;
  --sre-bg-page:        #1c1917;
  --sre-bg-card:        #292524;
  --sre-bg-elevated:    #44403c;
  --sre-bg-sunken:      #0c0a09;
  --sre-bg-hover:       rgba(255, 255, 255, 0.05);
  --sre-bg-active:      rgba(255, 255, 255, 0.08);
  --sre-bg-subtle:      rgba(255, 255, 255, 0.02);
  --sre-overlay-subtle: rgba(255, 255, 255, 0.02);

  --sre-glass-bg:       #292524;
  --sre-glass-border:   rgba(255, 255, 255, 0.08);

  --sre-border:         rgba(255, 255, 255, 0.08);
  --sre-border-strong:  rgba(255, 255, 255, 0.14);

  --sre-text-primary:   #fafaf9;
  --sre-text-secondary: #d6d3d1;
  --sre-text-tertiary:  #a8a29e;
  --sre-text-muted:     rgba(255, 255, 255, 0.18);
  --sre-text-inverse:   #1c1917;

  --sre-primary-soft:   rgba(13, 148, 136, 0.15);
  --sre-critical-soft:  rgba(239, 68, 68, 0.14);
  --sre-warning-soft:   rgba(245, 158, 11, 0.14);
  --sre-info-soft:      rgba(59, 130, 246, 0.14);
  --sre-success-soft:   rgba(13, 148, 136, 0.14);
  --sre-accent-soft:    rgba(245, 158, 11, 0.14);

  --sre-shadow-xs:  0 1px 2px rgba(0, 0, 0, 0.25);
  --sre-shadow-sm:  0 2px 6px rgba(0, 0, 0, 0.20);
  --sre-shadow-md:  0 8px 24px -6px rgba(0, 0, 0, 0.35);
  --sre-shadow-lg:  0 18px 40px -12px rgba(0, 0, 0, 0.45);
  --sre-shadow-xl:  0 32px 64px -16px rgba(0, 0, 0, 0.50);
}

body.dark-theme ::-webkit-scrollbar-thumb { background: rgba(128, 128, 128, 0.18); }
body.dark-theme ::-webkit-scrollbar-thumb:hover { background: rgba(128, 128, 128, 0.35); }
```

- [ ] **Step 10: Commit**

```bash
git add web/src/styles/global.css
git commit -m "feat(ui): update global.css tokens to Soft Warm palette (teal/amber)"
```

---

### Task 2: Update App.vue Theme Overrides

**Files:**
- Modify: `web/src/App.vue:1-256`

- [ ] **Step 1: Update `common` object (lines 17-40)**

Replace green/indigo with teal/amber, change font to Plus Jakarta Sans:

```js
const common = {
  primaryColor:        '#0d9488',
  primaryColorHover:   '#14b8a6',
  primaryColorPressed: '#0f766e',
  primaryColorSuppl:   '#14b8a6',
  errorColor:          '#ef4444',
  errorColorHover:     '#f87171',
  errorColorPressed:   '#dc2626',
  warningColor:        '#f59e0b',
  warningColorHover:   '#fbbf24',
  warningColorPressed: '#d97706',
  infoColor:           '#3b82f6',
  infoColorHover:      '#60a5fa',
  infoColorPressed:    '#2563eb',
  successColor:        '#0d9488',
  successColorHover:   '#14b8a6',
  successColorPressed: '#0f766e',
  borderRadius:        '8px',
  borderRadiusSmall:   '6px',
  fontFamily:
    '"Plus Jakarta Sans", -apple-system, BlinkMacSystemFont, "PingFang SC", "Hiragino Sans GB", "Microsoft YaHei", "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif',
  fontFamilyMono:
    '"JetBrains Mono", "Cascadia Code", "SF Mono", "Consolas", "Menlo", ui-monospace, monospace',
}
```

- [ ] **Step 2: Update `darkOverrides` (lines 42-125)**

Replace all green references (`#22c55e`, `#4ade80`, `rgba(34,197,94,...)`) with teal equivalents (`#0d9488`, `#14b8a6`, `rgba(13,148,136,...)`). Update surface colors to warm stone tones:

```js
const darkOverrides: GlobalThemeOverrides = {
  common: {
    ...common,
    bodyColor:     '#1c1917',
    cardColor:     '#292524',
    modalColor:    '#292524',
    popoverColor:  '#292524',
    tableColor:    '#292524',
    tableColorHover: 'rgba(255,255,255,0.04)',
    borderColor:   'rgba(255,255,255,0.08)',
    dividerColor:  'rgba(255,255,255,0.08)',
    hoverColor:    'rgba(255,255,255,0.05)',
    textColorBase:      '#fafaf9',
    textColor1:         '#fafaf9',
    textColor2:         '#d6d3d1',
    textColor3:         '#a8a29e',
    textColorDisabled:  'rgba(255,255,255,0.22)',
    placeholderColor:   'rgba(255,255,255,0.25)',
  },
  Card: {
    color:         '#292524',
    colorEmbedded: '#1c1917',
    borderColor:   'rgba(255,255,255,0.08)',
    borderRadius:  '12px',
  },
  DataTable: {
    thColor:           '#292524',
    tdColor:           '#1c1917',
    tdColorHover:      'rgba(255,255,255,0.04)',
    borderColor:       'rgba(255,255,255,0.08)',
    borderRadius:      '12px',
    thFontWeight:      '600',
  },
  Layout: {
    color:       '#1c1917',
    siderColor:  '#1c1917',
    headerColor: '#1c1917',
  },
  Modal:  { color: '#292524', borderRadius: '12px' },
  Drawer: { color: '#292524' },
  Menu: {
    itemColorActive:        'rgba(13,148,136,0.12)',
    itemColorActiveHover:   'rgba(13,148,136,0.16)',
    itemTextColorActive:    '#14b8a6',
    itemIconColorActive:    '#14b8a6',
    itemIconColorActiveHover:'#14b8a6',
    // ... keep other properties
  },
  Switch: { railColorActive: '#0d9488' },
  Slider: { fillColor: '#0d9488' },
  Progress: { fillColor: '#0d9488' },
  Popover: { color: '#292524', borderRadius: '12px' },
  Tooltip: { color: '#44403c', borderRadius: '8px' },
}
```

- [ ] **Step 3: Update `lightOverrides` (lines 127-213)**

Update to warm white palette with teal accents:

```js
const lightOverrides: GlobalThemeOverrides = {
  common: {
    ...common,
    bodyColor:     '#fafaf9',
    cardColor:     '#ffffff',
    modalColor:    '#ffffff',
    popoverColor:  '#ffffff',
    tableColor:    '#ffffff',
    tableColorHover: 'rgba(0,0,0,0.03)',
    borderColor:   'rgba(0,0,0,0.06)',
    dividerColor:  'rgba(0,0,0,0.06)',
    hoverColor:    'rgba(0,0,0,0.03)',
    textColorBase: '#1c1917',
    textColor1:    '#1c1917',
    textColor2:    '#78716c',
    textColor3:    '#a8a29e',
  },
  Card: {
    color:         '#ffffff',
    colorEmbedded: '#f5f5f4',
    borderColor:   'rgba(0,0,0,0.06)',
    borderRadius:  '12px',
  },
  DataTable: {
    tdColor:      '#ffffff',
    thColor:      '#f5f5f4',
    tdColorHover: 'rgba(0,0,0,0.03)',
    borderColor:  'rgba(0,0,0,0.06)',
    borderRadius: '12px',
    thFontWeight: '600',
  },
  Layout: {
    color:       '#fafaf9',
    siderColor:  '#ffffff',
    headerColor: '#ffffff',
  },
  Menu: {
    itemColorActive:        'rgba(13,148,136,0.08)',
    itemColorActiveHover:   'rgba(13,148,136,0.12)',
    itemTextColorActive:    '#0f766e',
    itemIconColorActive:    '#0f766e',
    itemIconColorActiveHover:'#0f766e',
    // ... keep other properties
  },
  Switch: { railColorActive: '#0d9488' },
  Slider: { fillColor: '#0d9488' },
  Progress: { fillColor: '#0d9488' },
  // ... rest unchanged
}
```

- [ ] **Step 4: Update `applyBodyClass` (lines 219-225)**

Swap the class logic — default is now light, dark is opt-in:

```js
function applyBodyClass(dark: boolean) {
  if (dark) {
    document.body.classList.add('dark-theme')
    document.body.classList.remove('light-theme')
  } else {
    document.body.classList.remove('dark-theme')
    document.body.classList.add('light-theme')
  }
}
```

And update the initial `isDark` default to `false`:

```js
const isDark = ref(savedTheme ? savedTheme === 'dark' : false)
```

- [ ] **Step 5: Commit**

```bash
git add web/src/App.vue
git commit -m "feat(ui): update Naive UI theme overrides to Soft Warm palette"
```

---

### Task 3: Update MainLayout.vue Scoped Styles

**Files:**
- Modify: `web/src/layouts/MainLayout.vue` (scoped CSS, lines 537-750)

- [ ] **Step 1: Update `.topbar-tab.active` background color (line 582)**

Change `var(--sre-primary-soft)` — this already uses CSS variables, no change needed. Verify all CSS variable references still work with new tokens.

- [ ] **Step 2: Update `.topbar-clock` border and hover (lines 600-608)**

These use CSS variables — no code changes needed, they auto-adapt.

- [ ] **Step 3: Update `.tz-abbr` color (line 619)**

Already uses `var(--sre-primary)` — auto-adapts.

- [ ] **Step 4: Verify all scoped styles use CSS variables**

The MainLayout scoped styles already use CSS variables consistently. No hardcoded colors found. This task is a verification pass only.

- [ ] **Step 5: Commit (if any changes needed)**

```bash
git add web/src/layouts/MainLayout.vue
git commit -m "fix(ui): verify MainLayout styles adapt to new tokens"
```

---

### Task 4: Update Default Theme in App.vue + global.css Body Class

**Files:**
- Modify: `web/src/App.vue:12-13`
- Modify: `web/src/styles/global.css:175-215`

- [ ] **Step 1: Change default theme to light in App.vue**

Update line 13:
```js
const isDark = ref(savedTheme ? savedTheme === 'dark' : false)
```

- [ ] **Step 2: In global.css, change body class logic**

The current code uses `body.light-theme` for light overrides. Since light is now the default, the `:root` should contain the light values and `body.dark-theme` should contain dark overrides (already covered in Task 1 Step 9).

- [ ] **Step 3: Update `applyBodyClass` in App.vue**

```js
function applyBodyClass(dark: boolean) {
  if (dark) {
    document.body.classList.add('dark-theme')
    document.body.classList.remove('light-theme')
  } else {
    document.body.classList.remove('dark-theme')
    document.body.classList.add('light-theme')
  }
}
```

- [ ] **Step 4: Commit**

```bash
git add web/src/App.vue web/src/styles/global.css
git commit -m "feat(ui): default to light theme, add dark-theme class support"
```

---

### Task 5: Fix Hardcoded Colors in Notification Pages

**Files:**
- Modify: `web/src/pages/notification/Subscribe.vue:345-350`
- Modify: `web/src/pages/notification/Rules.vue:328-332`

- [ ] **Step 1: Update Subscribe.vue hardcoded colors**

Replace `#a5b4fc` → `var(--sre-accent)`, `#86efac` → `var(--sre-primary)`, `rgba(129,140,248,0.18)` → `var(--sre-accent-soft)`, `rgba(34,197,94,0.18)` → `var(--sre-primary-soft)`, `rgba(255,255,255,0.04)` → `var(--sre-bg-hover)`.

- [ ] **Step 2: Update Rules.vue hardcoded colors**

Same pattern — replace hardcoded purple/green with CSS variable references.

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/notification/Subscribe.vue web/src/pages/notification/Rules.vue
git commit -m "fix(ui): replace hardcoded colors with CSS variables in notification pages"
```

---

### Task 6: Fix Hardcoded Colors in Dashboard ECharts

**Files:**
- Modify: `web/src/pages/dashboard/Index.vue:116-138`

- [ ] **Step 1: Update ECharts theme function**

Replace the hardcoded dark/light palettes with values derived from CSS variables. Read CSS variables via `getComputedStyle`:

```ts
const root = getComputedStyle(document.documentElement)
const chartTheme = computed(() => {
  if (isDark.value) {
    return {
      bgColor: root.getPropertyValue('--sre-bg-card').trim(),
      textColor: root.getPropertyValue('--sre-text-secondary').trim(),
      // ...
    }
  }
  return {
    bgColor: '#ffffff',
    textColor: '#78716c',
    // ...
  }
})
```

Alternatively, keep the JS palette but update the hex values to match the new warm palette:
- Dark: `#292524` bg, `#d6d3d1` text, `rgba(250,250,249,0.90)` tooltip text
- Light: `#ffffff` bg, `#78716c` text

- [ ] **Step 2: Commit**

```bash
git add web/src/pages/dashboard/Index.vue
git commit -m "fix(ui): update ECharts theme to match Soft Warm palette"
```

---

### Task 7: Fix Hardcoded Colors in Explore & Login Pages

**Files:**
- Modify: `web/src/pages/explore/Index.vue:252,254,326,548,637,1038`
- Modify: `web/src/pages/Login.vue:427`

- [ ] **Step 1: Update explore/Index.vue**

Replace `#fff` with `var(--sre-text-inverse)`, `#666`/`#888` with `var(--sre-text-tertiary)`.

- [ ] **Step 2: Update Login.vue**

Replace `color: #fff` with `color: var(--sre-text-inverse)`.

- [ ] **Step 3: Commit**

```bash
git add web/src/pages/explore/Index.vue web/src/pages/Login.vue
git commit -m "fix(ui): replace hardcoded colors in explore and login pages"
```

---

### Task 8: Fix Hardcoded Colors in Components

**Files:**
- Modify: `web/src/components/common/CommandPalette.vue:158,178,230`
- Modify: `web/src/components/query/PromQLEditor.vue:111,117,118`
- Modify: `web/src/components/query/QueryResultChart.vue:215`
- Modify: `web/src/components/time/TimeRangePicker.vue:95`
- Modify: `web/src/components/noise/QuickSilenceModal.vue:222`
- Modify: `web/src/components/common/PageHeader.vue:57`

- [ ] **Step 1: CommandPalette.vue**

Replace `#f8f9fa` → `var(--sre-bg-card)`, `rgba(0,0,0,0.55)` → `rgba(0,0,0,0.40)`, scrollbar-color → use CSS variable.

- [ ] **Step 2: PromQLEditor.vue**

Replace `#e0e0e0` → `var(--sre-border)`, `#18a058` → `var(--sre-primary)`, green box-shadow → `var(--sre-primary-ring)`.

- [ ] **Step 3: QueryResultChart.vue, TimeRangePicker.vue**

Replace `#999`/`#666` → `var(--sre-text-tertiary)`.

- [ ] **Step 4: QuickSilenceModal.vue, PageHeader.vue**

Replace green rgba with `var(--sre-primary-soft)` / `var(--sre-primary-ring)`.

- [ ] **Step 5: Commit**

```bash
git add web/src/components/common/CommandPalette.vue web/src/components/query/PromQLEditor.vue web/src/components/query/QueryResultChart.vue web/src/components/time/TimeRangePicker.vue web/src/components/noise/QuickSilenceModal.vue web/src/components/common/PageHeader.vue
git commit -m "fix(ui): replace hardcoded colors with CSS variables in components"
```

---

### Task 9: Add Plus Jakarta Sans Font

**Files:**
- Modify: `web/index.html` (add Google Fonts link)

- [ ] **Step 1: Add font import to index.html `<head>`**

```html
<link rel="preconnect" href="https://fonts.googleapis.com">
<link rel="preconnect" href="https://fonts.gstatic.com" crossorigin>
<link href="https://fonts.googleapis.com/css2?family=Plus+Jakarta+Sans:wght@300;400;500;600;700&display=swap" rel="stylesheet">
```

- [ ] **Step 2: Commit**

```bash
git add web/index.html
git commit -m "feat(ui): add Plus Jakarta Sans font from Google Fonts"
```

---

### Task 10: Update Severity Chip & Dot Colors for Light BG

**Files:**
- Modify: `web/src/styles/global.css:328-333,656-660`

- [ ] **Step 1: Update `.sev-chip` text colors for light background**

The severity chip text colors need better contrast on light bg:
```css
.sev-chip[data-sev="critical"] { background: var(--sre-critical-soft); color: #dc2626; }
.sev-chip[data-sev="warning"]  { background: var(--sre-warning-soft); color: #d97706; }
.sev-chip[data-sev="info"]     { background: var(--sre-info-soft); color: #2563eb; }
```

- [ ] **Step 2: Commit**

```bash
git add web/src/styles/global.css
git commit -m "fix(ui): adjust severity chip contrast for light background"
```

---

### Task 11: Verify Build & Typecheck

- [ ] **Step 1: Run typecheck**

```bash
cd web && npx vue-tsc --noEmit
```

- [ ] **Step 2: Run build**

```bash
cd web && npx vite build
```

- [ ] **Step 3: Fix any errors found**

- [ ] **Step 4: Commit fixes if any**

---

### Task 12: Version Bump & Tag

- [ ] **Step 1: Bump version to v3.1.0**

Update in: `CLAUDE.md`, `MODULES.md`, `web/package.json`

- [ ] **Step 2: Add CHANGELOG entry**

```markdown
## [v3.1.0] — 2026-05-11

### Changed — UI 重构：Soft Warm 清新温暖设计风格

- 主色调从 green-500 `#22c55e` 改为 teal-600 `#0d9488`
- 强调色从 indigo-500 `#6366f1` 改为 amber-500 `#f59e0b`
- 默认主题从 Dark 切换为 Light（暖白 `#fafaf9`）
- Dark 模式改为 opt-in，使用 stone 色系暖灰
- 字体从 Inter 改为 Plus Jakarta Sans
- 圆角 lg 从 10px 增大到 12px
- 13 个页面/组件中的硬编码颜色迁移为 CSS 变量
- 阴影改为更柔和的低透明度版本
```

- [ ] **Step 3: Commit, tag, push**

```bash
git add CLAUDE.md MODULES.md web/package.json CHANGELOG.md
git commit -m "chore: bump version to v3.1.0"
git tag v3.1.0
git push origin main --tags
```
