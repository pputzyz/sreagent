<script setup lang="ts">
import { ref, computed, watch } from 'vue'
import { useI18n } from 'vue-i18n'

const { t } = useI18n()

const props = withDefaults(defineProps<{
  src?: string
  presetId?: string
  name?: string
  size?: number
  showRing?: boolean
}>(), {
  size: 32,
  showRing: false,
})

const colorPalettes = [
  { bg: '#F43F5E', fg: '#FFF' },
  { bg: '#3B82F6', fg: '#FFF' },
  { bg: '#8B5CF6', fg: '#FFF' },
  { bg: '#F59E0B', fg: '#FFF' },
  { bg: '#10B981', fg: '#FFF' },
  { bg: '#EC4899', fg: '#FFF' },
  { bg: '#0D9488', fg: '#FFF' },
  { bg: '#06B6D4', fg: '#FFF' },
]

const palette = computed(() => {
  const str = props.name || 'U'
  let hash = 0
  for (let i = 0; i < str.length; i++) {
    hash = str.charCodeAt(i) + ((hash << 5) - hash)
  }
  return colorPalettes[Math.abs(hash) % colorPalettes.length]
})

const initial = computed(() =>
  (props.name || 'U').charAt(0).toUpperCase()
)

const imgError = ref(false)
watch(() => props.src, () => { imgError.value = false })

// Preset SVG avatar IDs
const presetAvatars: Record<string, string> = {
  'engineer': 'engineer',
  'firefighter': 'firefighter',
  'detective': 'detective',
  'pilot': 'pilot',
  'scientist': 'scientist',
  'wizard': 'wizard',
  'ninja': 'ninja',
  'chef': 'chef',
  'astronaut': 'astronaut',
  'artist': 'artist',
  'doctor': 'doctor',
  'pirate': 'pirate',
}
</script>

