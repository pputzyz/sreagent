<script setup lang="ts">
/**
 * Skeleton loader for sre-row-card lists.
 * Use during initial fetch to avoid blank flash.
 *
 *   <LoadingSkeleton :rows="6" />
 *   <LoadingSkeleton variant="card-grid" :rows="4" />
 *   <LoadingSkeleton variant="kpi" :rows="4" />
 */
withDefaults(defineProps<{
  rows?: number
  variant?: 'row' | 'card-grid' | 'kpi'
}>(), { rows: 5, variant: 'row' })
</script>

<template>
  <!-- Row variant: matches sre-row-card layout -->
  <div v-if="variant === 'row'" class="skel-list" :class="{}">
    <div v-for="i in rows" :key="i" class="skel-row">
      <div class="skel-stripe"></div>
      <div class="skel-content">
        <div class="skel-line shimmer" style="width: 60%"></div>
        <div class="skel-line shimmer" style="width: 40%"></div>
        <div class="skel-line shimmer" style="width: 75%"></div>
      </div>
      <div class="skel-action shimmer"></div>
    </div>
  </div>

  <!-- Card grid variant: for channel/integration/datasource grids -->
  <div v-else-if="variant === 'card-grid'" class="skel-grid">
    <div v-for="i in rows" :key="i" class="skel-card">
      <div class="skel-stripe-top shimmer"></div>
      <div class="skel-line shimmer" style="width: 30%; height: 11px"></div>
      <div class="skel-line shimmer" style="width: 70%; height: 16px; margin-top: 8px"></div>
      <div class="skel-line shimmer" style="width: 90%"></div>
      <div class="skel-line shimmer" style="width: 60%"></div>
      <div class="skel-card-footer">
        <div class="skel-line shimmer" style="width: 40%"></div>
        <div class="skel-action shimmer" style="width: 60px"></div>
      </div>
    </div>
  </div>

  <!-- KPI variant: 4 stat cards -->
  <div v-else-if="variant === 'kpi'" class="skel-kpi-row">
    <div v-for="i in rows" :key="i" class="skel-kpi">
      <div class="skel-line shimmer" style="width: 50%; height: 28px; margin-bottom: 8px"></div>
      <div class="skel-line shimmer" style="width: 40%; height: 11px"></div>
      <div class="skel-stripe-bottom shimmer"></div>
    </div>
  </div>
</template>

<style scoped>
@keyframes shimmer-anim {
  0%   { background-position: -200% 0; }
  100% { background-position: 200% 0; }
}
.shimmer {
  background: linear-gradient(
    90deg,
    var(--sre-bg-elevated) 0%,
    var(--sre-overlay-subtle) 50%,
    var(--sre-bg-elevated) 100%
  );
  background-size: 200% 100%;
  animation: shimmer-anim 1.4s ease-in-out infinite;
  border-radius: 4px;
}

/* Row variant */
.skel-list { display: flex; flex-direction: column; gap: 6px; }
.skel-row {
  display: flex; gap: 14px;
  padding: 14px 18px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  position: relative;
  padding-left: calc(18px + 4px + 6px);
}
.skel-stripe {
  position: absolute; left: 0; top: 0; bottom: 0;
  width: 4px;
  background: var(--sre-bg-elevated);
  border-top-left-radius: var(--sre-radius-md);
  border-bottom-left-radius: var(--sre-radius-md);
}
.skel-content { flex: 1; display: flex; flex-direction: column; gap: 6px; }
.skel-line { height: 13px; border-radius: 4px; }
.skel-action { width: 24px; height: 24px; border-radius: 50%; align-self: center; }

/* Card grid variant */
.skel-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 16px;
}
.skel-card {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  padding: 20px;
  display: flex; flex-direction: column; gap: 8px;
  position: relative;
  overflow: hidden;
  min-height: 180px;
}
.skel-stripe-top {
  position: absolute; top: 0; left: 0; right: 0;
  height: 3px;
}
.skel-card-footer {
  display: flex; justify-content: space-between; align-items: center;
  margin-top: auto;
  padding-top: 10px;
  border-top: var(--sre-hairline);
}

/* KPI variant */
.skel-kpi-row {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(220px, 1fr));
  gap: 16px;
}
.skel-kpi {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
  padding: 20px;
  position: relative;
  overflow: hidden;
  min-height: 100px;
}
.skel-stripe-bottom {
  position: absolute; bottom: 0; left: 0; right: 0;
  height: 3px;
}
</style>
