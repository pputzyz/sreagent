<script setup lang="ts">
import { reactive, ref, shallowRef, computed, onMounted, h } from 'vue'
import { useMessage, NDropdown } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { subscribeRuleApi, notifyRuleApi, userApi, teamApi } from '@/api'
import type { SubscribeRule, NotifyRule, User, Team } from '@/types'
import { AddOutline, SearchOutline, NotificationsOutline } from '@vicons/ionicons5'
import LabelMatcherEditor from '@/components/common/LabelMatcherEditor.vue'
import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const subscriptions = shallowRef<SubscribeRule[]>([])
const showModal = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)
const search = ref('')

const notifyRules = shallowRef<NotifyRule[]>([])
const users = shallowRef<User[]>([])
const teams = shallowRef<Team[]>([])

const form = reactive({
  name: '',
  description: '',
  match_labels: [] as LabelMatcher[],
  severities: [] as string[],
  notify_rule_id: null as number | null,
  subscriber_type: 'user' as 'user' | 'team',
  user_id: null as number | null,
  team_id: null as number | null,
  is_enabled: true,
})

const severityOptions = computed(() => [
  { label: t('alert.critical'), value: 'critical' },
  { label: t('alert.warning'), value: 'warning' },
  { label: t('alert.info'), value: 'info' },
])

const notifyRuleOptions = computed(() => notifyRules.value.map(r => ({ label: r.name, value: r.id })))
const userOptions = computed(() => users.value.map(u => ({ label: u.display_name || u.username, value: u.id })))
const teamOptions = computed(() => teams.value.map(t => ({ label: t.name, value: t.id })))

function getNotifyRuleName(ruleId: number | null) {
  if (ruleId == null) return '—'
  return notifyRules.value.find(r => r.id === ruleId)?.name || `#${ruleId}`
}

function getSubscriberLabel(row: SubscribeRule): { type: 'user' | 'team' | null; name: string; initial: string } {
  if (row.user_id) {
    const u = users.value.find(x => x.id === row.user_id)
    const name = u ? (u.display_name || u.username) : `User #${row.user_id}`
    return { type: 'user', name, initial: (name[0] || '?').toUpperCase() }
  }
  if (row.team_id) {
    const tm = teams.value.find(x => x.id === row.team_id)
    const name = tm ? tm.name : `Team #${row.team_id}`
    return { type: 'team', name, initial: (name[0] || '?').toUpperCase() }
  }
  return { type: null, name: '—', initial: '?' }
}

function severityDot(s: string) {
  return s === 'critical' ? 'critical' : s === 'warning' ? 'warning' : 'info'
}

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  if (!q) return subscriptions.value
  return subscriptions.value.filter(s =>
    s.name.toLowerCase().includes(q) || (s.description || '').toLowerCase().includes(q),
  )
})

async function fetchData() {
  loading.value = true
  try {
    const { data } = await subscribeRuleApi.list({ page: 1, page_size: 100 })
    subscriptions.value = data.data.list || []
  } catch (err: any) { message.error(err.message) } finally { loading.value = false }
}

async function fetchRefData() {
  try {
    const [rulesRes, usersRes, teamsRes] = await Promise.all([
      notifyRuleApi.list({ page: 1, page_size: 100 }),
      userApi.list({ page: 1, page_size: 200 }),
      teamApi.list({ page: 1, page_size: 100 }),
    ])
    notifyRules.value = rulesRes.data.data.list || []
    users.value = usersRes.data.data.list || []
    teams.value = teamsRes.data.data.list || []
  } catch (err: any) { message.error(err.message) }
}

function resetForm() {
  Object.assign(form, {
    name: '', description: '', match_labels: [], severities: [],
    notify_rule_id: null, subscriber_type: 'user', user_id: null, team_id: null,
    is_enabled: true,
  })
}

function openCreate() {
  editingId.value = null
  modalTitle.value = t('subscribe.create')
  resetForm()
  showModal.value = true
}

