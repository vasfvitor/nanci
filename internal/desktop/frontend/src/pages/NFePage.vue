<template>
  <q-page padding>
    <div class="row items-center q-gutter-sm q-mb-sm">
      <h5 class="q-my-none">NF-e (Modelo 55)</h5>
      <q-badge
        v-if="status"
        :color="badgeColor(ambienteColor(status.TpAmb), $q.dark.isActive)"
        :text-color="badgeTextColor(ambienteColor(status.TpAmb), $q.dark.isActive)"
        :label="ambienteLabel(status.TpAmb)"
        class="text-weight-bold"
      />
      <q-space />
      <q-btn
        flat
        color="negative"
        icon="restart_alt"
        label="Redefinir NF-e"
        title="Remove as NF-e da empresa e reinicia a sincronização NF-e"
        :loading="isResetting"
        :disable="!filter.CNPJ || isSyncing"
        @click="confirmResetNFe"
      />
      <q-btn
        color="primary"
        icon="sync"
        label="Sincronizar NF-e"
        :loading="isSyncing"
        :disable="!filter.CNPJ || isResetting || Boolean(syncBlockedUntil)"
        @click="syncNFe"
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
          color="negative"
          :text-color="badgeTextColor('negative', $q.dark.isActive)"
          class="q-ml-sm"
          :label="pendingCount"
        />
      </q-tab>
    </q-tabs>

    <q-tab-panels v-model="activeTab" animated keep-alive>
      <q-tab-panel name="notas" class="q-pa-none">
        <div class="row q-gutter-sm items-center q-mb-md q-pa-sm rounded-borders shadow-1">
          <q-select
            v-model="filter.CNPJ"
            class="col-12 col-md-4"
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
            v-model="filter.Situacao"
            class="col-6 col-sm-3 col-md-2 nfe-filter-select"
            :options="situacaoFilterOptions"
            label="Situação"
            emit-value
            map-options
            outlined
            dense
            options-dense
            :disable="loading"
          />
          <q-select
            v-model="filter.Completeness"
            class="col-6 col-sm-3 col-md-2 nfe-filter-select"
            :options="completenessFilterOptions"
            label="Completude"
            emit-value
            map-options
            outlined
            dense
            options-dense
            :disable="loading"
          />
          <q-select
            v-model="filter.Manifestacao"
            class="col-6 col-sm-3 col-md-2 nfe-filter-select"
            :options="manifestacaoFilterOptions"
            label="Manifestação"
            emit-value
            map-options
            outlined
            dense
            options-dense
            :disable="loading"
          />
          <q-select
            v-model="filter.Role"
            class="col-6 col-sm-3 col-md-2 nfe-filter-select"
            :options="nfeRoleFilterOptions"
            label="Papel"
            emit-value
            map-options
            outlined
            dense
            options-dense
            :disable="loading"
          />

          <q-space />

          <q-btn
            color="primary"
            icon="search"
            label="Buscar"
            :disable="loading || !filter.CNPJ"
            :loading="loading"
            dense
            flat
            @click="search"
          />
          <q-btn
            color="primary"
            icon="task_alt"
            :label="`Registrar ciência (${eligibleSelection.length})`"
            :disable="eligibleSelection.length === 0 || planning || Boolean(cienciaInFlight)"
            :loading="planning"
            dense
            flat
            @click="registerSelectedCiencia"
          />
          <q-btn
            color="secondary"
            icon="folder_zip"
            label="Exportar XML (ZIP)"
            :disable="exporting || filteredRows.length === 0"
            :loading="exporting"
            dense
            flat
            @click="exportZIP"
          />
        </div>

        <q-table
          v-model:pagination="pagination"
          v-model:selected="selected"
          :rows="filteredRows"
          :columns="columns"
          row-key="ChaveAcesso"
          selection="multiple"
          :loading="loading"
          no-data-label="Nenhuma NF-e encontrada."
          class="nfe-table"
          binary-state-sort
          flat
          bordered
          dense
        >
          <template #top>
            <div class="row items-center justify-between full-width">
              <div class="text-subtitle1 text-weight-bold">Notas fiscais eletrônicas</div>
              <q-input
                v-model="filterText"
                class="nfe-search-input"
                placeholder="Filtrar por chave, número ou emitente..."
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

          <template #body-cell-acoes="cellProps">
            <q-td :props="cellProps" auto-width>
              <q-btn
                dense
                flat
                round
                size="sm"
                color="grey-7"
                icon="more_vert"
                aria-label="Ações da nota"
                :disable="isChaveBusy(cellProps.row.ChaveAcesso)"
              >
                <q-menu auto-close>
                  <q-list dense class="nfe-row-menu">
                    <q-item clickable :disable="conclusiveBlockReason(cellProps.row) !== null" @click="openManifestacao(cellProps.row)">
                      <q-item-section>
                        <q-item-label>Manifestar…</q-item-label>
                        <q-item-label v-if="conclusiveBlockReason(cellProps.row)" caption>{{ conclusiveBlockReason(cellProps.row) }}</q-item-label>
                      </q-item-section>
                    </q-item>
                    <q-item
                      clickable
                      :disable="cienciaBlockReason(cellProps.row) !== null || Boolean(cienciaInFlight)"
                      @click="startCiencia([cellProps.row.ChaveAcesso])"
                    >
                      <q-item-section>
                        <q-item-label>Registrar ciência</q-item-label>
                        <q-item-label v-if="cienciaBlockReason(cellProps.row)" caption>{{ cienciaBlockReason(cellProps.row) }}</q-item-label>
                      </q-item-section>
                    </q-item>
                    <q-item clickable @click="openEvents(cellProps.row.ChaveAcesso)">
                      <q-item-section>Eventos</q-item-section>
                    </q-item>
                    <q-item
                      clickable
                      :disable="exporting || cellProps.row.Completeness !== 'completa'"
                      @click="exportXML(cellProps.row.ChaveAcesso)"
                    >
                      <q-item-section>
                        <q-item-label>Exportar XML</q-item-label>
                        <q-item-label v-if="cellProps.row.Completeness !== 'completa'" caption>
                          XML completo ainda não baixado
                        </q-item-label>
                      </q-item-section>
                    </q-item>
                  </q-list>
                </q-menu>
              </q-btn>
            </q-td>
          </template>

          <template #body-cell-chave="cellProps">
            <q-td :props="cellProps">
              <div class="row no-wrap items-center q-gutter-x-xs">
                <span
                  :title="formatChaveNFe(cellProps.row.ChaveAcesso)"
                  class="cursor-pointer text-weight-medium text-mono"
                  @click="copyChave(cellProps.row.ChaveAcesso)"
                >
                  {{ formatChaveAcesso(cellProps.row.ChaveAcesso) }}
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
                  @click.stop="copyChave(cellProps.row.ChaveAcesso)"
                />
              </div>
            </q-td>
          </template>

          <template #body-cell-emitente="cellProps">
            <q-td :props="cellProps">
              <div class="text-weight-medium text-mono">{{ formatCpfCnpj(cellProps.row.EmitenteCNPJ) || '-' }}</div>
              <div class="text-caption text-app-muted partner-name" :title="cellProps.row.EmitenteName">
                {{ cellProps.row.EmitenteName || '-' }}
              </div>
            </q-td>
          </template>

          <template #body-cell-situacao="cellProps">
            <q-td :props="cellProps">
              <q-badge
                :color="badgeColor(situacaoColor(cellProps.row.Situacao), $q.dark.isActive)"
                :text-color="badgeTextColor(situacaoColor(cellProps.row.Situacao), $q.dark.isActive)"
                :label="situacaoLabel(cellProps.row.Situacao)"
              />
            </q-td>
          </template>

          <template #body-cell-completude="cellProps">
            <q-td :props="cellProps">
              <q-badge
                :color="badgeColor(completenessColor(cellProps.row.Completeness), $q.dark.isActive)"
                :text-color="badgeTextColor(completenessColor(cellProps.row.Completeness), $q.dark.isActive)"
                :label="completenessLabel(cellProps.row.Completeness)"
              />
            </q-td>
          </template>

          <template #body-cell-manifestacao="cellProps">
            <q-td :props="cellProps">
              <div class="row no-wrap items-center q-gutter-x-xs">
                <q-spinner v-if="isChaveBusy(cellProps.row.ChaveAcesso)" size="xs" color="primary" />
                <q-badge
                  :color="badgeColor(manifestacaoColor(cellProps.row.Manifestacao), $q.dark.isActive)"
                  :text-color="badgeTextColor(manifestacaoColor(cellProps.row.Manifestacao), $q.dark.isActive)"
                  :label="manifestacaoLabel(cellProps.row.Manifestacao)"
                />
                <q-chip
                  v-if="cellProps.row.Manifestacao === 'ciencia' && cellProps.row.ConclusiveDue"
                  dense
                  square
                  outline
                  size="sm"
                  :color="deadlineColor(cellProps.row.DaysLeft, 'conclusiva')"
                  :label="conclusiveDeadlineLabel(cellProps.row.DaysLeft)"
                  :title="`Prazo da manifestação conclusiva: ${formatDate(cellProps.row.ConclusiveDue)}`"
                />
              </div>
            </q-td>
          </template>

          <template #body-cell-papel="cellProps">
            <q-td :props="cellProps">
              <q-badge
                :color="badgeColor(nfeRoleColor(cellProps.row.CompanyRole), $q.dark.isActive)"
                :text-color="badgeTextColor(nfeRoleColor(cellProps.row.CompanyRole), $q.dark.isActive)"
                :label="nfeRoleLabel(cellProps.row.CompanyRole)"
              />
            </q-td>
          </template>
        </q-table>
      </q-tab-panel>

      <q-tab-panel name="pendencias" class="q-pa-none">
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

    <NFeEventsDialog v-model="showEventsDialog" :cnpj="filter.CNPJ" :chave-acesso="eventsChave" />
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import CienciaConfirmDialog from '../components/CienciaConfirmDialog.vue'
import CompetencePicker from '../components/CompetencePicker.vue'
import ManifestacaoDialog from '../components/ManifestacaoDialog.vue'
import NFeEventResultsDialog from '../components/NFeEventResultsDialog.vue'
import NFeEventsDialog from '../components/NFeEventsDialog.vue'
import NFePendingPanel from '../components/NFePendingPanel.vue'
import { useNFeDocuments } from '@/composables/useNFeDocuments'
import { useNFeManifestation } from '@/composables/useNFeManifestation'
import { useNotify } from '@/composables/useNotify'
import { wailsErrorCode } from '@/platform/wails/client'
import type {
  ISODateValue,
  NFeConclusiveTipo,
  NFeEventBatchResult,
  NFeEventResult,
  NFeRow,
} from '@/types/desktop'
import {
  formatChaveAcesso,
  formatChaveNFe,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
  formatDateTime,
  formatNFeNumber,
  formatTime,
  normalizeText,
} from '@/utils/formatters'
import {
  ambienteColor,
  ambienteLabel,
  badgeColor,
  badgeTextColor,
  blockedMessage,
  completenessColor,
  completenessLabel,
  completenessFilterOptions,
  conclusiveDeadlineLabel,
  deadlineColor,
  manifestacaoColor,
  manifestacaoLabel,
  manifestacaoFilterOptions,
  nfeRoleColor,
  nfeRoleLabel,
  nfeRoleFilterOptions,
  situacaoColor,
  situacaoLabel,
  situacaoFilterOptions,
} from '@/utils/nfeDisplay'
import { cienciaBlockReason, conclusiveBlockReason, countOutcomes } from '@/utils/nfeManifestation'

