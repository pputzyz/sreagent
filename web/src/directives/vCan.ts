/**
 * v-can directive — conditional rendering based on RBAC permissions.
 *
 * Usage:
 *   v-can="'rules.create'"           — single permission
 *   v-can="['rules.create','rules.edit']" — any-of (OR)
 *
 * Elements are hidden via display:none when the user lacks permission.
 * Uses hidden attribute to avoid removing Vue-managed DOM nodes.
 *
 * When permissions finish loading asynchronously, all tracked elements
 * are re-evaluated so that initially-hidden elements become visible
 * if the user actually has the required permission.
 */
import type { Directive, DirectiveBinding } from 'vue'
import { watch } from 'vue'
import { usePermissions } from '@/composables/usePermissions'

// Track elements pending re-evaluation after async permission load
const pendingElements = new Map<HTMLElement, DirectiveBinding<string | string[]>>()
let unwatch: (() => void) | null = null

function applyCheck(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
  const { hasPerm, hasAnyPerm } = usePermissions()
  const perms = Array.isArray(binding.value) ? binding.value : [binding.value]
  const ok = perms.length === 1 ? hasPerm(perms[0]) : hasAnyPerm(...perms)

  if (!ok) {
    el.hidden = true
    el.style.pointerEvents = 'none'
  } else {
    el.hidden = false
    el.style.pointerEvents = ''
  }
}

function check(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
  const { loaded } = usePermissions()

  applyCheck(el, binding)

  // If permissions haven't loaded yet, track this element for re-evaluation
  if (!loaded.value) {
    pendingElements.set(el, binding)

    // Set up a single global watcher for when permissions finish loading
    if (!unwatch) {
      unwatch = watch(loaded, (isLoaded) => {
        if (isLoaded) {
          // Re-evaluate all pending elements
          for (const [pendingEl, pendingBinding] of pendingElements) {
            applyCheck(pendingEl, pendingBinding)
          }
          pendingElements.clear()
          if (unwatch) {
            unwatch()
            unwatch = null
          }
        }
      })
    }
  } else {
    // Permissions already loaded — remove from pending if present
    pendingElements.delete(el)
  }
}

export const vCan: Directive = {
  mounted: check,
  updated: check,
  beforeUnmount(el) {
    pendingElements.delete(el)
  },
}
