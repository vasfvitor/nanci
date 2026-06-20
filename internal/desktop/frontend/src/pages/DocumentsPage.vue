<template>
  <q-page padding>
    <div class="row items-center justify-between q-mb-md">
      <h5 class="q-my-none">Documentos Fiscais</h5>
    </div>

    <div class="row q-gutter-sm items-center q-mb-md q-pa-sm rounded-borders shadow-1">
      <q-select
v-model="filter.CNPJ" class="col-12 col-md-3" :options="companyOptions" label="Empresa" emit-value
        map-options outlined dense options-dense :disable="loading" @update:model-value="handleCompanyChange" />

      <div class="col-12 col-md-3">
        <div class="row no-wrap items-center q-gutter-xs">
          <q-btn
color="grey-7" icon="chevron_left" dense flat round :disable="loading || !filter.Competence"
            title="Competência anterior" aria-label="Competência anterior" @click="shiftCompetence(-1)" />

          <q-input
v-model="filter.Competence" class="col" label="Competência" outlined dense clearable mask="####-##"
            :disable="loading">
            <template #append>
              <q-icon name="event" class="cursor-pointer">
                <q-popup-proxy ref="datePopup" cover transition-show="scale" transition-hide="scale">
                  <q-date
v-model="filter.Competence" minimal mask="YYYY-MM" emit-immediately default-view="Months"
                    years-in-month-view @update:model-value="onDateChange">
                    <div class="row items-center justify-end">
                      <q-btn label="Mês Atual" color="primary" flat @click="setToday" />
                      <q-btn v-close-popup label="Fechar" color="primary" flat />
                    </div>
                  </q-date>
                </q-popup-proxy>
              </q-icon>
            </template>
          </q-input>

          <q-btn
color="grey-7" icon="chevron_right" dense flat round :disable="loading || !filter.Competence"
            title="Próxima competência" aria-label="Próxima competência" @click="shiftCompetence(1)" />
        </div>
      </div>

      <q-select
v-model="filter.Direction" class="col-12 col-md-2" :options="directionOptions" label="Direção"
        emit-value map-options outlined dense options-dense :disable="loading" />

      <q-space />

      <q-btn
color="primary" icon="search" label="Buscar" :disable="loading || !filter.CNPJ" :loading="loading" dense
        flat @click="search" />

      <q-btn-dropdown
color="secondary" label="Exportar" :disable="exporting || documents.length === 0"
        :loading="exporting" dense flat>
        <q-list dense>
          <q-item v-close-popup clickable @click="exportData('csv')">
            <q-item-section avatar>
              <q-icon name="table_view" />
            </q-item-section>
            <q-item-section>
              <q-item-label>Exportar CSV</q-item-label>
            </q-item-section>
          </q-item>

          <q-item v-close-popup clickable @click="exportData('xlsx')">
            <q-item-section avatar>
              <q-icon name="grid_on" />
            </q-item-section>
            <q-item-section>
              <q-item-label>Exportar XLSX</q-item-label>
            </q-item-section>
          </q-item>

          <q-item v-close-popup clickable @click="exportData('zip')">
            <q-item-section avatar>
              <q-icon name="folder_zip" />
            </q-item-section>
            <q-item-section>
              <q-item-label>Exportar XMLs (ZIP)</q-item-label>
            </q-item-section>
          </q-item>

          <q-item v-close-popup clickable @click="exportDanfseZip">
            <q-item-section avatar>
              <q-icon name="picture_as_pdf" />
            </q-item-section>
            <q-item-section>
              <q-item-label>Exportar DANFSes (ZIP)</q-item-label>
            </q-item-section>
          </q-item>
        </q-list>
      </q-btn-dropdown>
    </div>

    <q-table
v-model:pagination="pagination" :rows="filteredDocuments" :columns="columns" row-key="RelationID"
      :loading="loading" no-data-label="Nenhum documento encontrado." binary-state-sort flat bordered dense
      class="full-height">
      <template #top>
        <div class="column full-width q-gutter-y-sm">
          <div class="row items-center justify-between full-width">
            <div class="text-subtitle1 text-weight-bold">
              Documentos Fiscais Carregados
            </div>

            <q-input
