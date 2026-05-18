import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { petApi } from '@/api'
import { getErrorMessage } from '@/utils/format'
import type { Pet, PetInteraction } from '@/types'
import i18n from '@/i18n'

export type PetType = 'fox' | 'cat' | 'owl' | 'panda' | 'tiger' | 'bunny' | 'dragon' | 'penguin'

const petEmojiMap: Record<PetType, string> = {
  fox: '\u{1F98A}',
  cat: '\u{1F431}',
  owl: '\u{1F989}',
  panda: '\u{1F43C}',
  tiger: '\u{1F42F}',
  bunny: '\u{1F430}',
  dragon: '\u{1F409}',
  penguin: '\u{1F427}',
}

export const usePetStore = defineStore('pet', () => {
  const pet = ref<Pet | null>(null)
  const petType = ref<PetType>((localStorage.getItem('sre-pet-type') as PetType) || 'fox')
  const interactions = ref<PetInteraction[]>([])
  const loading = ref(false)
  const error = ref<string | null>(null)

  const petEmoji = computed(() => petEmojiMap[petType.value] || '\u{1F98A}')
  const expForNextLevel = computed(() => (pet.value?.level || 1) * 100)
  const expProgress = computed(() => {
    if (!pet.value) return 0
    return Math.min((pet.value.exp / expForNextLevel.value) * 100, 100)
  })
  const hungerPercent = computed(() => pet.value?.hunger ?? 0)
  const moodPercent = computed(() => pet.value?.mood ?? 0)

  async function fetchPet() {
    loading.value = true
    error.value = null
    try {
      const resp = await petApi.get()
      pet.value = resp.data.data
    } catch (e: unknown) {
      error.value = getErrorMessage(e) || i18n.global.t('pet.loadFailed')
    } finally {
      loading.value = false
    }
  }

  async function updateName(name: string) {
    const resp = await petApi.update({ name })
    pet.value = resp.data.data
  }

  async function feed() {
    const resp = await petApi.feed()
    pet.value = resp.data.data
  }

  async function play() {
    const resp = await petApi.play()
    pet.value = resp.data.data
  }

  async function fetchInteractions() {
    try {
      const resp = await petApi.getInteractions()
      interactions.value = resp.data.data || []
    } catch (e: unknown) {
      error.value = getErrorMessage(e) || i18n.global.t('pet.loadInteractionsFailed')
    }
  }

  function setPetType(type: PetType) {
    petType.value = type
    localStorage.setItem('sre-pet-type', type)
  }

  return {
    pet,
    petType,
    petEmoji,
    interactions,
    loading,
    error,
    expForNextLevel,
    expProgress,
    hungerPercent,
    moodPercent,
    fetchPet,
    updateName,
    feed,
    play,
    fetchInteractions,
    setPetType,
  }
})
