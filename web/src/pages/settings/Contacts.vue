<script setup lang="ts">
import { computed, ref, shallowRef, onMounted, h } from 'vue'
import {
  useMessage,
  useDialog,
  NButton,
  NIcon,
  NInput,
  NModal,
  NForm,
  NFormItem,
  NSpace,
  NSelect,
  NSwitch,
  NSpin,
  NTag,
  NDropdown,
} from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { userContactApi } from '@/api'
import type { UserContact } from '@/api/user-contact'
import { getErrorMessage } from '@/utils/format'
import {
  AddOutline,
  EllipsisHorizontal,
  MailOutline,
  CallOutline,
  ChatbubbleEllipsesOutline,
  LinkOutline,
  StarOutline,
  Star,
} from '@vicons/ionicons5'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const message = useMessage()
const dialog = useDialog()
const { t } = useI18n()

const loading = ref(false)
const list = shallowRef<UserContact[]>([])
const showModal = ref(false)
const saving = ref(false)
const editingId = ref<number | null>(null)

const form = ref({
  type: 'email',
  value: '',
  name: '',
  is_default: false,
})

// Contact type options
const contactTypes = computed(() => [
  { label: t('settings.contactTypeEmail'), value: 'email' },
  { label: t('settings.contactTypePhone'), value: 'phone' },
  { label: t('settings.contactTypeFeishu'), value: 'feishu' },
  { label: t('settings.contactTypeWecom'), value: 'wecom' },
  { label: t('settings.contactTypeDingtalk'), value: 'dingtalk' },
  { label: t('settings.contactTypeWebhook'), value: 'webhook' },
])

// Dynamic placeholder based on selected contact type
const contactValuePlaceholder = computed(() => {
  const map: Record<string, string> = {
    email: t('settings.contactPlaceholderEmail'),
    phone: t('settings.contactPlaceholderPhone'),
    feishu: t('settings.contactPlaceholderFeishu'),
    wecom: t('settings.contactPlaceholderWecom'),
    dingtalk: t('settings.contactPlaceholderDingtalk'),
    webhook: t('settings.contactPlaceholderWebhook'),
  }
  return map[form.value.type] || t('settings.contactValuePlaceholder')
})

// Type icon mapping
function typeIcon(type: string) {
  const map: Record<string, typeof MailOutline> = {
    email: MailOutline,
    phone: CallOutline,
    feishu: ChatbubbleEllipsesOutline,
    wecom: ChatbubbleEllipsesOutline,
    dingtalk: ChatbubbleEllipsesOutline,
    webhook: LinkOutline,
  }
  return map[type] || MailOutline
}

function typeLabel(type: string): string {
  const map: Record<string, string> = {
    email: t('settings.contactTypeEmail'),
    phone: t('settings.contactTypePhone'),
    feishu: t('settings.contactTypeFeishu'),
    wecom: t('settings.contactTypeWecom'),
    dingtalk: t('settings.contactTypeDingtalk'),
    webhook: t('settings.contactTypeWebhook'),
  }
  return map[type] || type
}

// Validation
function validate(): string | null {
  if (!form.value.value.trim()) return t('common.required')
  if (!form.value.name.trim()) return t('common.required')

  if (form.value.type === 'email') {
    const emailRe = /^[^\s@]+@[^\s@]+\.[^\s@]+$/
    if (!emailRe.test(form.value.value)) return t('settings.invalidEmail')
  }
  if (form.value.type === 'phone') {
    const phoneRe = /^[\d\-+\s()]{6,20}$/
    if (!phoneRe.test(form.value.value)) return t('settings.invalidPhone')
  }
  if (form.value.type === 'webhook') {
    try {
      new URL(form.value.value)
    } catch {
      return t('settings.invalidUrl')
    }
  }
  return null
}

// Group contacts by type
const groupedContacts = computed(() => {
  const groups: Record<string, UserContact[]> = {}
  for (const c of list.value) {
    if (!groups[c.type]) groups[c.type] = []
    groups[c.type].push(c)
  }
  return groups
})

