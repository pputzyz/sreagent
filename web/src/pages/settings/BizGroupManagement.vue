<script setup lang="ts">
import { ref, shallowRef, reactive, computed, onMounted, defineComponent, h } from 'vue'
import { useMessage, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { bizGroupApi } from '@/api'
import type { User, BizGroup } from '@/types'
import { kvArrayToRecord } from '@/utils/format'
import {
  AddOutline,
  FolderOutline,
  DocumentOutline,
  ChevronDownOutline,
  ChevronForwardOutline,
  PeopleOutline,
  TrashOutline,
  CreateOutline,
} from '@vicons/ionicons5'
import KVEditor from '@/components/common/KVEditor.vue'
import LabelMatcherEditor from '@/components/common/LabelMatcherEditor.vue'
import type { LabelMatcher } from '@/components/common/LabelMatcherEditor.vue'

interface TreeNode {
  id: number | string
  name: string
  fullPath: string
  group: BizGroup | null
  children: TreeNode[]
}

const props = defineProps<{ allUsers: User[] }>()

const message = useMessage()
const { t } = useI18n()

const loading = ref(false)
const list = shallowRef<BizGroup[]>([])
const selected = ref<BizGroup | null>(null)
const members = ref<any[]>([])
const membersLoading = ref(false)
const expanded = ref<Set<string | number>>(new Set())

const showModal = ref(false)
const modalTitle = ref('')
const editingId = ref<number | null>(null)
const saving = ref(false)
const showAddMemberModal = ref(false)
const selectedMemberUserId = ref<number | null>(null)
const selectedMemberRole = ref<string>('member')

const form = reactive({
  name: '',
  description: '',
  labels: [] as { key: string; value: string }[],
  match_labels: [] as LabelMatcher[],
})

const memberRoleOptions = [
  { label: t('settings.admin'), value: 'admin' },
  { label: t('settings.member'), value: 'member' },
]

const allUserOptions = computed(() =>
  props.allUsers.map(u => ({ label: u.display_name || u.username, value: u.id }))
)

/** Build hierarchical tree from flat list using "/" path notation. */
const tree = computed<TreeNode[]>(() => {
  const root: TreeNode[] = []
  const map = new Map<string, TreeNode>()
  const sorted = [...list.value].sort((a, b) => a.name.localeCompare(b.name))

  for (const g of sorted) {
    const parts = g.name.split('/')
    let path = ''
    let parentList = root

    for (let i = 0; i < parts.length; i++) {
      path = path ? `${path}/${parts[i]}` : parts[i]
      const isLeaf = i === parts.length - 1
      let node = map.get(path)

      if (!node) {
        node = {
          id: isLeaf ? g.id : `__path__${path}`,
          name: parts[i],
          fullPath: path,
          group: isLeaf ? g : null,
          children: [],
        }
        parentList.push(node)
        map.set(path, node)
      } else if (isLeaf) {
        node.id = g.id
        node.group = g
      }
      parentList = node.children
    }
  }
  return root
})

const totalGroups = computed(() => list.value.length)
const selectedDescendants = computed(() => {
  if (!selected.value) return 0
  const prefix = selected.value.name + '/'
  return list.value.filter(g => g.name.startsWith(prefix)).length
})
const selectedParent = computed(() => {
  if (!selected.value) return null
  const idx = selected.value.name.lastIndexOf('/')
  return idx > 0 ? selected.value.name.slice(0, idx) : null
})

function relTime(iso?: string): string {
  if (!iso) return '—'
  const d = Date.now() - new Date(iso).getTime()
  if (d < 60_000) return 'just now'
  if (d < 3600_000) return `${Math.floor(d / 60_000)}m ago`
  if (d < 86_400_000) return `${Math.floor(d / 3600_000)}h ago`
  if (d < 30 * 86_400_000) return `${Math.floor(d / 86_400_000)}d ago`
  return new Date(iso).toLocaleDateString()
}

async function fetchList() {
  loading.value = true
  try {
    const { data } = await bizGroupApi.list({ page: 1, page_size: 500 })
    list.value = data.data.list || []
    if (expanded.value.size === 0) {
      const next = new Set<string | number>()
      tree.value.forEach(n => next.add(n.fullPath))
      expanded.value = next
    }
  } catch (err: any) {
    message.error(err.message)
  } finally {
    loading.value = false
  }
}

function toggleExpand(node: TreeNode, e: Event) {
  e.stopPropagation()
  const next = new Set(expanded.value)
  if (next.has(node.fullPath)) next.delete(node.fullPath)
  else next.add(node.fullPath)
  expanded.value = next
}

function selectNode(node: TreeNode) {
  if (!node.group) {
    const next = new Set(expanded.value)
    if (next.has(node.fullPath)) next.delete(node.fullPath)
    else next.add(node.fullPath)
    expanded.value = next
    return
  }
  selected.value = node.group
  fetchMembers(node.group.id)
}

async function fetchMembers(groupId: number) {
  membersLoading.value = true
  try {
    const { data } = await bizGroupApi.listMembers(groupId)
    members.value = data.data || []
  } catch (err: any) {
    message.error(err.message)
    members.value = []
  } finally {
    membersLoading.value = false
  }
}

function recordToMatchers(record: Record<string, string> | undefined): LabelMatcher[] {
  return Object.entries(record || {}).map(([key, raw]) => {
    for (const op of ['!=', '=~', '!~'] as const) {
      if (raw.startsWith(op)) return { key, op, value: raw.slice(op.length) }
    }
    return { key, op: '=' as const, value: raw }
  })
}

function matchersToRecord(matchers: LabelMatcher[]): Record<string, string> {
  return Object.fromEntries(matchers.map(m => {
    const v = m.op === '=' ? m.value : `${m.op}${m.value}`
    return [m.key, v]
  }))
}

function openCreate() {
  editingId.value = null
  modalTitle.value = t('bizGroup.create')
  Object.assign(form, { name: '', description: '', labels: [], match_labels: [] })
  showModal.value = true
}

function openEdit() {
  if (!selected.value) return
  const g = selected.value
  editingId.value = g.id
  modalTitle.value = t('bizGroup.edit')
  Object.assign(form, {
    name: g.name,
    description: g.description,
    labels: Object.entries(g.labels || {}).map(([key, value]) => ({ key, value })),
    match_labels: recordToMatchers(g.match_labels),
  })
  showModal.value = true
}

async function handleSave() {
  if (!form.name.trim()) {
    message.warning(t('settings.nameRequired'))
    return
  }
  saving.value = true
  try {
    const payload = {
      name: form.name,
      description: form.description,
      labels: kvArrayToRecord(form.labels),
      match_labels: matchersToRecord(form.match_labels),
    }
    if (editingId.value) {
      await bizGroupApi.update(editingId.value, payload)
      message.success(t('bizGroup.updated'))
    } else {
      await bizGroupApi.create(payload)
      message.success(t('bizGroup.created'))
    }
    showModal.value = false
    await fetchList()
  } catch (err: any) {
    message.error(err.message)
  } finally {
    saving.value = false
  }
}

async function handleDelete() {
  if (!selected.value) return
  try {
    await bizGroupApi.delete(selected.value.id)
    message.success(t('bizGroup.deleted'))
    selected.value = null
    members.value = []
    await fetchList()
  } catch (err: any) {
    message.error(err.message)
  }
}

function openAddMember() {
  selectedMemberUserId.value = null
  selectedMemberRole.value = 'member'
  showAddMemberModal.value = true
}

async function handleAddMember() {
  if (!selected.value || !selectedMemberUserId.value) return
  try {
    await bizGroupApi.addMember(selected.value.id, {
      user_id: selectedMemberUserId.value,
      role: selectedMemberRole.value,
    })
    message.success(t('settings.memberAdded'))
    showAddMemberModal.value = false
    fetchMembers(selected.value.id)
  } catch (err: any) {
    message.error(err.message)
  }
}

async function handleRemoveMember(userId: number) {
  if (!selected.value) return
  try {
    await bizGroupApi.removeMember(selected.value.id, userId)
    message.success(t('settings.memberRemoved'))
    fetchMembers(selected.value.id)
  } catch (err: any) {
    message.error(err.message)
  }
}

/** Recursive tree-node renderer. Defined inline to keep this a single SFC. */
const TreeNodeRow: any = defineComponent({
  name: 'TreeNodeRow',
  props: {
    node: { type: Object as () => TreeNode, required: true },
    depth: { type: Number, required: true },
    expandedSet: { type: Object as () => Set<string | number>, required: true },
    selectedId: { type: [Number, String, null] as any, default: null },
    onPick: { type: Function, required: true },
    onToggleNode: { type: Function, required: true },
  },
  setup(p) {
    return () => {
      const node = p.node
      const hasChildren = node.children && node.children.length > 0
      const isOpen = p.expandedSet.has(node.fullPath)
      const isActive = !!node.group && p.selectedId === node.group.id
      const isPathOnly = !node.group

      return h('div', { class: 'tn-wrap' }, [
        h(
          'div',
          {
            class: ['tn-row', { 'tn-active': isActive, 'tn-path': isPathOnly }],
            style: { paddingLeft: `${p.depth * 14 + 8}px` },
            onClick: () => p.onPick(node),
          },
          [
            h(
              'span',
              {
                class: ['tn-caret', { 'tn-caret-empty': !hasChildren }],
                onClick: (e: Event) => hasChildren && p.onToggleNode(node, e),
              },
              hasChildren
                ? [h(NIcon, { size: 11, component: isOpen ? ChevronDownOutline : ChevronForwardOutline })]
                : []
            ),
            h(NIcon, {
              class: 'tn-icon',
              size: 13,
              component: hasChildren ? FolderOutline : DocumentOutline,
            }),
            h('span', { class: 'tn-name' }, node.name),
          ]
        ),
        hasChildren && isOpen
          ? h(
              'div',
              { class: 'tn-children' },
              node.children.map((c: TreeNode) =>
                h(TreeNodeRow, {
                  node: c,
                  depth: p.depth + 1,
                  expandedSet: p.expandedSet,
                  selectedId: p.selectedId,
                  onPick: p.onPick,
                  onToggleNode: p.onToggleNode,
                })
              )
            )
          : null,
      ])
    }
  },
})

onMounted(fetchList)
</script>

<template>
  <div class="bg-page sre-stagger">
    <header class="page-header">
      <div>
        <h2 class="page-title">{{ t('bizGroup.bizGroupsTitle') }}</h2>
        <p class="page-subtitle">
          {{ t('bizGroup.bizGroupsSubtitle') }}
          <span class="sre-meta-divider" />
          <span class="tnum">{{ totalGroups }}</span> {{ t('bizGroup.groupsCount') }}
        </p>
      </div>
      <NButton type="primary" size="small" @click="openCreate">
        <template #icon><NIcon :component="AddOutline" /></template>
        {{ t('bizGroup.newGroup') }}
      </NButton>
    </header>

    <div class="bg-layout">
      <!-- Tree -->
      <aside class="bg-tree sre-lift">
        <div class="bg-tree-head">
          <span class="sre-label-eyebrow">{{ t('bizGroup.hierarchy') }}</span>
        </div>
        <NSpin :show="loading">
          <div v-if="tree.length === 0 && !loading" class="bg-tree-empty">
            <NIcon :component="FolderOutline" :size="22" />
            <span>{{ t('bizGroup.noData') }}</span>
          </div>
          <div v-else class="bg-tree-body">
            <TreeNodeRow
              v-for="node in tree"
              :key="node.fullPath"
              :node="node"
              :depth="0"
              :expanded-set="expanded"
              :selected-id="selected?.id ?? null"
              :on-pick="selectNode"
              :on-toggle-node="toggleExpand"
            />
          </div>
        </NSpin>
      </aside>

      <!-- Detail -->
      <section class="bg-detail">
        <div v-if="!selected" class="bg-empty sre-lift">
          <NIcon :component="FolderOutline" :size="36" />
          <p class="bg-empty-title">{{ t('bizGroup.selectGroup') }}</p>
          <p class="bg-empty-sub">{{ t('bizGroup.pickGroupHint') }}</p>
        </div>

        <template v-else>
          <div class="bg-card sre-lift">
            <div class="bg-card-head">
              <div class="bg-title-block">
                <code class="bg-path tnum">{{ selected.name }}</code>
                <h3 class="bg-name">{{ selected.name.split('/').pop() }}</h3>
                <p class="bg-desc">{{ selected.description || '—' }}</p>
              </div>
              <div class="bg-actions">
                <NButton size="small" quaternary @click="openEdit">
                  <template #icon><NIcon :component="CreateOutline" /></template>
                  {{ t('common.edit') }}
                </NButton>
                <NPopconfirm @positive-click="handleDelete">
                  <template #trigger>
                    <NButton size="small" quaternary type="error">
                      <template #icon><NIcon :component="TrashOutline" /></template>
                      {{ t('common.delete') }}
                    </NButton>
                  </template>
                  {{ t('bizGroup.deleteConfirm') }}
                </NPopconfirm>
              </div>
            </div>

            <dl class="bg-meta">
              <div class="bg-meta-cell">
                <dt class="sre-label-eyebrow">{{ t('bizGroup.parent') }}</dt>
                <dd>{{ selectedParent || '—' }}</dd>
              </div>
              <div class="bg-meta-cell">
                <dt class="sre-label-eyebrow">{{ t('bizGroup.path') }}</dt>
                <dd class="mono">/{{ selected.name }}</dd>
              </div>
              <div class="bg-meta-cell">
                <dt class="sre-label-eyebrow">{{ t('bizGroup.children') }}</dt>
                <dd class="tnum">{{ selectedDescendants }}</dd>
              </div>
              <div class="bg-meta-cell">
                <dt class="sre-label-eyebrow">{{ t('bizGroup.created') }}</dt>
                <dd>{{ relTime(selected.created_at) }}</dd>
              </div>
            </dl>

            <div
              v-if="Object.keys(selected.labels || {}).length || Object.keys(selected.match_labels || {}).length"
              class="bg-tags"
            >
              <div v-if="Object.keys(selected.labels || {}).length" class="bg-tags-row">
                <span class="sre-label-eyebrow">{{ t('bizGroup.labels') }}</span>
                <span v-for="(v, k) in selected.labels" :key="`l-${k}`" class="bg-chip">{{ k }}={{ v }}</span>
              </div>
              <div v-if="Object.keys(selected.match_labels || {}).length" class="bg-tags-row">
                <span class="sre-label-eyebrow">{{ t('bizGroup.match') }}</span>
                <span v-for="(v, k) in selected.match_labels" :key="`m-${k}`" class="bg-chip bg-chip-info">
                  {{ k }}{{ (v.startsWith('!=') || v.startsWith('=~') || v.startsWith('!~')) ? v : '=' + v }}
                </span>
              </div>
            </div>
          </div>

          <div class="bg-card sre-lift">
            <div class="bg-section-head">
              <div class="bg-section-title">
                <NIcon :component="PeopleOutline" :size="14" />
                <span class="sre-label-eyebrow">{{ t('bizGroup.members') }}</span>
                <span class="bg-count tnum">{{ members.length }}</span>
              </div>
              <NButton size="small" type="primary" tertiary @click="openAddMember">
                <template #icon><NIcon :component="AddOutline" /></template>
                {{ t('bizGroup.addMember') }}
              </NButton>
            </div>

            <NSpin :show="membersLoading">
              <div v-if="members.length === 0 && !membersLoading" class="bg-members-empty">
                <span>{{ t('settings.noMembers') }}</span>
              </div>
              <ul v-else class="bg-members">
                <li v-for="m in members" :key="m.id" class="sre-row-card bg-member">
                  <NAvatar :size="28" round>
                    {{ (m.display_name || m.username).charAt(0).toUpperCase() }}
                  </NAvatar>
                  <div class="bg-member-info">
                    <div class="bg-member-name">{{ m.display_name || m.username }}</div>
                    <div class="bg-member-meta">{{ m.email || m.username }}</div>
                  </div>
                  <span class="bg-role" :class="m.role === 'admin' ? 'bg-role-admin' : 'bg-role-member'">
                    <span class="sre-dot" />
                    {{ m.role || 'member' }}
                  </span>
                  <NPopconfirm @positive-click="handleRemoveMember(m.id)">
                    <template #trigger>
                      <NButton size="tiny" quaternary type="error">{{ t('common.remove') }}</NButton>
                    </template>
                    {{ t('settings.removeMemberConfirm') }}
                  </NPopconfirm>
                </li>
              </ul>
            </NSpin>
          </div>
        </template>
      </section>
    </div>

    <!-- Create/Edit modal -->
    <NModal v-model:show="showModal" preset="card" :title="modalTitle" style="width: 520px" :bordered="false">
      <NForm label-placement="top">
        <NFormItem :label="t('common.name')" required>
          <NInput v-model:value="form.name" :placeholder="t('bizGroup.namePlaceholder')" />
          <template #feedback>
            <span class="bg-hint">{{ t('bizGroup.nameHint') }}</span>
          </template>
        </NFormItem>
        <NFormItem :label="t('common.description')">
          <NInput v-model:value="form.description" type="textarea" :rows="2" />
        </NFormItem>
        <NFormItem :label="t('settings.labels')">
          <KVEditor v-model="form.labels" :add-label="t('settings.addTeamLabel')" />
        </NFormItem>
        <NFormItem :label="t('bizGroup.matchLabels')">
          <template #feedback>
            <span class="bg-hint">{{ t('bizGroup.matchLabelsDesc') }}</span>
          </template>
          <LabelMatcherEditor v-model:modelValue="form.match_labels" :add-label="t('bizGroup.addMatchLabel')" />
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

    <!-- Add member modal -->
    <NModal
      v-model:show="showAddMemberModal"
      preset="card"
      :title="t('bizGroup.addMember')"
      style="width: 420px"
      :bordered="false"
    >
      <NForm label-placement="top">
        <NFormItem :label="t('settings.user')" required>
          <NSelect
            v-model:value="selectedMemberUserId"
            :options="allUserOptions"
            :placeholder="t('settings.selectUser')"
            filterable
          />
        </NFormItem>
        <NFormItem :label="t('settings.role')">
          <NSelect v-model:value="selectedMemberRole" :options="memberRoleOptions" />
        </NFormItem>
      </NForm>
      <template #action>
        <NSpace justify="end">
          <NButton @click="showAddMemberModal = false">{{ t('common.cancel') }}</NButton>
          <NButton type="primary" :disabled="!selectedMemberUserId" @click="handleAddMember">
            {{ t('common.add') }}
          </NButton>
        </NSpace>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.bg-page {
  font-family: var(--sre-font-sans, var(--sre-font-sans), system-ui, sans-serif);
  display: flex;
  flex-direction: column;
  gap: 18px;
}

/* Header */
.page-header {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 16px;
  padding-bottom: 14px;
  border-bottom: var(--sre-hairline);
}
.page-title {
  margin: 0;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.01em;
  font-family: var(--sre-font-sans, 'Geist');
}
.page-subtitle {
  margin: 4px 0 0;
  font-size: 12.5px;
  color: var(--sre-text-muted);
  display: flex;
  align-items: center;
  flex-wrap: wrap;
}

/* Layout */
.bg-layout {
  display: grid;
  grid-template-columns: 300px 1fr;
  gap: 20px;
  align-items: start;
}

/* Tree */
.bg-tree {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 10px);
  padding: 14px 12px;
  max-height: 70vh;
  overflow-y: auto;
  position: sticky;
  top: 12px;
}
.bg-tree-head {
  padding: 0 4px 10px;
  border-bottom: var(--sre-hairline);
  margin-bottom: 8px;
}
.bg-tree-body {
  display: flex;
  flex-direction: column;
}
.bg-tree-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  padding: 32px 0;
  font-size: 12.5px;
  color: var(--sre-text-muted);
}
:deep(.tn-wrap) { display: flex; flex-direction: column; }
:deep(.tn-row) {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 5px 8px 5px 0;
  cursor: pointer;
  border-radius: var(--sre-radius-sm);
  font-size: 13px;
  user-select: none;
  transition: background 120ms ease, color 120ms ease;
  line-height: 1.3;
}
:deep(.tn-row:hover) { background: var(--sre-bg-hover); }
:deep(.tn-active) {
  background: var(--sre-primary-soft) !important;
  color: var(--sre-primary);
}
:deep(.tn-active .tn-icon) { color: var(--sre-primary); }
:deep(.tn-path) { color: var(--sre-text-muted); }
:deep(.tn-caret) {
  width: 14px;
  display: inline-flex;
  justify-content: center;
  align-items: center;
  color: var(--sre-text-muted);
  flex-shrink: 0;
}
:deep(.tn-caret-empty) { visibility: hidden; }
:deep(.tn-icon) { opacity: 0.65; flex-shrink: 0; }
:deep(.tn-name) {
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
  font-family: var(--sre-font-sans, 'Geist');
}

