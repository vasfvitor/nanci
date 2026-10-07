<template>
  <q-page padding>
    <DocumentPageHeader
      title="CT-e"
      :ambiente="ambiente"
      :context-line="contextLine"
      :status-line="statusLine"
      :idle-warning="idleWarning"
      :blocked-text="blockedText"
      :syncing="isSyncing"
      :sync-disabled="!cnpj || isResetting || Boolean(syncBlockedUntil)"
      reset-title="Remove os CT-e da empresa e reinicia a sincronização CT-e"
      :resetting="isResetting || previewingReset"
      :reset-disabled="!cnpj || isSyncing"
      @sync="syncCTe"
      @reset="confirmResetCTe"
    />

    <DocumentFilterBar
      v-model:only-unviewed="onlyUnviewed"
      :loading="loading"
      :exporting="exporting"
      :export-disabled="scopeRows.length === 0"
      :search-disabled="!cnpj || Boolean(listError)"
      :mark-viewed-count="unviewedChaves.length"
      @search="search"
      @mark-viewed="confirmMarkViewed"
      @export="openExportDialog"
    >
      <template #default="{ search: searchOnEnter }">
        <DocumentFilterSelect v-model="filter.Role" :options="ctePapelFilterOptions" label="Papel" :disable="loading" />
        <DocumentFilterSelect v-model="filter.Modelo" :options="cteModeloFilterOptions" label="Modelo" :disable="loading" />
        <DocumentFilterSelect v-model="filter.Situacao" :options="cteSituacaoFilterOptions" label="Situação" :disable="loading" />
        <q-input
          v-model="filter.TomadorCNPJ"
          class="cte-filter-cnpj"
          label="CNPJ do tomador"
          outlined
          dense
          clearable
          :disable="loading"
          @keyup.enter="searchOnEnter"
        />
        <q-input
          v-model="filter.NFeChave"
          class="cte-filter-chave"
          label="Chave de NF-e transportada"
          outlined
          dense
          clearable
          hide-bottom-space
          :error="Boolean(listError)"
          :error-message="listError"
          :disable="loading"
          @keyup.enter="searchOnEnter"
        />
      </template>
    </DocumentFilterBar>

    <StateLegend :sections="CTE_LEGEND" class="q-mb-md" />

    <q-table
      v-model:pagination="pagination"
      v-model:selected="selected"
      :rows="filteredRows"
      :columns="columns"
      row-key="ChaveAcesso"
      selection="multiple"
      :loading="loading"
      :no-data-label="cnpj ? 'Nenhum CT-e encontrado.' : noCompanyLabel"
      class="document-table"
      binary-state-sort
      flat
      bordered
      dense
    >
      <template #top>
        <DocumentTableTop v-model="filterText" title="Conhecimentos de transporte eletrônicos" />
      </template>

      <template #body="rowProps">
        <q-tr :props="rowProps">
          <q-td auto-width>
            <q-checkbox v-model="rowProps.selected" dense />
          </q-td>
          <q-td v-for="col in rowProps.cols" :key="col.name" :props="rowProps">
            <RowActionsMenu v-if="col.name === 'acoes'" v-model:expanded="rowProps.expand" source="cte">
              <RowMenuItem
                :label="`Eventos (${rowProps.row.EventCount})`"
                :disable="rowProps.row.EventCount === 0"
                @click="openEvents(rowProps.row.ChaveAcesso)"
              />
              <RowMenuItem
                label="Exportar XML"
                :disable="exporting"
                @click="exportXML(rowProps.row.ChaveAcesso)"
              />
            </RowActionsMenu>

            <NumeroCell
              v-else-if="col.name === 'numero'"
              :numero="col.value"
              :serie="rowProps.row.Serie"
              :viewed-at="rowProps.row.ViewedAt"
            />

            <ChaveCell v-else-if="col.name === 'chave'" :chave="rowProps.row.ChaveAcesso" />

            <PartyCell
              v-else-if="col.name === 'emitente'"
              :name="rowProps.row.EmitenteName"
              :cnpj="rowProps.row.EmitenteCNPJ"
            />

            <PartyCell
              v-else-if="col.name === 'destinatario'"
              :name="rowProps.row.TomadorName"
              :cnpj="rowProps.row.TomadorCNPJ"
            />

            <StateBadges v-else-if="col.name === 'estados'" :badges="badgesByChave.get(rowProps.row.ChaveAcesso) ?? []" />

            <template v-else>{{ col.value }}</template>
          </q-td>
        </q-tr>

        <DocumentDetailRow v-if="rowProps.expand" :row-props="rowProps">
          <div class="row q-col-gutter-md">
            <DetailList class="col-12 col-md-4" title="Prestação" :items="prestacaoItems(rowProps.row)" />
            <DetailList
              class="col-12 col-md-4"
              title="Participantes"
              :items="participantesItems(rowProps.row)"
            />
            <DetailList class="col-12 col-md-4" title="Valores" :items="valoresItems(rowProps.row)" />

            <div v-if="rowProps.row.NFeChaves.length > 0" class="col-12">
              <div class="text-subtitle2 text-primary q-mb-xs">
                NF-e transportadas ({{ rowProps.row.NFeChaves.length }})
              </div>
              <div class="row q-gutter-x-md q-gutter-y-xs">
                <div
                  v-for="chave in rowProps.row.NFeChaves"
                  :key="chave"
                  class="row no-wrap items-center q-gutter-x-xs"
                >
                  <span class="text-mono text-body2">{{ formatChaveDFe(chave) }}</span>
                  <q-btn
                    dense
                    flat
                    round
                    size="xs"
                    color="grey-7"
                    icon="content_copy"
                    title="Copiar chave da NF-e"
                    aria-label="Copiar chave da NF-e"
                    @click.stop="copyChave(chave)"
                  />
                </div>
              </div>
            </div>

            <ParseWarnings class="col-12" :warnings="rowProps.row.ParseWarnings" />
          </div>
        </DocumentDetailRow>
      </template>
    </q-table>

    <CTeEventsDialog v-model="showEventsDialog" :cnpj="cnpj" :chave-acesso="eventsChave" />
  </q-page>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { useQuasar } from 'quasar'
