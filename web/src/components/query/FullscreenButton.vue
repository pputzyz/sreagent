<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { NButton, NIcon } from 'naive-ui'
import { ExpandOutline, ContractOutline } from '@vicons/ionicons5'

const props = defineProps<{
  targetRef?: HTMLElement | null
}>()

const isFullscreen = ref(false)

function toggle() {
  const el = props.targetRef || document.documentElement
  if (!document.fullscreenElement) {
    el.requestFullscreen?.()
  } else {
    document.exitFullscreen?.()
  }
}

function onChange() {
  isFullscreen.value = !!document.fullscreenElement
}

onMounted(() => document.addEventListener('fullscreenchange', onChange))
onUnmounted(() => document.removeEventListener('fullscreenchange', onChange))
</script>

<template>
  <NButton size="small" quaternary @click="toggle">
    <template #icon>
      <NIcon :size="16">
        <ContractOutline v-if="isFullscreen" />
        <ExpandOutline v-else />
      </NIcon>
    </template>
  </NButton>
</template>
