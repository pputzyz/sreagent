<script setup lang="ts">
import { ref, watch, onMounted, onUnmounted, shallowRef } from 'vue'
import { EditorView, keymap, placeholder as cmPlaceholder } from '@codemirror/view'
import { EditorState } from '@codemirror/state'
import { defaultKeymap, history, historyKeymap } from '@codemirror/commands'
import { oneDark } from '@codemirror/theme-one-dark'

const props = withDefaults(defineProps<{
  modelValue: string
  datasourceId?: number | null
  placeholder?: string
  disabled?: boolean
}>(), {
  datasourceId: null,
  placeholder: 'Enter LogsQL expression...',
  disabled: false,
})

const emit = defineEmits<{
  (e: 'update:modelValue', value: string): void
  (e: 'execute'): void
}>()

const editorRef = ref<HTMLDivElement>()
const view = shallowRef<EditorView>()

function createExtensions() {
  const exts = [
    history(),
    keymap.of([
      ...defaultKeymap,
      ...historyKeymap,
      { key: 'Ctrl-Enter', run: () => { emit('execute'); return true } },
      { key: 'Cmd-Enter', run: () => { emit('execute'); return true } },
    ]),
    cmPlaceholder(props.placeholder),
    EditorView.updateListener.of((update) => {
      if (update.docChanged) {
        emit('update:modelValue', update.state.doc.toString())
      }
    }),
    EditorView.lineWrapping,
    oneDark,
  ]
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
    console.error('Failed to initialize LogsQL editor:', e)
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
        console.error('Failed to recreate LogsQL editor:', e)
      }
    }
  }
})
</script>

<template>
  <div ref="editorRef" class="logsql-editor" :class="{ disabled }" />
</template>

<style scoped>
.logsql-editor {
  border: 1px solid var(--sre-border);
  border-radius: 6px;
  overflow: hidden;
  min-height: 42px;
}
.logsql-editor:focus-within {
  border-color: var(--sre-primary);
  box-shadow: 0 0 0 2px var(--sre-primary-soft);
}
.logsql-editor.disabled {
  opacity: 0.6;
  pointer-events: none;
}
.logsql-editor :deep(.cm-editor) {
  min-height: 42px;
  font-size: 13px;
}
.logsql-editor :deep(.cm-content) {
  padding: 8px 12px;
}
</style>