function openEdit(row: SubscribeRule) {
  editingId.value = row.id
  modalTitle.value = t('subscribe.edit')
  Object.assign(form, {
    name: row.name,
    description: row.description,
    match_labels: Object.entries(row.match_labels || {}).map(([key, raw]) => {
      for (const op of ['!=', '=~', '!~'] as const) {
        if (raw.startsWith(op)) return { key, op, value: raw.slice(op.length) }
      }
      return { key, op: '=' as const, value: raw }
    }),
    severities: (row.severities || '').split(',').filter(Boolean),
    notify_rule_id: row.notify_rule_id,
    subscriber_type: row.team_id ? 'team' : 'user',
    user_id: row.user_id,
    team_id: row.team_id,
    is_enabled: row.is_enabled,
  })
  showModal.value = true
}

async function handleSave() {
  if (!form.name.trim()) { message.warning(t('subscribe.nameRequired')); return }
  saving.value = true
  try {
    const payload: Partial<SubscribeRule> = {
      name: form.name,
      description: form.description,
      match_labels: Object.fromEntries(form.match_labels.map(m => {
        const v = m.op === '=' ? m.value : `${m.op}${m.value}`
        return [m.key, v]
      })),
      severities: form.severities.join(','),
      notify_rule_id: form.notify_rule_id || null,
      user_id: form.subscriber_type === 'user' ? form.user_id : null,
      team_id: form.subscriber_type === 'team' ? form.team_id : null,
      is_enabled: form.is_enabled,
    }
    if (editingId.value) {
      await subscribeRuleApi.update(editingId.value, payload)
      message.success(t('subscribe.updated'))
    } else {
      await subscribeRuleApi.create(payload)
      message.success(t('subscribe.created'))
    }
    showModal.value = false
    fetchData()
  } catch (err: any) { message.error(err.message) } finally { saving.value = false }
}

async function handleDelete(id: number) {
  try {
    await subscribeRuleApi.delete(id)
    message.success(t('subscribe.deleted'))
    fetchData()
  } catch (err: any) { message.error(err.message) }
}

async function toggleEnabled(row: SubscribeRule, val: boolean) {
  try {
    await subscribeRuleApi.update(row.id, { ...row, is_enabled: val })
    subscriptions.value = subscriptions.value.map(r => r.id === row.id ? { ...r, is_enabled: val } : r)
  } catch (err: any) { message.error(err.message) }
}

function rowMenu(row: SubscribeRule) {
  return [
    { label: t('common.edit'), key: 'edit' },
    { type: 'divider', key: 'd1' },
    { label: t('common.delete'), key: 'delete', props: { style: 'color: var(--sre-danger, #ef4444)' } },
  ]
}
function onRowMenu(key: string, row: SubscribeRule) {
  if (key === 'edit') openEdit(row)
  else if (key === 'delete' && confirm(t('subscribe.deleteConfirm'))) handleDelete(row.id)
}
const RowMenu = (row: SubscribeRule) => h(NDropdown, {
  trigger: 'click', options: rowMenu(row),
  onSelect: (k: string) => onRowMenu(k, row),
}, { default: () => h('button', { class: 'sre-icon-btn' }, h('span', { class: 'sre-dots' })) })

onMounted(() => { fetchData(); fetchRefData() })
</script>

