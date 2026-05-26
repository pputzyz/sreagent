<template>
  <div class="skill-manager">
    <n-card :bordered="false">
      <template #header>
        <div class="card-header">
          <span>{{ t('aiSkills.title') }}</span>
          <div class="header-actions">
            <n-upload
              :show-file-list="false"
              :custom-request="handleImport"
              accept=".zip,.tar.gz"
            >
              <n-button>{{ t('aiSkills.import') }}</n-button>
            </n-upload>
            <n-button type="primary" @click="openCreate">
              {{ t('aiSkills.create') }}
            </n-button>
          </div>
        </div>
      </template>

      <n-data-table
        :columns="columns"
        :data="skills"
        :loading="loading"
        :row-key="(row: AISkill) => row.id"
        :bordered="false"
        striped
      />
    </n-card>

    <!-- Create/Edit Drawer -->
    <n-drawer v-model:show="drawerVisible" :width="640" placement="right">
      <n-drawer-content :title="isEdit ? t('aiSkills.edit') : t('aiSkills.create')">
        <n-form ref="formRef" :model="form" :rules="rules" label-placement="left" label-width="100">
          <n-form-item :label="t('aiSkills.name')" path="name">
            <n-input v-model:value="form.name" :placeholder="t('aiSkills.namePlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('aiSkills.description')" path="description">
            <n-input v-model:value="form.description" type="textarea" :rows="2" :placeholder="t('aiSkills.descriptionPlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('aiSkills.instructions')" path="instructions">
            <n-input v-model:value="form.instructions" type="textarea" :rows="8" :placeholder="t('aiSkills.instructionsPlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('aiSkills.license')">
            <n-input v-model:value="form.license" placeholder="MIT" />
          </n-form-item>
          <n-form-item :label="t('aiSkills.compatibility')">
            <n-input v-model:value="form.compatibility" placeholder="claude-3, gpt-4" />
          </n-form-item>
          <n-form-item :label="t('aiSkills.allowedTools')">
            <n-input v-model:value="form.allowed_tools" :placeholder="t('aiSkills.allowedToolsPlaceholder')" />
          </n-form-item>
          <n-form-item :label="t('aiSkills.enabled')">
            <n-switch v-model:value="form.enabled" />
          </n-form-item>
        </n-form>

        <template #footer>
          <n-space>
            <n-button @click="drawerVisible = false">{{ t('common.cancel') }}</n-button>
            <n-button type="primary" :loading="saving" @click="handleSave">
              {{ t('common.save') }}
            </n-button>
          </n-space>
        </template>
      </n-drawer-content>
    </n-drawer>

    <!-- Files Drawer -->
    <n-drawer v-model:show="filesDrawerVisible" :width="700" placement="right">
      <n-drawer-content :title="t('aiSkills.files') + ' - ' + currentSkill?.name">
        <div class="files-header">
          <n-button size="small" type="primary" @click="openAddFile">
            {{ t('aiSkills.addFile') }}
          </n-button>
        </div>
        <n-data-table
          :columns="fileColumns"
          :data="skillFiles"
          :loading="filesLoading"
          :row-key="(row: AISkillFile) => row.id"
          :bordered="false"
          size="small"
        />
      </n-drawer-content>
    </n-drawer>

    <!-- Add File Modal -->
    <n-modal v-model:show="addFileModalVisible" preset="dialog" :title="t('aiSkills.addFile')">
      <n-form ref="fileFormRef" :model="fileForm" :rules="fileRules">
        <n-form-item :label="t('aiSkills.fileName')" path="name">
          <n-input v-model:value="fileForm.name" placeholder="scripts/foo.sh" />
        </n-form-item>
        <n-form-item :label="t('aiSkills.fileContent')" path="content">
          <n-input v-model:value="fileForm.content" type="textarea" :rows="10" placeholder="#!/bin/bash" />
        </n-form-item>
      </n-form>
      <template #action>
        <n-button @click="addFileModalVisible = false">{{ t('common.cancel') }}</n-button>
        <n-button type="primary" :loading="addFileLoading" @click="handleAddFile">{{ t('common.save') }}</n-button>
      </template>
    </n-modal>
  </div>
