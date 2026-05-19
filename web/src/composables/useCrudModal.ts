import { ref, type Ref } from 'vue'

interface CrudModalReturn {
  showModal: Ref<boolean>
  modalTitle: Ref<string>
  editingId: Ref<number | null>
  saving: Ref<boolean>
  openCreate: (title: string) => void
  openEdit: (id: number, title: string) => void
  closeModal: () => void
  withSaving: <T>(fn: () => Promise<T>) => Promise<T | undefined>
}

/**
 * @deprecated Prefer useCrudPage (provides full list+modal+CRUD loop).
 *   useCrudModal only provides modal state; you still need manual list/refresh management.
 *   Planned for removal in v4.11.
 */
export function useCrudModal(onClose?: () => void): CrudModalReturn {
  const showModal = ref(false)
  const modalTitle = ref('')
  const editingId = ref<number | null>(null)
  const saving = ref(false)

  function openCreate(title: string) {
    editingId.value = null
    modalTitle.value = title
    showModal.value = true
  }

  function openEdit(id: number, title: string) {
    editingId.value = id
    modalTitle.value = title
    showModal.value = true
  }

  function closeModal() {
    showModal.value = false
    editingId.value = null
    modalTitle.value = ''
  }

  /**
   * Wraps an async save function with saving state management.
   * Sets saving=true before, false after. Closes modal and calls onClose on success.
   * Returns the result or undefined if an error was thrown.
   */
  async function withSaving<T>(fn: () => Promise<T>): Promise<T | undefined> {
    saving.value = true
    try {
      const result = await fn()
      showModal.value = false
      onClose?.()
      return result
    } finally {
      saving.value = false
    }
  }

  return {
    showModal,
    modalTitle,
    editingId,
    saving,
    openCreate,
    openEdit,
    closeModal,
    withSaving,
  }
}
