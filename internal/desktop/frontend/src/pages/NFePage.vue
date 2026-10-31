<template>
  <q-page padding>
    <DocumentPageHeader
      title="NF-e"
      :ambiente="ambiente"
      :context-line="contextLine"
      :status-line="statusLine"
      :idle-warning="idleWarning"
      :blocked-text="blockedText"
      :syncing="isSyncing"
      :cancelable="canCancelSync"
      :cancelling="isCancelling"
      :sync-disabled="!cnpj || isResetting || Boolean(syncBlockedUntil)"
      reset-title="Remove as NF-e da empresa e reinicia a sincronização NF-e"
      :resetting="isResetting"
      :reset-disabled="!cnpj || isSyncing"
      @sync="syncNFe"
      @cancel="nfe.cancelSync"
      @reset="confirmResetNFe"
    />

    <q-tabs
      v-model="activeTab"
      dense
      inline-label
      align="left"
      active-color="primary"
      indicator-color="primary"
      class="q-mb-md"
    >
      <q-tab name="notas" label="Notas" />
      <q-tab name="pendencias" label="Pendências">
        <q-badge
          v-if="pendingCount > 0"
          v-bind="badgeProps('negative', $q.dark.isActive)"
          class="q-ml-sm"
          :label="pendingCount"
        />
      </q-tab>
    </q-tabs>

    <q-tab-panels v-model="activeTab" animated keep-alive class="bg-transparent">
      <q-tab-panel name="notas" class="q-pa-none">
        <DocumentFilterBar
          v-model:only-unviewed="onlyUnviewed"
          :loading="loading"
          :exporting="exporting"
          :export-disabled="scopeRows.length === 0"
          :search-disabled="!cnpj"
          :mark-viewed-count="unviewedChaves.length"
          @search="search"
          @mark-viewed="confirmMarkViewed"
          @export="openExportDialog"
        >
          <DocumentFilterSelect
            v-model="filter.Situacao"
            :options="situacaoFilterOptions"
            label="Situação"
            :disable="loading"
          />
          <DocumentFilterSelect
            v-model="filter.Completeness"
            :options="completenessFilterOptions"
            label="Completude"
            :disable="loading"
          />
          <DocumentFilterSelect
            v-model="filter.Manifestacao"
            :options="manifestacaoFilterOptions"
            label="Manifestação"
            :disable="loading"
          />
          <DocumentFilterSelect
            v-model="filter.Role"
            :options="nfeRoleFilterOptions"
            label="Papel"
            :disable="loading"
          />

          <template #actions>
            <q-btn
              color="primary"
              icon="task_alt"
              :label="`Registrar ciência (${eligibleSelection.length})`"
              :disable="
                eligibleSelection.length === 0 || planningCiencia || Boolean(cienciaInFlight)
              "
              :loading="planningCiencia"
              dense
              flat
              @click="registerSelectedCiencia"
            />
            <q-btn
              color="primary"
              icon="upload_file"
              label="Importar XML"
              title="Importa XMLs que a empresa já tem, sem consultar a SEFAZ"
              :disable="importDisabled"
              dense
              flat
              @click="importXML"
            />
            <q-btn
              color="primary"
              icon="travel_explore"
              label="Buscar pela chave"
              title="Pede uma NF-e à SEFAZ pela chave de acesso"
              :disable="importDisabled"
              dense
              flat
              @click="openFetchDialog"
            />
          </template>
        </DocumentFilterBar>

        <StateLegend :sections="NFE_LEGEND" class="q-mb-md" />

        <q-table
          v-model:pagination="pagination"
          v-model:selected="selected"
          :rows="filteredRows"
          :columns="columns"
          row-key="ChaveAcesso"
          selection="multiple"
          :loading="loading"
          :no-data-label="cnpj ? 'Nenhuma NF-e encontrada.' : noCompanyLabel"
          class="document-table"
          binary-state-sort
          flat
          bordered
          dense
        >
          <template #top>
            <DocumentTableTop v-model="filterText" title="Notas fiscais eletrônicas" />
          </template>

          <template #body="rowProps">
            <q-tr :props="rowProps">
              <q-td auto-width>
                <q-checkbox v-model="rowProps.selected" dense />
              </q-td>
              <q-td v-for="col in rowProps.cols" :key="col.name" :props="rowProps">
                <NFeRowMenu
                  v-if="col.name === 'acoes'"
                  v-model:expanded="rowProps.expand"
                  :row="rowProps.row"
                  :busy="isChaveBusy(rowProps.row.ChaveAcesso)"
                  :ciencia-in-flight="Boolean(cienciaInFlight)"
                  :exporting="exporting"
                  @manifest="openManifestacao(rowProps.row)"
                  @ciencia="startCiencia([rowProps.row.ChaveAcesso])"
                  @events="openEvents(rowProps.row.ChaveAcesso)"
                  @export-xml="exportXML(rowProps.row.ChaveAcesso)"
                  @export-danfe="exportDANFE(rowProps.row.ChaveAcesso)"
                />

                <NumeroCell
                  v-else-if="col.name === 'numero'"
                  :numero="col.value"
                  :serie="rowProps.row.Serie"
                  :viewed-at="rowProps.row.ViewedAt"
                />

                <div v-else-if="col.name === 'chave'" class="column items-start no-wrap">
                  <ChaveCell :chave="rowProps.row.ChaveAcesso" />
                  <q-badge
                    v-if="rowProps.row.FirstSeenNSU === null"
                    outline
                    color="primary"
                    label="Importado"
                    title="Veio de um XML importado, não da SEFAZ"
                  />
                </div>

                <PartyCell
                  v-else-if="col.name === 'emitente'"
                  :name="rowProps.row.EmitenteName"
                  :cnpj="rowProps.row.EmitenteCNPJ"
                />

                <PartyCell
                  v-else-if="col.name === 'destinatario'"
                  :name="rowProps.row.DestinatarioName"
                  :cnpj="rowProps.row.DestinatarioCNPJ"
                />

                <StateBadges
                  v-else-if="col.name === 'estados'"
                  :badges="badgesByChave.get(rowProps.row.ChaveAcesso) ?? []"
                >
                  <template
                    v-if="
                      isChaveBusy(rowProps.row.ChaveAcesso) || showsConclusiveDeadline(rowProps.row)
                    "
                    #default
                  >
                    <q-spinner
                      v-if="isChaveBusy(rowProps.row.ChaveAcesso)"
                      size="xs"
                      color="primary"
                    />
                    <q-chip
                      v-if="showsConclusiveDeadline(rowProps.row)"
                      dense
                      square
                      outline
                      size="sm"
                      class="q-ma-none"
                      :color="deadlineColor(rowProps.row.DaysLeft, 'conclusiva')"
                      :label="conclusiveDeadlineLabel(rowProps.row.DaysLeft)"
                      :title="`Prazo da manifestação conclusiva: ${formatDate(rowProps.row.ConclusiveDue)}`"
                    />
                  </template>
                </StateBadges>

                <template v-else>{{ col.value }}</template>
              </q-td>
            </q-tr>

            <DocumentDetailRow v-if="rowProps.expand" :row-props="rowProps">
              <div class="row q-col-gutter-md">
                <DetailList class="col-12 col-md-5" title="Nota" :items="notaItems(rowProps.row)" />
                <DetailList
                  class="col-12 col-md-4"
                  title="Emitente"
                  :items="emitenteItems(rowProps.row)"
                />
                <div class="col-12 col-md-3">
                  <DetailList
                    title="Destinatário"
                    :items="destinatarioItems(rowProps.row)"
                    class="q-mb-md"
                  />
                  <DetailList title="Valores" :items="valoresItems(rowProps.row)" />
                </div>

                <ParseWarnings class="col-12" :warnings="rowProps.row.ParseWarnings" />
              </div>
            </DocumentDetailRow>
          </template>
        </q-table>
      </q-tab-panel>

      <q-tab-panel name="pendencias" class="q-pa-none">
        <StateLegend :sections="NFE_PENDING_LEGEND" class="q-mb-md" />
        <NFePendingPanel
          :rows="pending"
          :loading="pendingLoading"
          :busy="isChaveBusy"
          @ciencia="(pendingRows) => startCiencia(pendingRows.map((row) => row.ChaveAcesso))"
          @manifest="openManifestacao"
          @events="(row) => openEvents(row.ChaveAcesso)"
        />
      </q-tab-panel>
    </q-tab-panels>

    <NFeEventsDialog v-model="showEventsDialog" :cnpj="cnpj" :chave-acesso="eventsChave" />
  </q-page>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useQuasar } from 'quasar'
