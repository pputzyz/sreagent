<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'

type MascotState = 'wave' | 'idle' | 'sleep'

const state = ref<MascotState>('wave')
const showMascot = ref(localStorage.getItem('sre-hide-mascot') !== 'true')
let inactivityTimer: ReturnType<typeof setTimeout>

function resetInactivity() {
  clearTimeout(inactivityTimer)
  if (state.value === 'sleep') state.value = 'idle'
  inactivityTimer = setTimeout(() => { state.value = 'sleep' }, 5 * 60 * 1000)
}

function handleClick() {
  // Easter egg: cycle states
  if (state.value === 'wave') state.value = 'idle'
  else if (state.value === 'idle') state.value = 'sleep'
  else state.value = 'idle'
}

onMounted(() => {
  setTimeout(() => { if (state.value === 'wave') state.value = 'idle' }, 2000)
  document.addEventListener('mousemove', resetInactivity)
  document.addEventListener('keydown', resetInactivity)
  resetInactivity()
})

onUnmounted(() => {
  clearTimeout(inactivityTimer)
  document.removeEventListener('mousemove', resetInactivity)
  document.removeEventListener('keydown', resetInactivity)
})
</script>

<template>
  <div
    v-if="showMascot"
    class="mascot-container"
    :class="`mascot-${state}`"
    :title="state === 'sleep' ? 'Sleeping...' : 'Hi!'"
    @click="handleClick"
  >
    <svg class="mascot-svg" viewBox="0 0 32 32" xmlns="http://www.w3.org/2000/svg" aria-hidden="true">
      <!-- Ears -->
      <g class="mascot-ear">
        <polygon points="6,4 12,14 0,14" fill="#f59e0b"/>
        <polygon points="26,4 32,14 20,14" fill="#f59e0b"/>
        <polygon points="7,6 11,13 3,13" fill="#0d9488" opacity="0.6"/>
        <polygon points="25,6 29,13 21,13" fill="#0d9488" opacity="0.6"/>
      </g>
      <!-- Face -->
      <g class="mascot-body">
        <ellipse cx="16" cy="18" rx="12" ry="11" fill="#f59e0b"/>
        <ellipse cx="11" cy="20" rx="4" ry="3" fill="white" opacity="0.7"/>
        <ellipse cx="21" cy="20" rx="4" ry="3" fill="white" opacity="0.7"/>
        <!-- Eyes change with state -->
        <template v-if="state === 'sleep'">
          <path d="M9 17 Q11 15 13 17" stroke="#1c1917" stroke-width="1" fill="none"/>
          <path d="M19 17 Q21 15 23 17" stroke="#1c1917" stroke-width="1" fill="none"/>
        </template>
        <template v-else>
          <circle cx="11" cy="17" r="1.5" fill="#1c1917"/>
          <circle cx="21" cy="17" r="1.5" fill="#1c1917"/>
          <circle cx="11.5" cy="16.5" r="0.5" fill="white"/>
          <circle cx="21.5" cy="16.5" r="0.5" fill="white"/>
        </template>
        <ellipse cx="16" cy="21" rx="1.5" ry="1" fill="#1c1917"/>
        <path d="M14 23 Q16 25 18 23" stroke="#1c1917" stroke-width="0.7" fill="none"/>
      </g>
      <!-- Wave paw (only in wave state) -->
      <g v-if="state === 'wave'" class="mascot-paw">
        <ellipse cx="28" cy="12" rx="3" ry="2" fill="#fbbf24" transform="rotate(-20 28 12)"/>
      </g>
      <!-- Zzz (only in sleep state) -->
      <g v-if="state === 'sleep'" class="mascot-zzz-group">
        <text class="mascot-zzz" x="24" y="8" font-size="6" fill="#0d9488" opacity="0">z</text>
        <text class="mascot-zzz" x="27" y="5" font-size="5" fill="#0d9488" opacity="0">z</text>
        <text class="mascot-zzz" x="29" y="2" font-size="4" fill="#0d9488" opacity="0">z</text>
      </g>
    </svg>
  </div>
</template>

<style scoped>
.mascot-container {
  width: 32px;
  height: 32px;
  cursor: pointer;
  transition: transform var(--sre-duration-base) var(--sre-ease-out);
}

.mascot-container:hover {
  transform: scale(1.15);
}

.mascot-svg {
  width: 100%;
  height: 100%;
}

/* Idle: subtle ear twitch */
.mascot-idle .mascot-ear {
  animation: mascot-ear-twitch 4s ease-in-out infinite;
}

@keyframes mascot-ear-twitch {
  0%, 92%, 100% { transform: rotate(0deg); transform-origin: 16px 14px; }
  95% { transform: rotate(5deg); transform-origin: 16px 14px; }
  97% { transform: rotate(-3deg); transform-origin: 16px 14px; }
}

/* Wave: paw movement */
.mascot-wave .mascot-paw {
  animation: mascot-wave-paw 0.6s ease-in-out 3;
  transform-origin: 28px 14px;
}

@keyframes mascot-wave-paw {
  0%, 100% { transform: rotate(0deg); }
  25% { transform: rotate(15deg); }
  75% { transform: rotate(-10deg); }
}

/* Sleep: breathing + Zzz */
.mascot-sleep .mascot-body {
  animation: mascot-breathe 3s ease-in-out infinite;
}

@keyframes mascot-breathe {
  0%, 100% { transform: scaleY(1); transform-origin: 16px 24px; }
  50% { transform: scaleY(0.97); transform-origin: 16px 24px; }
}

.mascot-zzz {
  animation: mascot-zzz-float 2s ease-in-out infinite;
}

.mascot-zzz:nth-child(1) { animation-delay: 0s; }
.mascot-zzz:nth-child(2) { animation-delay: 0.7s; }
.mascot-zzz:nth-child(3) { animation-delay: 1.4s; }

@keyframes mascot-zzz-float {
  0% { opacity: 0; transform: translate(0, 0) scale(0.8); }
  20% { opacity: 0.6; }
  100% { opacity: 0; transform: translate(8px, -16px) scale(1); }
}
</style>
