<script setup lang="ts">
import { computed } from 'vue'
import type { PetType } from '@/stores/pet'

const props = withDefaults(defineProps<{
  type?: PetType
  size?: number
  animated?: boolean
}>(), {
  type: 'fox',
  size: 48,
  animated: true,
})

const viewBox = '0 0 64 64'
</script>

<template>
  <div
    class="pet-avatar"
    :class="{ 'pet-avatar--animated': animated }"
    :style="{ width: `${size}px`, height: `${size}px` }"
  >
    <!-- Fox -->
    <svg v-if="type === 'fox'" :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <defs>
        <radialGradient id="fox-body" cx="50%" cy="45%" r="50%">
          <stop offset="0%" stop-color="#F59E0B"/>
          <stop offset="100%" stop-color="#D97706"/>
        </radialGradient>
        <radialGradient id="fox-belly" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#FEF3C7"/>
          <stop offset="100%" stop-color="#FDE68A"/>
        </radialGradient>
        <radialGradient id="fox-bg" cx="50%" cy="40%" r="55%">
          <stop offset="0%" stop-color="#FEF3C7"/>
          <stop offset="100%" stop-color="#FDE68A"/>
        </radialGradient>
      </defs>
      <circle cx="32" cy="32" r="30" fill="url(#fox-bg)" opacity="0.4"/>
      <!-- Ears -->
      <polygon points="14,8 22,22 6,22" fill="#D97706"/>
      <polygon points="50,8 58,22 42,22" fill="#D97706"/>
      <polygon points="15,12 20,21 10,21" fill="#FEF3C7" opacity="0.6"/>
      <polygon points="49,12 54,21 44,21" fill="#FEF3C7" opacity="0.6"/>
      <!-- Body -->
      <ellipse cx="32" cy="40" rx="20" ry="18" fill="url(#fox-body)"/>
      <!-- Belly -->
      <ellipse cx="32" cy="44" rx="12" ry="10" fill="url(#fox-belly)"/>
      <!-- Face -->
      <ellipse cx="32" cy="32" rx="16" ry="14" fill="#F59E0B"/>
      <!-- White face patches -->
      <ellipse cx="24" cy="34" rx="5" ry="4" fill="#FEF3C7" opacity="0.7"/>
      <ellipse cx="40" cy="34" rx="5" ry="4" fill="#FEF3C7" opacity="0.7"/>
      <!-- Eyes -->
      <g class="pet-eyes">
        <ellipse cx="24" cy="30" rx="3" ry="3.5" fill="#1F2937"/>
        <ellipse cx="40" cy="30" rx="3" ry="3.5" fill="#1F2937"/>
        <ellipse cx="25" cy="29" rx="1.2" ry="1.2" fill="white"/>
        <ellipse cx="41" cy="29" rx="1.2" ry="1.2" fill="white"/>
      </g>
      <!-- Nose -->
      <ellipse cx="32" cy="35" rx="2.5" ry="2" fill="#1F2937"/>
      <!-- Mouth -->
      <path d="M29 37 Q32 40 35 37" stroke="#1F2937" stroke-width="1" fill="none" stroke-linecap="round"/>
      <!-- Whiskers -->
      <line x1="12" y1="33" x2="22" y2="34" stroke="#D97706" stroke-width="0.8" opacity="0.5"/>
      <line x1="12" y1="36" x2="22" y2="36" stroke="#D97706" stroke-width="0.8" opacity="0.5"/>
      <line x1="52" y1="33" x2="42" y2="34" stroke="#D97706" stroke-width="0.8" opacity="0.5"/>
      <line x1="52" y1="36" x2="42" y2="36" stroke="#D97706" stroke-width="0.8" opacity="0.5"/>
      <!-- Tail -->
      <path d="M48 50 Q60 42 56 30 Q54 24 50 28 Q48 32 52 40 Q50 46 48 50" fill="#D97706" class="pet-tail"/>
      <path d="M50 30 Q52 26 52 30 Q50 36 52 40" fill="#FEF3C7" opacity="0.5"/>
    </svg>

    <!-- Cat -->
    <svg v-else-if="type === 'cat'" :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <defs>
        <radialGradient id="cat-body" cx="50%" cy="45%" r="50%">
          <stop offset="0%" stop-color="#94A3B8"/>
          <stop offset="100%" stop-color="#64748B"/>
        </radialGradient>
      </defs>
      <circle cx="32" cy="32" r="30" fill="#E2E8F0" opacity="0.35"/>
      <!-- Ears -->
      <polygon points="14,6 22,20 6,18" fill="#64748B"/>
      <polygon points="50,6 58,18 42,20" fill="#64748B"/>
      <polygon points="15,10 20,18 10,17" fill="#FDA4AF" opacity="0.5"/>
      <polygon points="49,10 54,17 44,18" fill="#FDA4AF" opacity="0.5"/>
      <!-- Body -->
      <ellipse cx="32" cy="42" rx="18" ry="16" fill="url(#cat-body)"/>
      <!-- Belly -->
      <ellipse cx="32" cy="46" rx="10" ry="8" fill="#E2E8F0" opacity="0.6"/>
      <!-- Face -->
      <ellipse cx="32" cy="32" rx="15" ry="13" fill="#94A3B8"/>
      <!-- Eyes -->
      <g class="pet-eyes">
        <ellipse cx="24" cy="30" rx="3.5" ry="4" fill="#10B981"/>
        <ellipse cx="40" cy="30" rx="3.5" ry="4" fill="#10B981"/>
        <ellipse cx="24" cy="30" rx="1.5" ry="3.5" fill="#1F2937"/>
        <ellipse cx="40" cy="30" rx="1.5" ry="3.5" fill="#1F2937"/>
        <ellipse cx="25" cy="29" rx="0.8" ry="0.8" fill="white"/>
        <ellipse cx="41" cy="29" rx="0.8" ry="0.8" fill="white"/>
      </g>
      <!-- Nose -->
      <polygon points="32,34 30,36 34,36" fill="#FDA4AF"/>
      <!-- Mouth -->
      <path d="M29 37 Q32 39 35 37" stroke="#475569" stroke-width="0.8" fill="none" stroke-linecap="round"/>
      <!-- Whiskers -->
      <line x1="8" y1="32" x2="21" y2="34" stroke="#94A3B8" stroke-width="0.7" opacity="0.6"/>
      <line x1="8" y1="36" x2="21" y2="36" stroke="#94A3B8" stroke-width="0.7" opacity="0.6"/>
      <line x1="56" y1="32" x2="43" y2="34" stroke="#94A3B8" stroke-width="0.7" opacity="0.6"/>
      <line x1="56" y1="36" x2="43" y2="36" stroke="#94A3B8" stroke-width="0.7" opacity="0.6"/>
      <!-- Tail -->
      <path d="M46 52 Q58 48 60 36 Q60 30 56 32 Q54 36 56 44 Q54 50 46 52" fill="#64748B" class="pet-tail"/>
    </svg>

    <!-- Owl -->
    <svg v-else-if="type === 'owl'" :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <defs>
        <radialGradient id="owl-body" cx="50%" cy="45%" r="50%">
          <stop offset="0%" stop-color="#92400E"/>
          <stop offset="100%" stop-color="#78350F"/>
        </radialGradient>
      </defs>
      <circle cx="32" cy="32" r="30" fill="#FEF3C7" opacity="0.35"/>
      <!-- Ear tufts -->
      <polygon points="16,8 22,18 10,16" fill="#78350F"/>
      <polygon points="48,8 54,16 42,18" fill="#78350F"/>
      <!-- Body -->
      <ellipse cx="32" cy="38" rx="18" ry="20" fill="url(#owl-body)"/>
      <!-- Chest feathers -->
      <ellipse cx="32" cy="42" rx="10" ry="12" fill="#B45309" opacity="0.5"/>
      <ellipse cx="32" cy="44" rx="7" ry="8" fill="#D97706" opacity="0.3"/>
      <!-- Face disc -->
      <ellipse cx="32" cy="30" rx="14" ry="12" fill="#D97706" opacity="0.4"/>
      <!-- Eyes -->
      <g class="pet-eyes">
        <circle cx="24" cy="28" r="6" fill="#FEF3C7"/>
        <circle cx="40" cy="28" r="6" fill="#FEF3C7"/>
        <circle cx="24" cy="28" r="4" fill="#F59E0B"/>
        <circle cx="40" cy="28" r="4" fill="#F59E0B"/>
        <circle cx="24" cy="28" r="2.5" fill="#1F2937"/>
        <circle cx="40" cy="28" r="2.5" fill="#1F2937"/>
        <circle cx="25" cy="27" r="1" fill="white"/>
        <circle cx="41" cy="27" r="1" fill="white"/>
      </g>
      <!-- Beak -->
      <polygon points="32,32 29,36 35,36" fill="#F59E0B"/>
      <!-- Wings -->
      <path d="M14 36 Q8 42 10 52 Q12 56 16 54 Q14 46 16 40" fill="#78350F" opacity="0.7"/>
      <path d="M50 36 Q56 42 54 52 Q52 56 48 54 Q50 46 48 40" fill="#78350F" opacity="0.7"/>
      <!-- Feet -->
      <path d="M26 56 L24 60 L28 60 L26 56" fill="#F59E0B"/>
      <path d="M38 56 L36 60 L40 60 L38 56" fill="#F59E0B"/>
    </svg>

    <!-- Panda -->
    <svg v-else-if="type === 'panda'" :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <circle cx="32" cy="32" r="30" fill="#F3F4F6" opacity="0.4"/>
      <!-- Ears -->
      <circle cx="16" cy="14" r="7" fill="#1F2937"/>
      <circle cx="48" cy="14" r="7" fill="#1F2937"/>
      <!-- Body -->
      <ellipse cx="32" cy="42" rx="18" ry="16" fill="#F9FAFB"/>
      <!-- Arms -->
      <ellipse cx="14" cy="44" rx="6" ry="8" fill="#1F2937" transform="rotate(-15 14 44)"/>
      <ellipse cx="50" cy="44" rx="6" ry="8" fill="#1F2937" transform="rotate(15 50 44)"/>
      <!-- Face -->
      <ellipse cx="32" cy="30" rx="16" ry="14" fill="#F9FAFB"/>
      <!-- Eye patches -->
      <ellipse cx="23" cy="28" rx="6" ry="5" fill="#1F2937" transform="rotate(-10 23 28)"/>
      <ellipse cx="41" cy="28" rx="6" ry="5" fill="#1F2937" transform="rotate(10 41 28)"/>
      <!-- Eyes -->
      <g class="pet-eyes">
        <ellipse cx="23" cy="28" rx="2.5" ry="3" fill="white"/>
        <ellipse cx="41" cy="28" rx="2.5" ry="3" fill="white"/>
        <ellipse cx="23" cy="28" rx="1.5" ry="2" fill="#1F2937"/>
        <ellipse cx="41" cy="28" rx="1.5" ry="2" fill="#1F2937"/>
        <ellipse cx="24" cy="27" rx="0.7" ry="0.7" fill="white"/>
        <ellipse cx="42" cy="27" rx="0.7" ry="0.7" fill="white"/>
      </g>
      <!-- Nose -->
      <ellipse cx="32" cy="34" rx="3" ry="2" fill="#1F2937"/>
      <!-- Mouth -->
      <path d="M29 36 Q32 38 35 36" stroke="#1F2937" stroke-width="0.8" fill="none" stroke-linecap="round"/>
      <!-- Blush -->
      <ellipse cx="19" cy="34" rx="3" ry="2" fill="#FDA4AF" opacity="0.3"/>
      <ellipse cx="45" cy="34" rx="3" ry="2" fill="#FDA4AF" opacity="0.3"/>
      <!-- Feet -->
      <ellipse cx="24" cy="56" rx="5" ry="4" fill="#1F2937"/>
      <ellipse cx="40" cy="56" rx="5" ry="4" fill="#1F2937"/>
    </svg>

    <!-- Tiger -->
    <svg v-else-if="type === 'tiger'" :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <defs>
        <radialGradient id="tiger-body" cx="50%" cy="45%" r="50%">
          <stop offset="0%" stop-color="#F59E0B"/>
          <stop offset="100%" stop-color="#D97706"/>
        </radialGradient>
      </defs>
      <circle cx="32" cy="32" r="30" fill="#FEF3C7" opacity="0.35"/>
      <!-- Ears -->
      <polygon points="14,6 22,18 6,16" fill="#D97706"/>
      <polygon points="50,6 58,16 42,18" fill="#D97706"/>
      <polygon points="15,10 19,16 11,15" fill="#FEF3C7" opacity="0.5"/>
      <polygon points="49,10 53,15 45,16" fill="#FEF3C7" opacity="0.5"/>
      <!-- Body -->
      <ellipse cx="32" cy="42" rx="18" ry="16" fill="url(#tiger-body)"/>
      <!-- Belly -->
      <ellipse cx="32" cy="46" rx="10" ry="8" fill="#FEF3C7" opacity="0.6"/>
      <!-- Face -->
      <ellipse cx="32" cy="30" rx="16" ry="14" fill="#F59E0B"/>
      <!-- Stripes -->
      <path d="M20 22 Q22 18 24 22" stroke="#92400E" stroke-width="2" fill="none" stroke-linecap="round"/>
      <path d="M26 20 Q28 16 30 20" stroke="#92400E" stroke-width="2" fill="none" stroke-linecap="round"/>
      <path d="M34 20 Q36 16 38 20" stroke="#92400E" stroke-width="2" fill="none" stroke-linecap="round"/>
      <path d="M40 22 Q42 18 44 22" stroke="#92400E" stroke-width="2" fill="none" stroke-linecap="round"/>
      <!-- Side stripes -->
      <path d="M16 28 Q14 26 16 24" stroke="#92400E" stroke-width="1.5" fill="none" stroke-linecap="round"/>
      <path d="M48 28 Q50 26 48 24" stroke="#92400E" stroke-width="1.5" fill="none" stroke-linecap="round"/>
      <!-- White face patches -->
      <ellipse cx="24" cy="32" rx="5" ry="4" fill="#FEF3C7" opacity="0.7"/>
      <ellipse cx="40" cy="32" rx="5" ry="4" fill="#FEF3C7" opacity="0.7"/>
      <!-- Eyes -->
      <g class="pet-eyes">
        <ellipse cx="24" cy="28" rx="3" ry="3.5" fill="#1F2937"/>
        <ellipse cx="40" cy="28" rx="3" ry="3.5" fill="#1F2937"/>
        <ellipse cx="25" cy="27" rx="1.2" ry="1.2" fill="white"/>
        <ellipse cx="41" cy="27" rx="1.2" ry="1.2" fill="white"/>
      </g>
      <!-- Nose -->
      <ellipse cx="32" cy="33" rx="3" ry="2.5" fill="#FDA4AF"/>
      <!-- Mouth -->
      <path d="M28 36 Q32 39 36 36" stroke="#1F2937" stroke-width="1" fill="none" stroke-linecap="round"/>
      <!-- Whiskers -->
      <line x1="10" y1="32" x2="21" y2="33" stroke="#D97706" stroke-width="0.8" opacity="0.5"/>
      <line x1="10" y1="36" x2="21" y2="36" stroke="#D97706" stroke-width="0.8" opacity="0.5"/>
      <line x1="54" y1="32" x2="43" y2="33" stroke="#D97706" stroke-width="0.8" opacity="0.5"/>
      <line x1="54" y1="36" x2="43" y2="36" stroke="#D97706" stroke-width="0.8" opacity="0.5"/>
    </svg>

    <!-- Bunny -->
    <svg v-else-if="type === 'bunny'" :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <defs>
        <radialGradient id="bunny-body" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#F9FAFB"/>
          <stop offset="100%" stop-color="#E5E7EB"/>
        </radialGradient>
      </defs>
      <circle cx="32" cy="32" r="30" fill="#FCE7F3" opacity="0.3"/>
      <!-- Ears -->
      <ellipse cx="22" cy="12" rx="5" ry="14" fill="#F9FAFB" transform="rotate(-8 22 12)"/>
      <ellipse cx="42" cy="12" rx="5" ry="14" fill="#F9FAFB" transform="rotate(8 42 12)"/>
      <ellipse cx="22" cy="12" rx="3" ry="10" fill="#FDA4AF" opacity="0.4" transform="rotate(-8 22 12)"/>
      <ellipse cx="42" cy="12" rx="3" ry="10" fill="#FDA4AF" opacity="0.4" transform="rotate(8 42 12)"/>
      <!-- Body -->
      <ellipse cx="32" cy="46" rx="16" ry="14" fill="url(#bunny-body)"/>
      <!-- Face -->
      <ellipse cx="32" cy="32" rx="14" ry="12" fill="#F9FAFB"/>
      <!-- Eyes -->
      <g class="pet-eyes">
        <ellipse cx="25" cy="30" rx="3" ry="3.5" fill="#E11D48"/>
        <ellipse cx="39" cy="30" rx="3" ry="3.5" fill="#E11D48"/>
        <ellipse cx="25" cy="30" rx="1.5" ry="3" fill="#1F2937"/>
        <ellipse cx="39" cy="30" rx="1.5" ry="3" fill="#1F2937"/>
        <ellipse cx="26" cy="29" rx="0.8" ry="0.8" fill="white"/>
        <ellipse cx="40" cy="29" rx="0.8" ry="0.8" fill="white"/>
      </g>
      <!-- Nose -->
      <ellipse cx="32" cy="34" rx="2" ry="1.5" fill="#FDA4AF"/>
      <!-- Mouth -->
      <path d="M30 36 L32 38 L34 36" stroke="#9CA3AF" stroke-width="0.8" fill="none" stroke-linecap="round"/>
      <!-- Cheeks -->
      <ellipse cx="20" cy="34" rx="3" ry="2" fill="#FDA4AF" opacity="0.25"/>
      <ellipse cx="44" cy="34" rx="3" ry="2" fill="#FDA4AF" opacity="0.25"/>
      <!-- Tail -->
      <circle cx="46" cy="52" r="4" fill="#F9FAFB"/>
      <circle cx="46" cy="52" r="2.5" fill="#E5E7EB" opacity="0.5"/>
    </svg>

    <!-- Dragon -->
    <svg v-else-if="type === 'dragon'" :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <defs>
        <radialGradient id="dragon-body" cx="50%" cy="45%" r="50%">
          <stop offset="0%" stop-color="#8B5CF6"/>
          <stop offset="100%" stop-color="#7C3AED"/>
        </radialGradient>
        <radialGradient id="dragon-belly" cx="50%" cy="50%" r="50%">
          <stop offset="0%" stop-color="#EDE9FE"/>
          <stop offset="100%" stop-color="#DDD6FE"/>
        </radialGradient>
      </defs>
      <circle cx="32" cy="32" r="30" fill="#EDE9FE" opacity="0.35"/>
      <!-- Horns -->
      <polygon points="18,10 22,20 14,18" fill="#F59E0B"/>
      <polygon points="46,10 50,18 42,20" fill="#F59E0B"/>
      <!-- Wings -->
      <path d="M10 30 Q2 20 6 12 Q10 8 14 14 Q12 20 14 28" fill="#7C3AED" opacity="0.5" class="pet-wing-l"/>
      <path d="M54 30 Q62 20 58 12 Q54 8 50 14 Q52 20 50 28" fill="#7C3AED" opacity="0.5" class="pet-wing-r"/>
      <!-- Body -->
      <ellipse cx="32" cy="42" rx="16" ry="16" fill="url(#dragon-body)"/>
      <!-- Belly -->
      <ellipse cx="32" cy="44" rx="10" ry="10" fill="url(#dragon-belly)"/>
      <!-- Face -->
      <ellipse cx="32" cy="30" rx="14" ry="12" fill="#8B5CF6"/>
      <!-- Eyes -->
      <g class="pet-eyes">
        <ellipse cx="25" cy="28" rx="3.5" ry="3" fill="#F59E0B"/>
        <ellipse cx="39" cy="28" rx="3.5" ry="3" fill="#F59E0B"/>
        <ellipse cx="25" cy="28" rx="1.5" ry="2.5" fill="#1F2937"/>
        <ellipse cx="39" cy="28" rx="1.5" ry="2.5" fill="#1F2937"/>
        <ellipse cx="26" cy="27" rx="0.7" ry="0.7" fill="white"/>
        <ellipse cx="40" cy="27" rx="0.7" ry="0.7" fill="white"/>
      </g>
      <!-- Nostrils -->
      <circle cx="29" cy="34" r="1.5" fill="#6D28D9"/>
      <circle cx="35" cy="34" r="1.5" fill="#6D28D9"/>
      <!-- Spikes on back -->
      <polygon points="24,22 26,16 28,22" fill="#F59E0B" opacity="0.7"/>
      <polygon points="30,20 32,14 34,20" fill="#F59E0B" opacity="0.7"/>
      <polygon points="36,22 38,16 40,22" fill="#F59E0B" opacity="0.7"/>
      <!-- Tail -->
      <path d="M46 50 Q56 52 58 46 Q60 42 56 44 Q54 46 56 50 Q54 54 46 52" fill="#7C3AED" class="pet-tail"/>
      <polygon points="58,44 62,42 60,46" fill="#F59E0B" opacity="0.8"/>
    </svg>

    <!-- Penguin -->
    <svg v-else-if="type === 'penguin'" :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <circle cx="32" cy="32" r="30" fill="#DBEAFE" opacity="0.3"/>
      <!-- Body -->
      <ellipse cx="32" cy="38" rx="18" ry="22" fill="#1F2937"/>
      <!-- Belly -->
      <ellipse cx="32" cy="42" rx="12" ry="16" fill="#F9FAFB"/>
      <!-- Face -->
      <ellipse cx="32" cy="26" rx="14" ry="12" fill="#1F2937"/>
      <!-- White face patches -->
      <ellipse cx="24" cy="26" rx="5" ry="4" fill="#F9FAFB"/>
      <ellipse cx="40" cy="26" rx="5" ry="4" fill="#F9FAFB"/>
      <!-- Eyes -->
      <g class="pet-eyes">
        <ellipse cx="24" cy="26" rx="2.5" ry="3" fill="#1F2937"/>
        <ellipse cx="40" cy="26" rx="2.5" ry="3" fill="#1F2937"/>
        <ellipse cx="25" cy="25" rx="1" ry="1" fill="white"/>
        <ellipse cx="41" cy="25" rx="1" ry="1" fill="white"/>
      </g>
      <!-- Beak -->
      <polygon points="32,30 28,34 36,34" fill="#F59E0B"/>
      <line x1="28" y1="34" x2="36" y2="34" stroke="#D97706" stroke-width="0.5"/>
      <!-- Wings -->
      <path d="M14 34 Q8 42 10 52 Q12 56 16 52 Q14 44 16 38" fill="#1F2937"/>
      <path d="M50 34 Q56 42 54 52 Q52 56 48 52 Q50 44 48 38" fill="#1F2937"/>
      <!-- Feet -->
      <ellipse cx="26" cy="58" rx="5" ry="3" fill="#F59E0B"/>
      <ellipse cx="38" cy="58" rx="5" ry="3" fill="#F59E0B"/>
      <!-- Blush -->
      <ellipse cx="19" cy="30" rx="2.5" ry="1.5" fill="#FDA4AF" opacity="0.3"/>
      <ellipse cx="45" cy="30" rx="2.5" ry="1.5" fill="#FDA4AF" opacity="0.3"/>
    </svg>

    <!-- Default fallback -->
    <svg v-else :viewBox="viewBox" xmlns="http://www.w3.org/2000/svg" class="pet-svg">
      <circle cx="32" cy="32" r="28" fill="#E5E7EB"/>
      <text x="32" y="38" text-anchor="middle" font-size="24">?</text>
    </svg>
  </div>
