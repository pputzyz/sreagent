<script setup lang="ts">
import { computed, ref } from 'vue'
import { NButton, NProgress, NIcon } from 'naive-ui'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'
import { usePetStore } from '@/stores/pet'
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
</script>

<template>
  <div class="pet-panel">
    <div v-if="petStore.pet" class="pet-panel-body">
      <!-- Header -->
      <div class="pet-panel-header">
        <span class="pet-panel-emoji">🦊</span>
        <div>
          <div class="pet-panel-name">{{ petStore.pet.name }}</div>
          <div class="pet-panel-level">Lv.{{ petStore.pet.level }}</div>
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
  font-size: 32px;
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
}
</style>
