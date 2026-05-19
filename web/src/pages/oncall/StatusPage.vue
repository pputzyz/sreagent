<script setup lang="ts">
import { ref, computed, onMounted, reactive, watch, type Component } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, NButton, NInput, NSpin, NModal, NForm, NFormItem, NSelect, NInputNumber, NPopconfirm } from 'naive-ui'
import { Activity, CheckCircle, AlertCircle, Clock, Bell, Globe, Shield, Zap, Layers, Server, Settings, Plus, Pencil, Trash2 } from 'lucide-vue-next'
import { statusServiceApi, type StatusServiceItem } from '@/api'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const { t } = useI18n()
const message = useMessage()

const email = ref('')
const submitting = ref(false)
const services = ref<StatusServiceItem[]>([])
const loading = ref(true)
const firstLoad = ref(true)

watch(loading, (isLoading) => {
  if (!isLoading) firstLoad.value = false
})

// --- Management modal ---
const showManage = ref(false)
const showForm = ref(false)
const editingId = ref<number | null>(null)
const saving = ref(false)

const form = reactive({
  name: '',
  status: 'operational' as string,
  description: '',
  url: '',
  icon: 'Server',
  sort_order: 0,
})

const iconMap: Record<string, Component> = { Server, Globe, Layers, Activity, Zap, Shield, AlertCircle, Clock }

const iconOptions = ['Server', 'Globe', 'Layers', 'Activity', 'Zap', 'Shield', 'AlertCircle', 'Clock'].map(name => ({
  label: name,
  value: name,
}))

const statusOptions = computed(() => [
  { label: t('statusPageModule.serviceOperational'), value: 'operational' },
  { label: t('statusPageModule.serviceDegraded'), value: 'degraded' },
  { label: t('statusPageModule.serviceOutage'), value: 'outage' },
  { label: t('statusPageModule.serviceMaintenance'), value: 'maintenance' },
])

async function loadServices() {
  try {
    const res = await statusServiceApi.list()
    services.value = res.data.data || []
  } catch {
    // fallback to empty
  }
}

onMounted(async () => {
  try {
    await loadServices()
  } finally {
    loading.value = false
  }
})

const allOperational = computed(() => services.value.length > 0 && services.value.every(s => s.status === 'operational'))

function handleNotify() {
  if (!email.value || !email.value.includes('@')) {
    message.warning(t('statusPageModule.invalidEmail'))
    return
  }
  submitting.value = true
  setTimeout(() => {
    submitting.value = false
    email.value = ''
    message.success(t('statusPageModule.notifySuccess'))
  }, 800)
}

function getIcon(iconName: string) {
  return iconMap[iconName] || Server
}

function statusColor(status: string) {
  if (status === 'operational') return 'var(--sre-success)'
  if (status === 'degraded') return 'var(--sre-warning)'
  if (status === 'maintenance') return 'var(--sre-info)'
  return 'var(--sre-critical)'
}

function statusBg(status: string) {
  if (status === 'operational') return 'var(--sre-success-soft)'
  if (status === 'degraded') return 'var(--sre-warning-soft)'
  if (status === 'maintenance') return 'var(--sre-info-soft)'
  return 'var(--sre-critical-soft)'
}

function statusLabel(status: string) {
  if (status === 'operational') return t('statusPageModule.serviceOperational')
  if (status === 'degraded') return t('statusPageModule.serviceDegraded')
  if (status === 'maintenance') return t('statusPageModule.serviceMaintenance')
  return t('statusPageModule.serviceOutage')
}

// --- CRUD handlers ---
function openManage() {
  showManage.value = true
}

function openCreate() {
  editingId.value = null
  form.name = ''
  form.status = 'operational'
  form.description = ''
  form.url = ''
  form.icon = 'Server'
  form.sort_order = 0
  showForm.value = true
}

function openEdit(svc: StatusServiceItem) {
  editingId.value = svc.id
  form.name = svc.name
  form.status = svc.status
  form.description = svc.description || ''
  form.url = svc.url || ''
  form.icon = svc.icon || 'Server'
  form.sort_order = svc.sort_order || 0
  showForm.value = true
}