/* Detail */
.bg-detail {
  display: flex;
  flex-direction: column;
  gap: 16px;
  min-width: 0;
}
.bg-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  gap: 10px;
  padding: 80px 20px;
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 10px);
  color: var(--sre-text-muted);
}
.bg-empty-title { margin: 0; font-size: 14px; font-weight: 500; }
.bg-empty-sub { margin: 0; font-size: 12px; opacity: 0.7; }

.bg-card {
  background: var(--sre-bg-card);
  border: var(--sre-hairline);
  border-radius: var(--sre-radius-md, 10px);
  padding: 18px 20px;
}

.bg-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 14px;
  margin-bottom: 16px;
}
.bg-title-block { min-width: 0; flex: 1; }
.bg-path {
  font-size: 11px;
  font-family: var(--sre-font-mono, ui-monospace, 'SF Mono', monospace);
  color: var(--sre-text-muted);
  letter-spacing: 0.02em;
}
.bg-name {
  margin: 4px 0 4px;
  font-size: 18px;
  font-weight: 600;
  letter-spacing: -0.01em;
}
.bg-desc {
  margin: 0;
  font-size: 12.5px;
  color: var(--sre-text-muted);
  line-height: 1.5;
}
.bg-actions { display: flex; gap: 6px; flex-shrink: 0; }

