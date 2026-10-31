<template>
  <div class="row q-col-gutter-md">
    <DetailList class="col-12 col-md-5" title="Nota" :items="notaItems" />
    <DetailList class="col-12 col-md-4" title="Emitente" :items="emitenteItems" />
    <div class="col-12 col-md-3">
      <DetailList title="Destinatário" :items="destinatarioItems" class="q-mb-md" />
      <DetailList title="Valores" :items="valoresItems" />
    </div>

    <ParseWarnings class="col-12" :warnings="row.ParseWarnings" />
  </div>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import DetailList, { type DetailItem } from './DetailList.vue'
import ParseWarnings from './ParseWarnings.vue'
import type { NFeRow } from '@/types/desktop'
import {
  formatCpfCnpj,
  formatCurrencyCents,
  formatDateTime,
  formatNFeNumber,
} from '@/utils/formatters'
import { situacaoLabel } from '@/utils/nfeDisplay'
import { ambienteLabel } from '@/utils/sefazDisplay'

// NFeDetails is the expanded row of an NF-e: the note, its parties, its
// values and the parse warnings.
const props = defineProps<{
  row: NFeRow
}>()

const notaItems = computed<DetailItem[]>(() => [
  { label: 'Número', value: formatNFeNumber(props.row.Numero), mono: true },
  { label: 'Série', value: props.row.Serie, mono: true },
  { label: 'Natureza da operação', value: props.row.NatOp },
  { label: 'Layout', value: props.row.LayoutVersion, mono: true },
  { label: 'Ambiente', value: props.row.TpAmb ? ambienteLabel(props.row.TpAmb) : '' },
  { label: 'Situação', value: props.row.Situacao ? situacaoLabel(props.row.Situacao) : '' },
  { label: 'Protocolo', value: props.row.Protocolo, mono: true },
  { label: 'Autorizada em', value: formatDateTime(props.row.AuthorizedAt, '') },
])

const emitenteItems = computed<DetailItem[]>(() => [
  { label: 'Nome', value: props.row.EmitenteName },
  { label: 'CNPJ', value: formatCpfCnpj(props.row.EmitenteCNPJ), mono: true },
  { label: 'IE', value: props.row.EmitenteIE, mono: true },
  { label: 'UF', value: props.row.EmitenteUF },
])

const destinatarioItems = computed<DetailItem[]>(() => [
  { label: 'Nome', value: props.row.DestinatarioName },
  { label: 'CNPJ', value: formatCpfCnpj(props.row.DestinatarioCNPJ), mono: true },
])

// A resumo carries only the total, so its taxes show as "—".
const valoresItems = computed<DetailItem[]>(() => {
  const completa = props.row.Completeness === 'completa'
  return [
    { label: 'Total', value: formatCurrencyCents(props.row.TotalValue), mono: true },
    {
      label: 'ICMS',
      value: completa ? formatCurrencyCents(props.row.ICMSValue) : '',
      mono: true,
    },
    { label: 'IPI', value: completa ? formatCurrencyCents(props.row.IPIValue) : '', mono: true },
  ]
})
</script>
