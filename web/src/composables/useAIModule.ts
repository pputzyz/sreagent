import { ref } from 'vue'
import { aiModuleApi, aiApi } from '@/api'
import type { AIModuleConfig, AIProvidersConfig } from '@/types/ai-module'

const modules = ref<AIModuleConfig | null>(null)
const globalEnabled = ref(false)
const providers = ref<AIProvidersConfig | null>(null)

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

  async function loadProviders() {
    try {
      const res = await aiApi.getProviders()
      providers.value = res.data.data
    } catch {
      providers.value = null
    }
  }

  function isEnabled(module: keyof AIModuleConfig): boolean {
    if (!globalEnabled.value || !modules.value) return false
    return modules.value[module]?.enabled ?? false
  }

  /** Returns the provider key assigned to a module, or empty string for default. */
  function getProviderForModule(module: keyof AIModuleConfig): string {
    if (!modules.value) return ''
    return modules.value[module]?.provider_key ?? ''
  }

  /** Check if a specific provider is enabled. */
  function isProviderEnabled(providerKey: string): boolean {
    if (!providers.value || !providers.value.providers) return false
    const p = providers.value.providers.find(pr => pr.key === providerKey)
    return p?.enabled ?? false
  }

  return {
    modules,
    globalEnabled,
    providers,
    loadModules,
    loadProviders,
    isEnabled,
    getProviderForModule,
    isProviderEnabled,
  }
}
