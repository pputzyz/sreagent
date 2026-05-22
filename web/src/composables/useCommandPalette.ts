import { ref, computed } from 'vue'
import { useRouter } from 'vue-router'
import { useI18n } from 'vue-i18n'

export interface PaletteItem {
  id: string
  label: string
  hint?: string
  group: 'navigate' | 'action' | 'recent'
  icon?: string     // ionicons5 name string — rendered by caller
  action: () => void
}

const visible = ref(false)
const query = ref('')

// Registry: actions registered by external composables/components
const registeredActions = ref<PaletteItem[]>([])

/** Reset module-level singleton state (call on logout) */
export function resetCommandPalette() {
  registeredActions.value = []
  query.value = ''
  visible.value = false
}

export function useCommandPalette() {
  const router = useRouter()
  const { t } = useI18n()

  function open() {
    query.value = ''
    visible.value = true
  }
  function close() { visible.value = false }
  function toggle() { visible.value ? close() : open() }

  // ── Navigate items (all app routes) ──────────────────────────────────
  const navigateItems = computed<PaletteItem[]>(() => [
    // On-Call
    { id: 'nav-overview',      label: t('commandPalette.oncallOverview'),    hint: t('commandPalette.hintOncall'), group: 'navigate', icon: 'grid-outline',            action: () => router.push('/oncall/overview') },
    { id: 'nav-schedule',      label: t('commandPalette.oncallSchedule'),    hint: t('commandPalette.hintOncall'), group: 'navigate', icon: 'calendar-outline',        action: () => router.push('/oncall/schedule') },
    { id: 'nav-spaces',        label: t('commandPalette.spaces'),           hint: t('commandPalette.hintOncall'), group: 'navigate', icon: 'people-outline',          action: () => router.push('/oncall/spaces') },
    { id: 'nav-incidents',     label: t('commandPalette.incidents'),        hint: t('commandPalette.hintOncall'), group: 'navigate', icon: 'warning-outline',         action: () => router.push('/oncall/incidents') },
    { id: 'nav-integrations',  label: t('commandPalette.integrations'),     hint: t('commandPalette.hintOncall'), group: 'navigate', icon: 'link-outline',            action: () => router.push('/oncall/integrations') },
    // Alert
    { id: 'nav-datasources',   label: t('commandPalette.dataSources'),      hint: t('commandPalette.hintAlert'),  group: 'navigate', icon: 'server-outline',          action: () => router.push('/alert/datasources') },
    { id: 'nav-explore',       label: t('commandPalette.dataQuery'),        hint: t('commandPalette.hintAlert'),  group: 'navigate', icon: 'search-outline',          action: () => router.push('/alert/explore') },
    { id: 'nav-dashboards',    label: t('commandPalette.dashboards'),       hint: t('commandPalette.hintAlert'),  group: 'navigate', icon: 'bar-chart-outline',       action: () => router.push('/alert/dashboards') },
    { id: 'nav-rules',         label: t('commandPalette.alertRules'),       hint: t('commandPalette.hintAlert'),  group: 'navigate', icon: 'alert-circle-outline',    action: () => router.push('/alert/rules') },
    { id: 'nav-events',        label: t('commandPalette.activeAlerts'),     hint: t('commandPalette.hintAlert'),  group: 'navigate', icon: 'flash-outline',           action: () => router.push('/alert/events') },
    { id: 'nav-history',       label: t('commandPalette.alertHistory'),     hint: t('commandPalette.hintAlert'),  group: 'navigate', icon: 'time-outline',            action: () => router.push('/alert/history') },
    { id: 'nav-suppression',   label: t('commandPalette.muteRules'),        hint: t('commandPalette.hintAlert'),  group: 'navigate', icon: 'volume-mute-outline',     action: () => router.push('/alert/suppression') },
    { id: 'nav-inhibition',    label: t('commandPalette.inhibitionRules'),  hint: t('commandPalette.hintAlert'),  group: 'navigate', icon: 'shield-outline',          action: () => router.push('/alert/suppression/inhibition') },
    { id: 'nav-notification',  label: t('commandPalette.notificationPolicies'), hint: t('commandPalette.hintAlert'), group: 'navigate', icon: 'notifications-outline',  action: () => router.push('/alert/notify/policies') },
    // Platform
    { id: 'nav-profile',       label: t('commandPalette.profile'),          hint: t('commandPalette.hintPlatform'), group: 'navigate', icon: 'settings-outline',        action: () => router.push('/platform/profile') },
  ])

  // ── Recent (last 5 navigations from localStorage) ────────────────────
  const RECENT_KEY = 'sre-cmd-recent'
  function getRecent(): PaletteItem[] {
    try {
      const ids: string[] = JSON.parse(localStorage.getItem(RECENT_KEY) || '[]')
      return ids
        .map(id => navigateItems.value.find(i => i.id === id))
        .filter(Boolean)
        .map(i => ({ ...i!, group: 'recent' as const }))
    } catch { return [] }
  }

  function pushRecent(id: string) {
    try {
      const ids: string[] = JSON.parse(localStorage.getItem(RECENT_KEY) || '[]')
      const next = [id, ...ids.filter(x => x !== id)].slice(0, 5)
      localStorage.setItem(RECENT_KEY, JSON.stringify(next))
    } catch { /**/ }
  }

  function runItem(item: PaletteItem) {
    if (item.group === 'navigate' || item.group === 'recent') {
      pushRecent(item.id)
    }
    close()
    item.action()
  }

  // ── Fuzzy filter ─────────────────────────────────────────────────────
  function score(text: string, q: string): number {
    const t = text.toLowerCase()
    const ql = q.toLowerCase()
    if (!ql) return 1
    if (t === ql) return 100
    if (t.startsWith(ql)) return 80
    if (t.includes(ql)) return 60
    // word-boundary: any word starts with q
    const words = t.split(/[\s\-_/]+/)
    if (words.some(w => w.startsWith(ql))) return 50
    // character subsequence
    let ci = 0
    for (const ch of ql) {
      const idx = t.indexOf(ch, ci)
      if (idx === -1) return 0
      ci = idx + 1
    }
    return 20
  }

  const filteredItems = computed(() => {
    const q = query.value.trim()
    const recent = getRecent()

    if (!q) {
      return {
        recent: recent.slice(0, 5),
        navigate: navigateItems.value.slice(0, 8),
        action: registeredActions.value.slice(0, 6),
      }
    }

    const filter = (items: PaletteItem[]) =>
      items
        .map(i => ({ item: i, s: Math.max(score(i.label, q), score(i.hint || '', q)) }))
        .filter(x => x.s > 0)
        .sort((a, b) => b.s - a.s)
        .map(x => x.item)

    return {
      recent: [],
      navigate: filter(navigateItems.value),
      action: filter(registeredActions.value),
    }
  })

  function registerAction(item: Omit<PaletteItem, 'group'>) {
    // De-dup: CommandPalette.vue's onMounted registers built-in actions on
    // every remount (HMR, route remount). Without this guard the actions
    // list doubled/tripled every hot reload.
    if (registeredActions.value.some(a => a.id === item.id)) return
    registeredActions.value.push({ ...item, group: 'action' })
  }

  return {
    visible,
    query,
    open,
    close,
    toggle,
    filteredItems,
    runItem,
    registerAction,
  }
}
