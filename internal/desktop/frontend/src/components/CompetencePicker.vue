<template>
  <div class="row no-wrap items-center q-gutter-xs">
    <q-btn
      color="grey-7"
      icon="chevron_left"
      dense
      flat
      round
      :disable="disable || !model"
      title="Competência anterior"
      aria-label="Competência anterior"
      @click="shift(-1)"
    />

    <q-input
      v-model="model"
      class="col"
      label="Competência"
      outlined
      dense
      clearable
      mask="####-##"
      :disable="disable"
    >
      <template #append>
        <q-icon name="event" class="cursor-pointer">
          <q-popup-proxy ref="datePopup" cover transition-show="scale" transition-hide="scale">
            <q-date
              v-model="model"
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
      :disable="disable || !model"
      title="Próxima competência"
      aria-label="Próxima competência"
      @click="shift(1)"
    />
  </div>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { competenceOf, shiftCompetence } from '@/utils/competence'

// CompetencePicker edits a YYYY-MM competence: typed, picked from a month
// calendar, or stepped a month at a time.
const model = defineModel<string>({ required: true })

defineProps<{
  disable?: boolean
}>()

const datePopup = ref<{ hide: () => void } | null>(null)

function onDateChange(_value: string, reason: string) {
  if (reason === 'month') {
    datePopup.value?.hide()
  }
}

function setToday() {
  model.value = competenceOf(new Date())
  datePopup.value?.hide()
}

function shift(monthDelta: number) {
  model.value = shiftCompetence(model.value, monthDelta)
}
</script>
