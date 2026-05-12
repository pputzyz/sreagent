<script setup lang="ts">
import { ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, NButton, NInput, NGrid, NGi, NCard, NSpin } from 'naive-ui'
import { Activity, CheckCircle, AlertCircle, Clock, Bell, Globe, Shield, Zap, Layers, Server } from 'lucide-vue-next'

const { t } = useI18n()
const message = useMessage()

const email = ref('')
const submitting = ref(false)

function handleNotify() {
  if (!email.value || !email.value.includes('@')) {
    message.warning(t('statusPageModule.notifyPlaceholder'))
    return
  }
  submitting.value = true
  setTimeout(() => {
    submitting.value = false
    email.value = ''
    message.success(t('statusPageModule.notifySuccess'))
  }, 800)
}

const mockServices = [
  { name: 'statusPageModule.apiGateway', status: 'operational', icon: Server, uptime: '99.99%' },
  { name: 'statusPageModule.webApp', status: 'operational', icon: Globe, uptime: '99.98%' },
  { name: 'statusPageModule.database', status: 'operational', icon: Layers, uptime: '99.99%' },
  { name: 'statusPageModule.monitoring', status: 'operational', icon: Activity, uptime: '100%' },
  { name: 'statusPageModule.messageQueue', status: 'degraded', icon: Zap, uptime: '99.85%' },
  { name: 'statusPageModule.cdn', status: 'operational', icon: Shield, uptime: '99.97%' },
]

const features = [
  { title: 'statusPageModule.feature1Title', desc: 'statusPageModule.feature1Desc', icon: Activity },
  { title: 'statusPageModule.feature2Title', desc: 'statusPageModule.feature2Desc', icon: Globe },
  { title: 'statusPageModule.feature3Title', desc: 'statusPageModule.feature3Desc', icon: Bell },
  { title: 'statusPageModule.feature4Title', desc: 'statusPageModule.feature4Desc', icon: Shield },
]

function statusColor(status: string) {
  if (status === 'operational') return 'var(--sre-success)'
  if (status === 'degraded') return 'var(--sre-warning)'
  return 'var(--sre-critical)'
}

function statusBg(status: string) {
  if (status === 'operational') return 'var(--sre-success-soft)'
  if (status === 'degraded') return 'var(--sre-warning-soft)'
  return 'var(--sre-critical-soft)'
}

function statusLabel(status: string) {
  if (status === 'operational') return t('statusPageModule.serviceOperational')
  if (status === 'degraded') return t('statusPageModule.serviceDegraded')
  return t('statusPageModule.serviceOutage')
}
</script>

<template>
  <div class="page-container">
    <!-- Hero Section -->
    <div class="status-hero">
      <div class="status-hero-badge">
        <Clock :size="14" />
        <span>{{ t('statusPageModule.comingSoon') }}</span>
      </div>
      <h1 class="page-title" style="margin-bottom: 8px;">{{ t('statusPageModule.title') }}</h1>
      <p class="page-subtitle" style="max-width: 560px; margin: 0 auto 24px;">
        {{ t('statusPageModule.subtitle') }}
      </p>
      <p class="status-hero-desc">{{ t('statusPageModule.heroDesc') }}</p>
    </div>

    <!-- Preview: Service Status Cards -->
    <div class="status-preview-section">
      <div class="status-preview-header">
        <div class="status-preview-title-row">
          <span class="eyebrow">{{ t('statusPageModule.previewTitle') }}</span>
          <div class="status-all-ok">
            <CheckCircle :size="14" style="color: var(--sre-success);" />
            <span>{{ t('statusPageModule.allSystemsOperational') }}</span>
          </div>
        </div>
      </div>

      <div class="status-services-grid stagger-card">
        <div
          v-for="svc in mockServices"
          :key="svc.name"
          class="status-service-card surface-card"
        >
          <div class="svc-icon" :style="{ background: statusBg(svc.status), color: statusColor(svc.status) }">
            <component :is="svc.icon" :size="18" />
          </div>
          <div class="svc-info">
            <span class="svc-name">{{ t(svc.name) }}</span>
            <span class="svc-status" :style="{ color: statusColor(svc.status) }">
              <span class="svc-dot" :style="{ background: statusColor(svc.status) }" />
              {{ statusLabel(svc.status) }}
            </span>
          </div>
          <span class="svc-uptime number-display">{{ svc.uptime }}</span>
        </div>
      </div>

      <div class="status-meta-row">
        <span class="text-caption text-tertiary">
          <Clock :size="12" style="vertical-align: -2px; margin-right: 4px;" />
          {{ t('statusPageModule.lastUpdated') }}: 2 min ago
        </span>
      </div>
    </div>

    <!-- Notify CTA -->
    <div class="status-cta-card content-card" style="text-align: center;">
      <div class="cta-icon-wrap">
        <Bell :size="28" />
      </div>
      <h3 class="section-title" style="margin-bottom: 8px;">{{ t('statusPageModule.notifyMe') }}</h3>
      <p class="text-caption text-secondary" style="margin-bottom: 20px; max-width: 380px; margin-left: auto; margin-right: auto;">
        {{ t('statusPageModule.notifyHint') }}
      </p>
      <div class="cta-input-row">
        <NInput
          v-model:value="email"
          :placeholder="t('statusPageModule.notifyPlaceholder')"
          size="large"
          style="max-width: 320px;"
          @keyup.enter="handleNotify"
        />
        <NButton
          type="primary"
          size="large"
          :loading="submitting"
          @click="handleNotify"
        >
          {{ t('statusPageModule.notifyMe') }}
        </NButton>
      </div>
    </div>

    <!-- Feature Cards -->
    <div class="status-features">
      <NGrid :x-gap="16" :y-gap="16" :cols="4" responsive="screen" :item-responsive="true">
        <NGi v-for="feat in features" :key="feat.title" span="4 m:2 l:1">
          <div class="feature-card surface-card">
            <div class="feature-icon">
              <component :is="feat.icon" :size="20" />
            </div>
            <span class="feature-title">{{ t(feat.title) }}</span>
            <span class="feature-desc">{{ t(feat.desc) }}</span>
          </div>
        </NGi>
      </NGrid>
    </div>
  </div>
