import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { petApi } from '@/api'
import type { Pet, PetInteraction } from '@/types'

export const usePetStore = defineStore('pet', () => {
  const pet = ref<Pet | null>(null)
  const interactions = ref<PetInteraction[]>([])
  const loading = ref(false)

  const expForNextLevel = computed(() => (pet.value?.level || 1) * 100)
  const expProgress = computed(() => {
    if (!pet.value) return 0
    return Math.min((pet.value.exp / expForNextLevel.value) * 100, 100)
  })

  async function fetchPet() {
    loading.value = true
    try {
      const resp = await petApi.get()
      pet.value = resp.data.data
    } catch {
      /* ignore */
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
    } catch {
      /* ignore */
    }
  }

  return {
    pet,
    interactions,
    loading,
    expForNextLevel,
    expProgress,
    fetchPet,
    updateName,
    feed,
    play,
    fetchInteractions,
  }
})