v-model="filterText" class="document-search-input"
              placeholder="Filtrar por nome, CNPJ, número ou chave..." outlined dense clearable debounce="300">
              <template #append>
                <q-icon name="search" />
              </template>
            </q-input>
          </div>

          <div :class="['column q-gutter-y-xs text-caption q-pa-md rounded-borders full-width custom-border-dashed', $q.dark.isActive ? 'bg-grey-10 text-grey-4' : 'bg-grey-2 text-grey-8']">
            <div :class="['text-weight-bold q-mb-xs', $q.dark.isActive ? 'text-grey-3' : 'text-grey-9']">
              Legenda dos Indicadores:
            </div>

            <div class="row items-center q-gutter-x-sm">
              <span :class="['text-weight-bold legend-label-width', $q.dark.isActive ? 'text-grey-3' : 'text-grey-9']">Direção (D):</span>

              <div class="row items-center q-gutter-x-md">
                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="primary" label="P" class="text-weight-bold text-mono" dense />
                  <span>Prestada</span>
                </span>

                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="secondary" label="T" class="text-weight-bold text-mono" dense />
                  <span>Tomada</span>
                </span>

                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="accent" label="I" class="text-weight-bold text-mono" dense />
                  <span>Intermediária</span>
                </span>
              </div>
            </div>

            <div class="row items-center q-gutter-x-sm">
              <span :class="['text-weight-bold legend-label-width', $q.dark.isActive ? 'text-grey-3' : 'text-grey-9']">Visibilidade (V):</span>

              <div class="row items-center q-gutter-x-md">
                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="positive" label="PE" class="text-weight-bold text-mono" dense />
                  <span>Prestador Exato</span>
                </span>

                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="positive" label="TE" class="text-weight-bold text-mono" dense />
                  <span>Tomador Exato</span>
                </span>

                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="positive" label="IE" class="text-weight-bold text-mono" dense />
                  <span>Intermediário Exato</span>
                </span>

                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="warning" label="MR" class="text-weight-bold text-mono text-dark" dense />
                  <span>Mesmo Raiz</span>
                </span>
              </div>
            </div>

            <div class="row items-center q-gutter-x-sm">
              <span :class="['text-weight-bold legend-label-width', $q.dark.isActive ? 'text-grey-3' : 'text-grey-9']">Status (S):</span>

              <div class="row items-center q-gutter-x-md">
                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="positive" label="N" class="text-weight-bold text-mono" dense />
                  <span>Normal</span>
                </span>

                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="negative" label="C" class="text-weight-bold text-mono" dense />
                  <span>Cancelada</span>
                </span>

                <span class="row items-center q-gutter-x-xs">
                  <q-badge color="negative" label="S" class="text-weight-bold text-mono" dense />
                  <span>Substituída</span>
                </span>
              </div>
            </div>
          </div>
        </div>
      </template>

      <template #body="props">
        <q-tr :props="props">
          <q-td v-for="col in props.cols" :key="col.name" :props="props">
            <template v-if="col.name === 'actions'">
              <div class="row no-wrap items-center justify-center q-gutter-x-xs">
                <q-btn
dense flat round size="sm" :color="props.expand ? 'primary' : 'grey-7'"
                  :icon="props.expand ? 'expand_less' : 'expand_more'" title="Ver detalhes do serviço e impostos"
                  aria-label="Ver detalhes do serviço e impostos" @click.stop="props.expand = !props.expand" />

                <q-btn
dense flat round size="sm" color="primary" icon="code" title="Exportar XML Original"
                  aria-label="Exportar XML Original" :disable="exporting || !props.row.ChaveAcesso"
                  @click.stop="exportXML(props.row.ChaveAcesso)" />

                <q-btn
dense flat round size="sm" color="negative" icon="picture_as_pdf" title="Exportar DANFSe"
                  aria-label="Exportar DANFSe" :disable="exporting || !props.row.ChaveAcesso"
                  @click.stop="exportDanfse(props.row.ChaveAcesso)" />

                <q-btn
v-if="hasEvents(props.row)" dense flat round size="sm" color="warning" icon="history"
                  title="Ver Eventos" aria-label="Ver Eventos" :disable="!props.row.DocumentID"
                  @click.stop="openEventsDialog(props.row.DocumentID)" />
              </div>
            </template>

            <template v-else-if="col.name === 'chaveAcesso'">
              <div class="row no-wrap items-center q-gutter-x-xs chave-acesso-cell">
                <span
