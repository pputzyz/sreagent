<script setup lang="ts">
import { NButton, NProgress, NIcon } from 'naive-ui'
import { useI18n } from 'vue-i18n'
import { usePetStore } from '@/stores/pet'
import { ChatbubbleEllipsesOutline } from '@vicons/ionicons5'

const emit = defineEmits<{
  close: []
  chat: []
}>()

const { t } = useI18n()
const petStore = usePetStore()

function handleChat() {
  emit('close')
  emit('chat')
}

function goToDetail() {
  emit('close')
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
        <div class="pet-bar-row">
          <span class="pet-bar-label">EXP</span>
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
        <n-button size="small" @click="petStore.feed()">{{ t('pet.feed') }}</n-button>
        <n-button size="small" @click="petStore.play()">{{ t('pet.play') }}</n-button>
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
      {{ t('pet.loading') }}
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
</style>