const $q = useQuasar()
const nfe = useNFeDocuments()
const manifestation = useNFeManifestation()
const { notifyError, copyChave } = useNotify()

const {
  filter,
  rows,
  selected,
  loading,
  exporting,
  status,
  activeTab,
  pagination,
  companyOptions,
  isSyncing,
  isResetting,
  syncBlockedUntil,
} = nfe
const { pending, pendingLoading, cienciaInFlight, isChaveBusy } = manifestation

const filterText = ref('')
const planning = ref(false)
const showEventsDialog = ref(false)
const eventsChave = ref('')

const columns: QTableColumn<NFeRow>[] = [
  { name: 'acoes', label: 'Ações', field: () => '', align: 'center' },
  {
    name: 'emissao',
    label: 'Emissão',
    field: 'IssueDate',
    sortable: true,
    align: 'left',
    classes: 'text-no-wrap text-mono',
    format: (value: ISODateValue) => formatDate(value),
  },
  {
    name: 'numero',
    label: 'Número / Série',
    field: (row) => formatNFeNumber(row.Numero, row.Serie),
    align: 'left',
    classes: 'text-no-wrap text-mono',
  },
  { name: 'chave', label: 'Chave de Acesso', field: 'ChaveAcesso', align: 'left' },
  { name: 'emitente', label: 'Emitente', field: 'EmitenteCNPJ', sortable: true, align: 'left' },
  {
    name: 'valor',
    label: 'Valor (R$)',
    field: 'TotalValue',
    sortable: true,
    align: 'right',
    classes: 'text-mono',
    format: (value: number) => formatCurrencyCents(value),
  },
  { name: 'situacao', label: 'Situação', field: 'Situacao', align: 'left' },
  { name: 'completude', label: 'Completude', field: 'Completeness', align: 'left' },
  { name: 'manifestacao', label: 'Manifestação', field: 'Manifestacao', align: 'left' },
  { name: 'papel', label: 'Papel', field: 'CompanyRole', align: 'left' },
]