import ChaveCell from '../components/ChaveCell.vue'
import CTeEventsDialog from '../components/CTeEventsDialog.vue'
import DetailList, { type DetailItem } from '../components/DetailList.vue'
import DocumentDetailRow from '../components/DocumentDetailRow.vue'
import DocumentFilterBar from '../components/DocumentFilterBar.vue'
import DocumentFilterSelect from '../components/DocumentFilterSelect.vue'
import DocumentPageHeader from '../components/DocumentPageHeader.vue'
import DocumentTableTop from '../components/DocumentTableTop.vue'
import type { ExportChoice } from '../components/ExportDialog.vue'
import NumeroCell from '../components/NumeroCell.vue'
import ParseWarnings from '../components/ParseWarnings.vue'
import PartyCell from '../components/PartyCell.vue'
import RowActionsMenu from '../components/RowActionsMenu.vue'
import RowMenuItem from '../components/RowMenuItem.vue'
import StateBadges from '../components/StateBadges.vue'
import StateLegend from '../components/StateLegend.vue'
import { useCTeDocuments } from '@/composables/useCTeDocuments'
import { useDocumentListActions } from '@/composables/useDocumentListActions'
import { copyChave, useNotify } from '@/composables/useNotify'
import { useWorkspaceList } from '@/composables/useWorkspaceList'
import { wailsErrorCode } from '@/platform/wails/client'
import type { CTeResetResult, CTeRow } from '@/types/desktop'
import { documentColumns } from '@/utils/documentColumns'
import {
  cteModalLabel,
  cteModeloFilterOptions,
  ctePapelFilterOptions,
  cteParticipantes,
  ctePercurso,
  cteSituacaoFilterOptions,
  cteTpServLabel,
} from '@/utils/cteDisplay'
import {
  formatChaveDFe,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDateTime,
  formatNFeNumber,
} from '@/utils/formatters'
import { CTE_LEGEND } from '@/utils/stateLegends'

const $q = useQuasar()
const cte = useCTeDocuments()
const { notifyError, notifySuccess, notifyWarning, notifyExported, notifySyncError } = useNotify()

const {
  filter,
  selected,
  loading,
  exporting,
  pagination,
  filterText,
  filteredRows,
  scopeRows,
  unviewedChaves,
  badgesByChave,
  onlyUnviewed,
  status,
  companyName,
  ambiente,
  statusLine,
  idleWarning,
  listError,
  isSyncing,
  isResetting,
  syncBlockedUntil,
  blockedText,
} = cte

const showEventsDialog = ref(false)
const eventsChave = ref('')

// previewingReset is true while the reset counts load for the confirmation.
const previewingReset = ref(false)

const { confirmMarkViewed, openExportDialog } = useDocumentListActions({
  source: 'cte',
  selected,
  scopeRows,
  unviewedChaves,
  markViewed: cte.markViewed,
  exportFormats: [{ label: 'XMLs (ZIP)', value: 'zip' }],
  exportList: exportZIP,
})

const columns = documentColumns<CTeRow>({
  emitenteLabel: 'Emitente',
  destinatarioLabel: 'Tomador',
  numero: (row) => formatNFeNumber(row.Numero),
  emitente: (row) => row.EmitenteName || row.EmitenteCNPJ,
  destinatario: (row) => row.TomadorName || row.TomadorCNPJ,
  valor: (row) => row.TotalValue,
})

