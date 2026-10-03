<template>
  <q-page padding>
    <DocumentPageHeader
      title="NFS-e"
      :ambiente="ambiente"
      :status-line="statusLine"
      :syncing="isSyncing"
      :sync-disabled="!filter.CNPJ || isResetting"
      reset-title="Reinicia a sincronização da NFS-e a partir do NSU 0; os documentos ficam"
      :resetting="isResetting"
      :reset-disabled="!filter.CNPJ || isSyncing"
      @sync="syncNFSe"
      @reset="confirmResetSync"
    />

    <DocumentFilterBar
      v-model:cnpj="filter.CNPJ"
      v-model:competence="filter.Competence"
      v-model:only-unviewed="onlyUnviewed"
      :company-options="companyOptions"
      :loading="loading"
      :exporting="exporting"
      :export-disabled="scopeRows.length === 0"
      :mark-viewed-count="unviewedChaves.length"
      @search="search"
      @company-change="handleCompanyChange"
      @mark-viewed="confirmMarkViewed"
      @export="openExportDialog"
    >
      <DocumentFilterSelect v-model="filter.Direction" :options="nfseRoleFilterOptions" label="Papel" :disable="loading" />
    </DocumentFilterBar>

    <StateLegend :sections="NFSE_LEGEND" class="q-mb-md" />

    <q-table
      v-model:pagination="pagination"
      v-model:selected="selected"
      :rows="filteredRows"
      :columns="columns"
      row-key="ChaveAcesso"
      selection="multiple"
      :loading="loading"
      :no-data-label="filter.CNPJ ? 'Nenhuma NFS-e encontrada.' : 'Selecione uma empresa.'"
      class="document-table"
      binary-state-sort
      flat
      bordered
      dense
    >
      <template #top>
        <DocumentTableTop v-model="filterText" title="Notas fiscais de serviço" />
      </template>

      <template #body="rowProps">
        <q-tr :props="rowProps">
          <q-td auto-width>
            <q-checkbox v-model="rowProps.selected" dense />
          </q-td>
          <q-td v-for="col in rowProps.cols" :key="col.name" :props="rowProps">
            <RowActionsMenu v-if="col.name === 'acoes'" v-model:expanded="rowProps.expand" source="nfse">
              <RowMenuItem label="Eventos" @click="openEvents(rowProps.row)" />
              <RowMenuItem
                label="Exportar XML"
                :disable="exporting"
                @click="exportXML(rowProps.row.ChaveAcesso)"
              />
              <RowMenuItem
                label="Exportar DANFSe"
                :disable="exporting"
                @click="exportDANFSe(rowProps.row.ChaveAcesso)"
              />
            </RowActionsMenu>

            <NumeroCell v-else-if="col.name === 'numero'" :numero="col.value" :viewed-at="rowProps.row.ViewedAt" />

            <ChaveCell v-else-if="col.name === 'chave'" :chave="rowProps.row.ChaveAcesso" />

            <PartyCell
              v-else-if="col.name === 'emitente'"
              :name="rowProps.row.PrestadorName"
              :cnpj="rowProps.row.PrestadorCNPJ"
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
            <div class="col-12 col-md-7">
              <DetailList title="Serviço" :items="servicoItems(rowProps.row)" />
              <div class="text-weight-medium text-body2 q-mt-sm">Descrição do serviço</div>
              <div class="text-body2 service-description">
                {{ rowProps.row.ServiceDescription || '—' }}
              </div>
            </div>
            <DetailList
              class="col-12 col-md-5"
              title="Retenções e tributos"
              :items="retencoesItems(rowProps.row)"
            />

            <ParseWarnings class="col-12" :warnings="rowProps.row.ParseWarnings" />
          </div>
        </DocumentDetailRow>
      </template>
    </q-table>

    <DocumentEventsDialog
      v-model="showEventsDialog"
      :document-id="eventsDocument.id"
      :chave-acesso="eventsDocument.chave"
    />
  </q-page>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar } from 'quasar'
import ChaveCell from '../components/ChaveCell.vue'
import DetailList, { type DetailItem } from '../components/DetailList.vue'
import DocumentDetailRow from '../components/DocumentDetailRow.vue'
import DocumentEventsDialog from '../components/DocumentEventsDialog.vue'
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
import { useDocumentListActions } from '@/composables/useDocumentListActions'
import { useDocuments } from '@/composables/useDocuments'
import { useNotify } from '@/composables/useNotify'
import type { DocumentRow, ExportFormat, ExportResult } from '@/types/desktop'
import { documentColumns } from '@/utils/documentColumns'
import { formatCompetence, formatCpfCnpj, formatCurrencyCents } from '@/utils/formatters'
import { nfseRoleFilterOptions, nfseSyncSummary } from '@/utils/nfseDisplay'
import { NFSE_LEGEND } from '@/utils/stateLegends'

const $q = useQuasar()
const route = useRoute()
const nfse = useDocuments()
const { notifyError, notifySuccess, notifyExported, notifySyncError } = useNotify()

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
  companyOptions,
  selectedCompany,
  ambiente,
  statusLine,
  isSyncing,
  isResetting,
} = nfse

const showEventsDialog = ref(false)
const eventsDocument = ref({ id: '', chave: '' })


const exportFormats = [
  { label: 'Planilha CSV', value: 'csv' },
  { label: 'Planilha Excel (XLSX)', value: 'xlsx' },
  { label: 'XMLs originais (ZIP)', value: 'zip' },
  { label: 'DANFSes (ZIP)', value: 'danfse-zip' },
]

