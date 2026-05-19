/**
 * v-can directive — conditional rendering based on RBAC permissions.
 *
 * Usage:
 *   v-can="'rules.create'"           — single permission
 *   v-can="['rules.create','rules.edit']" — any-of (OR)
 *
 * Elements are removed from DOM when the user lacks permission.
 */
import type { Directive, DirectiveBinding } from 'vue'
import { usePermissions } from '@/composables/usePermissions'

function check(el: HTMLElement, binding: DirectiveBinding<string | string[]>) {
  const { hasPerm, hasAnyPerm } = usePermissions()
  const perms = Array.isArray(binding.value) ? binding.value : [binding.value]
  const ok = perms.length === 1 ? hasPerm(perms[0]) : hasAnyPerm(...perms)

  if (!ok) {
    // Replace element with invisible placeholder to preserve layout flow
    el.parentNode?.removeChild(el)
  }
}

export const vCan: Directive = {
  mounted: check,
  updated: check,
}
