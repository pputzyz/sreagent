<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NProgress, NIcon } from 'naive-ui'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { usePetStore } from '@/stores/pet'
import type { PetType } from '@/stores/pet'
import { ChatbubbleEllipsesOutline } from '@vicons/ionicons5'

const emit = defineEmits<{
  close: []
  chat: []
}>()

const router = useRouter()
const { t } = useI18n()
const petStore = usePetStore()

const feedPressed = ref(false)
const playPressed = ref(false)
const showTypeSelector = ref(false)

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

const expNearlyFull = computed(() => petStore.expProgress > 85)

async function handleFeed() {
  feedPressed.value = true
  setTimeout(() => { feedPressed.value = false }, 200)
  try {
    await petStore.feed()
  } catch {
    // error already handled by store
  }
}

async function handlePlay() {
  playPressed.value = true
  setTimeout(() => { playPressed.value = false }, 200)
  try {
    await petStore.play()
  } catch {
    // error already handled by store
  }
}

function handleChat() {
  emit('close')
  emit('chat')
}

function goToDetail() {
  emit('close')
  router.push('/pet')
}

function selectPetType(type: PetType) {
  petStore.setPetType(type)
  showTypeSelector.value = false
}
</script>

<template>
  <div class="pet-panel">
    <div v-if="petStore.pet" class="pet-panel-body">
      <!-- Header -->
      <div class="pet-panel-header">
        <span class="pet-panel-emoji pet-float">{{ petStore.petEmoji }}</span>
        <div>
          <div class="pet-panel-name">{{ petStore.pet.name }}</div>
          <div class="pet-panel-level">Lv.{{ petStore.pet.level }}</div>
        </div>
      </div>

      <!-- Pet Type Selector -->
      <div v-if="showTypeSelector" class="pet-type-selector">
        <div class="pet-type-grid">
          <button
            v-for="pt in petTypes"
            :key="pt.type"
            class="pet-type-option"
            :class="{ 'pet-type-option--active': pt.type === petStore.petType }"
            @click="selectPetType(pt.type)"
          >
            <span class="pet-type-emoji">{{ pt.emoji }}</span>
            <span class="pet-type-label">{{ pt.label }}</span>
          </button>
        </div>
      </div>

      <!-- Status bars -->
      <div class="pet-bars">
        <div class="pet-bar-row">
          <span class="pet-bar-label">{{ t('pet.hunger') }}</span>
          <n-progress
            type="line"
            :percentage="petStore.hungerPercent"
            :show-indicator="false"
            :height="8"
            :border-radius="4"
            :status="petStore.hungerPercent > 70 ? 'error' : petStore.hungerPercent > 40 ? 'warning' : 'success'"
            style="flex: 1"
          />
          <span class="pet-bar-value">{{ petStore.hungerPercent }}%</span>
        </div>
        <div class="pet-bar-row">
          <span class="pet-bar-label">{{ t('pet.mood') }}</span>
          <n-progress
            type="line"
            :percentage="petStore.moodPercent"
            :show-indicator="false"
            :height="8"
            :border-radius="4"
            style="flex: 1"
          />
          <span class="pet-bar-value">{{ petStore.moodPercent }}%</span>
        </div>
        <div class="pet-bar-row" :class="{ 'pet-bar-row--celebrate': expNearlyFull }">
          <span class="pet-bar-label">{{ t('pet.exp') }}</span>
          <n-progress
            type="line"
            :percentage="petStore.expProgress"
            :show-indicator="false"
            :height="8"
            :border-radius="4"
            style="flex: 1"
          />
          <span class="pet-bar-value">{{ petStore.pet.exp }}/{{ petStore.expForNextLevel }}</span>
        </div>
      </div>

      <!-- Actions -->
      <div class="pet-actions">
        <n-button size="small" :class="{ 'pet-action--pressed': feedPressed }" @click="handleFeed">{{ t('pet.feed') }}</n-button>
        <n-button size="small" :class="{ 'pet-action--pressed': playPressed }" @click="handlePlay">{{ t('pet.play') }}</n-button>
        <n-button size="small" @click="handleChat">
          <template #icon>
            <n-icon :component="ChatbubbleEllipsesOutline" />
          </template>
          {{ t('pet.chat') }}
        </n-button>
      </div>

      <!-- Change Pet Type -->
      <n-button size="tiny" quaternary block @click="showTypeSelector = !showTypeSelector">
        {{ showTypeSelector ? t('common.close') : t('pet.changeType') }}
      </n-button>

      <!-- Detail link -->
      <div class="pet-detail-link" @click="goToDetail">
        {{ t('pet.viewDetail') }} &rarr;
      </div>
    </div>

    <div v-else-if="petStore.loading" class="pet-panel-empty">
      {{ t('pet.loading') }}
    </div>

    <div v-else-if="petStore.error" class="pet-panel-error">
      <span>{{ petStore.error }}</span>
      <n-button size="tiny" quaternary @click="petStore.fetchPet()">
        {{ t('pet.retry') }}
      </n-button>
    </div>

    <div v-else class="pet-panel-empty">
      {{ t('pet.emptyState') }}
    </div>
  </div>