<template>
  <div
    class="user-avatar"
    :class="{ 'user-avatar--ring': showRing }"
    :style="{ width: `${size}px`, height: `${size}px` }"
  >
    <!-- Uploaded image -->
    <img
      v-if="src && !imgError"
      :src="src"
      :alt="name || t('profile.avatar')"
      class="user-avatar-img"
      loading="lazy"
      @error="imgError = true"
    />

    <!-- Preset SVG avatars -->
    <svg
      v-else-if="presetId && presetAvatars[presetId]"
      :viewBox="'0 0 64 64'"
      xmlns="http://www.w3.org/2000/svg"
      class="user-avatar-svg"
    >
      <defs>
        <radialGradient id="avatar-shine" cx="40%" cy="35%" r="60%">
          <stop offset="0%" stop-color="#ffffff" stop-opacity="0.25"/>
          <stop offset="100%" stop-color="#ffffff" stop-opacity="0"/>
        </radialGradient>
      </defs>
      <!-- Engineer -->
      <template v-if="presetId === 'engineer'">
        <circle cx="32" cy="32" r="30" fill="#3B82F6"/>
        <circle cx="32" cy="28" r="12" fill="#FDE68A"/>
        <rect x="20" y="40" width="24" height="16" rx="4" fill="#1E40AF"/>
        <rect x="22" y="42" width="20" height="4" rx="2" fill="#3B82F6"/>
        <circle cx="26" cy="26" r="2" fill="#1F2937"/>
        <circle cx="38" cy="26" r="2" fill="#1F2937"/>
        <path d="M28 32 Q32 35 36 32" stroke="#1F2937" stroke-width="1.2" fill="none" stroke-linecap="round"/>
        <rect x="18" y="16" width="28" height="6" rx="3" fill="#1E40AF"/>
        <rect x="28" y="12" width="8" height="6" rx="2" fill="#F59E0B"/>
      </template>

      <!-- Firefighter -->
      <template v-else-if="presetId === 'firefighter'">
        <circle cx="32" cy="32" r="30" fill="#EF4444"/>
        <circle cx="32" cy="30" r="12" fill="#FDE68A"/>
        <path d="M18 22 Q20 10 32 10 Q44 10 46 22" fill="#DC2626"/>
        <rect x="20" y="42" width="24" height="14" rx="4" fill="#B91C1C"/>
        <rect x="28" y="42" width="8" height="14" rx="2" fill="#DC2626"/>
        <circle cx="26" cy="28" r="2" fill="#1F2937"/>
        <circle cx="38" cy="28" r="2" fill="#1F2937"/>
        <path d="M28 34 Q32 37 36 34" stroke="#1F2937" stroke-width="1.2" fill="none" stroke-linecap="round"/>
        <rect x="30" y="18" width="4" height="4" rx="1" fill="#F59E0B"/>
      </template>

      <!-- Detective -->
      <template v-else-if="presetId === 'detective'">
        <circle cx="32" cy="32" r="30" fill="#6B7280"/>
        <circle cx="32" cy="28" r="12" fill="#FDE68A"/>
        <path d="M16 24 Q18 14 32 14 Q46 14 48 24" fill="#374151"/>
        <ellipse cx="32" cy="22" rx="14" ry="4" fill="#4B5563"/>
        <rect x="20" y="40" width="24" height="16" rx="4" fill="#374151"/>
        <circle cx="26" cy="26" r="2" fill="#1F2937"/>
        <circle cx="38" cy="26" r="2" fill="#1F2937"/>
        <path d="M28 32 Q32 35 36 32" stroke="#1F2937" stroke-width="1.2" fill="none" stroke-linecap="round"/>
        <rect x="36" y="20" width="12" height="8" rx="4" fill="#1F2937" opacity="0.3"/>
      </template>

      <!-- Pilot -->
      <template v-else-if="presetId === 'pilot'">
        <circle cx="32" cy="32" r="30" fill="#1E40AF"/>
        <circle cx="32" cy="30" r="12" fill="#FDE68A"/>
        <path d="M16 26 Q18 12 32 12 Q46 12 48 26" fill="#1E3A8A"/>
        <rect x="20" y="42" width="24" height="14" rx="4" fill="#1E3A8A"/>
        <circle cx="26" cy="28" r="2" fill="#1F2937"/>
        <circle cx="38" cy="28" r="2" fill="#1F2937"/>
        <path d="M28 34 Q32 37 36 34" stroke="#1F2937" stroke-width="1.2" fill="none" stroke-linecap="round"/>
        <rect x="16" y="22" width="8" height="4" rx="2" fill="#F59E0B"/>
        <rect x="40" y="22" width="8" height="4" rx="2" fill="#F59E0B"/>
        <path d="M24 18 L32 14 L40 18" stroke="#F59E0B" stroke-width="1.5" fill="none"/>
      </template>

      <!-- Scientist -->
      <template v-else-if="presetId === 'scientist'">
        <circle cx="32" cy="32" r="30" fill="#8B5CF6"/>
        <circle cx="32" cy="28" r="12" fill="#FDE68A"/>
        <rect x="20" y="40" width="24" height="16" rx="4" fill="#F9FAFB"/>
        <circle cx="26" cy="26" r="3" fill="none" stroke="#1F2937" stroke-width="1.5"/>
        <circle cx="38" cy="26" r="3" fill="none" stroke="#1F2937" stroke-width="1.5"/>
        <line x1="29" y1="26" x2="35" y2="26" stroke="#1F2937" stroke-width="1"/>
        <path d="M28 32 Q32 35 36 32" stroke="#1F2937" stroke-width="1.2" fill="none" stroke-linecap="round"/>
        <path d="M20 18 Q24 8 32 8 Q40 8 44 18" fill="#6D28D9"/>
      </template>

      <!-- Wizard -->
      <template v-else-if="presetId === 'wizard'">
        <circle cx="32" cy="32" r="30" fill="#7C3AED"/>
        <circle cx="32" cy="30" r="10" fill="#FDE68A"/>
        <path d="M22 22 L32 4 L42 22" fill="#4C1D95"/>
        <polygon points="32,4 34,8 30,8" fill="#F59E0B"/>
        <rect x="20" y="42" width="24" height="14" rx="4" fill="#4C1D95"/>
        <circle cx="28" cy="28" r="1.5" fill="#1F2937"/>
        <circle cx="36" cy="28" r="1.5" fill="#1F2937"/>
        <path d="M29 32 Q32 34 35 32" stroke="#1F2937" stroke-width="1" fill="none" stroke-linecap="round"/>
        <circle cx="32" cy="10" r="2" fill="#F59E0B" opacity="0.8"/>
      </template>

      <!-- Ninja -->
      <template v-else-if="presetId === 'ninja'">
        <circle cx="32" cy="32" r="30" fill="#1F2937"/>
        <circle cx="32" cy="30" r="10" fill="#FDE68A"/>
        <rect x="18" y="24" width="28" height="6" rx="3" fill="#1F2937"/>
        <circle cx="26" cy="26" r="2" fill="white"/>
        <circle cx="38" cy="26" r="2" fill="white"/>
        <circle cx="26" cy="26" r="1" fill="#1F2937"/>
        <circle cx="38" cy="26" r="1" fill="#1F2937"/>
        <rect x="20" y="40" width="24" height="16" rx="4" fill="#374151"/>
        <path d="M44 24 L56 20 L52 28 Z" fill="#374151"/>
      </template>

      <!-- Chef -->
      <template v-else-if="presetId === 'chef'">
        <circle cx="32" cy="32" r="30" fill="#F9FAFB"/>
        <circle cx="32" cy="32" r="10" fill="#FDE68A"/>
        <path d="M20 26 Q20 10 32 10 Q44 10 44 26" fill="white" stroke="#E5E7EB" stroke-width="1"/>
        <path d="M24 26 L24 14 Q32 8 40 14 L40 26" fill="white"/>
        <circle cx="28" cy="30" r="1.5" fill="#1F2937"/>
        <circle cx="36" cy="30" r="1.5" fill="#1F2937"/>
        <path d="M30 34 Q32 36 34 34" stroke="#1F2937" stroke-width="1" fill="none" stroke-linecap="round"/>
        <rect x="22" y="42" width="20" height="14" rx="4" fill="#F9FAFB" stroke="#E5E7EB" stroke-width="1"/>
      </template>

      <!-- Astronaut -->
      <template v-else-if="presetId === 'astronaut'">
        <circle cx="32" cy="32" r="30" fill="#E5E7EB"/>
        <circle cx="32" cy="28" r="14" fill="#F9FAFB" stroke="#D1D5DB" stroke-width="1.5"/>
        <circle cx="32" cy="28" r="10" fill="#3B82F6" opacity="0.3"/>
        <circle cx="28" cy="26" r="1.5" fill="#1F2937"/>
        <circle cx="36" cy="26" r="1.5" fill="#1F2937"/>
        <path d="M30 31 Q32 33 34 31" stroke="#1F2937" stroke-width="1" fill="none" stroke-linecap="round"/>
        <rect x="22" y="42" width="20" height="14" rx="4" fill="#D1D5DB"/>
        <rect x="26" y="44" width="12" height="4" rx="2" fill="#3B82F6" opacity="0.5"/>
      </template>

      <!-- Artist -->
      <template v-else-if="presetId === 'artist'">
        <circle cx="32" cy="32" r="30" fill="#EC4899"/>
        <circle cx="32" cy="28" r="12" fill="#FDE68A"/>
        <path d="M18 22 Q20 12 32 12 Q44 12 46 22" fill="#BE185D"/>
        <rect x="20" y="40" width="24" height="16" rx="4" fill="#BE185D"/>
        <circle cx="26" cy="26" r="2" fill="#1F2937"/>
        <circle cx="38" cy="26" r="2" fill="#1F2937"/>
        <path d="M28 32 Q32 35 36 32" stroke="#1F2937" stroke-width="1.2" fill="none" stroke-linecap="round"/>
        <path d="M46 16 L54 8 L50 14 Z" fill="#F59E0B"/>
        <circle cx="54" cy="8" r="3" fill="#EF4444"/>
      </template>

      <!-- Doctor -->
      <template v-else-if="presetId === 'doctor'">
        <circle cx="32" cy="32" r="30" fill="#3B82F6"/>
        <circle cx="32" cy="28" r="12" fill="#FDE68A"/>
        <rect x="20" y="40" width="24" height="16" rx="4" fill="white"/>
        <rect x="28" y="42" width="8" height="4" rx="1" fill="#EF4444"/>
        <rect x="30" y="40" width="4" height="8" rx="1" fill="#EF4444"/>
        <circle cx="26" cy="26" r="2" fill="#1F2937"/>
        <circle cx="38" cy="26" r="2" fill="#1F2937"/>
        <path d="M28 32 Q32 35 36 32" stroke="#1F2937" stroke-width="1.2" fill="none" stroke-linecap="round"/>
        <path d="M20 20 Q22 10 32 10 Q42 10 44 20" fill="white"/>
      </template>

      <!-- Pirate -->
      <template v-else-if="presetId === 'pirate'">
        <circle cx="32" cy="32" r="30" fill="#92400E"/>
        <circle cx="32" cy="30" r="10" fill="#FDE68A"/>
        <path d="M16 24 Q18 10 32 10 Q46 10 48 24" fill="#1F2937"/>
        <circle cx="26" cy="28" r="2" fill="#1F2937"/>
        <circle cx="38" cy="28" r="3" fill="none" stroke="#1F2937" stroke-width="1.5"/>
        <circle cx="38" cy="28" r="1" fill="#1F2937"/>
        <path d="M28 33 Q32 36 36 33" stroke="#1F2937" stroke-width="1" fill="none" stroke-linecap="round"/>
        <rect x="20" y="40" width="24" height="16" rx="4" fill="#78350F"/>
        <path d="M46 18 L52 14 L50 22 Z" fill="#1F2937"/>
      </template>

      <!-- Default fallback for unknown preset -->
      <template v-else>
        <circle cx="32" cy="32" r="30" :fill="palette.bg"/>
        <text x="32" y="40" text-anchor="middle" :fill="palette.fg" font-size="24" font-weight="600">{{ initial }}</text>
      </template>
    </svg>

    <!-- Colorful initial (default when no image or preset) -->
    <div
      v-else
      class="user-avatar-initial"
      :style="{
        background: `linear-gradient(135deg, ${palette.bg}, ${palette.bg}dd)`,
        color: palette.fg,
        fontSize: `${Math.round(size * 0.4)}px`,
      }"
    >
      {{ initial }}
    </div>
  </div>
</template>

<style scoped>
.user-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border-radius: 50%;
  overflow: hidden;
  flex-shrink: 0;
  position: relative;
}

.user-avatar--ring {
  box-shadow: 0 0 0 2px var(--sre-bg-card), 0 0 0 3px var(--sre-primary);
}

.user-avatar-img {
  width: 100%;
  height: 100%;
  object-fit: cover;
  border-radius: 50%;
}

.user-avatar-svg {
  width: 100%;
  height: 100%;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.1));
}

.user-avatar-svg circle:first-of-type {
  stroke: rgba(255, 255, 255, 0.15);
  stroke-width: 1;
}

.user-avatar-initial {
  width: 100%;
  height: 100%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-weight: 700;
  line-height: 1;
  text-transform: uppercase;
}
</style>
