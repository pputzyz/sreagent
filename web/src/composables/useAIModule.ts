import { ref } from 'vue'
import { aiModuleApi } from '@/api/preset-rules'
import type { AIModuleConfig } from '@/types/preset-rule'

const modules = ref<AIModuleConfig | null>(null)
const globalEnabled = ref(false)

export function useAIModule() {
  async function loadModules() {
    try {
      const res = await aiModuleApi.getModules()
      modules.value = res.data.data
      globalEnabled.value = true
    } catch {
      globalEnabled.value = false
    }
  }

  function isEnabled(module: keyof AIModuleConfig): boolean {
    if (!globalEnabled.value || !modules.value) return false
    return modules.value[module]?.enabled ?? false
  }

  function isAIAvailable(): boolean {
    return globalEnabled.value
  }

  return { modules, globalEnabled, loadModules, isEnabled, isAIAvailable }
}