:title="props.row.ChaveAcesso"
                  class="cursor-pointer text-weight-medium ellipsis chave-acesso-text"
                  @click="copyChave(props.row.ChaveAcesso)">
                  {{ formatChave(props.row.ChaveAcesso) }}
                </span>

                <q-btn
dense flat round size="xs" color="grey-7" icon="content_copy" title="Copiar Chave Completa"
                  aria-label="Copiar Chave Completa" :disable="!props.row.ChaveAcesso"
                  @click.stop="copyChave(props.row.ChaveAcesso)" />
              </div>
            </template>

            <template v-else-if="col.name === 'info'">
              <div class="row no-wrap items-center justify-center q-gutter-x-xs">
                <q-badge :color="roleColor(props.row.CompanyRole)" class="text-weight-bold text-mono cursor-help">
                  {{ getRoleAbbreviation(props.row.CompanyRole) }}
                  <q-tooltip>Direção: {{ roleLabel(props.row.CompanyRole) }}</q-tooltip>
                </q-badge>

                <q-badge
:color="visibilityColor(props.row.VisibilityReason)"
                  class="text-weight-bold text-mono cursor-help">
                  {{ getVisibilityAbbreviation(props.row.VisibilityReason) }}
                  <q-tooltip>Visibilidade: {{ visibilityLabel(props.row.VisibilityReason) }}</q-tooltip>
                </q-badge>

                <q-badge :color="statusColor(props.row.Status)" class="text-weight-bold text-mono cursor-help">
                  {{ getStatusAbbreviation(props.row.Status) }}
                  <q-tooltip>Status: {{ getStatusLabel(props.row.Status) }}</q-tooltip>
                </q-badge>
              </div>
            </template>

            <template v-else-if="col.name === 'prestador'">
              <div class="text-weight-medium">
                {{ props.row.PrestadorCNPJ ? formatCpfCnpj(props.row.PrestadorCNPJ) : '-' }}
              </div>
              <div class="text-caption text-grey-6 ellipsis partner-name" :title="props.row.PrestadorName || ''">
                {{ props.row.PrestadorName || '-' }}
              </div>
            </template>

            <template v-else-if="col.name === 'tomador'">
              <div class="text-weight-medium">
                {{ props.row.TomadorCNPJ ? formatCpfCnpj(props.row.TomadorCNPJ) : '-' }}
              </div>
              <div class="text-caption text-grey-6 ellipsis partner-name" :title="props.row.TomadorName || ''">
                {{ props.row.TomadorName || '-' }}
              </div>
            </template>

            <template v-else>
              <span :class="{ 'text-mono': monospaceColumns.has(col.name) }">
                {{ col.value }}
              </span>
            </template>
          </q-td>
        </q-tr>

        <q-tr v-if="props.expand" :props="props" :class="['detail-container-borders', $q.dark.isActive ? 'bg-grey-10' : 'bg-grey-1']">
          <q-td :colspan="props.cols.length" class="q-pa-md">
            <div class="row q-col-gutter-md">
              <div class="col-12 col-md-7">
                <div class="text-subtitle2 text-primary q-mb-xs">
                  Dados do Serviço
                </div>

                <div class="row q-col-gutter-sm text-body2">
                  <div class="col-6">
                    <span class="text-weight-bold">Número da NFSe:</span>
                    {{ props.row.NFSeNumber || 'N/A' }}
                  </div>

                  <div class="col-6">
                    <span class="text-weight-bold">Versão Layout:</span>
                    {{ props.row.LayoutVersion || 'N/A' }}
                  </div>
                </div>

                <div class="q-mt-sm">
                  <span class="text-weight-bold text-body2">Descrição do Serviço:</span>
                  <div
                    :class="['q-mt-xs q-pa-sm text-body2 shadow-1 rounded-borders service-description custom-border-solid', $q.dark.isActive ? 'bg-grey-9 text-grey-4' : 'bg-white text-grey-8']">
                    {{ props.row.ServiceDescription || 'Sem descrição.' }}
                  </div>
                </div>
              </div>

              <div class="col-12 col-md-5">
                <div class="text-subtitle2 text-primary q-mb-xs">
                  Retenções e Tributos
                </div>

                <div :class="['q-pa-md shadow-1 rounded-borders custom-border-solid', $q.dark.isActive ? 'bg-grey-9 text-grey-4' : 'bg-white text-grey-8']">
                  <div class="row q-col-gutter-xs text-body2">
                    <div :class="['col-6', $q.dark.isActive ? 'text-grey-5' : 'text-grey-7']">ISS Retido:</div>
                    <div class="col-6 text-right text-weight-medium">
                      {{ formatCurrencyCents(props.row.ISSValue) }}
                    </div>

                    <div :class="['col-6', $q.dark.isActive ? 'text-grey-5' : 'text-grey-7']">IRRF:</div>
                    <div class="col-6 text-right text-weight-medium">
                      {{ formatCurrencyCents(props.row.IRRFValue) }}
                    </div>

                    <div :class="['col-6', $q.dark.isActive ? 'text-grey-5' : 'text-grey-7']">INSS:</div>
                    <div class="col-6 text-right text-weight-medium">
                      {{ formatCurrencyCents(props.row.INSSValue) }}
                    </div>

                    <div :class="['col-6', $q.dark.isActive ? 'text-grey-5' : 'text-grey-7']">PIS:</div>
                    <div class="col-6 text-right text-weight-medium">
                      {{ formatCurrencyCents(props.row.PISValue) }}
                    </div>

                    <div :class="['col-6', $q.dark.isActive ? 'text-grey-5' : 'text-grey-7']">COFINS:</div>
                    <div class="col-6 text-right text-weight-medium">
                      {{ formatCurrencyCents(props.row.COFINSValue) }}
                    </div>

                    <div :class="['col-6', $q.dark.isActive ? 'text-grey-5' : 'text-grey-7']">CSLL:</div>
                    <div class="col-6 text-right text-weight-medium">
                      {{ formatCurrencyCents(props.row.CSLLValue) }}
                    </div>

                    <q-separator class="col-12 q-my-xs" />

                    <div class="col-6 text-weight-bold text-primary">
                      Total Retenções:
                    </div>
                    <div class="col-6 text-right text-weight-bold text-primary">
                      {{ formatCurrencyCents(props.row.TotalRetentions) }}
                    </div>
                  </div>
                </div>
              </div>
            </div>
          </q-td>
        </q-tr>
      </template>
    </q-table>

    <DocumentEventsDialog v-model="showEventsDialog" :document-id="selectedDocumentId" />
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute } from 'vue-router'
import { copyToClipboard, date, useQuasar, type QTableColumn } from 'quasar'
import DocumentEventsDialog from '../components/DocumentEventsDialog.vue'
import { useDocuments } from '@/composables/useDocuments'
import {
  formatChaveAcesso,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
} from '@/utils/formatters'
import {
  getRoleAbbreviation,
  getStatusAbbreviation,
  getStatusLabel,
  getVisibilityAbbreviation,
  roleColor,
  roleLabel,
  statusColor,
  visibilityColor,
  visibilityLabel,
} from '@/utils/nfseDisplay'