import ChaveCell from '../components/ChaveCell.vue'
import DetailList, { type DetailItem } from '../components/DetailList.vue'
import DocumentDetailRow from '../components/DocumentDetailRow.vue'
import DocumentFilterBar from '../components/DocumentFilterBar.vue'
import DocumentFilterSelect from '../components/DocumentFilterSelect.vue'
import DocumentPageHeader from '../components/DocumentPageHeader.vue'
import DocumentTableTop from '../components/DocumentTableTop.vue'
import type { ExportChoice } from '../components/ExportDialog.vue'
import NFeEventsDialog from '../components/NFeEventsDialog.vue'
import NFePendingPanel from '../components/NFePendingPanel.vue'
import NFeRowMenu from '../components/NFeRowMenu.vue'
import NumeroCell from '../components/NumeroCell.vue'
import ParseWarnings from '../components/ParseWarnings.vue'
import PartyCell from '../components/PartyCell.vue'
import StateBadges from '../components/StateBadges.vue'
import StateLegend from '../components/StateLegend.vue'
import { useDocumentListActions } from '@/composables/useDocumentListActions'
import { useNFeDocuments } from '@/composables/useNFeDocuments'
import { useNFeImportActions } from '@/composables/useNFeImportActions'
import { useNFeManifestacao } from '@/composables/useNFeManifestacao'
import { useNFeManifestacaoActions } from '@/composables/useNFeManifestacaoActions'
import { useNotify } from '@/composables/useNotify'
import { useWorkspaceList } from '@/composables/useWorkspaceList'
import { wailsErrorCode } from '@/platform/wails/client'
import type { NFeExportFormat, NFeRow } from '@/types/desktop'
import { documentColumns } from '@/utils/documentColumns'
import { exportedNoun, NFE_EXPORT_FORMATS } from '@/utils/exportFormats'
import {
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
  formatDateTime,
  formatNFeNumber,
} from '@/utils/formatters'
import { ambienteLabel, badgeProps } from '@/utils/sefazDisplay'
import { NFE_LEGEND, NFE_PENDING_LEGEND } from '@/utils/stateLegends'
import {
  completenessFilterOptions,
  conclusiveDeadlineLabel,
  deadlineColor,
  manifestacaoFilterOptions,
  nfeRoleFilterOptions,
  situacaoFilterOptions,
  situacaoLabel,
  showsConclusiveDeadline,
} from '@/utils/nfeDisplay'

