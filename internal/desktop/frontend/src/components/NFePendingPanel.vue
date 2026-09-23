<template>
  <div class="column q-gutter-y-lg">
    <section>
      <div class="row items-center q-mb-sm">
        <div class="text-subtitle1 text-weight-bold">Sem ciência ({{ semCiencia.length }})</div>
        <q-space />
        <q-btn
          color="primary"
          icon="task_alt"
          :label="`Registrar ciência de todas (${semCiencia.length})`"
          dense
          flat
          :disable="loading || semCiencia.length === 0 || semCiencia.every((row) => busy(row.ChaveAcesso))"
          @click="emit('ciencia', semCiencia)"
        />
      </div>

      <q-banner dense rounded class="q-mb-sm" :class="$q.dark.isActive ? 'bg-grey-9 text-grey-4' : 'bg-grey-2 text-grey-8'">
        <template #avatar>
          <q-icon name="info" />
        </template>
        A ciência da operação libera o download do XML completo. Registre-a em até 10 dias da
        autorização da nota.
      </q-banner>

      <q-table
        :rows="semCiencia"
        :columns="columns"
        row-key="ChaveAcesso"
        :loading="loading"
        flat
        bordered
        dense
        :pagination="{ rowsPerPage: 0 }"
        hide-pagination
        no-data-label="Nenhuma nota aguardando ciência."
      >
        <template #body-cell-prazo="cellProps">
          <q-td :props="cellProps">
            <q-chip
              dense
              square
              :color="cienciaChipColor(cellProps.row)"
              outline
              :label="cienciaChipLabel(cellProps.row)"
            />
            <div class="text-caption text-app-muted">{{ formatDate(cellProps.row.CienciaDue) }}</div>
          </q-td>
        </template>

        <template #body-cell-acoes="cellProps">
          <q-td :props="cellProps" class="q-gutter-x-xs">
            <q-spinner v-if="busy(cellProps.row.ChaveAcesso)" size="sm" color="primary" />
            <template v-else>
              <q-btn dense flat size="sm" color="primary" label="Ciência" @click="emit('ciencia', [cellProps.row])" />
              <q-btn dense flat size="sm" color="secondary" label="Manifestar…" @click="emit('manifest', cellProps.row)" />
              <q-btn
                dense
                flat
                round
                size="sm"
                color="grey-7"
                icon="history"
                aria-label="Eventos"
                title="Eventos"
                @click="emit('events', cellProps.row)"
              />
            </template>
          </q-td>
        </template>
      </q-table>
    </section>

    <section>
      <div class="text-subtitle1 text-weight-bold q-mb-sm">
        Aguardando manifestação conclusiva ({{ semConclusiva.length }})
      </div>

      <q-banner dense rounded class="q-mb-sm" :class="$q.dark.isActive ? 'bg-grey-9 text-grey-4' : 'bg-grey-2 text-grey-8'">
        <template #avatar>
          <q-icon name="gavel" />
        </template>
        Confirmação, desconhecimento ou operação não realizada podem ser registrados em até 90 dias da
        autorização da nota. Passado esse prazo sem manifestação conclusiva, a operação é considerada
        confirmada por lei, mesmo que a ciência tenha sido registrada.
      </q-banner>

      <q-table
        :rows="semConclusiva"
        :columns="columns"
        row-key="ChaveAcesso"
        :loading="loading"
        flat
        bordered
        dense
        :pagination="{ rowsPerPage: 0 }"
        hide-pagination
        no-data-label="Nenhuma nota aguardando manifestação conclusiva."
      >
        <template #body-cell-prazo="cellProps">
          <q-td :props="cellProps">
            <q-chip
              dense
              square
              :color="deadlineColor(daysUntil(cellProps.row.Deadline), 'conclusiva')"
              outline
              :label="conclusiveChipLabel(cellProps.row)"
            />
            <div class="text-caption text-app-muted">{{ formatDate(cellProps.row.Deadline) }}</div>
          </q-td>
        </template>

        <template #body-cell-acoes="cellProps">
          <q-td :props="cellProps" class="q-gutter-x-xs">
            <q-spinner v-if="busy(cellProps.row.ChaveAcesso)" size="sm" color="primary" />
            <template v-else>
              <q-btn dense flat size="sm" color="secondary" label="Manifestar…" @click="emit('manifest', cellProps.row)" />
              <q-btn
                dense
                flat
                round
                size="sm"
                color="grey-7"
                icon="history"
                aria-label="Eventos"
                title="Eventos"
                @click="emit('events', cellProps.row)"
              />
            </template>
          </q-td>
        </template>
      </q-table>
    </section>
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import type { ISODateValue, NFePendingRow } from '@/types/desktop'
import {
  daysUntil,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
  formatNFeNumber,
} from '@/utils/formatters'
import {
  conclusiveDeadlineLabel,
  deadlineColor,
  deadlineLabel,
  TACIT_CONFIRMATION_LABEL,
} from '@/utils/nfeDisplay'

const props = defineProps<{
  rows: NFePendingRow[]
  loading: boolean
  busy: (chaveAcesso: string) => boolean
}>()

const emit = defineEmits<{
  ciencia: [rows: NFePendingRow[]]
  manifest: [row: NFePendingRow]
  events: [row: NFePendingRow]
}>()

const $q = useQuasar()

const semCiencia = computed(() => props.rows.filter((row) => row.Kind === 'sem_ciencia'))

const semConclusiva = computed(() =>
  props.rows
    .filter((row) => row.Kind === 'sem_conclusiva')
    .sort((a, b) => timeOf(a.Deadline) - timeOf(b.Deadline))
)

const columns: QTableColumn<NFePendingRow>[] = [
  {
    name: 'numero',
    label: 'Número / Série',
    field: (row) => formatNFeNumber(row.Numero, row.Serie),
    align: 'left',
    classes: 'text-mono',
  },
  {
    name: 'emitente',
    label: 'Emitente',
    field: (row) => row.EmitenteName || formatCpfCnpj(row.EmitenteCNPJ),
    align: 'left',
  },
  {
    name: 'emissao',
    label: 'Emissão',
    field: 'IssueDate',
    align: 'left',
    classes: 'text-mono',
    format: (value: ISODateValue) => formatDate(value),
  },
  {
    name: 'valor',
    label: 'Valor',
    field: 'TotalValue',
    align: 'right',
    classes: 'text-mono',
    format: (value: number) => formatCurrencyCents(value),
  },
  { name: 'prazo', label: 'Prazo', field: 'Deadline', align: 'left' },
  { name: 'acoes', label: 'Ações', field: () => '', align: 'right' },
]

// A note without ciência can also pass the conclusive deadline; the row is
// then Expired and the operation is already deemed confirmed.
function cienciaChipLabel(row: NFePendingRow) {
  if (row.Expired) return TACIT_CONFIRMATION_LABEL
  return deadlineLabel(daysUntil(row.CienciaDue))
}

function cienciaChipColor(row: NFePendingRow) {
  if (row.CienciaOverdue || row.Expired) return 'negative'
  return deadlineColor(daysUntil(row.CienciaDue), 'ciencia')
}

function conclusiveChipLabel(row: NFePendingRow) {
  if (row.Expired) return TACIT_CONFIRMATION_LABEL
  return conclusiveDeadlineLabel(daysUntil(row.Deadline))
}

function timeOf(value: ISODateValue) {
  if (!value) return Number.POSITIVE_INFINITY
  const time = new Date(value).getTime()
  return Number.isNaN(time) ? Number.POSITIVE_INFINITY : time
}
</script>
