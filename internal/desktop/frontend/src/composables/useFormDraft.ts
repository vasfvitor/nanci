import { computed, ref, toValue, type MaybeRefOrGetter, type Ref } from 'vue'
import { useFormDraftsStore } from '@/stores/formDrafts'

// useFormDraft holds a dialog form whose unsaved values survive closing the
// dialog, for the session. seed builds the starting values; a form that still
// equals them is not dirty and leaves no draft. With dropWhenSeedChanges, a
// draft typed over other starting values (the record changed since) is
// dropped instead of restored.
//
// The dialog calls open() when it opens, keep() when it closes without
// saving, done() after a save and clear() when the user discards the draft.
export function useFormDraft<T extends object>(
  key: MaybeRefOrGetter<string>,
  seed: () => T,
  opts: { dropWhenSeedChanges?: boolean } = {}
) {
  const store = useFormDraftsStore()
  const baseline = ref(JSON.stringify(seed()))
  const form = ref(seed()) as Ref<T>
  const restored = ref(false)
  const dirty = computed(() => JSON.stringify(form.value) !== baseline.value)

  function reset() {
    const initial = seed()
    baseline.value = JSON.stringify(initial)
    form.value = initial
  }

  function open() {
    reset()
    const draftKey = toValue(key)
    const draft = store.get(draftKey)
    const usable =
      draft !== undefined &&
      draft.value !== baseline.value &&
      (!opts.dropWhenSeedChanges || draft.seed === baseline.value)
    if (usable) {
      form.value = JSON.parse(draft.value) as T
      restored.value = true
      return
    }
    store.drop(draftKey)
    restored.value = false
  }

  function keep() {
    const draftKey = toValue(key)
    if (dirty.value) {
      store.set(draftKey, { seed: baseline.value, value: JSON.stringify(form.value) })
    } else {
      store.drop(draftKey)
    }
  }

  function clear() {
    store.drop(toValue(key))
    reset()
    restored.value = false
  }

  function done() {
    store.drop(toValue(key))
    reset()
  }

  return { form, dirty, restored, open, keep, clear, done }
}
