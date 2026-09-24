<template>
  <q-page padding>
    <div class="row items-center q-gutter-sm q-mb-sm">
      <h5 class="q-my-none">CT-e</h5>
      <q-badge
        v-if="status"
        v-bind="badgeProps(ambienteColor(status.TpAmb), $q.dark.isActive)"
        :label="ambienteLabel(status.TpAmb)"
        class="text-weight-bold"
      />
      <q-space />
      <q-btn
        flat
        color="negative"
        icon="restart_alt"
        label="Redefinir CT-e"
        title="Remove os CT-e da empresa e reinicia a sincronização CT-e"
        :loading="isResetting || previewingReset"
        :disable="!filter.CNPJ || isSyncing"
        @click="confirmResetCTe"
      />
      <q-btn
        color="primary"
        icon="sync"
        label="Sincronizar CT-e"
        :loading="isSyncing"
        :disable="!filter.CNPJ || isResetting || Boolean(syncBlockedUntil)"
        @click="syncCTe"
      />
    </div>

    <div v-if="status" class="text-caption text-app-muted q-mb-sm">{{ statusLine }}</div>

    <q-banner
      v-if="blockedText"
      dense
      rounded
      class="q-mb-md"
      :class="$q.dark.isActive ? 'bg-grey-9 text-orange-3' : 'bg-orange-1 text-orange-10'"
    >
      <template #avatar>
        <q-icon name="schedule" />
      </template>
      {{ blockedText }}
    </q-banner>

    <div class="row q-gutter-sm items-center q-mb-md q-pa-sm rounded-borders shadow-1">
      <q-select
        v-model="filter.CNPJ"
        class="col-12 col-md-3"
        :options="companyOptions"
        label="Empresa"
        emit-value
        map-options
        outlined
        dense
        options-dense
        :disable="loading"
        @update:model-value="handleCompanyChange"
      />

      <div class="col-12 col-sm-6 col-md-3" title="Competência pelo mês de emissão">
        <CompetencePicker v-model="filter.Competence" :disable="loading" />
      </div>

      <q-select
        v-model="filter.Role"
        class="col-6 col-sm-3 col-md-auto cte-filter-select"
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
        class="col-6 col-sm-3 col-md-auto cte-filter-select"
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
        class="col-6 col-sm-3 col-md-auto cte-filter-select"
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
        @keyup.enter="search"
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
        @keyup.enter="search"
      />

      <q-space />

      <div class="row no-wrap items-center q-gutter-sm">
        <q-btn
          color="primary"
          icon="search"
          label="Buscar"
          :disable="loading || !filter.CNPJ || Boolean(nfeChaveError)"
          :loading="loading"
          dense
          flat
          @click="search"
        />
        <q-toggle
          v-model="incremental"
          label="Somente novos"
          dense
          title="O ZIP leva só os CT-e ainda não exportados, ou cujo XML mudou desde a última exportação"
        />
        <q-btn
          color="secondary"
          icon="folder_zip"
          label="Exportar"
          title="Exporta em ZIP o XML dos CT-e listados, com seus eventos"
          :disable="exporting || filteredRows.length === 0"
          :loading="exporting"
          dense
          flat
          @click="exportZIP"
        />
      </div>
    </div>

    <q-table
      v-model:pagination="pagination"
      :rows="filteredRows"
      :columns="columns"
      row-key="ChaveAcesso"
      :loading="loading"
      no-data-label="Nenhum CT-e encontrado."
      class="cte-table"
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
            class="cte-search-input"
            placeholder="Filtrar por chave, número, emitente ou tomador..."
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
          <q-td v-for="col in rowProps.cols" :key="col.name" :props="rowProps">
            <div v-if="col.name === 'acoes'" class="row no-wrap items-center q-gutter-x-xs">
              <q-btn
                dense
                flat
                round
                size="sm"
                :color="rowProps.expand ? 'primary' : 'grey-7'"
                :icon="rowProps.expand ? 'expand_less' : 'expand_more'"
                title="Ver detalhes do CT-e"
                aria-label="Ver detalhes do CT-e"
                @click.stop="rowProps.expand = !rowProps.expand"
              />
              <q-btn dense flat round size="sm" color="grey-7" icon="more_vert" aria-label="Ações do CT-e">
                <q-menu auto-close>
                  <q-list dense class="cte-row-menu">
                    <q-item
                      clickable
                      :disable="rowProps.row.EventCount === 0"
                      @click="openEvents(rowProps.row.ChaveAcesso)"
                    >
                      <q-item-section>
                        <q-item-label>Eventos ({{ rowProps.row.EventCount }})</q-item-label>
                      </q-item-section>
                    </q-item>
                    <q-item clickable :disable="exporting" @click="exportXML(rowProps.row.ChaveAcesso)">
                      <q-item-section>Exportar XML</q-item-section>
                    </q-item>
                  </q-list>
                </q-menu>
              </q-btn>
            </div>

            <div v-else-if="col.name === 'chave'" class="row no-wrap items-center q-gutter-x-xs">
              <span
                :title="formatChaveDFe(rowProps.row.ChaveAcesso)"
                class="cursor-pointer text-weight-medium text-mono"
                @click="copyChave(rowProps.row.ChaveAcesso)"
              >
                {{ formatChaveAcesso(rowProps.row.ChaveAcesso) }}
              </span>
              <q-btn
                dense
                flat
                round
                size="xs"
                color="grey-7"
                icon="content_copy"
                title="Copiar chave completa"
                aria-label="Copiar chave completa"
                @click.stop="copyChave(rowProps.row.ChaveAcesso)"
              />
            </div>

            <template v-else-if="col.name === 'documento'">
              <q-badge
                v-bind="badgeProps(cteDocumentColor(rowProps.row), $q.dark.isActive)"
                :label="cteDocumentLabel(rowProps.row)"
              />
              <div class="text-mono">{{ col.value }}</div>
            </template>

            <template v-else-if="col.name === 'emitente' || col.name === 'tomador'">
              <div class="text-weight-medium text-mono">{{ formatCpfCnpj(col.value) || '-' }}</div>
              <div
                class="text-caption text-app-muted ellipsis partner-name"
                :title="partyName(rowProps.row, col.name)"
              >
                {{ partyName(rowProps.row, col.name) || '-' }}
              </div>
            </template>

            <div v-else-if="col.name === 'papel'" class="column items-start">
              <q-badge
                v-bind="badgeProps(ctePapelColor(rowProps.row.CompanyRole), $q.dark.isActive)"
                :label="ctePapelLabel(rowProps.row.CompanyRole)"
              />
              <div class="row no-wrap q-gutter-x-xs">
                <q-badge
                  v-for="papel in cteOtherPapeis(rowProps.row)"
                  :key="papel"
                  v-bind="badgeProps(ctePapelColor(papel), $q.dark.isActive)"
                  :label="ctePapelLabel(papel)"
                  class="cte-secondary-papel"
                  :title="`A empresa também é ${ctePapelLabel(papel).toLowerCase()}`"
                />
              </div>
            </div>

            <q-badge
              v-else-if="col.name === 'situacao'"
              v-bind="badgeProps(cteSituacaoColor(rowProps.row.Situacao), $q.dark.isActive)"
              :label="cteSituacaoLabel(rowProps.row.Situacao)"
            />

            <template v-else>{{ col.value }}</template>
          </q-td>
        </q-tr>

        <q-tr
          v-if="rowProps.expand"
          :props="rowProps"
          :class="$q.dark.isActive ? 'bg-grey-10' : 'bg-grey-1'"
        >
          <q-td :colspan="rowProps.cols.length" class="cte-detail-cell">
            <div class="cte-detail q-pa-md">
              <div class="row q-col-gutter-md">
                <div class="col-12 col-md-4">
                  <div class="text-subtitle2 text-primary q-mb-xs">Prestação</div>
                  <dl class="cte-detail-list text-body2">
                    <dt>CFOP</dt>
                    <dd>{{ [rowProps.row.CFOP, rowProps.row.NatOp].filter(Boolean).join(' · ') || '—' }}</dd>
                    <dt>Serviço</dt>
                    <dd>{{ cteTpServLabel(rowProps.row.TpServ) }}</dd>
                    <dt>Modal</dt>
                    <dd>{{ cteModalLabel(rowProps.row.Modal) }}</dd>
                    <dt>Percurso</dt>
                    <dd>{{ ctePercurso(rowProps.row) }}</dd>
                    <dt>Protocolo</dt>
                    <dd class="text-mono">{{ rowProps.row.Protocolo || '—' }}</dd>
                    <dt>Autorização</dt>
                    <dd>{{ formatDateTime(rowProps.row.AuthorizedAt, '—') }}</dd>
                  </dl>
                </div>

                <div class="col-12 col-md-4">
                  <div class="text-subtitle2 text-primary q-mb-xs">Participantes</div>
                  <dl class="cte-detail-list text-body2">
                    <template v-for="party in cteParticipantes(rowProps.row)" :key="party.label">
                      <dt>{{ party.label }}</dt>
                      <dd>
                        {{ party.name || '—' }}
                        <span class="text-caption text-app-muted text-mono">{{ formatCpfCnpj(party.cnpj) }}</span>
                      </dd>
                    </template>
                  </dl>
                </div>

                <div class="col-12 col-md-4">
                  <div class="text-subtitle2 text-primary q-mb-xs">Valores</div>
                  <dl class="cte-detail-list text-body2">
                    <dt>Prestação</dt>
                    <dd class="text-mono">{{ formatCurrencyCents(rowProps.row.TotalValue) }}</dd>
                    <dt>A receber</dt>
                    <dd class="text-mono">{{ formatCurrencyCents(rowProps.row.ReceivableValue) }}</dd>
                    <dt>ICMS</dt>
                    <dd class="text-mono">{{ formatCurrencyCents(rowProps.row.ICMSValue) }}</dd>
                    <dt>Tributos</dt>
                    <dd class="text-mono">{{ formatCurrencyCents(rowProps.row.TotTribValue) }}</dd>
                    <dt>Carga</dt>
                    <dd class="text-mono">{{ formatCurrencyCents(rowProps.row.CargaValue) }}</dd>
                    <dt>Produto</dt>
                    <dd>{{ rowProps.row.ProdutoPredominante || '—' }}</dd>
                  </dl>
                </div>

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
            </div>
          </q-td>
        </q-tr>
      </template>
    </q-table>

    <CTeEventsDialog v-model="showEventsDialog" :cnpj="filter.CNPJ" :chave-acesso="eventsChave" />
  </q-page>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import CompetencePicker from '../components/CompetencePicker.vue'
