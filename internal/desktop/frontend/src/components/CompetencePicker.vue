<template>
  <div class="row no-wrap items-center q-gutter-xs">
    <q-btn
      color="grey-7"
      icon="chevron_left"
      dense
      flat
      round
      :disable="disable || !base"
      title="Competência anterior"
      aria-label="Competência anterior"
      @click="shift(-1)"
    />

    <q-input
      :model-value="draft"
      class="col"
      label="Competência"
      outlined
      dense
      clearable
      mask="####-##"
      :disable="disable"
      :error="Boolean(draftError)"
      :error-message="draftError"
      hide-bottom-space
      @update:model-value="onType"
    >
      <template #append>
        <q-icon name="event" class="cursor-pointer">
          <q-popup-proxy ref="datePopup" cover transition-show="scale" transition-hide="scale">
            <q-date
              :model-value="draft"
              minimal
              mask="YYYY-MM"
              emit-immediately
              default-view="Months"
              years-in-month-view
              @update:model-value="onDateChange"
            >
              <div class="row items-center justify-end">
                <q-btn label="Mês Atual" color="primary" flat @click="setToday" />
                <q-btn v-close-popup label="Fechar" color="primary" flat />
              </div>
            </q-date>
          </q-popup-proxy>
        </q-icon>
      </template>
    </q-input>

    <q-btn
      color="grey-7"
      icon="chevron_right"
      dense
      flat
      round
      :disable="disable || !base"
      title="Próxima competência"
      aria-label="Próxima competência"
      @click="shift(1)"
    />
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, ref, watch } from 'vue'
import { competenceOf, isCompetence, shiftCompetence } from '@/utils/competence'

// CompetencePicker edits a YYYY-MM competence: typed, picked from a month
// calendar, or stepped a month at a time. The model only ever gets a
// complete competence or '' when cleared; a month typed in part stays in
// the field, flagged, and the model keeps the last complete one.
const model = defineModel<string>({ required: true })

defineProps<{
  disable?: boolean
}>()

// STEP_DELAY_MS lets quick chevron clicks or calendar picks reach the model
// once, so stepping several months starts one search.
const STEP_DELAY_MS = 200

// draft is the field text, ahead of the model while partial or while a
// step waits for STEP_DELAY_MS. The model watcher below fills it.
const draft = ref('')
let pendingStep: ReturnType<typeof setTimeout> | undefined

// cancelStep drops a step still waiting for STEP_DELAY_MS.
function cancelStep() {
  clearTimeout(pendingStep)
  pendingStep = undefined
}

// base is the competence the chevrons step from: the draft when complete,
// else the model.
const base = computed(() => (isCompetence(draft.value) ? draft.value : model.value))

// draftError says why the draft has not reached the model, or is ''.
const draftError = computed(() => {
  const value = draft.value
  if (!value || isCompetence(value)) return ''
  return value.length < 'AAAA-MM'.length ? 'Mês incompleto' : 'Mês inválido'
})

// A competência set elsewhere replaces the draft and any step in wait.
watch(
  model,
  (value) => {
    if (value === draft.value) return
    cancelStep()
    draft.value = value
  },
  { immediate: true }
)

function commit(value: string) {
  cancelStep()
  draft.value = value
  if (value !== model.value) model.value = value
}

function commitSoon(value: string) {
  cancelStep()
  draft.value = value
  pendingStep = setTimeout(() => commit(value), STEP_DELAY_MS)
}

// A step still waiting when the picker goes away is kept.
onBeforeUnmount(() => {
  if (pendingStep !== undefined) commit(draft.value)
})

// onType takes the typed text; clearable sets null. Only a complete or
// cleared value reaches the model, and typing drops a step in wait.
function onType(value: string | number | null) {
  const text = String(value ?? '')
  cancelStep()
  draft.value = text
  if (!text || isCompetence(text)) commit(text)
}

const datePopup = ref<{ hide: () => void } | null>(null)

// A month picked again in the calendar unselects it, as null.
function onDateChange(value: string | null, reason: string) {
  commitSoon(value ?? '')
  if (reason === 'month') {
    datePopup.value?.hide()
  }
}

function setToday() {
  commit(competenceOf(new Date()))
  datePopup.value?.hide()
}

function shift(monthDelta: number) {
  commitSoon(shiftCompetence(base.value, monthDelta))
}
</script>