</template>

<style scoped>
.pet-panel {
  min-width: 240px;
  max-width: 280px;
  padding: 12px;
}

.pet-panel-body {
  display: flex;
  flex-direction: column;
  gap: 12px;
}

.pet-panel-header {
  display: flex;
  align-items: center;
  gap: 10px;
}

.pet-panel-emoji {
  font-size: 48px;
  line-height: 1;
}

@keyframes pet-float {
  0%, 100% { transform: translateY(0); }
  50% { transform: translateY(-4px); }
}

.pet-float {
  animation: pet-float 3s ease-in-out infinite;
}

.pet-panel-name {
  font-size: 15px;
  font-weight: 600;
  color: var(--sre-text-primary);
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 180px;
}

.pet-panel-level {
  font-size: 12px;
  color: var(--sre-text-tertiary);
}

.pet-bars {
  display: flex;
  flex-direction: column;
  gap: 8px;
}

.pet-bar-row {
  display: flex;
  align-items: center;
  gap: 8px;
}

.pet-bar-label {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-secondary);
  width: 32px;
  flex-shrink: 0;
}

.pet-bar-value {
  font-size: 11px;
  color: var(--sre-text-tertiary);
  width: 60px;
  text-align: right;
  flex-shrink: 0;
}

.pet-actions {
  display: flex;
  gap: 6px;
  flex-wrap: wrap;
}

.pet-actions :deep(.n-button) {
  transition: transform 120ms var(--sre-ease-out);
}

.pet-actions :deep(.n-button:active) {
  transform: scale(0.94);
}

.pet-action--pressed :deep(.n-button),
.pet-action--pressed {
  transform: scale(0.94) !important;
}

.pet-bar-row--celebrate {
  position: relative;
  overflow: hidden;
}

.pet-bar-row--celebrate::after {
  content: '';
  position: absolute;
  top: 0;
  left: -60%;
  width: 40%;
  height: 100%;
  background: linear-gradient(90deg, transparent, rgba(255, 215, 0, 0.12), transparent);
  animation: pet-exp-shimmer 2.2s ease-in-out infinite;
  pointer-events: none;
}

@keyframes pet-exp-shimmer {
  0% { left: -60%; }
  100% { left: 120%; }
}

.pet-detail-link {
  font-size: 12px;
  color: var(--sre-primary);
  cursor: pointer;
  text-align: center;
  padding: 4px 0;
  border-top: 1px solid var(--sre-border);
  transition: opacity var(--sre-duration-fast) var(--sre-ease-out);
}

.pet-detail-link:hover {
  opacity: 0.8;
}

.pet-type-selector {
  padding: 8px;
  background: var(--sre-bg-secondary);
  border-radius: var(--sre-radius-sm);
}

.pet-type-grid {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 6px;
}

.pet-type-option {
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 2px;
  padding: 6px 4px;
  border: 1px solid var(--sre-border);
  border-radius: var(--sre-radius-sm);
  background: var(--sre-bg-primary);
  cursor: pointer;
  transition: all 150ms var(--sre-ease-out);
}

.pet-type-option:hover {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft);
}

.pet-type-option--active {
  border-color: var(--sre-primary);
  background: var(--sre-primary-soft);
}

.pet-type-option:active {
  transform: scale(0.92);
}

.pet-type-emoji {
  font-size: 22px;
  line-height: 1;
}

.pet-type-label {
  font-size: 9px;
  color: var(--sre-text-tertiary);
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.3px;
}

.pet-panel-empty {
  padding: 20px;
  text-align: center;
  color: var(--sre-text-tertiary);
  font-size: 13px;
}

.pet-panel-error {
  padding: 16px;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 8px;
  color: var(--sre-critical);
  font-size: 12px;
  text-align: center;
}

@media (prefers-reduced-motion: reduce) {
  .pet-actions :deep(.n-button:active),
  .pet-action--pressed {
    transform: none !important;
  }
  .pet-bar-row--celebrate::after {
    animation: none;
  }
  .pet-float {
    animation: none;
  }
  .pet-type-option:active {
    transform: none;
  }
}
</style>