.bg-meta {
  display: grid;
  grid-template-columns: repeat(auto-fit, minmax(140px, 1fr));
  gap: 12px 16px;
  margin: 0 0 14px;
  padding: 14px 0;
  border-top: var(--sre-hairline);
  border-bottom: var(--sre-hairline);
}
.bg-meta-cell { display: flex; flex-direction: column; gap: 4px; min-width: 0; }
.bg-meta-cell dd {
  margin: 0;
  font-size: 13px;
  font-weight: 500;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.bg-meta-cell dd.mono {
  font-family: var(--sre-font-mono, ui-monospace, 'SF Mono', monospace);
  font-size: 12px;
}

.bg-tags { display: flex; flex-direction: column; gap: 8px; }
.bg-tags-row { display: flex; align-items: center; gap: 8px; flex-wrap: wrap; }
.bg-chip {
  display: inline-flex;
  align-items: center;
  padding: 2px 8px;
  font-size: 11px;
  font-family: var(--sre-font-mono, ui-monospace, 'SF Mono', monospace);
  border-radius: 4px;
  background: var(--sre-bg-hover);
  border: var(--sre-hairline);
  color: var(--sre-text-muted);
}
.bg-chip-info {
  background: var(--sre-primary-soft);
  color: var(--sre-primary);
  border-color: var(--sre-primary-soft);
}

/* Members */
.bg-section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 14px;
}
.bg-section-title {
  display: flex;
  align-items: center;
  gap: 8px;
  color: var(--sre-text-muted);
}
.bg-count {
  font-size: 11px;
  font-weight: 500;
  padding: 1px 6px;
  border-radius: 999px;
  background: var(--sre-bg-hover);
  color: var(--sre-text-primary);
}
.bg-members {
  list-style: none;
  margin: 0;
  padding: 0;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.bg-member {
  display: flex;
  align-items: center;
  gap: 12px;
  padding: 10px 12px;
}
.bg-member-info { flex: 1; min-width: 0; }
.bg-member-name { font-size: 13.5px; font-weight: 500; line-height: 1.2; }
.bg-member-meta {
  font-size: 11.5px;
  color: var(--sre-text-muted);
  margin-top: 2px;
  font-family: var(--sre-font-mono, ui-monospace, monospace);
}
.bg-role {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-size: 11px;
  font-weight: 500;
  padding: 2px 8px;
  border-radius: 999px;
  text-transform: capitalize;
  border: var(--sre-hairline);
}
.bg-role-admin {
  background: var(--sre-warning-soft);
  color: var(--sre-warning);
  border-color: var(--sre-warning-soft);
}
.bg-role-admin .sre-dot { background: var(--sre-warning); }
.bg-role-member {
  background: var(--sre-bg-hover);
  color: var(--sre-text-muted);
}
.bg-role-member .sre-dot { background: var(--sre-text-tertiary); }
.bg-members-empty {
  padding: 24px 0;
  text-align: center;
  font-size: 12.5px;
  color: var(--sre-text-muted);
}

.bg-hint { font-size: 11px; opacity: 0.5; }

/* Responsive */
@media (max-width: 900px) {
  .bg-layout { grid-template-columns: 1fr; }
  .bg-tree { position: static; max-height: 320px; }
}
</style>
