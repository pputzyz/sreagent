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
 * Composable for CRUD modal state management.
 * Provides reactive state for modal visibility, title, edit/create mode, and save loading.
 *
 * @deprecated Use useCrudPage instead for new pages. This composable is still used
 * by existing pages but should not be used for new development.
 *
 * @param onClose - Optional callback when modal closes successfully after save
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
