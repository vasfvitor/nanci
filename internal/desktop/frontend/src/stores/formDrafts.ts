import { ref } from 'vue'
import { defineStore } from 'pinia'

// FormDraft is a form closed without saving: value is the form as JSON and
// seed is the JSON of the starting values it was typed over.
export type FormDraft = { seed: string; value: string }

// The store keeps the unsaved dialog forms for the session, in memory only,
// by form key (such as "company:add" or "company:edit:<CNPJ>").
export const useFormDraftsStore = defineStore('formDrafts', () => {
  const drafts = ref<Record<string, FormDraft>>({})

  function get(key: string): FormDraft | undefined {
    return drafts.value[key]
  }

  function set(key: string, draft: FormDraft) {
    drafts.value[key] = draft
  }

  function drop(key: string) {
    drafts.value = Object.fromEntries(
      Object.entries(drafts.value).filter(([draftKey]) => draftKey !== key)
    )
  }

  return { drafts, get, set, drop }
})