</template>

<script setup lang="ts">
import { ref, h, onMounted } from 'vue'
import { useI18n } from 'vue-i18n'
import {
  NButton, NCard, NDataTable, NDrawer, NDrawerContent, NForm, NFormItem,
  NInput, NSpace, NSwitch, NUpload, NModal, NPopconfirm, NTag, useMessage,
} from 'naive-ui'
import type { DataTableColumns, FormRules, UploadCustomRequestOptions } from 'naive-ui'
import { aiSkillApi } from '@/api'
import type { AISkill, AISkillFile, CreateAISkillRequest } from '@/api/ai-skill'

const { t } = useI18n()
const message = useMessage()

// --- State ---
const skills = ref<AISkill[]>([])
const loading = ref(false)
const drawerVisible = ref(false)
const isEdit = ref(false)
const saving = ref(false)
const formRef = ref()
const currentEditId = ref<number | null>(null)

const form = ref<CreateAISkillRequest>({
  name: '',
  description: '',
  instructions: '',
  license: '',
  compatibility: '',
  allowed_tools: '',
  enabled: true,
})

const rules: FormRules = {
  name: [{ required: true, message: t('aiSkills.nameRequired'), trigger: 'blur' }],
}

// Files state
const filesDrawerVisible = ref(false)
const filesLoading = ref(false)
const currentSkill = ref<AISkill | null>(null)
const skillFiles = ref<AISkillFile[]>([])

const addFileModalVisible = ref(false)
const addFileLoading = ref(false)
const fileFormRef = ref()
const fileForm = ref({ name: '', content: '' })
const fileRules: FormRules = {
  name: [{ required: true, message: t('aiSkills.fileNameRequired'), trigger: 'blur' }],
}

// --- Columns ---
const columns: DataTableColumns<AISkill> = [
  {
    title: t('aiSkills.name'),
    key: 'name',
    ellipsis: { tooltip: true },
    render: (row) => h('span', { class: 'skill-name' }, row.name),
  },
  {
    title: t('aiSkills.description'),
    key: 'description',
    ellipsis: { tooltip: true },
    width: 200,
  },
  {
    title: t('aiSkills.status'),
    key: 'enabled',
    width: 80,
    render: (row) => h(NTag, {
      type: row.enabled ? 'success' : 'default',
      size: 'small',
      bordered: false,
    }, { default: () => row.enabled ? t('common.enabled') : t('common.disabled') }),
  },
  {
    title: t('aiSkills.builtin'),
    key: 'builtin',
    width: 80,
    render: (row) => row.builtin ? h(NTag, { type: 'info', size: 'small', bordered: false }, { default: () => t('aiSkills.yes') }) : null,
  },
  {
    title: t('aiSkills.createdBy'),
    key: 'created_by',
    width: 100,
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 240,
    render: (row) => h(NSpace, { size: 'small' }, {
      default: () => [
        h(NButton, { size: 'small', onClick: () => openFiles(row) }, { default: () => t('aiSkills.files') }),
        h(NButton, { size: 'small', onClick: () => openEdit(row) }, { default: () => t('common.edit') }),
        row.builtin ? null : h(NPopconfirm, { onPositiveClick: () => handleDelete(row.id) }, {
          trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
          default: () => t('common.confirmDelete'),
        }),
      ],
    }),
  },
]

const fileColumns: DataTableColumns<AISkillFile> = [
  { title: t('aiSkills.fileName'), key: 'name', ellipsis: { tooltip: true } },
  {
    title: t('aiSkills.fileSize'),
    key: 'size',
    width: 100,
    render: (row) => `${(row.size / 1024).toFixed(1)} KB`,
  },
  {
    title: t('common.actions'),
    key: 'actions',
    width: 80,
    render: (row) => h(NPopconfirm, { onPositiveClick: () => handleDeleteFile(row.id) }, {
      trigger: () => h(NButton, { size: 'small', type: 'error' }, { default: () => t('common.delete') }),
      default: () => t('common.confirmDelete'),
    }),
  },
]

