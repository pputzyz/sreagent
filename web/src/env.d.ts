/// <reference types="vite/client" />

declare const __APP_VERSION__: string

declare module '*.vue' {
  import type { DefineComponent } from 'vue'
  const component: DefineComponent<Record<string, unknown>, Record<string, unknown>, unknown>
  export default component
}

declare module 'vue-virtual-scroller' {
  import type { DefineComponent, Plugin } from 'vue'

  export interface DynamicScrollerProps {
    items: unknown[]
    keyField?: string
    direction?: 'vertical' | 'horizontal'
    listTag?: string
    itemTag?: string
    minItemSize?: number
  }

  export interface DynamicScrollerItemProps {
    item: unknown
    active: boolean
    sizeDependencies?: unknown[]
    watchData?: boolean
    tag?: string
    emitResize?: boolean
    onResize?: () => void
  }

  export const DynamicScroller: DefineComponent<DynamicScrollerProps>
  export const DynamicScrollerItem: DefineComponent<DynamicScrollerItemProps>
  export const RecycleScroller: DefineComponent<Record<string, unknown>>
  export const IdState: () => Record<string, unknown>
  const plugin: Plugin
  export default plugin
}
