<script setup lang="ts">
import { computed, reactive, ref, shallowRef, onMounted, h } from 'vue'
import {
  useMessage,
  NButton,
  NIcon,
  NInput,
  NRadioGroup,
  NRadioButton,
  NDropdown,
  NModal,
  NForm,
  NFormItem,
  NSpace,
  NRadio,
  NSpin,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { userApi } from '@/api'
import type { User } from '@/types'
import { getErrorMessage } from '@/utils/format'
import {
  AddOutline,
  EllipsisHorizontal,
  HardwareChipOutline,
  ChatbubblesOutline,
  SearchOutline,
} from '@vicons/ionicons5'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const list = shallowRef<User[]>([])
const filterType = ref<'all' | 'bot' | 'channel'>('all')
const search = ref('')

const showModal = ref(false)
const saving = ref(false)

const form = reactive({
  display_name: '',
  user_type: 'bot' as 'bot' | 'channel',
  notify_target: '',
})

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return list.value.filter(u => {
    if (filterType.value !== 'all' && u.user_type !== filterType.value) return false
    if (q) {
      const hay = `${u.username} ${u.display_name} ${u.notify_target || ''}`.toLowerCase()
      if (!hay.includes(q)) return false
    }
    return true
  })
})

async function fetchList() {
  loading.value = true
  try {
    const { data } = await userApi.list({ page: 1, page_size: 200 })
    list.value = (data.data.list || []).filter(
      u => u.user_type === 'bot' || u.user_type === 'channel'
    )
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  Object.assign(form, { display_name: '', user_type: 'bot', notify_target: '' })
  showModal.value = true
}

async function handleSave() {
  if (!form.display_name.trim()) {
    message.warning(t('settings.displayNameRequired'))
    return
  }
  saving.value = true
  try {
    const username = `virtual_${form.display_name
      .toLowerCase()
      .replace(/\s+/g, '_')
      .replace(/[^a-z0-9_]/g, '')}_${Date.now().toString(36)}`
    await userApi.createVirtual({
      username,
      display_name: form.display_name,
      user_type: form.user_type,
      notify_target: form.notify_target || undefined,
    })
    message.success(t('settings.virtualUserCreated'))
    showModal.value = false
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await userApi.delete(id)
    message.success(t('settings.userDeleted'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function rowMenuOptions() {
  return [{ key: 'delete', label: t('common.delete') }]
}

function handleMenu(key: string, u: User) {
  if (key === 'delete') {
    if (confirm(t('settings.deleteVirtualConfirm'))) handleDelete(u.id)
  }
}

function typeLabel(type: string | undefined): string {
  if (type === 'bot') return t('settings.botType')
  if (type === 'channel') return t('settings.virtualChannelType')
  return type || '—'
}

const ellipsisIcon = () => h(NIcon, { component: EllipsisHorizontal })

onMounted(fetchList)
</script>

<template>
  <div class="vuser-mgmt">
    <header class="page-header">
      <div>
        <h2 class="page-title">{{ t('settings.virtualUsers') }}</h2>
        <p class="page-subtitle">{{ t('settings.virtualUsersDesc') }}</p>
      </div>
      <NButton type="primary" size="small" @click="openCreate">
        <template #icon><NIcon :component="AddOutline" /></template>
        {{ t('settings.createVirtual') }}
      </NButton>
    </header>

    <div class="filter-bar">
      <div class="filter-group">
        <span class="sre-label-eyebrow">{{ t('settings.userType') }}</span>
        <NRadioGroup v-model:value="filterType" size="small">
          <NRadioButton value="all">{{ t('common.all') }}</NRadioButton>
          <NRadioButton value="bot">{{ t('settings.botType') }}</NRadioButton>
          <NRadioButton value="channel">{{ t('settings.virtualChannelType') }}</NRadioButton>
        </NRadioGroup>
      </div>
      <NInput
        v-model:value="search"
        size="small"
        :placeholder="t('common.search')"
        clearable
        class="search-input"
      >
        <template #prefix><NIcon :component="SearchOutline" /></template>
      </NInput>
    </div>

    <LoadingSkeleton v-if="loading && filtered.length === 0" :rows="4" variant="row" />
    <NSpin v-else :show="loading">
    <div class="vuser-list sre-stagger">
      <div v-for="u in filtered" :key="u.id" class="sre-row-card vuser-row">
        <div class="vuser-icon" :data-type="u.user_type">
          <NIcon
            :component="u.user_type === 'bot' ? HardwareChipOutline : ChatbubblesOutline"
            :size="18"
          />
        </div>
        <div class="vuser-main">
          <div class="vuser-name">{{ u.display_name || u.username }}</div>
          <div class="vuser-meta">
            <span class="vuser-type-chip" :data-type="u.user_type">{{ typeLabel(u.user_type) }}</span>
            <span class="sre-meta-divider"></span>
            <span class="mono">{{ u.username }}</span>
            <template v-if="u.notify_target">
              <span class="sre-meta-divider"></span>
              <span class="vuser-target" :title="u.notify_target">{{ u.notify_target }}</span>
            </template>
          </div>
        </div>
        <NDropdown
          trigger="click"
          :options="rowMenuOptions()"
          @select="(k: string) => handleMenu(k, u)"
        >
          <NButton size="tiny" quaternary :render-icon="ellipsisIcon" />
        </NDropdown>
      </div>

      <div v-if="!loading && filtered.length === 0" class="empty-state">
        {{ t('settings.noVirtualUsers') }}
      </div>
    </div>
    </NSpin>

    <NModal v-model:show="showModal" preset="card" :title="t('settings.createVirtual')" style="width: 520px; max-width: 90vw" :bordered="false">
      <NForm label-placement="top">
        <NFormItem :label="t('settings.displayName')" required>
          <NInput v-model:value="form.display_name" :placeholder="t('settings.displayNamePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('settings.userType')">
          <NRadioGroup v-model:value="form.user_type">
            <NSpace>
              <NRadio value="bot">{{ t('settings.botType') }}</NRadio>
              <NRadio value="channel">{{ t('settings.virtualChannelType') }}</NRadio>
            </NSpace>
          </NRadioGroup>
        </NFormItem>
        <NFormItem :label="t('settings.notifyTarget')">
          <NInput
            v-model:value="form.notify_target"
            type="textarea"
            :rows="3"
            :placeholder="form.user_type === 'bot' ? t('settings.botNotifyTargetHint') : t('settings.channelNotifyTargetHint')"
          />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">{{ t('common.create') }}</NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.vuser-mgmt { font-family: var(--sre-font-sans); }

.page-header {
  display: flex; align-items: flex-start; justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: var(--sre-hairline);
  margin-bottom: 20px;
}
.page-title { font-size: 18px; font-weight: 600; margin: 0 0 4px; color: var(--sre-text-primary); }
.page-subtitle { font-size: 12px; color: var(--sre-text-secondary); margin: 0; }

.filter-bar {
  display: flex; align-items: center; gap: 20px;
  flex-wrap: wrap;
  margin-bottom: 16px;
  padding: 12px 14px;
  background: var(--sre-bg-elevated);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md);
}
.filter-group { display: flex; align-items: center; gap: 8px; }
.search-input { max-width: 220px; margin-left: auto; }

.vuser-list { display: flex; flex-direction: column; gap: 6px; }

.vuser-row {
  display: flex; align-items: center; gap: 12px;
  padding: 10px 14px;
}

.vuser-icon {
  width: 36px; height: 36px;
  border-radius: var(--sre-radius-sm);
  display: flex; align-items: center; justify-content: center;
  background: var(--sre-bg-elevated);
  flex-shrink: 0;
}
.vuser-icon[data-type="bot"]     { color: var(--sre-info); }
.vuser-icon[data-type="channel"] { color: var(--sre-info); }

.vuser-main { flex: 1; display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.vuser-name {
  font-size: 14px; font-weight: 600;
  color: var(--sre-text-primary);
}
.vuser-meta {
  display: flex; align-items: center; gap: 6px;
  font-size: 12px;
  color: var(--sre-text-tertiary);
  flex-wrap: wrap;
  min-width: 0;
}
.mono { font-family: var(--sre-font-mono); }
.vuser-target {
  overflow: hidden; text-overflow: ellipsis; white-space: nowrap;
  max-width: 320px;
}

.vuser-type-chip {
  font-size: 11px;
  padding: 2px 8px;
  border-radius: 4px;
  font-family: var(--sre-font-mono);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}
.vuser-type-chip[data-type="bot"]     { background: var(--sre-info-soft); color: var(--sre-info); }
.vuser-type-chip[data-type="channel"] { background: var(--sre-info-soft); color: var(--sre-info); }

.empty-state {
  padding: 40px 0; text-align: center;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}
</style>
