<script setup lang="ts">
/**
 * FieldValueToken — Tokenized field value with click-to-filter context menu.
 *
 * When segmented=false (default): renders the entire value as a clickable span.
 * When segmented=true: splits the value by common delimiters and renders each
 * token as a clickable span with delimiters as plain text.
 *
 * Popover menu offers: Copy (key:value), Copy Value, Filter AND, Filter NOT.
 */
import { ref, computed } from 'vue'
import { NPopover, NIcon, NDivider } from 'naive-ui'
import { CopyOutline, AddCircleOutline, RemoveCircleOutline } from '@vicons/ionicons5'

const props = withDefaults(defineProps<{
  fieldKey: string
  fieldValue: string
  segmented?: boolean
}>(), {
  segmented: false,
})

const emit = defineEmits<{
  (e: 'filter', key: string, value: string, operator: 'AND' | 'NOT'): void
  (e: 'copy', text: string): void
}>()

// --- Segmented tokenization ---
interface Token {
  text: string
  isDelimiter: boolean
}

const DELIMITER_RE = /([ ,=|;:]+)/
const MAX_TOKENS = 100

const tokens = computed<Token[]>(() => {
  if (!props.segmented) return [{ text: props.fieldValue, isDelimiter: false }]
  const parts = props.fieldValue.split(DELIMITER_RE)
  if (parts.length / 2 > MAX_TOKENS) {
    // Too many tokens — fall back to non-segmented rendering
    return [{ text: props.fieldValue, isDelimiter: false }]
  }
  return parts
    .filter(p => p.length > 0)
    .map(p => ({
      text: p,
      isDelimiter: DELIMITER_RE.test(p),
    }))
})

const useSegmented = computed(() => {
  if (!props.segmented) return false
  // Check if fallback was triggered
  return tokens.value.length > 1 || !tokens.value[0]?.isDelimiter
})

// --- Popover state ---
const popoverVisible = ref(false)
const activeTokenValue = ref('')
const activeTokenIndex = ref(-1)

function openMenu(tokenValue: string, tokenIndex: number) {
  activeTokenValue.value = tokenValue
  activeTokenIndex.value = tokenIndex
  popoverVisible.value = true
}

function closeMenu() {
  popoverVisible.value = false
  activeTokenIndex.value = -1
}

// --- Menu actions ---
function copyKeyValue() {
  const text = `${props.fieldKey}:${activeTokenValue.value}`
  navigator.clipboard?.writeText(text)
  emit('copy', text)
  closeMenu()
}

function copyValue() {
  const text = activeTokenValue.value
  navigator.clipboard?.writeText(text)
  emit('copy', text)
  closeMenu()
}

function filterAnd() {
  emit('filter', props.fieldKey, activeTokenValue.value, 'AND')
  closeMenu()
}

function filterNot() {
  emit('filter', props.fieldKey, activeTokenValue.value, 'NOT')
  closeMenu()
}
</script>

<template>
  <span class="field-value-token-root">
    <!-- Non-segmented: single clickable value -->
    <template v-if="!useSegmented">
      <NPopover
        :show="popoverVisible"
        trigger="manual"
        placement="bottom-start"
        :show-arrow="false"
        @update:show="(v: boolean) => { if (!v) closeMenu() }"
      >
        <template #trigger>
          <span
            class="fv-token"
            @click.stop="openMenu(fieldValue, 0)"
          >
            {{ fieldValue }}
          </span>
        </template>
        <div class="fv-menu">
          <button class="fv-menu-item" @click="copyKeyValue">
            <NIcon size="14"><CopyOutline /></NIcon>
            <span>Copy</span>
          </button>
          <button class="fv-menu-item" @click="copyValue">
            <NIcon size="14"><CopyOutline /></NIcon>
            <span>Copy Value</span>
          </button>
          <NDivider style="margin: 4px 0;" />
          <button class="fv-menu-item" @click="filterAnd">
            <NIcon size="14" color="var(--sre-primary)"><AddCircleOutline /></NIcon>
            <span>Filter AND</span>
          </button>
          <button class="fv-menu-item" @click="filterNot">
            <NIcon size="14" color="#e88080"><RemoveCircleOutline /></NIcon>
            <span>Filter NOT</span>
          </button>
        </div>
      </NPopover>
    </template>

    <!-- Segmented: each token is individually clickable -->
    <template v-else>
      <template v-for="(token, idx) in tokens" :key="idx">
        <span v-if="token.isDelimiter" class="fv-delimiter">{{ token.text }}</span>
        <NPopover
          v-else
          :show="popoverVisible && activeTokenIndex === idx"
          trigger="manual"
          placement="bottom-start"
          :show-arrow="false"
          @update:show="(v: boolean) => { if (!v) closeMenu() }"
        >
          <template #trigger>
            <span
              class="fv-token"
              @click.stop="openMenu(token.text, idx)"
            >
              {{ token.text }}
            </span>
          </template>
          <div class="fv-menu">
            <button class="fv-menu-item" @click="copyKeyValue">
              <NIcon size="14"><CopyOutline /></NIcon>
              <span>Copy</span>
            </button>
            <button class="fv-menu-item" @click="copyValue">
              <NIcon size="14"><CopyOutline /></NIcon>
              <span>Copy Value</span>
            </button>
            <NDivider style="margin: 4px 0;" />
            <button class="fv-menu-item" @click="filterAnd">
              <NIcon size="14" color="var(--sre-primary)"><AddCircleOutline /></NIcon>
              <span>Filter AND</span>
            </button>
            <button class="fv-menu-item" @click="filterNot">
              <NIcon size="14" color="#e88080"><RemoveCircleOutline /></NIcon>
              <span>Filter NOT</span>
            </button>
          </div>
        </NPopover>
      </template>
    </template>
  </span>
</template>

<style scoped>
.field-value-token-root {
  display: inline;
  font-family: var(--sre-font-mono, monospace);
  font-size: 12px;
  color: var(--sre-text-primary);
  word-break: break-all;
}

/* Clickable token */
.fv-token {
  cursor: pointer;
  border-radius: 2px;
  padding: 0 1px;
  transition: background-color 0.15s;
}

.fv-token:hover {
  text-decoration: underline;
  background-color: var(--sre-bg-sunken, rgba(0, 0, 0, 0.04));
}

/* Delimiter plain text */
.fv-delimiter {
  color: var(--sre-text-secondary);
  user-select: none;
}

/* Popover menu */
.fv-menu {
  display: flex;
  flex-direction: column;
  min-width: 140px;
  padding: 4px 0;
}

.fv-menu-item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  height: 28px;
  padding: 0 10px;
  border: none;
  background: transparent;
  color: var(--sre-text-primary);
  font-size: 12px;
  cursor: pointer;
  transition: background-color 0.12s;
  text-align: left;
}

.fv-menu-item:hover {
  background-color: var(--sre-bg-sunken, rgba(0, 0, 0, 0.04));
}

.fv-menu-item:active {
  background-color: var(--sre-bg-sunken, rgba(0, 0, 0, 0.08));
}
</style>