const { confirmMarkViewed, openExportDialog } = useDocumentListActions({
  source: 'nfse',
  selected,
  scopeRows,
  unviewedChaves,
  markViewed: nfse.markViewed,
  exportFormats,
  exportList,
})

// The NFS-e has no série, so the number column says only "Número".
const columns = documentColumns<DocumentRow>({
  emitenteLabel: 'Prestador',
  destinatarioLabel: 'Tomador',
  numeroLabel: 'Número',
  numero: (row) => row.NFSeNumber,
  emitente: (row) => row.PrestadorName || row.PrestadorCNPJ,
  destinatario: (row) => row.TomadorName || row.TomadorCNPJ,
  valor: (row) => row.ServiceValue,
})

function servicoItems(row: DocumentRow): DetailItem[] {
  const items: DetailItem[] = [
    { label: 'Número', value: row.NFSeNumber, mono: true },
    { label: 'Competência', value: formatCompetence(row.Competence), mono: true },
    { label: 'Versão do layout', value: row.LayoutVersion },
  ]
  if (row.IntermediarioCNPJ || row.IntermediarioName) {
    items.push({
      label: 'Intermediário',
      value: row.IntermediarioName,
      caption: formatCpfCnpj(row.IntermediarioCNPJ),
    })
  }
  return items
}

function retencoesItems(row: DocumentRow): DetailItem[] {
  return [
    { label: 'ISS retido', value: formatCurrencyCents(row.ISSValue), mono: true },
    { label: 'IRRF', value: formatCurrencyCents(row.IRRFValue), mono: true },
    { label: 'INSS', value: formatCurrencyCents(row.INSSValue), mono: true },
    { label: 'PIS', value: formatCurrencyCents(row.PISValue), mono: true },
    { label: 'COFINS', value: formatCurrencyCents(row.COFINSValue), mono: true },
    { label: 'CSLL', value: formatCurrencyCents(row.CSLLValue), mono: true },
    { label: 'Total das retenções', value: formatCurrencyCents(row.TotalRetentions), mono: true },
  ]
}

onMounted(() => {
  void loadCompanies()
})

function routeParam(param: unknown): string {
  return Array.isArray(param) ? String(param[0] ?? '') : String(param ?? '')
}

// loadCompanies opens the page on the company and competência of the route
// (?cnpj=&competence=, from the Empresas page), else on the company already
// picked, else on the first one.
async function loadCompanies() {
  const competence = routeParam(route.query['competence'])
  if (/^\d{4}-\d{2}$/.test(competence)) {
    filter.value.Competence = competence
  }
  try {
    await nfse.loadCompanies(routeParam(route.query['cnpj']))
    if (filter.value.CNPJ) {
      await search()
    }
  } catch (error) {
    notifyError('Erro ao carregar empresas', error)
  }
}

async function handleCompanyChange() {
  selected.value = []
  if (!filter.value.CNPJ) return
  await search()
}

async function search() {
  try {
    await nfse.search()
  } catch (error) {
    notifyError('Erro ao buscar NFS-e', error)
  }
}

async function syncNFSe() {
  try {
    const result = await nfse.syncNFSe()
    if (result) notifySuccess(nfseSyncSummary(result))
  } catch (error) {
    notifySyncError('Erro na sincronização da NFS-e', error)
  }
}

// confirmResetSync asks before restarting the NFS-e sync; it moves only the
// cursor, and the confirmation says so.
function confirmResetSync() {
  $q.dialog({
    title: 'Redefinir NFS-e',
    message: 'Reinicia a sincronização da NFS-e a partir do NSU 0. Os documentos já baixados ficam.',
    cancel: true,
    persistent: true,
    ok: { label: 'Redefinir', color: 'negative' },
  }).onOk(() => {
    void resetSync()
  })
}

async function resetSync() {
  const name = selectedCompany.value?.Name ?? ''
  try {
    if (!(await nfse.resetSync())) return
    notifySuccess(
      name ? `Sincronização da NFS-e redefinida para ${name}.` : 'Sincronização da NFS-e redefinida.'
    )
  } catch (error) {
    notifyError('Erro ao redefinir a sincronização da NFS-e', error)
  }
}

function openEvents(row: DocumentRow) {
  eventsDocument.value = { id: row.DocumentID, chave: row.ChaveAcesso }
  showEventsDialog.value = true
}

async function exportXML(chaveAcesso: string) {
  try {
    notifyExported(await nfse.exportXML(chaveAcesso), 'XML')
  } catch (error) {
    notifyError('Erro ao exportar XML', error)
  }
}

async function exportDANFSe(chaveAcesso: string) {
  try {
    notifyExported(await nfse.exportDANFSe(chaveAcesso), 'DANFSe')
  } catch (error) {
    notifyError('Erro ao exportar DANFSe', error)
  }
}

async function exportList(chaves: string[], choice: ExportChoice) {
  try {
    let result: ExportResult | null
    let noun: string
    if (choice.format === 'danfse-zip') {
      result = await nfse.exportDANFSeZIP(chaves, choice)
      noun = 'DANFSe'
    } else {
      result = await nfse.exportDocuments(choice.format as ExportFormat, chaves, choice)
      noun = choice.format === 'zip' ? 'XML' : 'documento'
    }
    notifyExported(result, noun)
  } catch (error) {
    notifyError('Erro ao exportar NFS-e', error)
  }
}
</script>

<style scoped>
.service-description {
  white-space: pre-wrap;
}
</style>
