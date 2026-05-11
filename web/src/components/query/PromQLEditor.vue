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

function createExtensions() {
  // Enable basic PromQL completion (keyword + function completion)
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

onMounted(() => {
  if (!editorRef.value) return
  try {
    const state = EditorState.create({
      doc: props.modelValue,
      extensions: createExtensions(),
    })
    view.value = new EditorView({ state, parent: editorRef.value })
  } catch (e) {
    console.error('Failed to initialize PromQL editor:', e)
  }
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

watch(() => props.datasourceId, () => {
  if (view.value) {
    const doc = view.value.state.doc.toString()
    view.value.destroy()
    view.value = undefined
    if (editorRef.value) {
      try {
        const state = EditorState.create({
          doc,
          extensions: createExtensions(),
        })
        view.value = new EditorView({ state, parent: editorRef.value })
      } catch (e) {
        console.error('Failed to recreate PromQL editor:', e)
      }
    }
  }
})
</script>

<template>
  <div ref="editorRef" class="promql-editor" :class="{ disabled }" />
</template>

<style scoped>
.promql-editor {
  border: 1px solid var(--sre-border);
  border-radius: 6px;
  overflow: hidden;
  min-height: 42px;
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
}
.promql-editor :deep(.cm-content) {
  padding: 8px 12px;
}
</style>
