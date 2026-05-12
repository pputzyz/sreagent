<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { NPopover } from 'naive-ui'
import { usePetStore } from '@/stores/pet'
import PetPanel from './PetPanel.vue'
import PetAvatar from '@/components/common/PetAvatar.vue'

const emit = defineEmits<{
  chat: []
}>()

const petStore = usePetStore()
const showPanel = ref(false)
const bouncing = ref(false)

const statusEmoji = computed(() => {
  if (!petStore.pet) return petStore.petEmoji
  if (petStore.pet.hunger > 80) return '😰'
  if (petStore.pet.mood > 80) return '😊'
  if (petStore.pet.mood < 30) return '😢'
  return petStore.petEmoji
})

const isHungry = computed(() => petStore.pet && petStore.pet.hunger > 80)

watch(statusEmoji, () => {
  bouncing.value = true
  setTimeout(() => { bouncing.value = false }, 500)
})

onMounted(() => {
  petStore.fetchPet()
})

function handleChat() {
  showPanel.value = false
  emit('chat')
}
</script>

<template>
  <n-popover
    :show="showPanel"
    trigger="click"
    placement="right"
    :show-arrow="false"
    style="padding: 0"
    @update:show="showPanel = $event"
  >
    <template #trigger>
      <button
        class="pet-corner"
        :class="{
          'pet-corner--active': showPanel,
          'pet-corner--hungry': isHungry,
        }"
        @click="showPanel = !showPanel"
      >
        <div class="pet-avatar-wrap" :class="{ 'pet-avatar--bounce': bouncing, 'pet-avatar--hungry': isHungry }">
          <PetAvatar :type="petStore.petType" :size="36" />
        </div>
        <span v-if="petStore.pet" class="pet-info">
          <span class="pet-name">{{ petStore.pet.name }}</span>
          <span class="pet-level">Lv.{{ petStore.pet.level }}</span>
        </span>
      </button>
    </template>

    <PetPanel @close="showPanel = false" @chat="handleChat" />
  </n-popover>
</template>

<style scoped>
.pet-corner {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 48px;
  height: 48px;
  border: none;
  border-radius: var(--sre-radius-md);
  background: transparent;
  cursor: pointer;
  padding: 0 4px;
  transition: background var(--sre-duration-fast) var(--sre-ease-out);
}

.pet-corner:hover {
  background: var(--sre-bg-hover);
}

.pet-corner--active {
  background: var(--sre-bg-active);
}

.pet-corner--hungry {
  animation: pet-hungry-pulse-corner 1.5s ease-in-out infinite;
}

@keyframes pet-hungry-pulse-corner {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.05); }
}

.pet-avatar-wrap {
  flex-shrink: 0;
  display: flex;
  align-items: center;
  justify-content: center;
  transition: transform 200ms var(--sre-ease-out);
}

.pet-avatar--bounce {
  animation: pet-status-bounce 500ms var(--sre-ease-out);
}

@keyframes pet-status-bounce {
  0% { transform: scale(1); }
  30% { transform: scale(1.2); }
  60% { transform: scale(0.95); }
  100% { transform: scale(1); }
}

.pet-avatar--hungry {
  animation: pet-hungry-pulse 1.8s ease-in-out infinite;
}

@keyframes pet-hungry-pulse {
  0%, 100% { transform: scale(1); }
  50% { transform: scale(1.06); }
}

.pet-info {
  display: flex;
  flex-direction: column;
  white-space: nowrap;
  opacity: 0;
  transform: translateX(-4px);
  transition:
    opacity var(--sre-duration-fast) var(--sre-ease-out),
    transform var(--sre-duration-fast) var(--sre-ease-out);
  pointer-events: none;
}

.pet-corner:hover .pet-info,
.pet-corner--active .pet-info {
  opacity: 1;
  transform: translateX(0);
  pointer-events: auto;
}

.pet-name {
  font-size: 13px;
  font-weight: 700;
  color: var(--sre-text-primary);
  line-height: 1.2;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  max-width: 120px;
}

.pet-level {
  font-size: 11px;
  color: var(--sre-primary);
  font-weight: 700;
  line-height: 1.2;
}

@media (prefers-reduced-motion: reduce) {
  .pet-emoji--bounce,
  .pet-emoji--hungry,
  .pet-float,
  .pet-corner--hungry {
    animation: none;
  }
}
</style>