// searchIndex normalizes the searchable fields once per result set, not on
// every keystroke.
const searchIndex = computed(() =>
  rows.value.map((row) => ({
    row,
    fields: [row.ChaveAcesso, row.Numero, row.EmitenteCNPJ, row.EmitenteName].map(normalizeText),
  }))
)

const filteredRows = computed(() => {
  const query = normalizeText(filterText.value)
  if (!query) return rows.value
  return searchIndex.value
    .filter(({ fields }) => fields.some((field) => field.includes(query)))
    .map(({ row }) => row)
})

const eligibleSelection = computed(() => selected.value.filter((row) => !cienciaBlockReason(row)))

const pendingCount = computed(
  () => (status.value?.PendingCiencia ?? 0) + (status.value?.PendingConclusiva ?? 0)
)

const statusLine = computed(() => {
  if (!status.value) return ''
  const maxNSU = status.value.MaxNSU ?? '—'
  return [
    `Última sincronização: ${formatDateTime(status.value.LastSyncAt, 'nunca')}`,
    `NSU ${status.value.LastCheckedNSU}/${maxNSU}`,
    `Pendências: ${pendingCount.value}`,
  ].join(' · ')
})

const blockedText = computed(() => {
  if (!syncBlockedUntil.value || !status.value) return ''
  return blockedMessage(status.value, formatTime(syncBlockedUntil.value))
})

