<script setup lang="ts">
import { onMounted, computed } from 'vue'
import { NCard, NButton, NProgress, NInput, NDataTable, NIcon } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { usePetStore } from '@/stores/pet'
import type { PetInteraction } from '@/types'
import { ChatbubbleEllipsesOutline } from '@vicons/ionicons5'

const { t } = useI18n()
const router = useRouter()
const petStore = usePetStore()

onMounted(async () => {
  await Promise.all([petStore.fetchPet(), petStore.fetchInteractions()])
})

const hungerDisplay = computed(() => {
  if (!petStore.pet) return 0
  return 100 - petStore.pet.hunger
})

const interactionColumns = computed<DataTableColumns<PetInteraction>>(() => [
  {
    title: t('pet.interactionType'),
    key: 'type',
    width: 100,
    render(row) {
      const labels: Record<string, string> = {
        feed: t('pet.feed'),
        play: t('pet.play'),
        chat: t('pet.chat'),
        level_up: t('pet.levelUp'),
      }
      return labels[row.type] || row.type
    },
  },
  {
    title: t('pet.interactionValue'),
    key: 'value',
    width: 80,
    align: 'center',
  },
  {
    title: t('pet.interactionTime'),
    key: 'created_at',
    width: 180,
    render(row) {
      return row.created_at ? new Date(row.created_at).toLocaleString() : ''
    },
  },
])

async function handleNameUpdate(val: string) {
  if (val.trim() && petStore.pet) {
    await petStore.updateName(val.trim())
  }
}
</script>

<template>
  <div class="pet-page">
    <div class="pet-grid">
      <!-- Left: Avatar + Status -->
      <n-card class="pet-card pet-card--main">
        <div class="pet-avatar-section">
          <div class="pet-avatar">🦊</div>
          <div v-if="petStore.pet" class="pet-main-info">
            <div class="pet-name-display">{{ petStore.pet.name }}</div>
            <div class="pet-level-display">Lv.{{ petStore.pet.level }}</div>
          </div>
        </div>

        <div v-if="petStore.pet" class="pet-bars">
          <div class="pet-bar-item">
            <div class="pet-bar-header">
              <span class="pet-bar-label">{{ t('pet.hunger') }}</span>
              <span class="pet-bar-value">{{ hungerDisplay }}%</span>
            </div>
            <n-progress
              type="line"
              :percentage="hungerDisplay"
              :show-indicator="false"
              :height="10"
              :border-radius="5"
            />
          </div>
          <div class="pet-bar-item">
            <div class="pet-bar-header">
              <span class="pet-bar-label">{{ t('pet.mood') }}</span>
              <span class="pet-bar-value">{{ petStore.pet.mood }}%</span>
            </div>
            <n-progress
              type="line"
              :percentage="petStore.pet.mood"
              :show-indicator="false"
              :height="10"
              :border-radius="5"
            />
          </div>
          <div class="pet-bar-item">
            <div class="pet-bar-header">
              <span class="pet-bar-label">EXP</span>
              <span class="pet-bar-value">{{ petStore.pet.exp }}/{{ petStore.expForNextLevel }}</span>
            </div>
            <n-progress
              type="line"
              :percentage="petStore.expProgress"
              :show-indicator="false"
              :height="10"
              :border-radius="5"
            />
          </div>
        </div>

        <div class="pet-actions-row">
          <n-button type="primary" @click="petStore.feed()">{{ t('pet.feed') }}</n-button>
          <n-button @click="petStore.play()">{{ t('pet.play') }}</n-button>
          <n-button @click="router.push('/')">
            <template #icon>
              <n-icon :component="ChatbubbleEllipsesOutline" />
            </template>
            {{ t('pet.chat') }}
          </n-button>
        </div>
      </n-card>

      <!-- Right: Settings -->
      <n-card :title="t('pet.settings')" class="pet-card pet-card--settings">
        <div v-if="petStore.pet" class="pet-settings-form">
          <div class="pet-form-item">
            <label class="pet-form-label">{{ t('pet.name') }}</label>
            <n-input
              :default-value="petStore.pet.name"
              :placeholder="t('pet.namePlaceholder')"
              @blur="(e: FocusEvent) => handleNameUpdate((e.target as HTMLInputElement).value)"
              @keydown.enter="(e: KeyboardEvent) => handleNameUpdate((e.target as HTMLInputElement).value)"
            />
          </div>
        </div>
      </n-card>
    </div>

    <!-- Bottom: Interaction History -->
    <n-card :title="t('pet.interactionHistory')" class="pet-card pet-card--history">
      <n-data-table
        :columns="interactionColumns"
        :data="petStore.interactions"
        :bordered="false"
        size="small"
      />
    </n-card>
  </div>
</template>

<style scoped>
.pet-page {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.pet-grid {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

@media (max-width: 768px) {
  .pet-grid {
    grid-template-columns: 1fr;
  }
}

.pet-card {
  border-radius: var(--sre-radius-md);
}

.pet-avatar-section {
  display: flex;
  align-items: center;
  gap: 16px;
  margin-bottom: 20px;
}

.pet-avatar {
  font-size: 64px;
  line-height: 1;
}

.pet-main-info {
  display: flex;
  flex-direction: column;
}

.pet-name-display {
  font-size: 20px;
  font-weight: 700;
  color: var(--sre-text-primary);
}

.pet-level-display {
  font-size: 14px;
  color: var(--sre-text-tertiary);
  font-weight: 600;
}

.pet-bars {
  display: flex;
  flex-direction: column;
  gap: 14px;
  margin-bottom: 20px;
}

.pet-bar-item {
  display: flex;
  flex-direction: column;
  gap: 4px;
}

.pet-bar-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
}

.pet-bar-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
}

.pet-bar-value {
  font-size: 12px;
  color: var(--sre-text-tertiary);
  font-family: var(--sre-font-mono);
}

.pet-actions-row {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
}

.pet-settings-form {
  display: flex;
  flex-direction: column;
  gap: 16px;
}

.pet-form-item {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.pet-form-label {
  font-size: 13px;
  font-weight: 600;
  color: var(--sre-text-secondary);
}
</style>
