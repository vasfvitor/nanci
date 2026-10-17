<template>
  <div class="row items-center justify-between full-width">
    <div class="text-subtitle1 text-weight-bold">{{ title }}</div>
    <q-input
      :model-value="text"
      class="document-search-input"
      placeholder="Filtrar por chave, número, nome ou CNPJ..."
      outlined
      dense
      clearable
      @update:model-value="onInput"
      @blur="flush"
      @keyup.esc="clear"
    >
      <template #append>
        <q-icon name="search" />
      </template>
    </q-input>
  </div>
</template>

<script setup lang="ts">
// DocumentTableTop is the top of a document table: its title and the box
// that narrows the listed rows. The placeholder names only the fields that
// fit in the 350px box; each page searches a few more. Escape clears it.
//
// The box shows each keystroke at once and hands the text to the model
// 300 ms after the last one, on blur or on unmount. The debounce lives here
// and not in the q-input so that Escape and the clear button can drop a
// pending update and always leave the filter empty.
import { onBeforeUnmount, ref, watch } from 'vue'

defineProps<{
  title: string
}>()

const model = defineModel<string>({ required: true })

const debounceMs = 300

const text = ref<string>('')
let timer: ReturnType<typeof setTimeout> | null = null

function cancel() {
  if (timer !== null) {
    clearTimeout(timer)
    timer = null
  }
}

function flush() {
  if (timer === null) {
    return
  }
  cancel()
  model.value = text.value
}

function clear() {
  cancel()
  text.value = ''
  model.value = ''
}

// onInput takes a keystroke, or null from the clear button.
function onInput(value: string | number | null) {
  if (value === null) {
    clear()
    return
  }
  text.value = String(value)
  cancel()
  timer = setTimeout(flush, debounceMs)
}

// The model, at mount or when set from outside such as by a page reset,
// replaces what is typed.
watch(
  model,
  (value) => {
    if (value !== text.value) {
      cancel()
      text.value = value
    }
  },
  { immediate: true }
)

onBeforeUnmount(flush)
</script>
