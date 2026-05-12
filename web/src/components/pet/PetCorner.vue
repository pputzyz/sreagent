<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { NPopover } from 'naive-ui'
import { usePetStore } from '@/stores/pet'
import PetPanel from './PetPanel.vue'

const petStore = usePetStore()
const showPanel = ref(false)

const statusEmoji = computed(() => {
  if (!petStore.pet) return '🦊'
  if (petStore.pet.hunger > 80) return '😰'
  if (petStore.pet.mood > 80) return '😊'
  if (petStore.pet.mood < 30) return '😢'
  return '🦊'
})

onMounted(() => {
  petStore.fetchPet()
})
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
        :class="{ 'pet-corner--active': showPanel }"
        @click="showPanel = !showPanel"
      >
        <span class="pet-emoji">{{ statusEmoji }}</span>
        <span v-if="petStore.pet" class="pet-info">
          <span class="pet-name">{{ petStore.pet.name }}</span>
          <span class="pet-level">Lv.{{ petStore.pet.level }}</span>
        </span>
      </button>
    </template>

    <PetPanel @close="showPanel = false" />
  </n-popover>
</template>

<style scoped>
.pet-corner {
  display: flex;
  align-items: center;
  gap: 6px;
  width: 40px;
  height: 40px;
  border: none;
  border-radius: var(--sre-radius-md);
  background: transparent;
  cursor: pointer;
  padding: 0;
  overflow: hidden;
  transition:
    background var(--sre-duration-fast) var(--sre-ease-out),
    width var(--sre-duration-fast) var(--sre-ease-out);
}

.pet-corner:hover {
  background: var(--sre-bg-hover);
  width: auto;
  padding: 0 8px;
}

.pet-corner--active {
  background: var(--sre-bg-active);
  width: auto;
  padding: 0 8px;
}

.pet-emoji {
  font-size: 20px;
  flex-shrink: 0;
  width: 40px;
  height: 40px;
  display: flex;
  align-items: center;
  justify-content: center;
}

.pet-info {
  display: flex;
  flex-direction: column;
  white-space: nowrap;
}

.pet-name {
  font-size: 11px;
  font-weight: 600;
  color: var(--sre-text-primary);
  line-height: 1.2;
}

.pet-level {
  font-size: 10px;
  color: var(--sre-text-tertiary);
  line-height: 1.2;
}
</style>
