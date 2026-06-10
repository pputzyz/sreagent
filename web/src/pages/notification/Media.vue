<script setup lang="ts">
import { ref, computed, onMounted, h, type Ref, type Component } from 'vue'
import { useMessage, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { notifyMediaApi } from '@/api'
import type { NotifyMedia } from '@/types'
import { getErrorMessage } from '@/utils/format'
import { useCrudPage } from '@/composables/useCrudPage'
import type { CrudApiModule } from '@/composables/useCrudPage'
import {
  AddOutline,
  SearchOutline,
  ChatbubblesOutline,
  MailOutline,
  GlobeOutline,
  TerminalOutline,
  FlashOutline,
} from '@vicons/ionicons5'
import {
  MessageCircle,
  Hash,
  Send,
  CreditCard,
  Smartphone,
  Zap,
  BellRing,
  MessageSquareText,
  AppWindow,
} from 'lucide-vue-next'
import KVEditor from '@/components/common/KVEditor.vue'
import PageHeader from '@/components/common/PageHeader.vue'
import LoadingSkeleton from '@/components/common/LoadingSkeleton.vue'

const message = useMessage()
const { t } = useI18n()

type MediaType =
  | 'lark_webhook' | 'email' | 'http' | 'script'
  | 'dingtalk_webhook' | 'wecom_webhook' | 'slack_webhook' | 'discord_webhook'
  | 'telegram_bot' | 'feishu_webhook' | 'feishu_card' | 'feishu_app'
  | 'wecom_app' | 'flashduty' | 'pagerduty' | 'tencent_sms' | 'aliyun_sms'

interface MediaForm {
  name: string
  description: string
  type: MediaType
  is_enabled: boolean
  config: string
  variables: string
  webhook_url: string
  smtp_host: string
  smtp_port: number
  username: string
  password: string
  from: string
  method: string
  url: string
  headers: { key: string; value: string }[]
  body: string
  path: string
  args: string
  // telegram_bot
  bot_token: string
  chat_id: string
  // feishu_app / wecom_app
  app_id: string
  app_secret: string
  receive_id: string
  receive_id_type: string
  corp_id: string
  corp_secret: string
  agent_id: string
  to_user: string
  // flashduty
  integration_url: string
  // pagerduty
  routing_key: string
  // tencent_sms
  secret_id: string
  secret_key: string
  sdk_app_id: string
  template_id: string
  sign_name: string
  phone_numbers: string
  // aliyun_sms
  access_key_id: string
  access_key_secret: string
  template_code: string
}

function parseConfig(configStr: string): Record<string, unknown> {
  try { return JSON.parse(configStr || '{}') } catch { return {} }
}

function buildConfigString(f: Record<string, unknown>): string {
  switch (f.type) {
    case 'lark_webhook':
    case 'dingtalk_webhook':
    case 'wecom_webhook':
    case 'slack_webhook':
    case 'discord_webhook':
    case 'feishu_webhook':
    case 'feishu_card':
      return JSON.stringify({ webhook_url: f.webhook_url }, null, 2)
    case 'email':
      return JSON.stringify({
        smtp_host: f.smtp_host, smtp_port: f.smtp_port,
        username: f.username, password: f.password, from: f.from,
      }, null, 2)
    case 'http': {
      const hdrs: Record<string, string> = {}
      for (const hdr of (f.headers as { key: string; value: string }[] || [])) { if (hdr.key?.trim()) hdrs[hdr.key.trim()] = hdr.value }
      return JSON.stringify({ method: f.method, url: f.url, headers: hdrs, body: f.body }, null, 2)
    }
    case 'script':
      return JSON.stringify({ path: f.path, args: f.args }, null, 2)
    case 'telegram_bot':
      return JSON.stringify({ bot_token: f.bot_token, chat_id: f.chat_id }, null, 2)
    case 'feishu_app':
      return JSON.stringify({
        app_id: f.app_id, app_secret: f.app_secret,
        receive_id: f.receive_id, receive_id_type: f.receive_id_type,
      }, null, 2)
    case 'wecom_app':
      return JSON.stringify({
        corp_id: f.corp_id, corp_secret: f.corp_secret,
        agent_id: f.agent_id, to_user: f.to_user,
      }, null, 2)
    case 'flashduty':
      return JSON.stringify({ integration_url: f.integration_url }, null, 2)
    case 'pagerduty':
      return JSON.stringify({ routing_key: f.routing_key }, null, 2)
    case 'tencent_sms':
      return JSON.stringify({
        secret_id: f.secret_id, secret_key: f.secret_key,
        sdk_app_id: f.sdk_app_id, template_id: f.template_id,
        sign_name: f.sign_name, phone_numbers: f.phone_numbers,
      }, null, 2)
    case 'aliyun_sms':
      return JSON.stringify({
        access_key_id: f.access_key_id, access_key_secret: f.access_key_secret,
        sign_name: f.sign_name, template_code: f.template_code,
        phone_numbers: f.phone_numbers,
      }, null, 2)
    default:
      return '{}'
  }
}

const crud = useCrudPage<NotifyMedia>({
  api: notifyMediaApi as unknown as CrudApiModule<NotifyMedia>,
  defaultForm: () => ({
    name: '', description: '', type: 'lark_webhook' as MediaType,
    is_enabled: true, config: '{}', variables: '{}',
    webhook_url: '', smtp_host: '', smtp_port: 25,
    username: '', password: '', from: '',
    method: 'POST', url: '', headers: [] as { key: string; value: string }[],
    body: '', path: '', args: '',
    bot_token: '', chat_id: '',
    app_id: '', app_secret: '', receive_id: '', receive_id_type: '',
    corp_id: '', corp_secret: '', agent_id: '', to_user: '',
    integration_url: '', routing_key: '',
    secret_id: '', secret_key: '', sdk_app_id: '', template_id: '',
    sign_name: '', phone_numbers: '',
    access_key_id: '', access_key_secret: '', template_code: '',
  } as unknown as Partial<NotifyMedia>),
  i18nKeys: {
    created: 'media.created',
    updated: 'media.updated',
    deleted: 'media.deleted',
    deleteConfirm: 'media.deleteConfirm',
    createTitle: 'media.create',
    editTitle: 'media.edit',
  },
  rowToForm: (row) => {
    const cfg = parseConfig(row.config || '{}')
    return {
      name: row.name, description: row.description, type: row.type,
      is_enabled: row.is_enabled, config: row.config, variables: row.variables || '{}',
      webhook_url: (cfg.webhook_url as string) || '',
      smtp_host: (cfg.smtp_host as string) || '', smtp_port: (cfg.smtp_port as number) || 25,
      username: (cfg.username as string) || '', password: (cfg.password as string) || '', from: (cfg.from as string) || '',
      method: (cfg.method as string) || 'POST', url: (cfg.url as string) || '',
      headers: Object.entries((cfg.headers as Record<string, string>) || {}).map(([key, value]) => ({ key, value: String(value) })),
      body: (cfg.body as string) || '', path: (cfg.path as string) || '', args: (cfg.args as string) || '',
      bot_token: (cfg.bot_token as string) || '', chat_id: (cfg.chat_id as string) || '',
      app_id: (cfg.app_id as string) || '', app_secret: (cfg.app_secret as string) || '',
      receive_id: (cfg.receive_id as string) || '', receive_id_type: (cfg.receive_id_type as string) || '',
      corp_id: (cfg.corp_id as string) || '', corp_secret: (cfg.corp_secret as string) || '',
      agent_id: (cfg.agent_id as string) || '', to_user: (cfg.to_user as string) || '',
      integration_url: (cfg.integration_url as string) || '', routing_key: (cfg.routing_key as string) || '',
      secret_id: (cfg.secret_id as string) || '', secret_key: (cfg.secret_key as string) || '',
      sdk_app_id: (cfg.sdk_app_id as string) || '', template_id: (cfg.template_id as string) || '',
      sign_name: (cfg.sign_name as string) || '', phone_numbers: (cfg.phone_numbers as string) || '',
      access_key_id: (cfg.access_key_id as string) || '', access_key_secret: (cfg.access_key_secret as string) || '',
      template_code: (cfg.template_code as string) || '',
    } as unknown as Partial<NotifyMedia>
  },
  formToPayload: (form) => {
    const f = form as Record<string, unknown>
    return {
      name: form.name, description: form.description, type: form.type,
      is_enabled: form.is_enabled, config: buildConfigString(f), variables: form.variables,
    }
  },
  validate: (form) => {
    if (!form.name?.trim()) return t('media.nameRequired')
    try { JSON.parse(form.variables || '{}') } catch { return t('media.variables') + ': ' + t('media.invalidJson') }
    return null
  },
  pageSize: 100,
})

const {
  loading,
  items: mediaList,
  search,
  showModal,
  modalTitle,
  editingId,
  saving,
  fetchList,
  openCreate,
  openEdit,
  handleSave,
  confirmDelete,
} = crud
const form = crud.form as Ref<MediaForm>

const testingId = ref<number | null>(null)
const typeFilter = ref<string>('')

const typeOptions = computed(() => [
  { label: t('media.larkWebhook'), value: 'lark_webhook' },
  { label: t('media.email'), value: 'email' },
  { label: t('media.http'), value: 'http' },
  { label: t('media.script'), value: 'script' },
  { label: t('media.channelType.dingtalk_webhook'), value: 'dingtalk_webhook' },
  { label: t('media.channelType.wecom_webhook'), value: 'wecom_webhook' },
  { label: t('media.channelType.slack_webhook'), value: 'slack_webhook' },
  { label: t('media.channelType.discord_webhook'), value: 'discord_webhook' },
  { label: t('media.channelType.telegram_bot'), value: 'telegram_bot' },
  { label: t('media.channelType.feishu_webhook'), value: 'feishu_webhook' },
  { label: t('media.channelType.feishu_card'), value: 'feishu_card' },
  { label: t('media.channelType.feishu_app'), value: 'feishu_app' },
  { label: t('media.channelType.wecom_app'), value: 'wecom_app' },
  { label: t('media.channelType.flashduty'), value: 'flashduty' },
  { label: t('media.channelType.pagerduty'), value: 'pagerduty' },
  { label: t('media.channelType.tencent_sms') + ` (${t('common.notImplemented')})`, value: 'tencent_sms', disabled: true },
  { label: t('media.channelType.aliyun_sms') + ` (${t('common.notImplemented')})`, value: 'aliyun_sms', disabled: true },
])

const filterTypeOptions = computed(() => [
  { label: t('common.all'), value: '' },
  ...typeOptions.value,
])

const methodOptions = [
  { label: 'GET', value: 'GET' },
  { label: 'POST', value: 'POST' },
  { label: 'PUT', value: 'PUT' },
  { label: 'PATCH', value: 'PATCH' },
]

const receiveIdTypeOptions = [
  { label: 'open_id', value: 'open_id' },
  { label: 'user_id', value: 'user_id' },
  { label: 'union_id', value: 'union_id' },
  { label: 'email', value: 'email' },
  { label: 'chat_id', value: 'chat_id' },
]

function getTypeLabel(type: string) {
  const map: Record<string, string> = {
    lark_webhook: t('media.typeLark'),
    email: t('media.typeEmail'),
    http: t('media.typeHttp'),
    script: t('media.typeScript'),
    dingtalk_webhook: t('media.channelType.dingtalk_webhook'),
    wecom_webhook: t('media.channelType.wecom_webhook'),
    slack_webhook: t('media.channelType.slack_webhook'),
    discord_webhook: t('media.channelType.discord_webhook'),
    telegram_bot: t('media.channelType.telegram_bot'),
    feishu_webhook: t('media.channelType.feishu_webhook'),
    feishu_card: t('media.channelType.feishu_card'),
    feishu_app: t('media.channelType.feishu_app'),
    wecom_app: t('media.channelType.wecom_app'),
    flashduty: t('media.channelType.flashduty'),
    pagerduty: t('media.channelType.pagerduty'),
    tencent_sms: t('media.channelType.tencent_sms'),
    aliyun_sms: t('media.channelType.aliyun_sms'),
  }
  return map[type] || type
}

function getTypeIcon(type: string) {
  const map: Record<string, Component> = {
    lark_webhook: ChatbubblesOutline,
    email: MailOutline,
    http: GlobeOutline,
    script: TerminalOutline,
    dingtalk_webhook: MessageCircle,
    wecom_webhook: MessageCircle,
    slack_webhook: Hash,
    discord_webhook: MessageCircle,
    telegram_bot: Send,
    feishu_webhook: MessageCircle,
    feishu_card: CreditCard,
    feishu_app: AppWindow,
    wecom_app: Smartphone,
    flashduty: Zap,
    pagerduty: BellRing,
    tencent_sms: MessageSquareText,
    aliyun_sms: MessageSquareText,
  }
  return map[type] || FlashOutline
}

function getTargetSummary(row: NotifyMedia): string {
  try {
    const cfg = JSON.parse(row.config || '{}')
    switch (row.type) {
      case 'lark_webhook':
      case 'dingtalk_webhook':
      case 'wecom_webhook':
      case 'slack_webhook':
      case 'discord_webhook':
      case 'feishu_webhook':
      case 'feishu_card':
        return cfg.webhook_url ? cfg.webhook_url.replace(/^https?:\/\//, '') : '—'
      case 'email':
        return cfg.from ? `${cfg.from} via ${cfg.smtp_host}:${cfg.smtp_port}` : (cfg.smtp_host || '—')
      case 'http':
        return `${cfg.method || 'POST'} ${cfg.url || ''}`.trim()
      case 'script':
        return cfg.path || '—'
      case 'telegram_bot':
        return cfg.chat_id ? `chat: ${cfg.chat_id}` : '—'
      case 'feishu_app':
        return cfg.app_id ? `${cfg.app_id} -> ${cfg.receive_id || '?'}` : '—'
      case 'wecom_app':
        return cfg.corp_id ? `${cfg.corp_id}/${cfg.agent_id || '?'}` : '—'
      case 'flashduty':
        return cfg.integration_url ? cfg.integration_url.replace(/^https?:\/\//, '') : '—'
      case 'pagerduty':
        return cfg.routing_key || '—'
      case 'tencent_sms':
        return cfg.phone_numbers || '—'
      case 'aliyun_sms':
        return cfg.phone_numbers || '—'
      default:
        return '—'
    }
  } catch {
    return '—'
  }
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return mediaList.value.filter(m => {
    if (typeFilter.value && m.type !== typeFilter.value) return false
    if (!q) return true
    return (
      m.name.toLowerCase().includes(q) ||
      (m.description || '').toLowerCase().includes(q) ||
      getTargetSummary(m).toLowerCase().includes(q)
    )
  })
})

async function handleTest(id: number) {
  testingId.value = id
  try {
    const { data } = await notifyMediaApi.test(id)
    // Backend returns {message: ...} on success; HTTP 2xx means test passed
    const msg = data.data?.message
    message.success(msg || t('media.testSuccess'))
  } catch (err: unknown) { message.error(getErrorMessage(err)) } finally { testingId.value = null }
}

function rowMenuOptions(row: NotifyMedia) {
  return [
    { label: t('common.edit'), key: 'edit' },
    { label: t('common.test'), key: 'test' },
    { type: 'divider', key: 'd1' },
    {
      label: t('common.delete'), key: 'delete',
      disabled: row.is_builtin,
      props: { style: row.is_builtin ? '' : 'color: var(--sre-danger)' },
    },
  ]
}

function onRowMenu(key: string, row: NotifyMedia) {
  if (key === 'edit') openEdit(row)
  else if (key === 'test') handleTest(row.id)
  else if (key === 'delete' && !row.is_builtin) confirmDelete(row.id)
}

// Render dropdown trigger via h to keep template light
const RowMenu = (row: NotifyMedia) => h(NDropdown, {
  trigger: 'click',
  options: rowMenuOptions(row),
  onSelect: (key: string) => onRowMenu(key, row),
}, {
  default: () => h('button', { class: 'sre-icon-btn', 'aria-label': t('common.actions') },
    h('span', { class: 'sre-dots' })),
})

onMounted(fetchList)
</script>

<template>
  <div class="media-page">
    <PageHeader :title="t('media.title')" :subtitle="t('media.subtitle')">
      <template #actions>
        <n-button type="primary" size="small" @click="openCreate">
          <template #icon><n-icon :component="AddOutline" /></template>
          {{ t('media.create') }}
        </n-button>
      </template>
    </PageHeader>

    <div class="toolbar">
      <n-input v-model:value="search" size="small" :placeholder="t('common.search')" clearable style="width: 240px">
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <n-select v-model:value="typeFilter" size="small" :options="filterTypeOptions" style="width: 160px" />
      <span class="count tnum">{{ filtered.length }} / {{ mediaList.length }}</span>
    </div>

    <LoadingSkeleton v-if="loading && filtered.length === 0" :rows="4" variant="row" />

    <div v-else-if="filtered.length === 0" class="empty">
      <n-icon :component="FlashOutline" size="36" />
      <div class="empty-text">{{ t('media.noData') }}</div>
      <n-button type="primary" size="small" @click="openCreate">{{ t('media.create') }}</n-button>
    </div>

    <ul v-else class="row-list sre-stagger">
      <li v-for="m in filtered" :key="m.id" class="sre-notify-card sre-lift" :data-type="m.type">
        <div class="row-l1">
          <span class="type-icon" :data-type="m.type"><n-icon :component="getTypeIcon(m.type)" size="16" /></span>
          <span class="row-name">{{ m.name }}</span>
          <span class="type-chip" :data-type="m.type">{{ getTypeLabel(m.type) }}</span>
          <span v-if="m.is_builtin" class="builtin-chip">{{ t('media.builtin') }}</span>
          <span class="status-text" :class="{ off: !m.is_enabled }">
            {{ m.is_enabled ? t('common.on') : t('common.off') }}
          </span>
          <div class="row-actions">
            <n-button quaternary size="tiny" :loading="testingId === m.id" @click="handleTest(m.id)">
              {{ t('common.test') }}
            </n-button>
            <component :is="RowMenu(m)" />
          </div>
        </div>
        <div class="row-l2">
          <code class="target tnum">{{ getTargetSummary(m) }}</code>
        </div>
        <div class="row-l3" v-if="m.description">
          <span class="meta">{{ m.description }}</span>
        </div>
      </li>
    </ul>

    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" :bordered="false" class="media-modal">
      <n-form label-placement="top">
        <n-grid :x-gap="12" :cols="2">
          <n-gi>
            <n-form-item :label="t('media.name')" required>
              <n-input v-model:value="form.name" :placeholder="t('media.namePlaceholder')" />
            </n-form-item>
          </n-gi>
          <n-gi>
            <n-form-item :label="t('media.type')">
              <n-select v-model:value="form.type" :options="typeOptions" />
            </n-form-item>
          </n-gi>
        </n-grid>

        <n-form-item :label="t('media.description')">
          <n-input v-model:value="form.description" :placeholder="t('media.description')" />
        </n-form-item>

        <n-divider style="margin: 12px 0">{{ t('media.config') }}</n-divider>

        <template v-if="form.type === 'lark_webhook'">
          <n-form-item :label="t('media.webhookUrl')" required>
            <n-input v-model:value="form.webhook_url" :placeholder="t('mediaMgmt.webhookUrlPlaceholder')" />
          </n-form-item>
        </template>

        <template v-if="form.type === 'email'">
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.smtpHost')">
                <n-input v-model:value="form.smtp_host" :placeholder="t('mediaMgmt.smtpHostPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.smtpPort')">
                <n-input-number v-model:value="form.smtp_port" :min="1" :max="65535" style="width: 100%" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.username')">
                <n-input v-model:value="form.username" :placeholder="t('mediaMgmt.usernamePlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.password')">
                <n-input v-model:value="form.password" type="password" show-password-on="click" :placeholder="t('media.passwordPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-form-item :label="t('media.from')">
            <n-input v-model:value="form.from" :placeholder="t('mediaMgmt.fromPlaceholder')" />
          </n-form-item>
        </template>

        <template v-if="form.type === 'http'">
          <n-grid :x-gap="12" :cols="4">
            <n-gi>
              <n-form-item :label="t('media.method')">
                <n-select v-model:value="form.method" :options="methodOptions" />
              </n-form-item>
            </n-gi>
            <n-gi :span="3">
              <n-form-item :label="t('media.url')">
                <n-input v-model:value="form.url" :placeholder="t('mediaMgmt.httpUrlPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-form-item :label="t('media.headers')">
            <KVEditor v-model:modelValue="form.headers" :key-placeholder="t('media.headerName')" :value-placeholder="t('media.headerValue')" :add-label="t('media.addHeader')" />
          </n-form-item>
          <n-form-item :label="t('media.body')">
            <n-input v-model:value="form.body" type="textarea" :rows="4"
              :placeholder="t('mediaMgmt.httpBodyPlaceholder')"
              style="font-family: var(--sre-font-mono); font-size: 12px" />
          </n-form-item>
        </template>

        <template v-if="form.type === 'script'">
          <n-form-item :label="t('media.path')">
            <n-input v-model:value="form.path" :placeholder="t('mediaMgmt.scriptPathPlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('media.args')">
            <n-input v-model:value="form.args" :placeholder="t('mediaMgmt.scriptArgsPlaceholder')" />
          </n-form-item>
        </template>

        <!-- dingtalk_webhook / wecom_webhook / slack_webhook / discord_webhook / feishu_webhook / feishu_card -->
        <template v-if="['dingtalk_webhook','wecom_webhook','slack_webhook','discord_webhook','feishu_webhook','feishu_card'].includes(form.type)">
          <n-form-item :label="t('media.webhookUrl')" required>
            <n-input v-model:value="form.webhook_url" :placeholder="t('mediaMgmt.webhookUrlPlaceholder')" />
          </n-form-item>
        </template>

        <!-- telegram_bot -->
        <template v-if="form.type === 'telegram_bot'">
          <n-form-item :label="t('media.field.botToken')" required>
            <n-input v-model:value="form.bot_token" type="password" show-password-on="click" :placeholder="t('mediaMgmt.botTokenPlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('media.field.chatId')" required>
            <n-input v-model:value="form.chat_id" :placeholder="t('mediaMgmt.chatIdPlaceholder')" />
          </n-form-item>
        </template>

        <!-- feishu_app -->
        <template v-if="form.type === 'feishu_app'">
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.appId')" required>
                <n-input v-model:value="form.app_id" :placeholder="t('mediaMgmt.appIdPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.appSecret')" required>
                <n-input v-model:value="form.app_secret" type="password" show-password-on="click" :placeholder="t('mediaMgmt.appSecretPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.receiveId')" required>
                <n-input v-model:value="form.receive_id" :placeholder="t('mediaMgmt.receiveIdPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.receiveIdType')">
                <n-select v-model:value="form.receive_id_type" :options="receiveIdTypeOptions" />
              </n-form-item>
            </n-gi>
          </n-grid>
        </template>

        <!-- wecom_app -->
        <template v-if="form.type === 'wecom_app'">
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.corpId')" required>
                <n-input v-model:value="form.corp_id" :placeholder="t('mediaMgmt.corpIdPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.corpSecret')" required>
                <n-input v-model:value="form.corp_secret" type="password" show-password-on="click" :placeholder="t('mediaMgmt.corpSecretPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.agentId')" required>
                <n-input v-model:value="form.agent_id" :placeholder="t('mediaMgmt.agentIdPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.toUser')">
                <n-input v-model:value="form.to_user" :placeholder="t('mediaMgmt.toUserPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
        </template>

        <!-- flashduty -->
        <template v-if="form.type === 'flashduty'">
          <n-form-item :label="t('media.field.integrationUrl')" required>
            <n-input v-model:value="form.integration_url" :placeholder="t('mediaMgmt.integrationUrlPlaceholder')" />
          </n-form-item>
        </template>

        <!-- pagerduty -->
        <template v-if="form.type === 'pagerduty'">
          <n-form-item :label="t('media.field.routingKey')" required>
            <n-input v-model:value="form.routing_key" :placeholder="t('mediaMgmt.routingKeyPlaceholder')" />
          </n-form-item>
        </template>

        <!-- tencent_sms -->
        <template v-if="form.type === 'tencent_sms'">
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.secretId')" required>
                <n-input v-model:value="form.secret_id" :placeholder="t('mediaMgmt.secretIdPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.secretKey')" required>
                <n-input v-model:value="form.secret_key" type="password" show-password-on="click" :placeholder="t('mediaMgmt.secretKeyPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.sdkAppId')">
                <n-input v-model:value="form.sdk_app_id" :placeholder="t('mediaMgmt.sdkAppIdPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.templateId')">
                <n-input v-model:value="form.template_id" :placeholder="t('mediaMgmt.templateIdPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.signName')">
                <n-input v-model:value="form.sign_name" :placeholder="t('mediaMgmt.signNamePlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.phoneNumbers')" required>
                <n-input v-model:value="form.phone_numbers" :placeholder="t('mediaMgmt.phoneNumbersPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
        </template>

        <!-- aliyun_sms -->
        <template v-if="form.type === 'aliyun_sms'">
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.accessKeyId')" required>
                <n-input v-model:value="form.access_key_id" :placeholder="t('mediaMgmt.accessKeyIdPlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.accessKeySecret')" required>
                <n-input v-model:value="form.access_key_secret" type="password" show-password-on="click" :placeholder="t('mediaMgmt.accessKeySecretPlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-grid :x-gap="12" :cols="2">
            <n-gi>
              <n-form-item :label="t('media.field.signName')">
                <n-input v-model:value="form.sign_name" :placeholder="t('mediaMgmt.signNamePlaceholder')" />
              </n-form-item>
            </n-gi>
            <n-gi>
              <n-form-item :label="t('media.field.templateCode')">
                <n-input v-model:value="form.template_code" :placeholder="t('mediaMgmt.templateCodePlaceholder')" />
              </n-form-item>
            </n-gi>
          </n-grid>
          <n-form-item :label="t('media.field.phoneNumbers')" required>
            <n-input v-model:value="form.phone_numbers" :placeholder="t('mediaMgmt.phoneNumbersPlaceholder')" />
          </n-form-item>
        </template>

        <n-divider style="margin: 12px 0" />

        <n-form-item :label="t('media.variables')">
          <n-input v-model:value="form.variables" type="textarea" :rows="3"
            :placeholder="t('media.variablesHint')"
            style="font-family: var(--sre-font-mono); font-size: 12px" />
        </n-form-item>

        <n-form-item :label="t('common.enabled')">
          <n-switch v-model:value="form.is_enabled" />
        </n-form-item>
      </n-form>

      <template #action>
        <n-space justify="end">
          <n-button @click="showModal = false">{{ t('common.cancel') }}</n-button>
          <n-button type="primary" :loading="saving" @click="handleSave">
            {{ editingId ? t('common.update') : t('common.create') }}
          </n-button>
        </n-space>
      </template>
    </n-modal>
  </div>
</template>

<style scoped>
.media-page { font-family: var(--sre-font-sans); max-width: 1400px; }

.sub-header {
  display: flex; align-items: flex-end; justify-content: space-between;
  padding-bottom: 14px; border-bottom: 1px solid var(--sre-hairline, rgba(255,255,255,0.06));
  margin-bottom: 14px;
}
.sub-title { font: 600 18px/1.2 var(--sre-font-sans), sans-serif; margin: 0; letter-spacing: -0.01em; }
.sub-sub { font-size: 12px; color: var(--sre-text-secondary); margin: 4px 0 0; }

.toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }
.count { font-size: 12px; color: var(--sre-text-secondary); margin-left: auto; font-variant-numeric: tabular-nums; }

.loading, .empty { padding: 60px 20px; text-align: center; color: var(--sre-text-secondary); }
.empty { display: flex; flex-direction: column; gap: 12px; align-items: center; }
.empty-text { font-size: 13px; }

.row-list { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 6px; }

.row-l1 { display: flex; align-items: center; gap: 10px; }
.type-icon {
  width: 28px; height: 28px; display: inline-flex; align-items: center; justify-content: center;
  border-radius: 6px; background: var(--sre-bg-elevated);
}
.type-icon[data-type="lark_webhook"]     { color: var(--sre-info); background: var(--sre-info-soft); }
.type-icon[data-type="email"]            { color: var(--sre-text-secondary); background: var(--sre-bg-elevated); }
.type-icon[data-type="http"]             { color: var(--sre-success); background: var(--sre-success-soft); }
.type-icon[data-type="script"]           { color: var(--sre-warning); background: var(--sre-warning-soft); }
.type-icon[data-type="dingtalk_webhook"] { color: #0089ff; background: rgba(0,137,255,0.1); }
.type-icon[data-type="wecom_webhook"]    { color: #07c160; background: rgba(7,193,96,0.1); }
.type-icon[data-type="slack_webhook"]    { color: #4a154b; background: rgba(74,21,75,0.1); }
.type-icon[data-type="discord_webhook"]  { color: #5865f2; background: rgba(88,101,242,0.1); }
.type-icon[data-type="telegram_bot"]     { color: #0088cc; background: rgba(0,136,204,0.1); }
.type-icon[data-type="feishu_webhook"]   { color: var(--sre-info); background: var(--sre-info-soft); }
.type-icon[data-type="feishu_card"]      { color: var(--sre-info); background: var(--sre-info-soft); }
.type-icon[data-type="feishu_app"]       { color: var(--sre-info); background: var(--sre-info-soft); }
.type-icon[data-type="wecom_app"]        { color: #07c160; background: rgba(7,193,96,0.1); }
.type-icon[data-type="flashduty"]        { color: var(--sre-warning); background: var(--sre-warning-soft); }
.type-icon[data-type="pagerduty"]        { color: #06ac38; background: rgba(6,172,56,0.1); }
.type-icon[data-type="tencent_sms"]      { color: #006eff; background: rgba(0,110,255,0.1); }
.type-icon[data-type="aliyun_sms"]       { color: #ff6a00; background: rgba(255,106,0,0.1); }

.row-name { font: 600 14px/1.3 var(--sre-font-sans), sans-serif; letter-spacing: -0.005em; }

.type-chip {
  font: 500 10px/1 var(--sre-font-mono), monospace; text-transform: uppercase;
  padding: 3px 6px; border-radius: 4px; letter-spacing: .04em;
  background: var(--sre-bg-elevated); color: var(--sre-text-secondary);
}
.type-chip[data-type="lark_webhook"]     { background: var(--sre-info-soft); color: var(--sre-info); }
.type-chip[data-type="email"]            { background: var(--sre-bg-elevated); color: var(--sre-text-secondary); }
.type-chip[data-type="http"]             { background: var(--sre-success-soft); color: var(--sre-success); }
.type-chip[data-type="script"]           { background: var(--sre-warning-soft); color: var(--sre-warning); }
.type-chip[data-type="dingtalk_webhook"] { background: rgba(0,137,255,0.1); color: #0089ff; }
.type-chip[data-type="wecom_webhook"]    { background: rgba(7,193,96,0.1); color: #07c160; }
.type-chip[data-type="slack_webhook"]    { background: rgba(74,21,75,0.1); color: #4a154b; }
.type-chip[data-type="discord_webhook"]  { background: rgba(88,101,242,0.1); color: #5865f2; }
.type-chip[data-type="telegram_bot"]     { background: rgba(0,136,204,0.1); color: #0088cc; }
.type-chip[data-type="feishu_webhook"]   { background: var(--sre-info-soft); color: var(--sre-info); }
.type-chip[data-type="feishu_card"]      { background: var(--sre-info-soft); color: var(--sre-info); }
.type-chip[data-type="feishu_app"]       { background: var(--sre-info-soft); color: var(--sre-info); }
.type-chip[data-type="wecom_app"]        { background: rgba(7,193,96,0.1); color: #07c160; }
.type-chip[data-type="flashduty"]        { background: var(--sre-warning-soft); color: var(--sre-warning); }
.type-chip[data-type="pagerduty"]        { background: rgba(6,172,56,0.1); color: #06ac38; }
.type-chip[data-type="tencent_sms"]      { background: rgba(0,110,255,0.1); color: #006eff; }
.type-chip[data-type="aliyun_sms"]       { background: rgba(255,106,0,0.1); color: #ff6a00; }

.builtin-chip {
  font: 500 10px/1 var(--sre-font-mono), monospace; padding: 3px 6px; border-radius: 4px;
  background: var(--sre-info-soft); color: var(--sre-info); letter-spacing: .04em;
}
.status-text { font-size: 11px; color: var(--sre-success); }
.status-text.off { color: var(--sre-text-secondary); }

.row-actions { margin-left: auto; display: flex; align-items: center; gap: 4px; }

.row-l2 { padding-left: 38px; }
.target {
  font: 12px/1.4 var(--sre-font-mono), monospace;
  color: var(--sre-text-secondary);
  font-variant-numeric: tabular-nums;
  word-break: break-all;
}
.row-l3 { padding-left: 38px; }
.meta { font-size: 12px; color: var(--sre-text-secondary); }

.media-modal { width: 600px; }
</style>
