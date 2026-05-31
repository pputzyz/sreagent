import { ref } from 'vue'
import { aiModuleApi, aiApi } from '@/api'
import type { AIModuleConfig, AIProvidersConfig } from '@/types/ai-module'

// Design note: Module-level singleton used instead of Pinia store for simplicity.
// AI module config is fetched once and cached; this composable is not visible in Pinia devtools.
// The reset function is called on logout to clear stale cached config.
const modules = ref<AIModuleConfig | null>(null)
const globalEnabled = ref(false)
const providers = ref<AIProvidersConfig | null>(null)

/** Reset module-level singleton state (call on logout) */
export function resetAIModule() {
  modules.value = null
  globalEnabled.value = false
  providers.value = null
}

export function useAIModule() {
  async function loadModules() {
    try {
      const res = await aiModuleApi.getModules()
      modules.value = res.data.data
      globalEnabled.value = true
    } catch (err) {
      console.warn('[useAIModule] Failed to load AI modules:', err)
      globalEnabled.value = false
    }
  }

  async function loadProviders() {
    try {
      const res = await aiApi.getProviders()
      providers.value = res.data.data
    } catch (err) {
      console.warn('[useAIModule] Failed to load AI providers:', err)
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
