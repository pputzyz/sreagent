/**
 * v-can directive — conditional rendering based on RBAC permissions.
 *
 * Usage:
 *   v-can="'rules.create'"           — single permission
 *   v-can="['rules.create','rules.edit']" — any-of (OR)
 *
 * Elements are hidden via display:none when the user lacks permission.
 * Uses hidden attribute to avoid removing Vue-managed DOM nodes.
 */
import type { Directive, DirectiveBinding } from 'vue'
import { usePermissions } from '@/composables/usePermissions'

function check(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
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

export const vCan: Directive = {
  mounted: check,
  updated: check,
}