</template>

<style scoped>
.pet-avatar {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}

.pet-svg {
  width: 100%;
  height: 100%;
  filter: drop-shadow(0 2px 4px rgba(0, 0, 0, 0.12)) drop-shadow(0 1px 2px rgba(0, 0, 0, 0.06));
}

.pet-avatar--animated .pet-eyes {
  animation: pet-blink 4s ease-in-out infinite;
}

.pet-avatar--animated .pet-tail {
  animation: pet-wag 2s ease-in-out infinite;
  transform-origin: center bottom;
}

.pet-avatar--animated .pet-wing-l,
.pet-avatar--animated .pet-wing-r {
  animation: pet-wing-flap 3s ease-in-out infinite;
}

.pet-avatar--animated:hover .pet-svg {
  animation: pet-bounce 400ms var(--ease-out-quart, cubic-bezier(0.25, 1, 0.5, 1));
}

@keyframes pet-blink {
  0%, 92%, 100% { transform: scaleY(1); }
  95% { transform: scaleY(0.1); }
}

@keyframes pet-wag {
  0%, 100% { transform: rotate(0deg); }
  25% { transform: rotate(8deg); }
  75% { transform: rotate(-8deg); }
}

@keyframes pet-wing-flap {
  0%, 100% { transform: rotate(0deg); }
  50% { transform: rotate(-5deg); }
}

@keyframes pet-bounce {
  0% { transform: scale(1); }
  40% { transform: scale(1.1); }
  70% { transform: scale(0.97); }
  100% { transform: scale(1); }
}

@media (prefers-reduced-motion: reduce) {
  .pet-avatar--animated .pet-eyes,
  .pet-avatar--animated .pet-tail,
  .pet-avatar--animated .pet-wing-l,
  .pet-avatar--animated .pet-wing-r {
    animation: none;
  }
  .pet-avatar--animated:hover .pet-svg {
    animation: none;
  }
}
</style>
