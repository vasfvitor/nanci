<template>
  <q-page padding>
    <DocumentPageHeader
      title="NF-e"
      :ambiente="ambiente"
      :status-line="statusLine"
      :blocked-text="blockedText"
      :syncing="isSyncing"
      :sync-disabled="!cnpj || isResetting || Boolean(syncBlockedUntil)"
      reset-title="Remove as NF-e da empresa e reinicia a sincronização NF-e"
      :resetting="isResetting"
      :reset-disabled="!cnpj || isSyncing"
      @sync="syncNFe"
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
          v-model:cnpj="cnpj"
          v-model:competence="competence"
          v-model:only-unviewed="onlyUnviewed"
          :company-options="companyOptions"
          :loading="loading"
          :exporting="exporting"
          :export-disabled="scopeRows.length === 0"
          :mark-viewed-count="unviewedChaves.length"
          @search="search"
          @mark-viewed="confirmMarkViewed"
          @export="openExportDialog"
        >
          <DocumentFilterSelect v-model="filter.Situacao" :options="situacaoFilterOptions" label="Situação" :disable="loading" />
          <DocumentFilterSelect v-model="filter.Completeness" :options="completenessFilterOptions" label="Completude" :disable="loading" />
          <DocumentFilterSelect v-model="filter.Manifestacao" :options="manifestacaoFilterOptions" label="Manifestação" :disable="loading" />
          <DocumentFilterSelect v-model="filter.Role" :options="nfeRoleFilterOptions" label="Papel" :disable="loading" />

          <template #actions>
            <q-btn
              color="primary"
              icon="task_alt"
              :label="`Registrar ciência (${eligibleSelection.length})`"
              :disable="eligibleSelection.length === 0 || planningCiencia || Boolean(cienciaInFlight)"
              :loading="planningCiencia"
              dense
              flat
              @click="registerSelectedCiencia"
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
                <RowActionsMenu v-if="col.name === 'acoes'" v-model:expanded="rowProps.expand" source="nfe">
                  <RowMenuItem
                    label="Manifestar…"
                    :caption="manifestarCaption(rowProps.row)"
                    :disable="!canManifest(rowProps.row)"
                    @click="openManifestacao(rowProps.row)"
                  />
                  <RowMenuItem
                    label="Registrar ciência"
                    :caption="cienciaCaption(rowProps.row)"
                    :disable="!canRegisterCiencia(rowProps.row)"
                    @click="startCiencia([rowProps.row.ChaveAcesso])"
                  />
                  <RowMenuItem
                    :label="`Eventos (${rowProps.row.EventCount})`"
                    :disable="rowProps.row.EventCount === 0"
                    @click="openEvents(rowProps.row.ChaveAcesso)"
                  />
                  <RowMenuItem
                    label="Exportar XML"
                    :caption="nfeRowActions(rowProps.row).canExportXML ? '' : 'XML completo ainda não baixado'"
                    :disable="exporting || !nfeRowActions(rowProps.row).canExportXML"
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
                  :name="rowProps.row.DestinatarioName"
                  :cnpj="rowProps.row.DestinatarioCNPJ"
                />

                <StateBadges v-else-if="col.name === 'estados'" :badges="badgesByChave.get(rowProps.row.ChaveAcesso) ?? []">
                  <template
                    v-if="isChaveBusy(rowProps.row.ChaveAcesso) || showsConclusiveDeadline(rowProps.row)"
                    #default
                  >
                    <q-spinner v-if="isChaveBusy(rowProps.row.ChaveAcesso)" size="xs" color="primary" />
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
                <DetailList class="col-12 col-md-4" title="Emitente" :items="emitenteItems(rowProps.row)" />
                <div class="col-12 col-md-3">
                  <DetailList title="Destinatário" :items="destinatarioItems(rowProps.row)" class="q-mb-md" />
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
import NFeCienciaConfirmDialog from '../components/NFeCienciaConfirmDialog.vue'
import NFeEventResultsDialog from '../components/NFeEventResultsDialog.vue'
import NFeEventsDialog from '../components/NFeEventsDialog.vue'
import NFeManifestacaoDialog from '../components/NFeManifestacaoDialog.vue'
import NFePendingPanel from '../components/NFePendingPanel.vue'
import NumeroCell from '../components/NumeroCell.vue'
import ParseWarnings from '../components/ParseWarnings.vue'
import PartyCell from '../components/PartyCell.vue'
import RowActionsMenu from '../components/RowActionsMenu.vue'
import RowMenuItem from '../components/RowMenuItem.vue'
import StateBadges from '../components/StateBadges.vue'
import StateLegend from '../components/StateLegend.vue'
import { companyOption } from '@/composables/useCompanies'
import { useDocumentListActions } from '@/composables/useDocumentListActions'
import { useNFeDocuments } from '@/composables/useNFeDocuments'
import { useNFeManifestacao } from '@/composables/useNFeManifestacao'
import { useNotify } from '@/composables/useNotify'
import { useWorkspaceList } from '@/composables/useWorkspaceList'
import { wailsErrorCode } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import type { NFeConclusiveTipo, NFeEventBatchResult, NFeEventResult, NFeRow } from '@/types/desktop'
import { documentColumns } from '@/utils/documentColumns'
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
import {
  countOutcomes,
  isProblemOutcome,
  nfeRowActions,
  noEligibleCienciaMessage,
} from '@/utils/nfeManifestacao'