<template>
  <div class="sub-page">
    <header class="sub-header">
      <div>
        <h2 class="sub-title">{{ t('subscribe.title') }}</h2>
        <p class="sub-sub">{{ t('subscribe.subtitle') }}</p>
      </div>
      <n-button type="primary" size="small" @click="openCreate">
        <template #icon><n-icon :component="AddOutline" /></template>
        {{ t('subscribe.create') }}
      </n-button>
    </header>

    <div class="toolbar">
      <n-input v-model:value="search" size="small" :placeholder="t('common.search')" clearable style="width: 240px">
        <template #prefix><n-icon :component="SearchOutline" /></template>
      </n-input>
      <span class="count tnum">{{ filtered.length }} / {{ subscriptions.length }}</span>
    </div>

    <div v-if="loading" class="loading">{{ t('common.loading') }}…</div>

    <div v-else-if="filtered.length === 0" class="empty">
      <n-icon :component="NotificationsOutline" size="36" />
      <div class="empty-text">{{ t('subscribe.noData') }}</div>
      <n-button type="primary" size="small" @click="openCreate">{{ t('subscribe.create') }}</n-button>
    </div>

    <ul v-else class="row-list sre-stagger">
      <li v-for="s in filtered" :key="s.id" class="sre-notify-card sre-lift">
        <div class="row-l1">
          <span class="sre-dot" :class="s.is_enabled ? 'on' : 'off'"></span>
          <span class="row-name">{{ s.name }}</span>
          <span class="subscriber" :data-type="getSubscriberLabel(s).type">
            <span class="avatar">{{ getSubscriberLabel(s).initial }}</span>
            <span class="sub-name">{{ getSubscriberLabel(s).name }}</span>
            <span class="sub-kind">{{ getSubscriberLabel(s).type === 'team' ? t('subscribe.team') : t('subscribe.user') }}</span>
          </span>
          <div class="row-actions">
            <n-switch :value="s.is_enabled" size="small" @update:value="(v: boolean) => toggleEnabled(s, v)" />
            <component :is="RowMenu(s)" />
          </div>
        </div>

        <div class="row-l2">
          <template v-for="(v, k) in (s.match_labels || {})" :key="k">
            <code class="label-chip">{{ k }}={{ v }}</code>
          </template>
          <span v-if="!Object.keys(s.match_labels || {}).length" class="muted">—</span>
          <span class="severities">
            <span v-for="sv in (s.severities || '').split(',').filter(Boolean)" :key="sv"
              class="sev-chip" :data-sev="severityDot(sv)">{{ sv }}</span>
          </span>
        </div>

        <div class="row-l3">
          <span class="meta">→ {{ getNotifyRuleName(s.notify_rule_id) }}</span>
          <span v-if="s.description" class="sre-meta-divider">·</span>
          <span v-if="s.description" class="meta">{{ s.description }}</span>
        </div>
      </li>
    </ul>

    <n-modal v-model:show="showModal" preset="card" :title="modalTitle" :bordered="false" class="sub-modal">
      <n-form label-placement="top">
        <n-form-item :label="t('subscribe.name')" required>
          <n-input v-model:value="form.name" placeholder="e.g. My Critical Alert Sub" />
        </n-form-item>

        <n-form-item :label="t('subscribe.description')">
          <n-input v-model:value="form.description" :placeholder="t('subscribe.description')" />
        </n-form-item>

        <n-form-item :label="t('subscribe.matchLabels')">
          <LabelMatcherEditor v-model:modelValue="form.match_labels" :add-label="t('subscribe.addLabel')" />
        </n-form-item>

        <n-form-item :label="t('subscribe.severities')">
          <n-select v-model:value="form.severities" :options="severityOptions" multiple
            :placeholder="t('common.selectSeverities')" />
        </n-form-item>

        <n-form-item :label="t('subscribe.notifyRule')">
          <n-select v-model:value="form.notify_rule_id" :options="notifyRuleOptions"
            :placeholder="t('subscribe.selectNotifyRule')" clearable />
        </n-form-item>

        <n-form-item :label="t('subscribe.subscriberType')">
          <n-radio-group v-model:value="form.subscriber_type">
            <n-radio-button value="user">{{ t('subscribe.user') }}</n-radio-button>
            <n-radio-button value="team">{{ t('subscribe.team') }}</n-radio-button>
          </n-radio-group>
        </n-form-item>

        <n-form-item v-if="form.subscriber_type === 'user'" :label="t('subscribe.user')">
          <n-select v-model:value="form.user_id" :options="userOptions"
            :placeholder="t('subscribe.selectUser')" filterable clearable />
        </n-form-item>

        <n-form-item v-if="form.subscriber_type === 'team'" :label="t('subscribe.team')">
          <n-select v-model:value="form.team_id" :options="teamOptions"
            :placeholder="t('subscribe.selectTeam')" filterable clearable />
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
.sub-page { font-family: var(--sre-font-sans); max-width: 1400px; }