const typeOrder = ['email', 'phone', 'feishu', 'wecom', 'dingtalk', 'webhook']

const sortedTypes = computed(() => {
  return Object.keys(groupedContacts.value).sort(
    (a, b) => typeOrder.indexOf(a) - typeOrder.indexOf(b),
  )
})

// Fetch contacts
async function fetchList() {
  loading.value = true
  try {
    const { data } = await userContactApi.list()
    list.value = data.data || []
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

// Open create modal
function openCreate() {
  editingId.value = null
  form.value = { type: 'email', value: '', name: '', is_default: false }
  showModal.value = true
}

// Open edit modal
function openEdit(row: UserContact) {
  editingId.value = row.id
  form.value = {
    type: row.type,
    value: row.value,
    name: row.name,
    is_default: row.is_default,
  }
  showModal.value = true
}

// Save contact
async function handleSave() {
  const err = validate()
  if (err) {
    message.warning(err)
    return
  }

  saving.value = true
  try {
    if (editingId.value) {
      await userContactApi.update(editingId.value, { ...form.value })
      message.success(t('settings.contactUpdated'))
    } else {
      await userContactApi.create({ ...form.value })
      message.success(t('settings.contactCreated'))
    }
    showModal.value = false
    await fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

// Delete contact
async function handleDelete(id: number) {
  try {
    await userContactApi.delete(id)
    message.success(t('settings.contactDeleted'))
    await fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function confirmDelete(id: number) {
  dialog.warning({
    title: t('common.confirmDelete'),
    content: t('settings.deleteContactConfirm'),
    positiveText: t('common.confirmDelete'),
    negativeText: t('common.cancel'),
    onPositiveClick: () => handleDelete(id),
  })
}

// Set default
async function handleSetDefault(id: number) {
  try {
    await userContactApi.setDefault(id)
    message.success(t('settings.contactDefaultSet'))
    await fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

// Dropdown menu options
function rowMenuOptions(row: UserContact) {
  const options = [
    { key: 'edit', label: t('common.edit') },
    { key: 'default', label: t('settings.setDefault'), disabled: row.is_default },
    { key: 'delete', label: t('common.delete') },
  ]
  return options
}

function handleMenu(key: string, row: UserContact) {
  if (key === 'edit') {
    openEdit(row)
  } else if (key === 'default') {
    handleSetDefault(row.id)
  } else if (key === 'delete') {
    confirmDelete(row.id)
  }
}

const ellipsisIcon = () => h(NIcon, { component: EllipsisHorizontal })

onMounted(fetchList)
</script>

<template>
  <div class="contacts-page">
    <header class="page-header">
      <div>
        <h2 class="page-title">{{ t('settings.contacts') }}</h2>
        <p class="page-subtitle">{{ t('settings.contactsDesc') }}</p>
      </div>
      <NButton type="primary" size="small" @click="openCreate">
        <template #icon><NIcon :component="AddOutline" /></template>
        {{ t('settings.addContact') }}
      </NButton>
    </header>

    <LoadingSkeleton v-if="loading && list.length === 0" :rows="4" variant="row" />
    <NSpin v-else :show="loading">
      <div v-if="list.length === 0 && !loading" class="empty-state">
        {{ t('settings.noContacts') }}
      </div>

      <div v-for="type in sortedTypes" :key="type" class="contact-group">
        <div class="group-header">
          <NIcon :component="typeIcon(type)" :size="16" class="group-icon" :data-type="type" />
          <span class="group-label">{{ typeLabel(type) }}</span>
          <NTag size="tiny" :bordered="false">{{ groupedContacts[type].length }}</NTag>
        </div>

        <div class="contact-list sre-stagger">
          <div v-for="c in groupedContacts[type]" :key="c.id" class="sre-row-card contact-row">
            <div class="contact-icon" :data-type="c.type">
              <NIcon :component="typeIcon(c.type)" :size="18" />
            </div>
            <div class="contact-main">
              <div class="contact-name">
                {{ c.name }}
                <NTag v-if="c.is_default" type="success" size="tiny" :bordered="false">
                  <template #icon><NIcon :component="Star" :size="12" /></template>
                  {{ t('settings.contactDefault') }}
                </NTag>
              </div>
              <div class="contact-value mono">{{ c.value }}</div>
            </div>
            <NDropdown
              trigger="click"
              :options="rowMenuOptions(c)"
              @select="(k: string) => handleMenu(k, c)"
            >
              <NButton size="tiny" quaternary :render-icon="ellipsisIcon" />
            </NDropdown>
          </div>
        </div>
      </div>
    </NSpin>

    <!-- Add/Edit Modal -->
    <NModal
      v-model:show="showModal"
      preset="card"
      :title="editingId ? t('settings.editContact') : t('settings.addContact')"
      style="width: 480px; max-width: 90vw"
      :bordered="false"
    >
      <NForm label-placement="top">
        <NFormItem :label="t('settings.contactType')" required>
          <NSelect
            v-model:value="form.type"
            :options="contactTypes"
            :disabled="!!editingId"
          />
        </NFormItem>
        <NFormItem :label="t('settings.contactName')" required>
          <NInput
            v-model:value="form.name"
            :placeholder="t('settings.contactNamePlaceholder')"
          />
        </NFormItem>
        <NFormItem :label="t('settings.contactValue')" required>
          <NInput
            v-model:value="form.value"
            :placeholder="contactValuePlaceholder"
          />
        </NFormItem>
        <NFormItem :label="t('settings.contactDefault')">
          <NSwitch v-model:value="form.is_default" />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace justify="end">
          <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.contacts-page {
  font-family: var(--sre-font-sans);
}

.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  padding-bottom: 16px;
  border-bottom: var(--sre-hairline);
  margin-bottom: 20px;
}

.page-title {
  font-size: 18px;
  font-weight: 600;
  margin: 0 0 4px;
  color: var(--sre-text-primary);
}

.page-subtitle {
  font-size: 12px;
  color: var(--sre-text-secondary);
  margin: 0;
}

.contact-group {
  margin-bottom: 24px;
}

.group-header {
  display: flex;
  align-items: center;
  gap: 8px;
  margin-bottom: 10px;
  padding: 0 2px;
}

.group-icon {
  color: var(--sre-text-secondary);
}

.group-icon[data-type="email"]     { color: #3B82F6; }
.group-icon[data-type="phone"]     { color: #10B981; }
.group-icon[data-type="feishu"]    { color: #6366F1; }
.group-icon[data-type="wecom"]     { color: #14B8A6; }
.group-icon[data-type="dingtalk"]  { color: #F59E0B; }
.group-icon[data-type="webhook"]   { color: #8B5CF6; }

.group-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.contact-list {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.contact-row {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 14px;
}

.contact-icon {
  width: 36px;
  height: 36px;
  border-radius: var(--sre-radius-sm);
  display: flex;
  align-items: center;
  justify-content: center;
  background: var(--sre-bg-elevated);
  flex-shrink: 0;
  color: var(--sre-info);
}

.contact-icon[data-type="email"]     { color: #3B82F6; }
.contact-icon[data-type="phone"]     { color: #10B981; }
.contact-icon[data-type="feishu"]    { color: #6366F1; }
.contact-icon[data-type="wecom"]     { color: #14B8A6; }
.contact-icon[data-type="dingtalk"]  { color: #F59E0B; }
.contact-icon[data-type="webhook"]   { color: #8B5CF6; }

.contact-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  gap: 4px;
  min-width: 0;
}

.contact-name {
  font-size: 14px;
  font-weight: 600;
  color: var(--sre-text-primary);
  display: flex;
  align-items: center;
  gap: 8px;
}

.contact-value {
  font-size: 12px;
  color: var(--sre-text-secondary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}

.mono {
  font-family: var(--sre-font-mono);
}

.empty-state {
  padding: 40px 0;
  text-align: center;
  font-size: 13px;
  color: var(--sre-text-tertiary);
}
</style>