type Direction = '' | 'tomada' | 'prestada' | 'intermediario' | 'none'
type ExportFormat = 'csv' | 'xlsx' | 'zip'

type SelectOption<T = string> = {
  label: string
  value: T
}

type DocumentRow = {
  RelationID?: string
  DocumentID?: string
  ChaveAcesso?: string
  NFSeNumber?: string
  PrestadorCNPJ?: string
  PrestadorName?: string
  TomadorCNPJ?: string
  TomadorName?: string
  Status?: string
  ServiceDescription?: string
  CompanyRole?: string
  VisibilityReason?: string
  IssueDate?: string | Date | null
  Competence?: string
  ServiceValue?: number
  LayoutVersion?: string
  ISSValue?: number
  IRRFValue?: number
  INSSValue?: number
  PISValue?: number
  COFINSValue?: number
  CSLLValue?: number
  TotalRetentions?: number
}

type ExportResult = {
  OutPath?: string
}

const $q = useQuasar()
const route = useRoute()
const documentsApi = useDocuments()

const {
  filter,
  documents,
  pagination,
  companyOptions,
  loading,
  exporting,
} = documentsApi

const showEventsDialog = ref(false)
const selectedDocumentId = ref('')
const filterText = ref('')

const datePopup = ref<{ hide: () => void } | null>(null)

const directionOptions: SelectOption<Direction>[] = [
  { label: 'Todos', value: '' },
  { label: 'Tomados', value: 'tomada' },
  { label: 'Prestados', value: 'prestada' },
  { label: 'Intermediário', value: 'intermediario' },
  { label: 'Sem papel fiscal', value: 'none' },
]