.sub-header {
  display: flex; align-items: flex-end; justify-content: space-between;
  padding-bottom: 14px; border-bottom: 1px solid var(--sre-hairline, rgba(255,255,255,0.06));
  margin-bottom: 14px;
}
.sub-title { font: 600 18px/1.2 var(--sre-font-sans), sans-serif; margin: 0; letter-spacing: -0.01em; }
.sub-sub { font-size: 12px; color: var(--sre-text-secondary, #888); margin: 4px 0 0; }

.toolbar { display: flex; gap: 8px; align-items: center; margin-bottom: 12px; }
.count { font-size: 12px; color: var(--sre-text-secondary, #888); margin-left: auto; font-variant-numeric: tabular-nums; }

.loading, .empty { padding: 60px 20px; text-align: center; color: var(--sre-text-secondary, #888); }
.empty { display: flex; flex-direction: column; gap: 12px; align-items: center; }
.empty-text { font-size: 13px; }

.row-list { list-style: none; padding: 0; margin: 0; display: flex; flex-direction: column; gap: 6px; }

.row-l1 { display: flex; align-items: center; gap: 10px; }
.row-name { font: 600 14px/1.3 var(--sre-font-sans), sans-serif; letter-spacing: -0.005em; }

.subscriber { display: inline-flex; align-items: center; gap: 6px; padding: 3px 8px 3px 3px;
  border-radius: 999px; background: rgba(255,255,255,0.04); border: 1px solid var(--sre-hairline, rgba(255,255,255,0.06)); }
.avatar {
  width: 20px; height: 20px; border-radius: 50%; display: inline-flex; align-items: center; justify-content: center;
  font: 600 10px/1 var(--sre-font-sans), sans-serif; background: rgba(129,140,248,0.18); color: #a5b4fc;
}
.subscriber[data-type="team"] .avatar { background: rgba(34,197,94,0.18); color: #86efac; }
.sub-name { font-size: 12px; }
.sub-kind { font: 500 10px/1 var(--sre-font-mono), monospace; color: var(--sre-text-secondary, #888); text-transform: uppercase; letter-spacing: .04em; }

.row-actions { margin-left: auto; display: flex; align-items: center; gap: 6px; }

.row-l2 { padding-left: 18px; display: flex; flex-wrap: wrap; gap: 4px; align-items: center; }
.label-chip {
  font: 11px/1 var(--sre-font-mono), monospace; padding: 3px 6px; border-radius: 4px;
  background: rgba(255,255,255,0.05); color: var(--sre-text-secondary, #aaa);
}
.severities { display: inline-flex; gap: 4px; margin-left: 4px; }
.sev-chip {
  font: 500 10px/1 var(--sre-font-mono), monospace; padding: 3px 6px; border-radius: 4px;
  text-transform: uppercase; letter-spacing: .04em;
}
.sev-chip[data-sev="critical"] { background: rgba(239,68,68,0.14); color: #fca5a5; }
.sev-chip[data-sev="warning"]  { background: rgba(245,158,11,0.14); color: #fcd34d; }
.sev-chip[data-sev="info"]     { background: rgba(56,189,248,0.14); color: #7dd3fc; }
.muted { color: var(--sre-text-secondary, #666); font-size: 12px; }

.row-l3 { padding-left: 18px; display: flex; gap: 6px; align-items: center; }
.meta { font-size: 12px; color: var(--sre-text-secondary, #888); }

.sub-modal { width: 600px; }
</style>
