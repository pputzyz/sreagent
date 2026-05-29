<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, shallowRef } from 'vue'
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { completionKeymap, autocompletion } from '@codemirror/autocomplete'
import { PromQLExtension } from '@prometheus-io/codemirror-promql'
import { oneDark } from '@codemirror/theme-one-dark'

const props = withDefaults(defineProps<{
  modelValue: string
  datasourceId?: number | null
  placeholder?: string
  disabled?: boolean
  dark?: boolean
}>(), {
  datasourceId: null,
  placeholder: 'Enter PromQL expression...',
  disabled: false,
  dark: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'execute'): void
}>()

const editorRef = ref<HTMLDivElement>()
const view = shallowRef<EditorView>()

const promQLExt = new PromQLExtension()

/** fetchFn that attaches JWT auth and forwards to the datasource proxy */
function authFetch(input: RequestInfo | URL, init?: RequestInit): Promise<Response> {
  const token = localStorage.getItem('token')
  const headers = new Headers(init?.headers)
  if (token) headers.set('Authorization', `Bearer ${token}`)
  return fetch(input, { ...init, headers })
}

/** Configure remote completion from the datasource proxy endpoint */
function configureRemoteCompletion(dsId: number | null | undefined) {
  if (dsId) {
    promQLExt.setComplete({
      remote: {
        url: `/api/v1/datasources/${dsId}/proxy`,
        httpMethod: 'GET',
        fetchFn: authFetch,
      },
    })
  } else {
    // No datasource — offline-only completion (functions, operators)
    promQLExt.setComplete({})
  }
}

function createExtensions() {
  promQLExt.activateCompletion(true)
  const exts = [
    history(),
    keymap.of([
      ...defaultKeymap,
      ...historyKeymap,
      ...completionKeymap,
      { key: 'Ctrl-Enter', run: () => { emit('execute'); return true } },
      { key: 'Cmd-Enter', run: () => { emit('execute'); return true } },
    ]),
    promQLExt.asExtension(),
    autocompletion(),
    cmPlaceholder(props.placeholder),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        emit('update:modelValue', update.state.doc.toString())
      }
    }),
    EditorView.lineWrapping,
  ]
  if (props.dark) exts.push(oneDark)
  if (props.disabled) exts.push(EditorState.readOnly.of(true))
  return exts
}

function recreateView() {
  if (!editorRef.value) return
  const doc = view.value ? view.value.state.doc.toString() : props.modelValue
  view.value?.destroy()
  view.value = undefined
  try {
    const state = EditorState.create({
      doc,
      extensions: createExtensions(),
    })
    view.value = new EditorView({ state, parent: editorRef.value })
  } catch (e) {
    console.error('Failed to (re)create PromQL editor:', e)
  }
}

onMounted(() => {
  configureRemoteCompletion(props.datasourceId)
  recreateView()
})

onUnmounted(() => {
  view.value?.destroy()
})

watch(() => props.modelValue, (val) => {
  if (view.value && view.value.state.doc.toString() !== val) {
    view.value.dispatch({
      changes: { from: 0, to: view.value.state.doc.length, insert: val },
    })
  }
})

watch(() => props.datasourceId, (newId) => {
  configureRemoteCompletion(newId)
  recreateView()
})
</script>

<template>
  <div ref="editorRef" class="promql-editor" :class="{ disabled }" />
</template>

<style scoped>
.promql-editor {
  border: 1px solid var(--sre-border);
  border-radius: 6px;
  overflow: clip;
  min-height: 42px;
  width: 100%;
  cursor: text;
}
.promql-editor:focus-within {
  border-color: var(--sre-primary);
  box-shadow: 0 0 0 2px var(--sre-primary-soft);
}
.promql-editor.disabled {
  opacity: 0.6;
  pointer-events: none;
}
.promql-editor :deep(.cm-editor) {
  min-height: 42px;
  font-size: 13px;
  width: 100%;
  cursor: text;
}
.promql-editor :deep(.cm-content) {
  padding: 8px 12px;
  cursor: text;
}
.promql-editor :deep(.cm-line) {
  cursor: text;
}
</style>