const monospaceColumns = new Set(['issueDate', 'competence', 'value'])

const columns: QTableColumn<DocumentRow>[] = [
  {
    name: 'actions',
    label: 'Ações',
    field: () => '',
    align: 'center',
  },
  {
    name: 'issueDate',
    label: 'Emissão',
    field: 'IssueDate',
    sortable: true,
    classes: 'text-no-wrap text-mono',
    format: (value: string | Date | null | undefined) => formatDate(value ?? null),
  },
  {
    name: 'competence',
    label: 'Competência',
    field: 'Competence',
    sortable: true,
    classes: 'text-no-wrap text-mono',
    format: (value: string | undefined) => value || '-',
  },
  {
    name: 'chaveAcesso',
    label: 'Chave de Acesso',
    field: 'ChaveAcesso',
    classes: 'text-mono chave-acesso-column',
    headerClasses: 'text-mono chave-acesso-column',
    style: 'width: 1%; max-width: 220px;',
    headerStyle: 'width: 1%; max-width: 220px;',
  },
  {
    name: 'info',
    label: 'Info',
    align: 'center',
    field: () => '',
  },
  {
    name: 'prestador',
    label: 'Prestador',
    field: 'PrestadorCNPJ',
    sortable: true,
    classes: 'text-mono',
  },
  {
    name: 'tomador',
    label: 'Tomador',
    field: 'TomadorCNPJ',
    sortable: true,
    classes: 'text-mono',
  },
  {
    name: 'value',
    label: 'Valor (R$)',
    field: 'ServiceValue',
    classes: 'text-mono',
    format: (value: number | undefined) => formatCurrencyCents(value ?? 0),
    sortable: true,
  },
]

const filteredDocuments = computed(() => {
  const query = normalizeText(filterText.value)

  if (!query) {
    return documents.value
  }

  return documents.value.filter((document) => {
    return [
      document.ChaveAcesso,
      document.NFSeNumber,
      document.PrestadorCNPJ,
      document.PrestadorName,
      document.TomadorCNPJ,
      document.TomadorName,
      document.Status,
      document.ServiceDescription,
    ].some((value) => normalizeText(value).includes(query))
  })
})

watch(filterText, () => {
  pagination.value.page = 1
})

onMounted(() => {
  void loadCompanies()
})

function normalizeText(value: unknown): string {
  return String(value ?? '')
    .normalize('NFD')
    .replace(/\p{Diacritic}/gu, '')
    .toLowerCase()
    .trim()
}

function hasEvents(document: DocumentRow): boolean {
  return document.Status === 'cancelada' || document.Status === 'substituida'
}

function formatChave(chave?: string): string {
  return chave ? formatChaveAcesso(chave) : '-'
}

function onDateChange(_value: string, reason: string) {
  if (reason === 'month') {
    datePopup.value?.hide()
  }
}

function setToday() {
  filter.value.Competence = date.formatDate(Date.now(), 'YYYY-MM')
  datePopup.value?.hide()
}

function shiftCompetence(monthDelta: number) {
  if (!filter.value.Competence) {
    filter.value.Competence = date.formatDate(Date.now(), 'YYYY-MM')
  }

  const [yearText, monthText] = filter.value.Competence.split('-')
  const year = Number(yearText)
  const month = Number(monthText)

  if (!Number.isInteger(year) || !Number.isInteger(month) || month < 1 || month > 12) {
    return
  }

  const next = new Date(year, month - 1 + monthDelta, 1)
  filter.value.Competence = date.formatDate(next, 'YYYY-MM')
}

function parseRouteQueryParam(param: unknown): string {
  return Array.isArray(param) ? String(param[0] ?? '') : String(param ?? '')
}

function applyRouteFilters(): boolean {
  let shouldSearch = false

  const cnpjFromRoute = parseRouteQueryParam(route.query['cnpj'])
  const competenceFromRoute = parseRouteQueryParam(route.query['competence'])

  if (cnpjFromRoute) {
    const matchingCompany = companyOptions.value.find((option) => option.value === cnpjFromRoute)

    if (matchingCompany) {
      filter.value.CNPJ = matchingCompany.value
      shouldSearch = true
    }
  }

  if (/^\d{4}-\d{2}$/.test(competenceFromRoute)) {
    filter.value.Competence = competenceFromRoute
    shouldSearch = true
  }

  return shouldSearch
}