import CTeEventsDialog from '../components/CTeEventsDialog.vue'
import { useCTeDocuments } from '@/composables/useCTeDocuments'
import { useNotify } from '@/composables/useNotify'
import { wailsErrorCode } from '@/platform/wails/client'
import type { CTeResetResult, CTeRow, ISODateValue } from '@/types/desktop'
import {
  cteDocumentColor,
  cteDocumentLabel,
  cteModalLabel,
  cteModeloFilterOptions,
  cteOtherPapeis,
  ctePapelColor,
  ctePapelFilterOptions,
  ctePapelLabel,
  cteParticipantes,
  ctePercurso,
  cteSituacaoColor,
  cteSituacaoFilterOptions,
  cteSituacaoLabel,
  cteTpServLabel,
} from '@/utils/cteDisplay'
import {
  formatChaveAcesso,
  formatChaveDFe,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
  formatDateTime,
  formatNFeNumber,
} from '@/utils/formatters'
import { ambienteColor, ambienteLabel, badgeProps } from '@/utils/sefazDisplay'

const $q = useQuasar()
const cte = useCTeDocuments()
const { notifyError, notifySyncError, copyChave } = useNotify()

const {
  filter,
  loading,
  exporting,
  incremental,
  status,
  pagination,
  filterText,
  filteredRows,
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

const columns: QTableColumn<CTeRow>[] = [
  { name: 'acoes', label: 'Ações', field: () => '', align: 'left' },
  { name: 'chave', label: 'Chave de Acesso', field: 'ChaveAcesso', align: 'left' },
  {
    name: 'documento',
    label: 'Documento / Número',
    field: (row) => formatNFeNumber(row.Numero, row.Serie),
    align: 'left',
  },
  {
    name: 'issueDate',
    label: 'Emissão',
    field: 'IssueDate',
    sortable: true,
    align: 'left',
    classes: 'text-mono',
    format: (value: ISODateValue) => formatDate(value),
  },
  { name: 'emitente', label: 'Emitente', field: 'EmitenteCNPJ', sortable: true, align: 'left' },
  { name: 'tomador', label: 'Tomador', field: 'TomadorCNPJ', sortable: true, align: 'left' },
  { name: 'papel', label: 'Papel', field: 'CompanyRole', align: 'left' },
  {
    name: 'valor',
    label: 'Prestação (R$)',
    field: 'TotalValue',
    sortable: true,
    align: 'right',
    classes: 'text-mono',
    format: (value: number) => formatCurrencyCents(value),
  },
  { name: 'situacao', label: 'Situação', field: 'Situacao', align: 'left' },
]

function partyName(row: CTeRow, column: string) {
  return column === 'emitente' ? row.EmitenteName : row.TomadorName
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
    $q.notify({
      type: 'positive',
      message: `Sincronização CT-e ${result.Status || 'concluída'}: ${result.DocumentsSaved} CT-e e ${result.EventsSaved} eventos (NSU ${result.LastNSU}/${result.MaxNSU ?? '—'}).`,
    })
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
    $q.notify({
      type: 'positive',
      message: `CT-e redefinidos: ${result.CompanyDocuments} CT-e e ${result.Events} eventos removidos.`,
    })
  } catch (error) {
    if (wailsErrorCode(error) === 'sync_running') {
      $q.notify({ type: 'warning', message: 'Aguarde a sincronização CT-e terminar antes de redefinir.' })
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
    const result = await cte.exportXML(chaveAcesso)
    if (result) {
      $q.notify({ type: 'positive', message: `XML exportado para ${result.OutPath}.` })
    }
  } catch (error) {
    notifyError('Erro ao exportar XML', error)
  }
}

// exportZIP exports the rows the grid shows.
async function exportZIP() {
  try {
    const result = await cte.exportZIP()
    if (!result) return
    if (result.ExportedCount === 0) {
      const message = result.Incremental
        ? 'Nenhum CT-e novo para exportar.'
        : 'Nenhum CT-e encontrado para exportação.'
      $q.notify({ type: 'info', message })
      return
    }
    $q.notify({
      type: 'positive',
      message: `${result.ExportedCount} CT-e exportados para ${result.OutPath}.`,
    })
  } catch (error) {
    notifyError('Erro ao exportar XMLs', error)
  }
}
</script>

<style scoped>
/* Values stay on one line and the table scrolls sideways; party names are
   cut with an ellipsis and only the expanded details wrap. */
.cte-table :deep(td) {
  white-space: nowrap;
}

.cte-table :deep(td.cte-detail-cell) {
  padding: 0;
  white-space: normal;
}

/* The expanded details span every column, which is wider than the visible
   table when it scrolls sideways. Size them to the scroll area (100cqw) and
   pin them to its left edge so they wrap inside what the user sees. */
.cte-table :deep(.q-table__middle) {
  container-type: inline-size;
}

.cte-detail {
  position: sticky;
  left: 0;
  width: 100cqw;
  overflow-wrap: anywhere;
}

.cte-detail-list {
  display: grid;
  grid-template-columns: max-content 1fr;
  column-gap: 12px;
  row-gap: 2px;
  margin: 0;
}

.cte-detail-list dt {
  font-weight: 500;
}

.cte-detail-list dd {
  margin: 0;
}

.cte-filter-select {
  min-width: 130px;
}

.cte-filter-cnpj {
  width: 170px;
}

.cte-filter-chave {
  width: 240px;
}

.cte-search-input {
  width: 350px;
  max-width: 100%;
}

/* About the width of the formatted CNPJ above it, so the name does not widen
   the column. The full name is in the title tooltip. */
.partner-name {
  max-width: 130px;
}

.cte-secondary-papel {
  margin-top: 2px;
  font-size: 10px;
}

.cte-row-menu {
  min-width: 180px;
}
</style>