function prestacaoItems(row: CTeRow): DetailItem[] {
  return [
    { label: 'CFOP', value: [row.CFOP, row.NatOp].filter(Boolean).join(' · ') },
    { label: 'Serviço', value: cteTpServLabel(row.TpServ) },
    { label: 'Modal', value: cteModalLabel(row.Modal) },
    { label: 'Percurso', value: ctePercurso(row) },
    { label: 'Protocolo', value: row.Protocolo, mono: true },
    { label: 'Autorização', value: formatDateTime(row.AuthorizedAt, '—') },
  ]
}

function participantesItems(row: CTeRow): DetailItem[] {
  return cteParticipantes(row).map((party) => ({
    label: party.label,
    value: party.name,
    caption: formatCpfCnpj(party.cnpj),
  }))
}

function valoresItems(row: CTeRow): DetailItem[] {
  return [
    { label: 'Prestação', value: formatCurrencyCents(row.TotalValue), mono: true },
    { label: 'A receber', value: formatCurrencyCents(row.ReceivableValue), mono: true },
    { label: 'ICMS', value: formatCurrencyCents(row.ICMSValue), mono: true },
    { label: 'Tributos', value: formatCurrencyCents(row.TotTribValue), mono: true },
    { label: 'Carga', value: formatCurrencyCents(row.CargaValue), mono: true },
    { label: 'Produto', value: row.ProdutoPredominante },
  ]
}

// The page lists the workspace company and competência. A new company also
// reloads the status; a new competência only the list.
const { cnpj, noCompanyLabel, contextLine } = useWorkspaceList({
  rowsFor: () => cte.rowsFor.value,
  clearRows: () => cte.clearRows(),
  loadRows: search,
  clearCompany: (companyCNPJ) => {
    if (status.value?.CNPJ !== companyCNPJ) status.value = null
  },
  loadCompany: loadStatus,
})

async function search() {
  try {
    await cte.search()
  } catch (error) {
    notifyError('Erro ao buscar CT-e', error)
  }
}

async function loadStatus() {
  try {
    await cte.loadStatus()
  } catch (error) {
    notifyError('Erro ao carregar o status do CT-e', error)
  }
}

async function syncCTe() {
  try {
    const result = await cte.syncCTe()
    if (!result) return
    notifySuccess(
      `Sincronização CT-e ${result.Status || 'concluída'}: ${result.DocumentsSaved} CT-e e ${result.EventsSaved} eventos (NSU ${result.LastNSU}/${result.MaxNSU ?? '—'}).`
    )
  } catch (error) {
    notifySyncError('Erro na sincronização do CT-e', error)
  }
}

// confirmResetCTe loads what the reset would remove and asks the user to
// confirm with those counts.
async function confirmResetCTe() {
  let preview: CTeResetResult | null
  previewingReset.value = true
  try {
    preview = await cte.previewReset()
  } catch (error) {
    notifyError('Erro ao preparar a redefinição do CT-e', error)
    return
  } finally {
    previewingReset.value = false
  }
  if (!preview) return

  $q.dialog({
    title: 'Redefinir CT-e',
    message:
      `Remove ${preview.CompanyDocuments} CT-e de ${preview.CompanyName || companyName.value} nos dois ambientes, ` +
      `com ${preview.Events} eventos e ${preview.ExportMarks} marcas de exportação, ` +
      'e reinicia a sincronização CT-e desde o NSU 0. ' +
      'CT-e vistos por outra empresa continuam para ela.',
    cancel: true,
    persistent: true,
    ok: { label: 'Redefinir', color: 'negative' },
  }).onOk(() => {
    void resetCTe()
  })
}

async function resetCTe() {
  try {
    const result = await cte.resetCTe()
    if (!result) return
    notifySuccess(`CT-e redefinidos: ${result.CompanyDocuments} CT-e e ${result.Events} eventos removidos.`)
  } catch (error) {
    if (wailsErrorCode(error) === 'sync_running') {
      notifyWarning('Aguarde a sincronização CT-e terminar antes de redefinir.')
    } else {
      notifyError('Erro ao redefinir CT-e', error)
    }
  }
}

function openEvents(chaveAcesso: string) {
  eventsChave.value = chaveAcesso
  showEventsDialog.value = true
}

async function exportXML(chaveAcesso: string) {
  try {
    notifyExported(await cte.exportXML(chaveAcesso), 'XML')
  } catch (error) {
    notifyError('Erro ao exportar XML', error)
  }
}

async function exportZIP(chaves: string[], choice: ExportChoice) {
  try {
    notifyExported(await cte.exportZIP(chaves, choice), 'XML')
  } catch (error) {
    notifyError('Erro ao exportar XMLs', error)
  }
}
</script>

<style scoped>
.cte-filter-cnpj {
  width: 170px;
}

.cte-filter-chave {
  width: 240px;
}
</style>
