<template>
  <q-dialog ref="dialogRef" persistent @hide="onDialogHide">
    <q-card class="q-dialog-plugin manifestacao-dialog">
      <q-card-section class="row items-center q-gutter-sm">
        <div class="text-h6">Manifestação do destinatário</div>
        <q-space />
        <q-badge
          :color="ambienteColor(tpAmb)"
          :text-color="badgeTextColor(ambienteColor(tpAmb), $q.dark.isActive)"
          class="text-weight-bold q-pa-sm"
        >
          {{ environment || 'Ambiente desconhecido' }}
        </q-badge>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <div class="text-body2 text-weight-medium">
          NF-e {{ formatNFeNumber(note.Numero, note.Serie) }} · {{ note.EmitenteName || formatCpfCnpj(note.EmitenteCNPJ) }}
        </div>
        <div class="text-caption text-app-muted text-mono">{{ formatChaveNFe(note.ChaveAcesso) }}</div>
        <div class="text-caption text-app-muted">Valor: {{ formatCurrencyCents(note.TotalValue) }}</div>
      </q-card-section>

      <template v-if="step === 'escolha'">
        <q-card-section class="q-pt-none">
          <q-option-group v-model="tipo" :options="options" color="primary" />

          <div
            v-for="option in blockedOptions"
            :key="option.value"
            class="text-caption text-app-muted q-mt-xs"
          >
            {{ option.label }}: {{ option.reason }}
          </div>

          <q-input
            v-if="tipo === '210240'"
            v-model="justificativa"
            class="q-mt-md"
            type="textarea"
            label="Justificativa"
            outlined
            autogrow
            counter
            :maxlength="JUSTIFICATIVA_MAX_LENGTH"
            :error="justificativa.length > 0 && Boolean(justificativaError)"
            :error-message="justificativaError ?? ''"
            :hint="`Entre ${JUSTIFICATIVA_MIN_LENGTH} e ${JUSTIFICATIVA_MAX_LENGTH} caracteres`"
          />
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Cancelar" @click="onDialogCancel" />
          <q-btn color="primary" label="Revisar" :disable="!canReview" @click="step = 'revisao'" />
        </q-card-actions>
      </template>

      <template v-else>
        <q-card-section class="q-pt-none">
          <div class="row items-center q-gutter-sm">
            <span class="text-weight-medium">Evento:</span>
            <q-badge
              :color="nfeEventColor(tipo ?? '')"
              :text-color="badgeTextColor(nfeEventColor(tipo ?? ''), $q.dark.isActive)"
              :label="selectedLabel"
            />
            <span class="text-caption text-app-muted text-mono">{{ tipo }}</span>
          </div>
          <div v-if="tipo === '210240'" class="q-mt-sm">
            <div class="text-weight-medium">Justificativa:</div>
            <div class="text-body2 justificativa-text">{{ justificativa.trim() }}</div>
          </div>

          <q-banner
            dense
            rounded
            class="q-mt-md"
            :class="$q.dark.isActive ? 'bg-grey-9 text-orange-3' : 'bg-orange-1 text-orange-10'"
          >
            <template #avatar>
              <q-icon name="gavel" />
            </template>
            Este evento é registrado na SEFAZ e não pode ser desfeito pelo Nanci.
          </q-banner>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Voltar" @click="step = 'escolha'" />
          <q-btn
            :color="nfeEventColor(tipo ?? '')"
            label="Registrar na SEFAZ"
            :disable="!canReview"
            @click="onOKClick"
          />
        </q-card-actions>
      </template>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useDialogPluginComponent, useQuasar } from 'quasar'
import type { NFeConclusiveTipo, NFeRow } from '@/types/desktop'
import {
  formatChaveNFe,
  formatCpfCnpj,
  formatCurrencyCents,
  formatNFeNumber,
} from '@/utils/formatters'
import { ambienteColor, badgeTextColor, nfeEventColor } from '@/utils/nfeDisplay'
import {
  conclusiveBlockReason,
  JUSTIFICATIVA_MAX_LENGTH,
  JUSTIFICATIVA_MIN_LENGTH,
  validateJustificativa,
} from '@/utils/nfeManifestation'

const props = defineProps<{
  note: NFeRow
  environment: string
  tpAmb: string
}>()

defineEmits<{
  ok: [payload: { tipo: NFeConclusiveTipo; justificativa: string }]
  hide: []
}>()

const $q = useQuasar()
const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } = useDialogPluginComponent()

const tipoLabels: Record<NFeConclusiveTipo, string> = {
  '210200': 'Confirmação da operação',
  '210220': 'Desconhecimento da operação',
  '210240': 'Operação não realizada',
}

const tipos: NFeConclusiveTipo[] = ['210200', '210220', '210240']

const options = computed(() =>
  tipos.map((value) => {
    const reason = conclusiveBlockReason(props.note, value)
    return { value, label: tipoLabels[value], disable: reason !== null, reason }
  })
)

const blockedOptions = computed(() => options.value.filter((option) => option.disable))

const step = ref<'escolha' | 'revisao'>('escolha')
const tipo = ref<NFeConclusiveTipo | null>(null)
const justificativa = ref('')

const justificativaError = computed(() =>
  tipo.value === '210240' ? validateJustificativa(justificativa.value) : null
)

const canReview = computed(() => {
  const selected = options.value.find((option) => option.value === tipo.value)
  return Boolean(selected && !selected.disable && !justificativaError.value)
})

const selectedLabel = computed(() => (tipo.value ? tipoLabels[tipo.value] : ''))

function onOKClick() {
  if (!canReview.value || !tipo.value) return
  onDialogOK({
    tipo: tipo.value,
    justificativa: tipo.value === '210240' ? justificativa.value.trim() : '',
  })
}

defineExpose({ dialogRef })
</script>

<style scoped>
.manifestacao-dialog {
  width: 560px;
  max-width: 90vw;
}

.justificativa-text {
  white-space: pre-wrap;
}
</style>