function selectDefaultCompany(): boolean {
  if (filter.value.CNPJ || companyOptions.value.length === 0) {
    return false
  }

  const [firstCompany] = companyOptions.value

  if (!firstCompany) {
    return false
  }

  filter.value.CNPJ = firstCompany.value
  return true
}

function errorMessage(error: unknown): string {
  if (error instanceof Error) {
    return error.message
  }

  return String(error)
}

function notifyError(message: string, error: unknown) {
  $q.notify({
    type: 'negative',
    message: `${message}: ${errorMessage(error)}`,
  })
}

function notifyExportSuccess(label: string, result: ExportResult | null | undefined) {
  if (!result?.OutPath) {
    return
  }

  $q.notify({
    type: 'positive',
    message: `${label} exportado com sucesso para ${result.OutPath}`,
  })
}

async function handleCompanyChange() {
  if (!filter.value.CNPJ) {
    return
  }

  await search()
}

async function copyChave(chave?: string) {
  if (!chave) {
    return
  }

  try {
    await copyToClipboard(chave.replace(/^NFS/i, ''))

    $q.notify({
      type: 'positive',
      message: 'Chave copiada!',
      timeout: 1000,
    })
  } catch (error) {
    notifyError('Erro ao copiar chave', error)
  }
}

async function loadCompanies() {
  try {
    await documentsApi.loadCompanies()

    if (companyOptions.value.length === 0) {
      return
    }

    const shouldSearch = applyRouteFilters() || selectDefaultCompany() || documents.value.length === 0

    if (shouldSearch) {
      await search()
    }
  } catch (error) {
    notifyError('Erro ao carregar empresas', error)
  }
}

async function search() {
  if (!filter.value.CNPJ) {
    return
  }

  try {
    await documentsApi.search()
  } catch (error) {
    notifyError('Erro ao buscar documentos', error)
  }
}

async function exportData(format: ExportFormat) {
  try {
    const result = await documentsApi.exportDocuments(format)
    notifyExportSuccess(`Arquivo ${format.toUpperCase()}`, result)
  } catch (error) {
    notifyError('Erro ao exportar', error)
  }
}

async function exportDanfse(chaveAcesso?: string) {
  if (!chaveAcesso) {
    return
  }

  try {
    const result = await documentsApi.exportDANFSe(chaveAcesso)
    notifyExportSuccess('DANFSe', result)
  } catch (error) {
    notifyError('Erro ao exportar DANFSe', error)
  }
}

async function exportXML(chaveAcesso?: string) {
  if (!chaveAcesso) {
    return
  }

  try {
    const result = await documentsApi.exportXML(chaveAcesso)
    notifyExportSuccess('XML', result)
  } catch (error) {
    notifyError('Erro ao exportar XML', error)
  }
}

async function exportDanfseZip() {
  try {
    const result = await documentsApi.exportDANFSeZIP()
    notifyExportSuccess('ZIP de DANFSes', result)
  } catch (error) {
    notifyError('Erro ao exportar DANFSes', error)
  }
}

function openEventsDialog(documentId?: string) {
  if (!documentId) {
    return
  }

  selectedDocumentId.value = documentId
  showEventsDialog.value = true
}
</script>

<style scoped>
:deep(.q-table) {
  table-layout: auto;
}

:deep(.q-table td) {
  white-space: normal;
  word-break: break-word;
}

.document-search-input {
  width: 350px;
  max-width: 100%;
}

.custom-border-dashed {
  border: 1px dashed var(--q-separator-color);
}

.custom-border-solid {
  border: 1px solid var(--q-separator-color);
}

.legend-label-width {
  min-width: 100px;
}

.partner-name {
  max-width: 180px;
}

.service-description {
  white-space: pre-wrap;
  max-height: 150px;
  overflow-y: auto;
}

.detail-container-borders {
  border-top: 1px solid var(--q-separator-color);
  border-bottom: 1px solid var(--q-separator-color);
}

:deep(.chave-acesso-column) {
  width: 1%;
  max-width: 220px;
}

.chave-acesso-cell {
  width: fit-content;
  max-width: 220px;
}

.chave-acesso-text {
  display: inline-block;
  max-width: 170px;
  vertical-align: middle;
}
</style>