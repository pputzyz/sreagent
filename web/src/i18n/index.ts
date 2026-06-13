import { watch } from 'vue'
import { createI18n } from 'vue-i18n'
import zhCN from './zh-CN'
import en from './en'

const savedLocale = localStorage.getItem('locale') || 'zh-CN'

const i18n = createI18n({
  legacy: false,
  locale: savedLocale,
  fallbackLocale: 'en',
  messages: {
    'zh-CN': zhCN,
    en,
  },
})

// Keep <html lang> and persisted locale in sync with the active locale. Locale-aware
// formatting (e.g. utils/format.ts → toLocaleString) reads document.documentElement.lang,
// so without this, dates/times stayed in the initial locale even after switching language.
function applyLocaleSideEffects(locale: string) {
  document.documentElement.setAttribute('lang', locale)
  localStorage.setItem('locale', locale)
}
applyLocaleSideEffects(savedLocale)
watch(
  () => i18n.global.locale.value,
  (locale) => applyLocaleSideEffects(locale),
)

export default i18n