const $q = useQuasar()
const nfe = useNFeDocuments()
const workspace = useWorkspaceStore()
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
  competence,
  companyName,
  ambiente,
  pendingCount,
  noteCount,
  statusLine,
  isSyncing,
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
  exportFormats: [{ label: 'XMLs (ZIP)', value: 'zip' }],
  showIncludeResumos: true,
  exportList: exportZIP,
})

const columns = documentColumns<NFeRow>({
  emitenteLabel: 'Emitente',
  destinatarioLabel: 'Destinatário',
  numero: (row) => formatNFeNumber(row.Numero),
  emitente: (row) => row.EmitenteName || row.EmitenteCNPJ,
  destinatario: (row) => row.DestinatarioName || row.DestinatarioCNPJ,
  valor: (row) => row.TotalValue,
})

// A note with an event being sent takes no other; its menu says why.
const SENDING_CAPTION = 'Envio em andamento'

function canManifest(row: NFeRow) {
  return nfeRowActions(row).conclusiveBlockReason === null && !isChaveBusy(row.ChaveAcesso)
}

function manifestarCaption(row: NFeRow) {
  if (isChaveBusy(row.ChaveAcesso)) return SENDING_CAPTION
  return nfeRowActions(row).conclusiveBlockReason ?? ''
}

function canRegisterCiencia(row: NFeRow) {
  return (
    nfeRowActions(row).cienciaBlockReason === null &&
    !cienciaInFlight.value &&
    !isChaveBusy(row.ChaveAcesso)
  )
}

function cienciaCaption(row: NFeRow) {
  if (isChaveBusy(row.ChaveAcesso)) return SENDING_CAPTION
  return nfeRowActions(row).cienciaBlockReason ?? ''
}

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

// The filter bar picks from the workspace companies until the drawer does.
const companyOptions = computed(() => workspace.companies.map(companyOption))

