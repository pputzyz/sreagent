import type { Directive } from 'vue'

const RIPPLE_HANDLER_KEY = Symbol('_rippleHandler')

/**
 * v-ripple — Click ripple effect directive.
 * On mousedown, creates a <span class="sre-ripple-wave"> that expands from the click point.
 * Auto-removes after animation ends.
 */
export const vRipple: Directive<HTMLElement> = {
  mounted(el) {
    el.style.position = el.style.position || 'relative'
    el.style.overflow = el.style.overflow || 'hidden'

    const handler = (e: MouseEvent) => {
      const rect = el.getBoundingClientRect()
      const x = e.clientX - rect.left
      const y = e.clientY - rect.top

      const wave = document.createElement('span')
      wave.className = 'sre-ripple-wave'
      wave.style.setProperty('--ripple-x', `${x}px`)
      wave.style.setProperty('--ripple-y', `${y}px`)
      el.appendChild(wave)

      wave.addEventListener('animationend', () => {
        wave.remove()
      })
    }

    // Store handler reference for cleanup in unmounted
    ;(el as any)[RIPPLE_HANDLER_KEY] = handler
    el.addEventListener('mousedown', handler)
  },

  unmounted(el) {
    const handler = (el as any)[RIPPLE_HANDLER_KEY]
    if (handler) {
      el.removeEventListener('mousedown', handler)
      delete (el as any)[RIPPLE_HANDLER_KEY]
    }
  },
}