</template>

<style scoped>
.status-hero {
  text-align: center;
  padding: 48px 0 40px;
  animation: sre-fade-in 400ms var(--sre-ease-out) both;
}

.status-hero-badge {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 5px 14px;
  border-radius: var(--sre-radius-pill);
  background: var(--sre-accent-soft);
  color: var(--sre-accent);
  font-size: var(--sre-fs-sm);
  font-weight: var(--sre-fw-semibold);
  letter-spacing: 0.04em;
  text-transform: uppercase;
  margin-bottom: 16px;
}

.status-hero-desc {
  font-size: var(--sre-text-body-size);
  color: var(--sre-text-secondary);
  line-height: var(--sre-lh-normal);
  max-width: 520px;
  margin: 0 auto;
}

/* --- Preview Section --- */
.status-preview-section {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-xl);
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: var(--sre-shadow-sm);
  animation: sre-fade-in 400ms var(--sre-ease-out) 80ms both;
}

.status-preview-header {
  margin-bottom: 16px;
}

.status-preview-title-row {
  display: flex;
  align-items: center;
  justify-content: space-between;
}

.status-all-ok {
  display: flex;
  align-items: center;
  gap: 6px;
  font-size: var(--sre-fs-sm);
  font-weight: var(--sre-fw-medium);
  color: var(--sre-success);
}

.status-services-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}

.status-service-card {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 14px 16px;
  border-radius: var(--sre-radius-lg);
  border: 1px solid var(--sre-border);
  background: var(--sre-bg-card);
  transition: border-color var(--sre-duration-fast) var(--sre-ease-out),
              box-shadow var(--sre-duration-fast) var(--sre-ease-out);
}

.status-service-card:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-sm);
}

.svc-icon {
  width: 36px;
  height: 36px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.svc-info {
  display: flex;
  flex-direction: column;
  gap: 2px;
  flex: 1;
  min-width: 0;
}

.svc-name {
  font-size: var(--sre-fs-base);
  font-weight: var(--sre-fw-medium);
  color: var(--sre-text-primary);
}

.svc-status {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: var(--sre-fs-xs);
  font-weight: var(--sre-fw-medium);
}

.svc-dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}

.svc-uptime {
  font-size: var(--sre-fs-sm);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-secondary);
  flex-shrink: 0;
}

.status-meta-row {
  margin-top: 12px;
  text-align: right;
}

/* --- CTA Section --- */
.status-cta-card {
  margin-bottom: 24px;
  padding: 40px 24px !important;
  animation: sre-fade-in 400ms var(--sre-ease-out) 160ms both;
}

.cta-icon-wrap {
  width: 52px;
  height: 52px;
  border-radius: 14px;
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  margin: 0 auto 16px;
}

.cta-input-row {
  display: flex;
  align-items: center;
  gap: 10px;
  justify-content: center;
  max-width: 440px;
  margin: 0 auto;
}

/* --- Feature Cards --- */
.status-features {
  animation: sre-fade-in 400ms var(--sre-ease-out) 240ms both;
}

.feature-card {
  display: flex;
  flex-direction: column;
  gap: 10px;
  padding: 20px;
  border-radius: var(--sre-radius-lg);
  border: 1px solid var(--sre-border);
  background: var(--sre-bg-card);
  height: 100%;
  transition: border-color var(--sre-duration-fast) var(--sre-ease-out),
              box-shadow var(--sre-duration-fast) var(--sre-ease-out),
              transform var(--sre-duration-fast) var(--sre-ease-out);
}

.feature-card:hover {
  border-color: var(--sre-border-strong);
  box-shadow: var(--sre-shadow-lift);
  transform: translateY(-2px);
}

.feature-icon {
  width: 40px;
  height: 40px;
  border-radius: 12px;
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  display: flex;
  align-items: center;
  justify-content: center;
}

.feature-title {
  font-size: var(--sre-fs-base);
  font-weight: var(--sre-fw-semibold);
  color: var(--sre-text-primary);
}

.feature-desc {
  font-size: var(--sre-fs-sm);
  color: var(--sre-text-secondary);
  line-height: var(--sre-lh-normal);
}

/* Responsive */
@media (max-width: 768px) {
  .status-hero {
    padding: 32px 0 28px;
  }
  .status-services-grid {
    grid-template-columns: 1fr;
  }
  .cta-input-row {
    flex-direction: column;
  }
  .cta-input-row .n-input {
    max-width: 100% !important;
  }
}
</style>
