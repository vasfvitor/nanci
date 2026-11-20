<template>
  <q-dialog ref="dialogRef" persistent @hide="onDialogHide">
    <q-card class="q-dialog-plugin desacordo-dialog">
      <q-card-section class="row items-center q-gutter-sm">
        <div class="text-h6">Prestação do serviço em desacordo</div>
        <q-space />
        <q-badge
          v-bind="badgeProps(ambienteColor(row.TpAmb), $q.dark.isActive)"
          class="text-weight-bold q-pa-sm"
        >
          {{ ambienteLabel(row.TpAmb) }}
        </q-badge>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <div class="text-body2 text-weight-medium">
          CT-e {{ formatNFeNumber(row.Numero, row.Serie) }} ·
          {{ formatParty(row.EmitenteName, row.EmitenteCNPJ) }}
        </div>
        <div class="text-caption text-app-muted text-mono">
          {{ formatChaveDFe(row.ChaveAcesso) }}
        </div>
        <div class="text-caption text-app-muted">
          Valor da prestação: {{ formatCurrencyCents(row.TotalValue) }}
        </div>
      </q-card-section>

      <template v-if="step === 'observacao'">
        <q-card-section class="q-pt-none">
          <div class="text-body2 q-mb-sm">
            Como tomador, declare o que não foi prestado como consta no CT-e. A SEFAZ que autorizou
            o CT-e registra o evento.
          </div>
          <q-input
            v-model="observacao"
            type="textarea"
            label="Observação"
            outlined
            autogrow
            autofocus
            counter
            :maxlength="OBSERVACAO_MAX_LENGTH"
            :error="observacao.length > 0 && Boolean(observacaoError)"
            :error-message="observacaoError ?? ''"
            :hint="`Entre ${OBSERVACAO_MIN_LENGTH} e ${OBSERVACAO_MAX_LENGTH} caracteres`"
          />
          <div v-if="loadError" class="text-negative text-caption q-mt-sm">{{ loadError }}</div>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Cancelar" @click="onDialogCancel" />
          <q-btn
            color="primary"
            label="Revisar"
            :loading="loading"
            :disable="Boolean(observacaoError)"
            @click="review"
          />
        </q-card-actions>
      </template>

      <template v-else-if="plan">
        <q-card-section class="q-pt-none">
          <div class="row items-center q-gutter-sm">
            <span class="text-weight-medium">Evento:</span>
            <q-badge
              v-bind="badgeProps('negative', $q.dark.isActive)"
              label="Prestação do serviço em desacordo"
            />
            <span class="text-caption text-app-muted text-mono">610110</span>
          </div>
          <div class="q-mt-sm">
            <span class="text-weight-medium">Prazo:</span>
            {{ plan.Due ? formatDate(plan.Due) : '—' }} ({{ desacordoDeadlineText(plan.DaysLeft) }})
          </div>
          <div v-if="deadlinePassed" class="text-caption text-negative">
            Passaram os 45 dias da autorização. A SEFAZ deve rejeitar o evento (cStat 787).
          </div>
          <div class="q-mt-sm">
            <div class="text-weight-medium">Observação:</div>
            <div class="text-body2 observacao-text">{{ plan.Observacao }}</div>
          </div>

          <q-banner
            v-if="plan.BlockReason"
            dense
            rounded
            class="q-mt-md"
            :class="$q.dark.isActive ? 'bg-grey-9 text-red-3' : 'bg-red-1 text-red-10'"
          >
            <template #avatar>
              <q-icon name="block" />
            </template>
            {{ plan.BlockReason }}
          </q-banner>
          <q-banner
            v-else
            dense
            rounded
            class="q-mt-md"
            :class="warningBannerClass($q.dark.isActive)"
          >
            <template #avatar>
              <q-icon name="gavel" />
            </template>
            Este evento é registrado na SEFAZ e não pode ser desfeito pelo Nanci.
          </q-banner>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Voltar" @click="step = 'observacao'" />
          <q-btn
            color="negative"
            label="Registrar na SEFAZ"
            :disable="Boolean(plan.BlockReason)"
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
import { errorMessage } from '@/platform/wails/errors'
import type { CTeDesacordoPlan, CTeRow } from '@/types/desktop'
import {
  desacordoDeadlineText,
  OBSERVACAO_MAX_LENGTH,
  OBSERVACAO_MIN_LENGTH,
  validateObservacao,
} from '@/utils/cteDesacordo'
import {
  formatChaveDFe,
  formatCurrencyCents,
  formatDate,
  formatNFeNumber,
  formatParty,
} from '@/utils/formatters'
import { ambienteColor, ambienteLabel, badgeProps } from '@/utils/sefazDisplay'
import { warningBannerClass } from '@/utils/bannerClass'

const props = defineProps<{
  row: CTeRow
  // load asks the backend for the plan of the observação; it sends nothing.
  load: (observacao: string) => Promise<CTeDesacordoPlan>
}>()

defineEmits<{
  ok: [observacao: string]
  hide: []
}>()

const $q = useQuasar()
const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } = useDialogPluginComponent()

const step = ref<'observacao' | 'revisao'>('observacao')
const observacao = ref('')
const plan = ref<CTeDesacordoPlan | null>(null)
const loading = ref(false)
const loadError = ref('')

const observacaoError = computed(() => validateObservacao(observacao.value))
const deadlinePassed = computed(() => (plan.value?.DaysLeft ?? 0) < 0)

async function review() {
  if (observacaoError.value || loading.value) return
  loading.value = true
  loadError.value = ''
  try {
    plan.value = await props.load(observacao.value)
    step.value = 'revisao'
  } catch (error) {
    loadError.value = errorMessage(error)
  } finally {
    loading.value = false
  }
}

function onOKClick() {
  if (!plan.value || plan.value.BlockReason) return
  onDialogOK(plan.value.Observacao)
}

defineExpose({ dialogRef })
</script>

<style scoped>
.desacordo-dialog {
  width: 560px;
  max-width: 90vw;
}

.observacao-text {
  white-space: pre-wrap;
}
</style>
