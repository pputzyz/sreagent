<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import { useMessage, NButton, NIcon, NRadioGroup, NRadioButton, NSpin, NEmpty, NModal, NForm, NFormItem, NInput, NSelect, NDatePicker } from 'naive-ui'
import { AddOutline, CheckmarkOutline, TrashOutline, ListOutline } from '@vicons/ionicons5'
import { todoApi } from '@/api'
import type { TodoItem } from '@/api/center'
import { getErrorMessage, formatTime } from '@/utils/format'
import PageHeader from '@/components/common/PageHeader.vue'

const { t } = useI18n()
const message = useMessage()

const loading = ref(false)
const items = ref<TodoItem[]>([])
const total = ref(0)
const page = ref(1)
const pageSize = ref(20)
const statusFilter = ref<'all' | 'pending' | 'completed'>('all')

// Create modal
const showModal = ref(false)
const saving = ref(false)
const form = ref({ title: '', description: '', priority: 'medium' as string, due_at: null as number | null })

const priorityOptions = [
  { label: t('todo.high'), value: 'high' },
  { label: t('todo.medium'), value: 'medium' },
  { label: t('todo.low'), value: 'low' },
]

async function fetchList() {
  loading.value = true
  try {
    const params: Record<string, unknown> = { page: page.value, page_size: pageSize.value }
    if (statusFilter.value !== 'all') params.status = statusFilter.value
    const { data } = await todoApi.list(params)
    items.value = data.data.list || []
    total.value = data.data.total || 0
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  form.value = { title: '', description: '', priority: 'medium', due_at: null }
  showModal.value = true
}

async function handleCreate() {
  if (!form.value.title.trim()) return
  saving.value = true
  try {
    await todoApi.create({
      title: form.value.title,
      description: form.value.description,
      priority: form.value.priority as 'high' | 'medium' | 'low',
      due_at: form.value.due_at ? new Date(form.value.due_at).toISOString() : undefined,
    })
    showModal.value = false
    message.success(t('todo.created'))
    fetchList()
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  } finally {
    saving.value = false
  }
}

async function handleComplete(item: TodoItem) {
  try {
    await todoApi.complete(item.id)
    item.status = 'completed'
    message.success(t('todo.completed'))
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

async function handleDelete(id: number) {
  try {
    await todoApi.delete(id)
    items.value = items.value.filter(i => i.id !== id)
    total.value--
  } catch (err: unknown) {
    message.error(getErrorMessage(err))
  }
}

function priorityColor(p: string): string {
  if (p === 'high') return 'error'
  if (p === 'medium') return 'warning'
  return 'default'
}

onMounted(fetchList)
</script>

<template>
  <div class="todo-page">
    <PageHeader :title="t('todo.title')" :subtitle="t('todo.subtitle')">
      <template #actions>
        <NButton size="small" quaternary @click="fetchList">
          <template #icon><NIcon :component="ListOutline" /></template>
          {{ t('common.refresh') }}
        </NButton>
        <NButton size="small" type="primary" @click="openCreate">
          <template #icon><NIcon :component="AddOutline" /></template>
          {{ t('todo.create') }}
        </NButton>
      </template>
    </PageHeader>

    <div class="todo-filter">
      <NRadioGroup v-model:value="statusFilter" size="small" @update:value="fetchList">
        <NRadioButton value="all">{{ t('common.all') }}</NRadioButton>
        <NRadioButton value="pending">{{ t('todo.pending') }}</NRadioButton>
        <NRadioButton value="completed">{{ t('todo.completed') }}</NRadioButton>
      </NRadioGroup>
    </div>

    <NSpin :show="loading">
      <div v-if="items.length === 0 && !loading" class="todo-empty">
        <NEmpty :description="t('todo.noTodos')" />
      </div>
      <div v-else class="todo-list">
        <div
          v-for="item in items"
          :key="item.id"
          class="todo-item sre-row-card"
          :class="{ completed: item.status === 'completed' }"
        >
          <div class="todo-main">
            <div class="todo-head">
              <span class="todo-title">{{ item.title }}</span>
              <span class="todo-priority" :class="item.priority">{{ item.priority }}</span>
            </div>
            <div v-if="item.description" class="todo-desc">{{ item.description }}</div>
            <div class="todo-meta tnum">
              <span>{{ formatTime(item.created_at) }}</span>
              <span v-if="item.due_at"> &middot; {{ t('todo.due') }}: {{ formatTime(item.due_at) }}</span>
            </div>
          </div>
          <div class="todo-actions">
            <NButton
              v-if="item.status === 'pending'"
              size="tiny" quaternary type="success"
              @click="handleComplete(item)"
            >
              <template #icon><NIcon :component="CheckmarkOutline" :size="14" /></template>
            </NButton>
            <NButton size="tiny" quaternary @click="handleDelete(item.id)">
              <template #icon><NIcon :component="TrashOutline" :size="14" /></template>
            </NButton>
          </div>
        </div>
      </div>
    </NSpin>

    <NModal v-model:show="showModal" preset="card" :title="t('todo.create')" style="width: 480px" :bordered="false">
      <NForm label-placement="top">
        <NFormItem :label="t('todo.title')" required>
          <NInput v-model:value="form.title" :placeholder="t('todo.titlePlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('todo.description')">
          <NInput v-model:value="form.description" type="textarea" :rows="2" :placeholder="t('todo.descPlaceholder')" />
        </NFormItem>
        <NFormItem :label="t('todo.priority')">
          <NSelect v-model:value="form.priority" :options="priorityOptions" />
        </NFormItem>
        <NFormItem :label="t('todo.dueDate')">
          <NDatePicker v-model:value="form.due_at" type="datetime" clearable style="width: 100%" />
        </NFormItem>
      </NForm>
      <template #action>
        <NButton @click="showModal = false">{{ t('common.cancel') }}</NButton>
        <NButton type="primary" :loading="saving" @click="handleCreate">{{ t('common.create') }}</NButton>
      </template>
    </NModal>
  </div>
</template>

<style scoped>
.todo-page { font-family: var(--sre-font-sans); }
.todo-filter { margin: 12px 0 16px; }
.todo-list { display: flex; flex-direction: column; gap: 6px; }
.todo-empty { padding: 60px 0; text-align: center; }
.todo-item {
  display: flex; align-items: flex-start; gap: 12px;
  padding: 12px 14px;
}
.todo-item.completed { opacity: 0.55; }
.todo-item.completed .todo-title { text-decoration: line-through; }
.todo-main { flex: 1; min-width: 0; display: flex; flex-direction: column; gap: 4px; }
.todo-head { display: flex; align-items: center; gap: 8px; }
.todo-title { font-size: 14px; font-weight: 600; color: var(--sre-text-primary); }
.todo-priority {
  font-size: 10px; padding: 1px 6px; border-radius: 3px;
  font-weight: 600; text-transform: uppercase; letter-spacing: 0.5px;
}
.todo-priority.high { background: var(--sre-critical-soft); color: var(--sre-critical); }
.todo-priority.medium { background: var(--sre-warning-soft); color: var(--sre-warning); }
.todo-priority.low { background: var(--sre-bg-elevated); color: var(--sre-text-secondary); }
.todo-desc { font-size: 12px; color: var(--sre-text-secondary); }
.todo-meta { font-size: 11px; color: var(--sre-text-tertiary); }
.todo-actions { display: flex; gap: 4px; flex-shrink: 0; }
</style>