const companyName = computed(() => {
  if (status.value?.CompanyName) return status.value.CompanyName
  const option = companyOptions.value.find((item) => item.value === filter.value.CNPJ)
  return option?.label ?? ''
})

watch(filterText, () => {
  pagination.value.page = 1
})

onMounted(() => {
  void loadCompanies()
})

async function loadCompanies() {
  try {
    await nfe.loadCompanies()
    const known = companyOptions.value.some((option) => option.value === filter.value.CNPJ)
    if (!known) {
      filter.value.CNPJ = companyOptions.value[0]?.value ?? ''
    }
    if (filter.value.CNPJ) {
      await refreshAll()
    }
  } catch (error) {
    notifyError('Erro ao carregar empresas', error)
  }
}

async function refreshAll() {
  await Promise.all([search(), loadStatus(), loadPending()])
}

async function handleCompanyChange() {
  selected.value = []
  if (!filter.value.CNPJ) return
  await refreshAll()
}

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
    await manifestation.loadPending()
  } catch (error) {
    notifyError('Erro ao carregar pendências', error)
  }
}

async function syncNFe() {
  try {
    const result = await nfe.syncNFe()
    if (!result) return
    $q.notify({
      type: 'positive',
      message: `Sincronização NF-e ${result.Status || 'concluída'}: ${result.CompletasSaved} completas, ${result.ResumosSaved} resumos, ${result.EventsSaved} eventos (NSU ${result.UltNSU}/${result.MaxNSU ?? '—'}).`,
    })
  } catch (error) {
    const code = wailsErrorCode(error)
    if (code === 'canceled') {
      $q.notify({ type: 'warning', message: 'Sincronização cancelada.' })
    } else if (code === 'sync_running') {
      $q.notify({ type: 'warning', message: 'Sincronização já em andamento para esta empresa.' })
    } else if (code === 'sefaz_blocked') {
      $q.notify({ type: 'warning', message: 'Consultas bloqueadas no momento. Aguarde o horário indicado.' })
    } else {
      notifyError('Erro na sincronização da NF-e', error)
    }
  }
}