const $q = useQuasar()
const nfe = useNFeDocuments()
const manifestacao = useNFeManifestacao()
const { notifyError, notifySuccess, notifyInfo, notifyWarning, notifyExported, notifySyncError } =
  useNotify()

const {
  filter,
  selected,
  loading,
  exporting,
  status,
  activeTab,
  pagination,
  filterText,
  filteredRows,
  scopeRows,
  unviewedChaves,
  badgesByChave,
  onlyUnviewed,
  companyName,
  ambiente,
  pendingCount,
  noteCount,
  statusLine,
  idleWarning,
  isSyncing,
  canCancelSync,
  isCancelling,
  isResetting,
  syncBlockedUntil,
  blockedText,
} = nfe
const {
  pending,
  pendingLoading,
  planningCiencia,
  cienciaInFlight,
  eligibleSelection,
  isChaveBusy,
} = manifestacao

const showEventsDialog = ref(false)
const eventsChave = ref('')

const { confirmMarkViewed, openExportDialog } = useDocumentListActions({
  source: 'nfe',
  selected,
  scopeRows,
  unviewedChaves,
  markViewed: nfe.markViewed,
  exportFormats: NFE_EXPORT_FORMATS,
  exportList,
})

const columns = documentColumns<NFeRow>({
  emitenteLabel: 'Emitente',
  destinatarioLabel: 'Destinatário',
  numero: (row) => formatNFeNumber(row.Numero),
  emitente: (row) => row.EmitenteName || row.EmitenteCNPJ,
  destinatario: (row) => row.DestinatarioName || row.DestinatarioCNPJ,
  valor: (row) => row.TotalValue,
})