// The page lists the workspace company and competência. A new company also
// reloads the status; a new competência only the list.
const { cnpj, noCompanyLabel } = useWorkspaceList({
  rowsFor: () => nfe.rowsFor.value,
  clear: ({ company }) => {
    nfe.clearRows()
    if (!company) return
    status.value = null
    pending.value = []
  },
  reload: async ({ company }) => {
    await (company ? Promise.all([search(), loadStatus(), loadPending()]) : search())
  },
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

function registerSelectedCiencia() {
  if (eligibleSelection.value.length === 0) return
  void startCiencia(selected.value.map((row) => row.ChaveAcesso))
}

// startCiencia asks the backend which notes can receive ciência, then asks
// the user to confirm exactly those notes.
async function startCiencia(chavesAcesso: string[]) {
  let plan
  try {
    plan = await manifestacao.planCiencia(chavesAcesso)
  } catch (error) {
    notifyError('Erro ao preparar a ciência', error)
    return
  }
  if (!plan) return

  const noEligible = noEligibleCienciaMessage(plan)
  if (noEligible) {
    notifyWarning(noEligible)
    return
  }

  $q.dialog({
    component: NFeCienciaConfirmDialog,
    componentProps: {
      companyName: companyName.value,
      cnpj: cnpj.value,
      tpAmb: status.value?.TpAmb ?? '',
      plan,
    },
  }).onOk((eligible: string[]) => {
    void sendCiencia(eligible)
  })
}

async function sendCiencia(chavesAcesso: string[]) {
  try {
    const result = await manifestacao.registerCiencia(chavesAcesso)
    if (!result) {
      notifyWarning('Já existe um envio de ciência em andamento.')
      return
    }
    notifyCienciaResult(result)
  } catch (error) {
    if (wailsErrorCode(error) === 'canceled') {
      notifyWarning('Envio de ciência cancelado.')
    } else {
      notifyError('Erro ao registrar ciência', error)
    }
  }
}

function notifyCienciaResult(result: NFeEventBatchResult) {
  const counts = countOutcomes(result.Results)
  const hasProblems = result.Results.some(isProblemOutcome) || Boolean(result.Interrupted)
  const notify = hasProblems ? notifyWarning : notifySuccess
  notify(
    `Ciência: ${counts.registrada} registradas, ${counts.ja_registrada} já registradas, ${counts.rejeitada} rejeitadas, ${counts.nao_enviada} não enviadas.`,
    {
      caption: 'O XML completo chega na próxima sincronização.',
      timeout: 10000,
      actions: [
        {
          label: 'Sincronizar agora',
          handler: () => {
            void syncNFe()
          },
        },
      ],
    }
  )

  if (hasProblems) {
    $q.dialog({ component: NFeEventResultsDialog, componentProps: { result } })
  }
}

function openManifestacao(row: NFeRow) {
  $q.dialog({
    component: NFeManifestacaoDialog,
    componentProps: {
      note: row,
      tpAmb: status.value?.TpAmb ?? '',
    },
  }).onOk((payload: { tipo: NFeConclusiveTipo; justificativa: string }) => {
    void sendManifestacao(row.ChaveAcesso, payload.tipo, payload.justificativa)
  })
}

async function sendManifestacao(chaveAcesso: string, tipo: NFeConclusiveTipo, justificativa: string) {
  try {
    const result = await manifestacao.registerManifestacao(chaveAcesso, tipo, justificativa)
    if (!result) {
      notifyWarning('Esta nota já tem um envio em andamento.')
      return
    }
    notifyManifestacaoResult(result)
  } catch (error) {
    if (wailsErrorCode(error) === 'canceled') {
      notifyWarning('Envio da manifestação cancelado.')
    } else {
      notifyError('Erro ao registrar manifestação', error)
    }
  }
}

function notifyManifestacaoResult(result: NFeEventResult) {
  const detail = [result.CStat, result.XMotivo].filter(Boolean).join(' - ')
  switch (result.Status) {
    case 'registrada':
      notifySuccess(`Manifestação registrada. Protocolo ${result.Protocolo || '—'}.`)
      return
    case 'ja_registrada':
      notifyInfo('A manifestação já estava registrada na SEFAZ.')
      return
    case 'rejeitada':
      notifyError('Manifestação rejeitada pela SEFAZ', detail)
      return
    default:
      notifyWarning(`Manifestação não enviada. ${detail}`.trim())
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

// exportZIP reports the export and, apart, the resumos it left out because
// their complete XML has not arrived yet.
async function exportZIP(chaves: string[], choice: ExportChoice) {
  try {
    const result = await nfe.exportZIP(chaves, choice)
    notifyExported(result, 'XML')
    if (result && result.SkippedResumos > 0) {
      notifyInfo(
        result.SkippedResumos === 1
          ? '1 resumo ignorado: o XML completo ainda não foi baixado.'
          : `${result.SkippedResumos} resumos ignorados: o XML completo ainda não foi baixado.`
      )
    }
  } catch (error) {
    notifyError('Erro ao exportar XMLs', error)
  }
}
</script>
