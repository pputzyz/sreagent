<script setup lang="ts">
import { onMounted, computed, ref, watch } from 'vue'
import { NCard, NButton, NProgress, NInput, NDataTable, NIcon } from 'naive-ui'
import type { DataTableColumns } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { useRouter } from 'vue-router'
import { usePetStore } from '@/stores/pet'
import type { PetType } from '@/stores/pet'
import type { PetInteraction } from '@/types'
import { ChatbubbleEllipsesOutline } from '@vicons/ionicons5'

const { t } = useI18n()
const router = useRouter()
const petStore = usePetStore()

const celebrating = ref(false)
const prevLevel = ref(petStore.pet?.level ?? 0)

const petTypes: { type: PetType; emoji: string; label: string }[] = [
  { type: 'fox', emoji: '\u{1F98A}', label: 'Fox' },
  { type: 'cat', emoji: '\u{1F431}', label: 'Cat' },
  { type: 'owl', emoji: '\u{1F989}', label: 'Owl' },
  { type: 'panda', emoji: '\u{1F43C}', label: 'Panda' },
  { type: 'tiger', emoji: '\u{1F42F}', label: 'Tiger' },
  { type: 'bunny', emoji: '\u{1F430}', label: 'Bunny' },
  { type: 'dragon', emoji: '\u{1F409}', label: 'Dragon' },
  { type: 'penguin', emoji: '\u{1F427}', label: 'Penguin' },
]

// Watch for level-up celebration
watch(() => petStore.pet?.level, (newLevel) => {
  if (newLevel && newLevel > prevLevel.value && prevLevel.value > 0) {
    celebrating.value = true
    setTimeout(() => { celebrating.value = false }, 2000)
  }
  prevLevel.value = newLevel ?? 0
})

onMounted(async () => {
  await Promise.all([petStore.fetchPet(), petStore.fetchInteractions()])
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

async function handleFeed() {
  try {
    await petStore.feed()
  } catch {
    // error already handled by store
  }
}

async function handlePlay() {
  try {
    await petStore.play()
  } catch {
    // error already handled by store
  }
}
</script>

<template>
  <div class="pet-page">
    <div v-if="petStore.error && !petStore.pet" class="pet-error-banner">
      <span>{{ petStore.error }}</span>
      <n-button size="small" quaternary @click="petStore.fetchPet()">
        {{ t('pet.retry') }}
      </n-button>
    </div>

    <div class="pet-grid">
      <!-- Left: Avatar + Status -->
      <n-card class="pet-card pet-card--main">
        <div class="pet-avatar-section">
          <div class="pet-avatar pet-float" :class="{ 'pet-avatar--celebrate': celebrating }">
            {{ petStore.petEmoji }}
          </div>
          <div v-if="petStore.pet" class="pet-main-info">
            <div class="pet-name-display">{{ petStore.pet.name }}</div>
            <div class="pet-level-display" :class="{ 'pet-level--celebrate': celebrating }">
              Lv.{{ petStore.pet.level }}
              <span v-if="celebrating" class="pet-celebrate-stars">✨</span>
            </div>
          </div>
        </div>

        <div v-if="petStore.pet" class="pet-bars">
          <div class="pet-bar-item">
            <div class="pet-bar-header">
              <span class="pet-bar-label">{{ t('pet.hunger') }}</span>
              <span class="pet-bar-value">{{ petStore.hungerPercent }}%</span>
            </div>
            <n-progress
              type="line"
              :percentage="petStore.hungerPercent"
              :show-indicator="false"
              :height="10"
              :border-radius="5"
              :status="petStore.hungerPercent > 70 ? 'error' : petStore.hungerPercent > 40 ? 'warning' : 'success'"
            />
          </div>
          <div class="pet-bar-item">
            <div class="pet-bar-header">
              <span class="pet-bar-label">{{ t('pet.mood') }}</span>
              <span class="pet-bar-value">{{ petStore.moodPercent }}%</span>
            </div>
            <n-progress
              type="line"
              :percentage="petStore.moodPercent"
              :show-indicator="false"
              :height="10"
              :border-radius="5"
            />
          </div>
          <div class="pet-bar-item">
            <div class="pet-bar-header">
              <span class="pet-bar-label">{{ t('pet.exp') }}</span>
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
          <n-button type="primary" @click="handleFeed">{{ t('pet.feed') }}</n-button>
          <n-button @click="handlePlay">{{ t('pet.play') }}</n-button>
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
          <div class="pet-form-item">
            <label class="pet-form-label">{{ t('pet.type') }}</label>
            <div class="pet-type-grid">
              <button
                v-for="pt in petTypes"
                :key="pt.type"
                class="pet-type-option"
                :class="{ 'pet-type-option--active': pt.type === petStore.petType }"
                @click="petStore.setPetType(pt.type)"
              >
                <span class="pet-type-emoji">{{ pt.emoji }}</span>
                <span class="pet-type-label">{{ pt.label }}</span>
              </button>
            </div>
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

.pet-error-banner {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 8px;
  padding: 8px 12px;
  font-size: 12px;
  color: var(--sre-critical);
  background: var(--sre-critical-soft);
  border-radius: var(--sre-radius-sm);
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
  font-size: 80px;
  line-height: 1;
  transition: transform 200ms var(--sre-ease-out);
}

@keyframes pet-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-6px); }
}

.pet-float {
  animation: pet-float 3s ease-in-out infinite;
}

@keyframes pet-celebrate {
  0% { transform: scale(1) rotate(0deg); }
  25% { transform: scale(1.2) rotate(-5deg); }
  50% { transform: scale(1.3) rotate(5deg); }
  75% { transform: scale(1.2) rotate(-3deg); }
  100% { transform: scale(1) rotate(0deg); }
}

.pet-avatar--celebrate {
  animation: pet-celebrate 0.8s ease-in-out;
}

.pet-level--celebrate {
  color: var(--sre-primary) !important;
  font-size: 16px;
  transition: all 300ms var(--sre-ease-out);
}

.pet-celebrate-stars {
  display: inline-block;
  animation: pet-star-spin 1s ease-in-out infinite;
}

@keyframes pet-star-spin {
  0%, 100% { transform: rotate(0deg) scale(1); }
  50% { transform: rotate(180deg) scale(1.3); }
}

.pet-main-info {
  display: flex;
  flex-direction: column;
}

.pet-name-display {
  font-size: 20px;
  font-weight: 700;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 200px;
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

.pet-type-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 8px;
}

.pet-type-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 4px;
  padding: 10px 6px;
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-md);
  background: var(--sre-bg-primary);
  cursor: pointer;
  transition: all 150ms var(--sre-ease-out);
}

.pet-type-option:hover {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft);
  transform: translateY(-2px);
  box-shadow: 0 4px 12px rgba(0, 0, 0, 0.08);
}

.pet-type-option--active {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft);
  box-shadow: 0 0 0 2px var(--sre-primary-soft);
}

.pet-type-option:active {
  transform: scale(0.92);
}

.pet-type-emoji {
  font-size: 28px;
  line-height: 1;
}

.pet-type-label {
  font-size: 10px;
  color: var(--sre-text-tertiary);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

@media (prefers-reduced-motion: reduce) {
  .pet-float,
  .pet-avatar--celebrate,
  .pet-celebrate-stars {
    animation: none;
  }
  .pet-type-option:hover {
    transform: none;
  }
  .pet-type-option:active {
    transform: none;
  }
}
</style>