function notaItems(row: NFeRow): DetailItem[] {
  return [
    { label: 'Número', value: formatNFeNumber(row.Numero), mono: true },
    { label: 'Série', value: row.Serie, mono: true },
    { label: 'Natureza da operação', value: row.NatOp },
    { label: 'Layout', value: row.LayoutVersion, mono: true },
    { label: 'Ambiente', value: row.TpAmb ? ambienteLabel(row.TpAmb) : '' },
    { label: 'Situação', value: row.Situacao ? situacaoLabel(row.Situacao) : '' },
    { label: 'Protocolo', value: row.Protocolo, mono: true },
    { label: 'Autorizada em', value: formatDateTime(row.AuthorizedAt, '') },
  ]
}

function emitenteItems(row: NFeRow): DetailItem[] {
  return [
    { label: 'Nome', value: row.EmitenteName },
    { label: 'CNPJ', value: formatCpfCnpj(row.EmitenteCNPJ), mono: true },
    { label: 'IE', value: row.EmitenteIE, mono: true },
    { label: 'UF', value: row.EmitenteUF },
  ]
}

function destinatarioItems(row: NFeRow): DetailItem[] {
  return [
    { label: 'Nome', value: row.DestinatarioName },
    { label: 'CNPJ', value: formatCpfCnpj(row.DestinatarioCNPJ), mono: true },
  ]
}

// A resumo carries only the total, so its taxes show as "—".
function valoresItems(row: NFeRow): DetailItem[] {
  const completa = row.Completeness === 'completa'
  return [
    { label: 'Total', value: formatCurrencyCents(row.TotalValue), mono: true },
    { label: 'ICMS', value: completa ? formatCurrencyCents(row.ICMSValue) : '', mono: true },
    { label: 'IPI', value: completa ? formatCurrencyCents(row.IPIValue) : '', mono: true },
  ]
}

// The page lists the workspace company and competência. A new company also
// reloads the status and the pendências; a new competência only the list.
// The pendências are loaded with the status, so they go with it.
const { cnpj, noCompanyLabel, contextLine } = useWorkspaceList({
  rowsFor: () => nfe.rowsFor.value,
  clearRows: () => nfe.clearRows(),
  loadRows: search,
  clearCompany: (companyCNPJ) => {
    if (status.value?.CNPJ === companyCNPJ) return
    status.value = null
    pending.value = []
  },
  loadCompany: async () => {
    await Promise.all([loadStatus(), loadPending()])
  },
})

const { registerSelectedCiencia, startCiencia, openManifestacao } = useNFeManifestacaoActions({
  cnpj,
  companyName,
  tpAmb: computed(() => status.value?.TpAmb ?? ''),
  syncNFe,
})

