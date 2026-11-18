<template>
  <div>
    <div class="text-subtitle2 text-primary q-mb-xs">CT-e do frete</div>
    <q-spinner v-if="loading" size="sm" color="primary" />
    <div v-else-if="error" class="text-body2 text-negative">{{ error }}</div>
    <div v-else-if="rows.length === 0" class="text-body2">
      Nenhum CT-e guardado para esta empresa transporta esta NF-e.
    </div>
    <div v-else class="column q-gutter-y-xs">
      <div
        v-for="cte in rows"
        :key="cte.ChaveAcesso"
        class="row no-wrap items-center q-gutter-x-sm"
      >
        <span class="text-body2">{{ cteLine(cte) }}</span>
        <q-btn
          dense
          flat
          no-caps
          size="sm"
          color="primary"
          label="Ver CT-e"
          title="Abre este CT-e na página CT-e, de qualquer competência"
          @click.stop="emit('open', cte.ChaveAcesso)"
        />
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { useNFeLinkedCTe } from '@/composables/useNFeLinkedCTe'
import type { CTeRow } from '@/types/desktop'
import { formatCpfCnpj, formatCurrencyCents, formatDate, formatNFeNumber } from '@/utils/formatters'

// NFeLinkedCTe lists, in the expanded row of an NF-e, the CT-e of the
// company that transported it; open asks the page to show one of them.
const props = defineProps<{
  cnpj: string
  chave: string
}>()

const emit = defineEmits<{
  open: [chave: string]
}>()

const { rows, loading, error } = useNFeLinkedCTe(
  () => props.cnpj,
  () => props.chave
)

function cteLine(cte: CTeRow) {
  return [
    `CT-e ${formatNFeNumber(cte.Numero)}`,
    cte.EmitenteName || formatCpfCnpj(cte.EmitenteCNPJ),
    formatDate(cte.IssueDate),
    formatCurrencyCents(cte.TotalValue),
  ]
    .filter(Boolean)
    .join(' · ')
}
</script>