function confirmResetNFe() {
  if (!filter.value.CNPJ) return
  const notes =
    (status.value?.TotalDestinatario ?? 0) +
    (status.value?.TotalEmitente ?? 0) +
    (status.value?.TotalOutros ?? 0)
  $q.dialog({
    title: 'Redefinir NF-e',
    message:
      `Remove as ${notes} NF-e de ${companyName.value}, com seus eventos e marcas de exportação, ` +
      'e reinicia a sincronização NF-e desde o NSU 0. Notas vistas por outra empresa continuam para ela. ' +
      'O histórico das manifestações enviadas é mantido, e as manifestações registradas na SEFAZ não são afetadas. ' +
      'Depois disso o ambiente da empresa pode ser alterado.',
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
    $q.notify({
      type: 'positive',
      message: `NF-e redefinidas: ${result.CompanyDocuments} notas e ${result.Events} eventos removidos.`,
      caption: `${result.ManifestationsKept} manifestações enviadas mantidas no histórico.`,
    })
  } catch (error) {
    if (wailsErrorCode(error) === 'sync_running') {
      $q.notify({ type: 'warning', message: 'Aguarde a sincronização NF-e terminar antes de redefinir.' })
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
  if (chavesAcesso.length === 0 || planning.value) return

  planning.value = true
  let plan
  try {
    plan = await manifestation.planCiencia(chavesAcesso)
  } catch (error) {
    notifyError('Erro ao preparar a ciência', error)
    return
  } finally {
    planning.value = false
  }
  if (!plan) return

  if (plan.Eligible.length === 0) {
    const reason = plan.Skipped[0]?.Reason
    $q.notify({
      type: 'warning',
      message: reason
        ? `Nenhuma nota elegível para ciência. Motivo: ${reason}`
        : 'Nenhuma nota elegível para ciência.',
    })
    return
  }

  $q.dialog({
    component: CienciaConfirmDialog,
    componentProps: {
      companyName: companyName.value,
      cnpj: filter.value.CNPJ,
      tpAmb: status.value?.TpAmb ?? '',
      plan,
    },
  }).onOk((eligible: string[]) => {
    void sendCiencia(eligible)
  })
}

async function sendCiencia(chavesAcesso: string[]) {
  try {
    const result = await manifestation.registerCiencia(chavesAcesso)
    if (!result) {
      $q.notify({ type: 'warning', message: 'Já existe um envio de ciência em andamento.' })
      return
    }
    notifyCienciaResult(result)
  } catch (error) {
    if (wailsErrorCode(error) === 'canceled') {
      $q.notify({ type: 'warning', message: 'Envio de ciência cancelado.' })
    } else {
      notifyError('Erro ao registrar ciência', error)
    }
  }
}

function notifyCienciaResult(result: NFeEventBatchResult) {
  const counts = countOutcomes(result.Results)
  const hasProblems = counts.rejeitada > 0 || counts.nao_enviada > 0 || Boolean(result.Interrupted)
  $q.notify({
    type: hasProblems ? 'warning' : 'positive',
    message: `Ciência: ${counts.registrada} registradas, ${counts.ja_registrada} já registradas, ${counts.rejeitada} rejeitadas, ${counts.nao_enviada} não enviadas.`,
    caption: 'O XML completo chega na próxima sincronização.',
    timeout: 10000,
    actions: [
      {
        label: 'Sincronizar agora',
        color: 'white',
        handler: () => {
          void syncNFe()
        },
      },
    ],
  })

  if (hasProblems) {
    $q.dialog({ component: NFeEventResultsDialog, componentProps: { result } })
  }
}

function openManifestacao(row: NFeRow) {
  $q.dialog({
    component: ManifestacaoDialog,
    componentProps: {
      note: row,
      tpAmb: status.value?.TpAmb ?? '',
    },
  }).onOk((payload: { tipo: NFeConclusiveTipo; justificativa: string }) => {
    void sendManifestation(row.ChaveAcesso, payload.tipo, payload.justificativa)
  })
}

async function sendManifestation(chaveAcesso: string, tipo: NFeConclusiveTipo, justificativa: string) {
  try {
    const result = await manifestation.registerManifestation(chaveAcesso, tipo, justificativa)
    if (!result) {
      $q.notify({ type: 'warning', message: 'Esta nota já tem um envio em andamento.' })
      return
    }
    notifyManifestationResult(result)
  } catch (error) {
    if (wailsErrorCode(error) === 'canceled') {
      $q.notify({ type: 'warning', message: 'Envio da manifestação cancelado.' })
    } else {
      notifyError('Erro ao registrar manifestação', error)
    }
  }
}

function notifyManifestationResult(result: NFeEventResult) {
  const detail = [result.CStat, result.XMotivo].filter(Boolean).join(' - ')
  switch (result.Status) {
    case 'registrada':
      $q.notify({ type: 'positive', message: `Manifestação registrada. Protocolo ${result.Protocolo || '-'}.` })
      return
    case 'ja_registrada':
      $q.notify({ type: 'info', message: 'A manifestação já estava registrada na SEFAZ.' })
      return
    case 'rejeitada':
      $q.notify({ type: 'negative', message: `Manifestação rejeitada pela SEFAZ: ${detail}` })
      return
    default:
      $q.notify({ type: 'warning', message: `Manifestação não enviada. ${detail}`.trim() })
  }
}

function openEvents(chaveAcesso: string) {
  eventsChave.value = chaveAcesso
  showEventsDialog.value = true
}

async function exportXML(chaveAcesso: string) {
  try {
    const result = await nfe.exportXML(chaveAcesso)
    if (result) {
      $q.notify({ type: 'positive', message: `XML exportado para ${result.OutPath}.` })
    }
  } catch (error) {
    notifyError('Erro ao exportar XML', error)
  }
}

// exportZIP exports the selected rows, or else every row the grid shows
// after the filters and the text search.
async function exportZIP() {
  const target = selected.value.length > 0 ? selected.value : filteredRows.value
  const chavesAcesso = target.map((row) => row.ChaveAcesso)
  try {
    const result = await nfe.exportZIP(chavesAcesso)
    if (!result) return
    const skipped =
      result.SkippedResumos > 0
        ? ` ${result.SkippedResumos} resumos ignorados (XML completo ainda não baixado).`
        : ''
    if (result.ExportedCount === 0) {
      $q.notify({ type: 'info', message: `Nenhum XML completo encontrado para exportação.${skipped}` })
      return
    }
    $q.notify({
      type: 'positive',
      message: `${result.ExportedCount} XMLs exportados para ${result.OutPath}.${skipped}`,
    })
  } catch (error) {
    notifyError('Erro ao exportar XMLs', error)
  }
}
</script>

<style scoped>
/* Values stay on one line and the table scrolls sideways; only the emitente
   name wraps. */
.nfe-table :deep(td) {
  white-space: nowrap;
}

.nfe-filter-select {
  min-width: 140px;
}

.nfe-search-input {
  width: 350px;
  max-width: 100%;
}

.partner-name {
  min-width: 220px;
  max-width: 280px;
  white-space: normal;
  overflow-wrap: break-word;
}

.nfe-row-menu {
  min-width: 220px;
}
</style>
