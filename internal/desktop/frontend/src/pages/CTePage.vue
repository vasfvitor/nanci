<template>
  <q-page padding>
    <DocumentPageHeader
      title="CT-e"
      :ambiente="ambiente"
      :status-line="statusLine"
      :blocked-text="blockedText"
      :syncing="isSyncing"
      :sync-disabled="!filter.CNPJ || isResetting || Boolean(syncBlockedUntil)"
      reset-label="Redefinir CT-e"
      reset-title="Remove os CT-e da empresa e reinicia a sincronização CT-e"
      :resetting="isResetting || previewingReset"
      :reset-disabled="!filter.CNPJ || isSyncing"
      @sync="syncCTe"
      @reset="confirmResetCTe"
    />

    <DocumentFilterBar
      v-model:cnpj="filter.CNPJ"
      v-model:competence="filter.Competence"
      v-model:only-unviewed="onlyUnviewed"
      :company-options="companyOptions"
      :loading="loading"
      :exporting="exporting"
      :export-disabled="scopeRows.length === 0"
      :search-disabled="Boolean(nfeChaveError)"
      :mark-viewed-count="unviewedChaves.length"
      @search="search"
      @company-change="handleCompanyChange"
      @mark-viewed="confirmMarkViewed"
      @export="openExportDialog"
    >
      <q-select
        v-model="filter.Role"
        class="col-6 col-sm-3 col-md-auto document-filter-select"
        :options="ctePapelFilterOptions"
        label="Papel"
        emit-value
        map-options
        outlined
        dense
        options-dense
        :disable="loading"
      />
      <q-select
        v-model="filter.Modelo"
        class="col-6 col-sm-3 col-md-auto document-filter-select"
        :options="cteModeloFilterOptions"
        label="Modelo"
        emit-value
        map-options
        outlined
        dense
        options-dense
        :disable="loading"
      />
      <q-select
        v-model="filter.Situacao"
        class="col-6 col-sm-3 col-md-auto document-filter-select"
        :options="cteSituacaoFilterOptions"
        label="Situação"
        emit-value
        map-options
        outlined
        dense
        options-dense
        :disable="loading"
      />
      <q-input
        v-model="filter.TomadorCNPJ"
        class="cte-filter-cnpj"
        label="CNPJ do tomador"
        outlined
        dense
        clearable
        :disable="loading"
      />
      <q-input
        v-model="filter.NFeChave"
        class="cte-filter-chave"
        label="Chave de NF-e transportada"
        outlined
        dense
        clearable
        hide-bottom-space
        :error="Boolean(nfeChaveError)"
        :error-message="nfeChaveError"
        :disable="loading"
      />
    </DocumentFilterBar>

    <StateLegend :sections="cteLegend()" class="q-mb-md" />

    <q-table
      v-model:pagination="pagination"
      v-model:selected="selected"
      :rows="filteredRows"
      :columns="columns"
      row-key="ChaveAcesso"
      selection="multiple"
      :loading="loading"
      :no-data-label="filter.CNPJ ? 'Nenhum CT-e encontrado.' : 'Selecione uma empresa.'"
      class="document-table"
      binary-state-sort
      flat
      bordered
      dense
    >
      <template #top>
        <div class="row items-center justify-between full-width">
          <div class="text-subtitle1 text-weight-bold">Conhecimentos de transporte eletrônicos</div>
          <q-input
            v-model="filterText"
            class="document-search-input"
            placeholder="Filtrar por chave, número, nome ou CNPJ..."
            outlined
            dense
            clearable
            debounce="300"
          >
            <template #append>
              <q-icon name="search" />
            </template>
          </q-input>
        </div>
      </template>

      <template #body="rowProps">
        <q-tr :props="rowProps">
          <q-td auto-width>
            <q-checkbox v-model="rowProps.selected" dense />
          </q-td>
          <q-td v-for="col in rowProps.cols" :key="col.name" :props="rowProps">
            <RowActionsMenu v-if="col.name === 'acoes'" v-model:expanded="rowProps.expand" noun="do CT-e">
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

            <div v-else-if="col.name === 'numero'" :title="col.value">
              <div class="text-mono ellipsis numero-cell">{{ formatNFeNumber(rowProps.row.Numero) || '—' }}</div>
              <div class="row no-wrap items-center q-gutter-x-xs">
                <span v-if="rowProps.row.Serie" class="text-caption text-app-muted">
                  série {{ rowProps.row.Serie }}
                </span>
                <q-badge
                  v-if="!rowProps.row.ViewedAt"
                  v-bind="badgeProps('warning', $q.dark.isActive)"
                  label="Novo"
                />
              </div>
            </div>

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

            <StateBadges v-else-if="col.name === 'estados'" :badges="cteStateBadges(rowProps.row)" />

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

            <div v-if="rowProps.row.ParseWarnings.length > 0" class="col-12">
              <div class="text-subtitle2 text-primary q-mb-xs">Avisos da leitura do XML</div>
              <ul class="q-my-none q-pl-md text-body2">
                <li v-for="warning in rowProps.row.ParseWarnings" :key="warning">{{ warning }}</li>
              </ul>
            </div>
          </div>
        </DocumentDetailRow>
      </template>
    </q-table>

    <CTeEventsDialog v-model="showEventsDialog" :cnpj="filter.CNPJ" :chave-acesso="eventsChave" />
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useQuasar } from 'quasar'
import ChaveCell from '../components/ChaveCell.vue'
import CTeEventsDialog from '../components/CTeEventsDialog.vue'
import DetailList, { type DetailItem } from '../components/DetailList.vue'
import DocumentDetailRow from '../components/DocumentDetailRow.vue'
import DocumentFilterBar from '../components/DocumentFilterBar.vue'
import DocumentPageHeader from '../components/DocumentPageHeader.vue'
import type { ExportChoice } from '../components/ExportDialog.vue'
import PartyCell from '../components/PartyCell.vue'
import RowActionsMenu from '../components/RowActionsMenu.vue'
import RowMenuItem from '../components/RowMenuItem.vue'
import StateBadges from '../components/StateBadges.vue'
import StateLegend from '../components/StateLegend.vue'
import { useCTeDocuments } from '@/composables/useCTeDocuments'
import { useDocumentListActions } from '@/composables/useDocumentListActions'
import { useNotify } from '@/composables/useNotify'
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
  cteStateBadges,
  cteTpServLabel,
} from '@/utils/cteDisplay'
import {
  formatChaveDFe,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDateTime,
  formatNFeNumber,
} from '@/utils/formatters'
import { ambienteColor, ambienteLabel, badgeProps } from '@/utils/sefazDisplay'
import { cteLegend } from '@/utils/stateLegends'