// --- Methods ---
async function loadSkills() {
  loading.value = true
  try {
    const res = await aiSkillApi.list()
    skills.value = res.data.data || []
  } catch {
    message.error(t('aiSkills.loadError'))
  } finally {
    loading.value = false
  }
}

function openCreate() {
  isEdit.value = false
  currentEditId.value = null
  form.value = { name: '', description: '', instructions: '', license: '', compatibility: '', allowed_tools: '', enabled: true }
  drawerVisible.value = true
}

function openEdit(row: AISkill) {
  isEdit.value = true
  currentEditId.value = row.id
  form.value = {
    name: row.name,
    description: row.description,
    instructions: row.instructions,
    license: row.license,
    compatibility: row.compatibility,
    allowed_tools: row.allowed_tools,
    enabled: row.enabled,
  }
  drawerVisible.value = true
}

async function handleSave() {
  try {
    await formRef.value?.validate()
  } catch { return }

  saving.value = true
  try {
    if (isEdit.value && currentEditId.value) {
      await aiSkillApi.update(currentEditId.value, form.value)
      message.success(t('aiSkills.updateSuccess'))
    } else {
      await aiSkillApi.create(form.value)
      message.success(t('aiSkills.createSuccess'))
    }
    drawerVisible.value = false
    await loadSkills()
  } catch {
    message.error(t('aiSkills.saveError'))
  } finally {
    saving.value = false
  }
}

async function handleDelete(id: number) {
  try {
    await aiSkillApi.delete(id)
    message.success(t('aiSkills.deleteSuccess'))
    await loadSkills()
  } catch {
    message.error(t('aiSkills.deleteError'))
  }
}

async function handleImport(options: UploadCustomRequestOptions) {
  const file = options.file.file
  if (!file) return
  try {
    await aiSkillApi.import(file)
    message.success(t('aiSkills.importSuccess'))
    await loadSkills()
  } catch {
    message.error(t('aiSkills.importError'))
  }
}

// --- Files ---
async function openFiles(skill: AISkill) {
  currentSkill.value = skill
  filesDrawerVisible.value = true
  filesLoading.value = true
  try {
    const res = await aiSkillApi.getFiles(skill.id)
    skillFiles.value = res.data.data || []
  } catch {
    message.error(t('aiSkills.loadFilesError'))
  } finally {
    filesLoading.value = false
  }
}

function openAddFile() {
  fileForm.value = { name: '', content: '' }
  addFileModalVisible.value = true
}

async function handleAddFile() {
  try {
    await fileFormRef.value?.validate()
  } catch { return }

  if (!currentSkill.value) return
  addFileLoading.value = true
  try {
    await aiSkillApi.addFile(currentSkill.value.id, fileForm.value)
    message.success(t('aiSkills.addFileSuccess'))
    addFileModalVisible.value = false
    const res = await aiSkillApi.getFiles(currentSkill.value.id)
    skillFiles.value = res.data.data || []
  } catch {
    message.error(t('aiSkills.addFileError'))
  } finally {
    addFileLoading.value = false
  }
}

async function handleDeleteFile(fileId: number) {
  try {
    await aiSkillApi.deleteFile(fileId)
    message.success(t('aiSkills.deleteFileSuccess'))
    if (currentSkill.value) {
      const res = await aiSkillApi.getFiles(currentSkill.value.id)
      skillFiles.value = res.data.data || []
    }
  } catch {
    message.error(t('aiSkills.deleteFileError'))
  }
}

onMounted(loadSkills)
</script>

<style scoped>
.skill-manager {
  padding: 16px;
}
.card-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.header-actions {
  display: flex;
  gap: 8px;
}
.files-header {
  margin-bottom: 12px;
}
.skill-name {
  font-weight: 500;
}
</style>
