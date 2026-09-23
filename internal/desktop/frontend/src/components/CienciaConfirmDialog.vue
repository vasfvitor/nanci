<template>
  <q-dialog ref="dialogRef" persistent @hide="onDialogHide">
    <q-card class="q-dialog-plugin ciencia-dialog">
      <q-card-section class="row items-center q-gutter-sm">
        <div class="text-h6">Registrar ciência da operação</div>
        <q-space />
        <q-badge
          :color="badgeColor(ambienteColor(tpAmb), $q.dark.isActive)"
          :text-color="badgeTextColor(ambienteColor(tpAmb), $q.dark.isActive)"
          class="text-weight-bold q-pa-sm"
        >
          {{ environment || 'Ambiente desconhecido' }}
        </q-badge>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <div class="text-body2">
          <span class="text-weight-medium">{{ companyName }}</span>
          <span class="text-app-muted"> · {{ formatCpfCnpj(cnpj) }}</span>
        </div>
        <div class="text-subtitle2 q-mt-sm">
          {{ eligibleSummary }}
        </div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-list bordered separator dense class="rounded-borders ciencia-list">
          <q-item v-for="note in plan.Eligible" :key="note.ChaveAcesso">
            <q-item-section>
              <q-item-label>
                {{ formatNFeNumber(note.Numero, note.Serie) }} · {{ note.EmitenteName || formatCpfCnpj(note.EmitenteCNPJ) }}
              </q-item-label>
              <q-item-label caption class="text-mono">{{ formatChaveNFe(note.ChaveAcesso) }}</q-item-label>
            </q-item-section>
            <q-item-section side class="text-mono">
              {{ formatCurrencyCents(note.TotalValue) }}
            </q-item-section>
          </q-item>
          <q-item v-if="plan.Eligible.length === 0">
            <q-item-section class="text-app-muted">Nenhuma nota elegível para ciência.</q-item-section>
          </q-item>
        </q-list>

        <q-expansion-item
          v-if="plan.Skipped.length > 0"
          dense
          class="q-mt-sm"
          icon="block"
          :label="`Não serão enviadas (${plan.Skipped.length})`"
        >
          <q-list dense separator>
            <q-item v-for="skip in plan.Skipped" :key="skip.ChaveAcesso">
              <q-item-section>
                <q-item-label caption class="text-mono">{{ formatChaveNFe(skip.ChaveAcesso) }}</q-item-label>
                <q-item-label>{{ skip.Reason }}</q-item-label>
              </q-item-section>
            </q-item>
          </q-list>
        </q-expansion-item>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-banner dense rounded :class="$q.dark.isActive ? 'bg-grey-9 text-orange-3' : 'bg-orange-1 text-orange-10'">
          <template #avatar>
            <q-icon name="gavel" />
          </template>
          A ciência da operação é um evento fiscal registrado na SEFAZ em nome da empresa e não pode ser
          cancelada. Ela não interrompe o prazo de 90 dias da autorização para a manifestação conclusiva:
          sem manifestação conclusiva nesse prazo, a operação é considerada confirmada.
        </q-banner>

        <q-checkbox
          v-model="acknowledged"
          class="q-mt-sm"
          label="Entendo que a ciência será registrada na SEFAZ para as notas listadas"
        />
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat label="Cancelar" @click="onDialogCancel" />
        <q-btn
          color="primary"
          label="Registrar ciência na SEFAZ"
          :disable="!canConfirm"
          @click="onOKClick"
        />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useDialogPluginComponent, useQuasar } from 'quasar'
import type { NFeCienciaPlan } from '@/types/desktop'
import {
  formatChaveNFe,
  formatCpfCnpj,
  formatCurrencyCents,
  formatNFeNumber,
} from '@/utils/formatters'
import { ambienteColor, badgeColor, badgeTextColor } from '@/utils/nfeDisplay'

const props = defineProps<{
  companyName: string
  cnpj: string
  environment: string
  tpAmb: string
  plan: NFeCienciaPlan
}>()

defineEmits<{
  ok: [chavesAcesso: string[]]
  hide: []
}>()

const $q = useQuasar()
const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } = useDialogPluginComponent()

const acknowledged = ref(false)

const canConfirm = computed(() => acknowledged.value && props.plan.Eligible.length > 0)

const eligibleSummary = computed(() => {
  const count = props.plan.Eligible.length
  const total = props.plan.Eligible.reduce((sum, note) => sum + note.TotalValue, 0)
  const notes = count === 1 ? '1 nota' : `${count} notas`
  const lotes = props.plan.Lotes === 1 ? '1 lote' : `${props.plan.Lotes} lotes`
  return `${notes} · total ${formatCurrencyCents(total)} · ${lotes}`
})

function onOKClick() {
  if (!canConfirm.value) return
  onDialogOK(props.plan.Eligible.map((note) => note.ChaveAcesso))
}

defineExpose({ dialogRef })
</script>

<style scoped>
.ciencia-dialog {
  width: 640px;
  max-width: 90vw;
}

.ciencia-list {
  max-height: 280px;
  overflow-y: auto;
}
</style>