async function handleSave() {
  if (!form.name.trim()) {
    message.warning(t('statusPageModule.serviceName') + ' ' + t('common.required'))
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name.trim(),
      status: form.status,
      description: form.description.trim() || undefined,
      url: form.url.trim() || undefined,
      icon: form.icon,
      sort_order: form.sort_order,
    }
    if (editingId.value) {
      await statusServiceApi.update(editingId.value, payload)
      message.success(t('common.updateSuccess'))
    } else {
      await statusServiceApi.create(payload)
      message.success(t('common.createSuccess'))
    }
    showForm.value = false
    await loadServices()
  } catch {
    message.error(t('common.saveFailed'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await statusServiceApi.delete(id)
    message.success(t('common.deleteSuccess'))
    await loadServices()
  } catch {
    message.error(t('common.deleteFailed'))
  }
}
</script>

<template>
  <div class="page-container">
    <!-- Hero Section -->
    <div class="status-hero">
      <h1 class="page-title" style="margin-bottom: 8px;">{{ t('statusPageModule.title') }}</h1>
      <p class="page-subtitle" style="max-width: 560px; margin: 0 auto;">
        {{ t('statusPageModule.subtitle') }}
      </p>
    </div>

    <!-- Service Status Cards -->
    <div class="status-section">
      <LoadingSkeleton v-if="firstLoad && loading" :rows="4" variant="card-grid" />
      <NSpin v-else :show="loading">
        <div class="status-header">
          <div class="status-header-row">
            <span class="eyebrow">{{ t('statusPageModule.currentStatus') }}</span>
            <div class="status-header-actions">
              <div v-if="allOperational" class="status-all-ok">
                <CheckCircle :size="14" style="color: var(--sre-success);" />
                <span>{{ t('statusPageModule.allSystemsOperational') }}</span>
              </div>
              <div v-else-if="services.length > 0" class="status-all-ok" style="color: var(--sre-warning);">
                <AlertCircle :size="14" />
                <span>{{ t('statusPageModule.partialOutage') }}</span>
              </div>
              <NButton size="small" quaternary @click="openManage">
                <template #icon><Settings :size="14" /></template>
                {{ t('statusPageModule.manageServices') }}
              </NButton>
            </div>
          </div>
        </div>

        <div v-if="services.length > 0" class="status-services-grid stagger-card">
          <div
            v-for="svc in services"
            :key="svc.id"
            class="status-service-card surface-card"
          >
            <div class="svc-icon" :style="{ background: statusBg(svc.status), color: statusColor(svc.status) }">
              <component :is="getIcon(svc.icon)" :size="18" />
            </div>
            <div class="svc-info">
              <span class="svc-name">{{ svc.name }}</span>
              <span v-if="svc.description" class="svc-desc">{{ svc.description }}</span>
              <span class="svc-status" :style="{ color: statusColor(svc.status) }">
                <span class="svc-dot" :style="{ background: statusColor(svc.status) }" />
                {{ statusLabel(svc.status) }}
              </span>
            </div>
          </div>
        </div>

        <div v-else-if="!loading" class="status-empty">
          <Server :size="32" style="color: var(--sre-text-tertiary); margin-bottom: 8px;" />
          <span style="color: var(--sre-text-tertiary); font-size: 13px;">{{ t('statusPageModule.noServices') }}</span>
          <NButton size="small" type="primary" style="margin-top: 12px;" @click="openCreate">
            <template #icon><Plus :size="14" /></template>
            {{ t('statusPageModule.addService') }}
          </NButton>
        </div>
      </NSpin>
    </div>

    <!-- Subscribe CTA -->
    <div class="status-cta-card content-card" style="text-align: center;">
      <div class="cta-icon-wrap">
        <Bell :size="28" />
      </div>
      <h3 class="section-title" style="margin-bottom: 8px;">{{ t('statusPageModule.subscribe') }}</h3>
      <p class="text-caption text-secondary" style="margin-bottom: 20px; max-width: 380px; margin-left: auto; margin-right: auto;">
        {{ t('statusPageModule.subscribeHint') }}
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
          {{ t('statusPageModule.subscribe') }}
        </NButton>
      </div>
    </div>

    <!-- Manage Services Modal -->
    <NModal v-model:show="showManage" preset="card" :title="t('statusPageModule.manageServices')" style="width: 640px; max-width: 90vw;" :bordered="false">
      <template #header-extra>
        <NButton type="primary" size="small" @click="openCreate">
          <template #icon><Plus :size="14" /></template>
          {{ t('statusPageModule.addService') }}
        </NButton>
      </template>
      <div v-if="services.length === 0" class="manage-empty">
        <Server :size="28" style="color: var(--sre-text-tertiary); margin-bottom: 6px;" />
        <span style="color: var(--sre-text-tertiary); font-size: 13px;">{{ t('statusPageModule.noServices') }}</span>
      </div>
      <div v-else class="manage-list">
        <div v-for="svc in services" :key="svc.id" class="manage-row">
          <div class="manage-row-icon" :style="{ background: statusBg(svc.status), color: statusColor(svc.status) }">
            <component :is="getIcon(svc.icon)" :size="16" />
          </div>
          <div class="manage-row-info">
            <span class="manage-row-name">{{ svc.name }}</span>
            <span v-if="svc.description" class="manage-row-desc">{{ svc.description }}</span>
          </div>
          <span class="manage-row-status" :style="{ color: statusColor(svc.status) }">
            <span class="svc-dot" :style="{ background: statusColor(svc.status) }" />
            {{ statusLabel(svc.status) }}
          </span>
          <div class="manage-row-actions">
            <NButton size="tiny" quaternary @click="openEdit(svc)">
              <template #icon><Pencil :size="13" /></template>
            </NButton>
            <NPopconfirm @positive-click="handleDelete(svc.id)">
              <template #trigger>
                <NButton size="tiny" quaternary type="error">
                  <template #icon><Trash2 :size="13" /></template>
                </NButton>
              </template>
              {{ t('statusPageModule.deleteServiceConfirm') }}
            </NPopconfirm>
          </div>
        </div>
      </div>
    </NModal>

    <!-- Create / Edit Form Modal -->
    <NModal v-model:show="showForm" preset="card" :title="editingId ? t('statusPageModule.editService') : t('statusPageModule.addService')" style="width: 480px; max-width: 90vw;" :bordered="false">
      <NForm label-placement="left" label-width="80" :model="form">
        <NFormItem :label="t('statusPageModule.serviceName')" required>
          <NInput v-model:value="form.name" :placeholder="t('statusPageModule.serviceNamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('statusPageModule.serviceStatus')">
          <NSelect v-model:value="form.status" :options="statusOptions" />
        </NFormItem>
        <NFormItem :label="t('common.description')">
          <NInput v-model:value="form.description" type="textarea" :rows="2" :placeholder="t('statusPageModule.serviceDescPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('statusPageModule.serviceIcon')">
          <NSelect v-model:value="form.icon" :options="iconOptions" />
        </NFormItem>
        <NFormItem :label="t('statusPageModule.serviceUrl')">
          <NInput v-model:value="form.url" :placeholder="t('statusPageModule.serviceUrlPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('statusPageModule.sortOrder')">
          <NInputNumber v-model:value="form.sort_order" :min="0" style="width: 120px;" />
        </NFormItem>
      </NForm>
      <template #footer>
        <div style="display: flex; justify-content: flex-end; gap: 8px;">
          <NButton @click="showForm = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">{{ t('common.save') }}</NButton>
        </div>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.status-hero {
  text-align: center;
  padding: 48px 0 32px;
  animation: sre-fade-in 400ms var(--sre-ease-out) both;
}

/* --- Status Section --- */
.status-section {
  background: var(--sre-bg-card);
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-xl);
  padding: 24px;
  margin-bottom: 24px;
  box-shadow: var(--sre-shadow-sm);
  animation: sre-fade-in 400ms var(--sre-ease-out) 80ms both;
}