const { importXML, openFetchDialog } = useNFeImportActions({
  cnpj,
  status,
  importXML: nfe.importXML,
  fetchByChave: nfe.fetchByChave,
})

async function search() {
  try {
    await nfe.search()
  } catch (error) {
    notifyError('Erro ao buscar NF-e', error)
  }
}

async function loadStatus() {
  try {
    await nfe.loadStatus()
  } catch (error) {
    notifyError('Erro ao carregar o status da NF-e', error)
  }
}

async function loadPending() {
  try {
    await manifestacao.loadPending()
  } catch (error) {
    notifyError('Erro ao carregar pendências', error)
  }
}

async function syncNFe() {
  try {
    const result = await nfe.syncNFe()
    if (!result) return
    notifySuccess(
      `Sincronização NF-e ${result.Status || 'concluída'}: ${result.CompletasSaved} completas, ${result.ResumosSaved} resumos, ${result.EventsSaved} eventos (NSU ${result.LastNSU}/${result.MaxNSU ?? '—'}).`
    )
  } catch (error) {
    notifySyncError('Erro na sincronização da NF-e', error)
  }
}

// The import and the fetch hold the NF-e sync marker, so they wait for a
// pull, a reset or an export to finish.
const importDisabled = computed(
  () => exporting.value || isSyncing.value || isResetting.value || !cnpj.value
)

function confirmResetNFe() {
  if (!cnpj.value) return
  $q.dialog({
    title: 'Redefinir NF-e',
    message:
      `Remove as NF-e de ${companyName.value} nos dois ambientes (${noteCount.value} no ambiente atual), ` +
      'com seus eventos e marcas de exportação, e reinicia a sincronização NF-e desde o NSU 0. ' +
      'Notas vistas por outra empresa continuam para ela. O histórico das manifestações enviadas é mantido, ' +
      'e as manifestações registradas na SEFAZ não são afetadas.',
    cancel: true,
    persistent: true,
    ok: { label: 'Redefinir', color: 'negative' },
  }).onOk(() => {
    void resetNFe()
  })
}

async function resetNFe() {
  try {
    const result = await nfe.resetNFe()
    if (!result) return
    selected.value = []
    notifySuccess(
      `NF-e redefinidas: ${result.CompanyDocuments} notas e ${result.Events} eventos removidos.`,
      { caption: `${result.ManifestacoesKept} manifestações enviadas mantidas no histórico.` }
    )
  } catch (error) {
    if (wailsErrorCode(error) === 'sync_running') {
      notifyWarning('Aguarde a sincronização NF-e terminar antes de redefinir.')
    } else {
      notifyError('Erro ao redefinir NF-e', error)
    }
  }
}

function openEvents(chaveAcesso: string) {
  eventsChave.value = chaveAcesso
  showEventsDialog.value = true
}

async function exportXML(chaveAcesso: string) {
  try {
    notifyExported(await nfe.exportXML(chaveAcesso), 'XML')
  } catch (error) {
    notifyError('Erro ao exportar XML', error)
  }
}

async function exportDANFE(chaveAcesso: string) {
  try {
    notifyExported(await nfe.exportDANFE(chaveAcesso), 'DANFE')
  } catch (error) {
    notifyError('Erro ao exportar DANFE', error)
  }
}

// exportList reports the export and, apart, the resumos it left out because
// their complete XML has not arrived yet.
async function exportList(chaves: string[], choice: ExportChoice) {
  try {
    const result = await nfe.exportDocuments(choice.format as NFeExportFormat, chaves, choice)
    notifyExported(result, exportedNoun(choice.format))
    if (result && result.SkippedResumos > 0) {
      notifyInfo(
        result.SkippedResumos === 1
          ? '1 resumo ignorado: o XML completo ainda não foi baixado.'
          : `${result.SkippedResumos} resumos ignorados: o XML completo ainda não foi baixado.`
      )
    }
  } catch (error) {
    notifyError('Erro ao exportar NF-e', error)
  }
}
</script>