const $q = useQuasar()
const cte = useCTeDocuments()
const { notifyError, notifySuccess, notifyWarning, notifyExported, notifySyncError, copyChave } =
  useNotify()

const {
  filter,
  selected,
  loading,
  exporting,
  status,
  pagination,
  filterText,
  filteredRows,
  scopeRows,
  unviewedChaves,
  onlyUnviewed,
  companyName,
  companyOptions,
  statusLine,
  nfeChaveError,
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
  noun: 'CT-e',
  selected,
  scopeRows,
  unviewedChaves,
  markViewed: cte.markViewed,
  exportFormats: [{ label: 'XMLs (ZIP)', value: 'zip' }],
  exportList: exportZIP,
})

const ambiente = computed(() =>
  status.value
    ? { label: ambienteLabel(status.value.TpAmb), color: ambienteColor(status.value.TpAmb) }
    : null
)

const columns = documentColumns<CTeRow>({
  emitenteLabel: 'Emitente',
  destinatarioLabel: 'Tomador',
  // "000.021.502 / 1": the header already says the second part is the série.
  numero: (row) => [formatNFeNumber(row.Numero), row.Serie].filter(Boolean).join(' / '),
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

onMounted(() => {
  void loadCompanies()
})

async function loadCompanies() {
  try {
    await cte.loadCompanies()
    if (filter.value.CNPJ) {
      await refreshAll()
    }
  } catch (error) {
    notifyError('Erro ao carregar empresas', error)
  }
}

async function refreshAll() {
  await Promise.all([search(), loadStatus()])
}

async function handleCompanyChange() {
  selected.value = []
  if (!filter.value.CNPJ) return
  await refreshAll()
}

async function search() {
  if (nfeChaveError.value) return
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

.document-search-input {
  width: 350px;
  max-width: 100%;
}

.numero-cell {
  max-width: 100px;
}
</style>