.status-header {
  margin-bottom: 16px;
}

.status-header-row {
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

.svc-desc {
  font-size: var(--sre-fs-xs);
  color: var(--sre-text-tertiary);
  line-height: 1.3;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
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

.status-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 48px 24px;
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

/* --- Manage Modal --- */
.manage-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  padding: 32px 16px;
}

.manage-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.manage-row {
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 10px 12px;
  border-radius: var(--sre-radius-lg);
  border: 1px solid var(--sre-border);
  background: var(--sre-bg-card);
  transition: border-color var(--sre-duration-fast) var(--sre-ease-out);
}

.manage-row:hover {
  border-color: var(--sre-border-strong);
}

.manage-row-icon {
  width: 32px;
  height: 32px;
  border-radius: 8px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.manage-row-info {
  flex: 1;
  min-width: 0;
  display: flex;
  flex-direction: column;
  gap: 1px;
}

.manage-row-name {
  font-size: var(--sre-fs-sm);
  font-weight: var(--sre-fw-medium);
  color: var(--sre-text-primary);
}

.manage-row-desc {
  font-size: var(--sre-fs-xs);
  color: var(--sre-text-tertiary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.manage-row-status {
  display: flex;
  align-items: center;
  gap: 5px;
  font-size: var(--sre-fs-xs);
  font-weight: var(--sre-fw-medium);
  flex-shrink: 0;
}

.manage-row-actions {
  display: flex;
  align-items: center;
  gap: 2px;
  flex-shrink: 0;
}

.status-header-actions {
  display: flex;
  align-items: center;
  gap: 12px;
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
  .manage-row {
    flex-wrap: wrap;
  }
  .manage-row-info {
    flex-basis: 100%;
    order: -1;
  }
}
</style>
