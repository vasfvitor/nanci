<template>
  <q-page padding>
    <div class="row items-center justify-between q-mb-md">
      <h5 class="q-my-none">Documentos Fiscais</h5>
    </div>

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
      />
      <div class="col-12 col-md-3">
        <div class="row no-wrap items-center q-gutter-xs">
          <q-btn
            color="grey-7"
            icon="chevron_left"
            dense
            flat
            round
            :disable="!filter.Competence"
            title="Competência anterior"
            @click="shiftCompetence(-1)"
          />
          <q-input
            v-model="filter.Competence"
            class="col"
            label="Competência"
            outlined
            dense
            clearable
            mask="####-##"
          >
        <template #append>
          <q-icon name="event" class="cursor-pointer">
            <q-popup-proxy ref="datePopup" cover transition-show="scale" transition-hide="scale">
              <q-date
                v-model="filter.Competence"
                minimal
                mask="YYYY-MM"
                emit-immediately
                default-view="Months"
                years-in-month-view
                @update:model-value="onDateChange"
              >
                <div class="row items-center justify-end">
                  <q-btn v-close-popup label="Mês Atual" color="primary" flat @click="setToday" />
                  <q-btn v-close-popup label="Fechar" color="primary" flat />
                </div>
              </q-date>
            </q-popup-proxy>
          </q-icon>
        </template>
          </q-input>
          <q-btn
            color="grey-7"
            icon="chevron_right"
            dense
            flat
            round
            :disable="!filter.Competence"
            title="Próxima competência"
            @click="shiftCompetence(1)"
          />
        </div>
      </div>
      <q-select
        v-model="filter.Direction"
        class="col-12 col-md-2"
        :options="[
          { label: 'Todos', value: '' },
          { label: 'Tomados', value: 'tomada' },
          { label: 'Prestados', value: 'prestada' },
          { label: 'Intermediário', value: 'intermediario' },
          { label: 'Sem papel fiscal', value: 'none' },
        ]"
        label="Direção"
        emit-value
        map-options
        outlined
        dense
        options-dense
      />

      <q-space />

      <q-btn
        color="primary"
        icon="search"
        label="Buscar"
        :disable="!filter.CNPJ"
        dense
        flat
        @click="search"
      />
      <q-btn-dropdown
        color="secondary"
        label="Exportar"
        :disable="documents.length === 0"
        dense
        flat
      >
        <q-list dense>
          <q-item v-close-popup clickable @click="exportData('csv')">
            <q-item-section avatar><q-icon name="table_view" /></q-item-section>
            <q-item-section><q-item-label>Exportar CSV</q-item-label></q-item-section>
          </q-item>
          <q-item v-close-popup clickable @click="exportData('xlsx')">
            <q-item-section avatar><q-icon name="grid_on" /></q-item-section>
            <q-item-section><q-item-label>Exportar XLSX</q-item-label></q-item-section>
          </q-item>
          <q-item v-close-popup clickable @click="exportData('zip')">
            <q-item-section avatar><q-icon name="folder_zip" /></q-item-section>
            <q-item-section><q-item-label>Exportar XMLs (ZIP)</q-item-label></q-item-section>
          </q-item>
          <q-item v-close-popup clickable @click="exportDanfseZip">
            <q-item-section avatar><q-icon name="picture_as_pdf" /></q-item-section>
            <q-item-section><q-item-label>Exportar DANFSes (ZIP)</q-item-label></q-item-section>
          </q-item>
        </q-list>
      </q-btn-dropdown>
    </div>

    <q-table
      :rows="documents"
      :columns="columns"
      row-key="RelationID"
      v-model:pagination="pagination"
      :loading="loading"
      no-data-label="Nenhum documento encontrado."
      flat
      bordered
      dense
      class="full-height"
    >
      <template #body-cell-status="props">
        <q-td :props="props">
          <q-badge :color="statusColor(props.row.Status)">
            {{ props.row.Status }}
          </q-badge>
        </q-td>
      </template>
      <template #body-cell-chaveAcesso="props">
        <q-td :props="props" class="q-gutter-x-sm">
          <span :title="props.row.ChaveAcesso" class="cursor-pointer" @click="copyChave(props.row.ChaveAcesso)">
            {{ formatChave(props.row.ChaveAcesso) }}
          </span>
          <q-btn
            dense
            flat
            round
            size="sm"
            color="grey-7"
            icon="content_copy"
            title="Copiar Chave Completa"
            @click="copyChave(props.row.ChaveAcesso)"
          />
        </q-td>
      </template>
      <template #body-cell-companyRole="props">
        <q-td :props="props">
          <q-badge :color="roleColor(props.row.CompanyRole)" outline>
            {{ roleLabel(props.row.CompanyRole) }}
          </q-badge>
        </q-td>
      </template>
      <template #body-cell-visibilityReason="props">
        <q-td :props="props">
          <q-badge :color="visibilityColor(props.row.VisibilityReason)" outline>
            {{ visibilityLabel(props.row.VisibilityReason) }}
          </q-badge>
        </q-td>
      </template>
      <template #body-cell-actions="props">
        <q-td :props="props" class="q-gutter-x-sm">
          <q-btn
            dense
            flat
            round
            size="sm"
            color="primary"
            icon="code"
            title="Exportar XML Original"
            @click="exportXML(props.row.ChaveAcesso)"
          />
          <q-btn
            dense
            flat
            round
            size="sm"
            color="negative"
            icon="picture_as_pdf"
            title="Exportar DANFSe"
            @click="exportDanfse(props.row.ChaveAcesso)"
          />
          <q-btn
            v-if="props.row.Status === 'cancelada' || props.row.Status === 'substituida'"
            dense
            flat
            round
            color="warning"
            icon="history"
            title="Ver Eventos"
            @click="openEventsDialog(props.row.DocumentID)"
          />
        </q-td>
      </template>
    </q-table>

    <DocumentEventsDialog
      v-model="showEventsDialog"
      :document-id="selectedDocumentId"
    />
  </q-page>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useRoute } from 'vue-router'
import { useQuasar, date } from 'quasar'
import DocumentEventsDialog from '../components/DocumentEventsDialog.vue'
import { useDocuments } from '@/composables/useDocuments'
import {
  formatChaveAcesso,
  formatCpfCnpj,
  formatCurrencyCents,
  formatDate,
} from '@/utils/formatters'
import { roleColor, roleLabel, statusColor, visibilityColor, visibilityLabel } from '@/utils/nfseDisplay'

const $q = useQuasar()
const route = useRoute()
const documentsApi = useDocuments()
const { filter, documents, pagination, companyOptions, loading } = documentsApi

const showEventsDialog = ref(false)
const selectedDocumentId = ref('')

function openEventsDialog(docId: string) {
  selectedDocumentId.value = docId
  showEventsDialog.value = true
}

const datePopup = ref<{ hide: () => void } | null>(null)

function onDateChange(_val: string, reason: string) {
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
  if (!year || !month) return
  const next = new Date(year, month - 1 + monthDelta, 1)
  filter.value.Competence = date.formatDate(next, 'YYYY-MM')
}

const columns = [
  { name: 'actions', label: 'Ações', field: () => '', align: 'center' as const },
  {
    name: 'issueDate',
    label: 'Emissão',
    field: 'IssueDate',
    sortable: true,
    classes: 'text-no-wrap',
    format: (val: string | Date | null) => formatDate(val),
  },
  { name: 'competence', label: 'Competência', field: 'Competence', sortable: true, classes: 'text-no-wrap' },
  { name: 'chaveAcesso', label: 'Chave de Acesso', field: 'ChaveAcesso', sortable: true },
  { name: 'companyRole', label: 'Direção', field: 'CompanyRole', sortable: true },
  { name: 'visibilityReason', label: 'Visibilidade', field: 'VisibilityReason', sortable: true },
  { name: 'status', label: 'Status', field: 'Status', sortable: true },
  { name: 'prestador', label: 'Prestador', field: 'PrestadorCNPJ', sortable: true, format: (val: string) => formatCpfCnpj(val) },
  { name: 'tomador', label: 'Tomador', field: 'TomadorCNPJ', sortable: true, format: (val: string) => formatCpfCnpj(val) },
  {
    name: 'value',
    label: 'Valor (R$)',
    field: 'ServiceValue',
    format: (val: number) => formatCurrencyCents(val),
    sortable: true,
  },
]

function formatChave(chave: string) {
  return formatChaveAcesso(chave)
}

async function copyChave(chave: string) {
  if (!chave) return
  const clean = chave.replace(/^NFS/i, '')
  await navigator.clipboard.writeText(clean)
  $q.notify({ type: 'positive', message: 'Chave copiada!', timeout: 1000 })
}

async function loadCompanies() {
  try {
    await documentsApi.loadCompanies()
    if (companyOptions.value.length > 0) {
      const cnpjFromRoute = String(route.query['cnpj'] || '')
      let shouldSearch = false

      if (cnpjFromRoute) {
        const matchingOption = companyOptions.value.find((option) => option.value === cnpjFromRoute)
        if (matchingOption) {
          filter.value.CNPJ = matchingOption.value
          shouldSearch = true
        }
      } else if (filter.value && !filter.value.CNPJ) {
        const firstOption = companyOptions.value[0]
        if (firstOption) {
          filter.value.CNPJ = firstOption.value
          shouldSearch = true
        }
      }

      const competenceFromRoute = String(route.query['competence'] || '')
      if (competenceFromRoute) {
        filter.value.Competence = competenceFromRoute
        shouldSearch = true
      }

      if (shouldSearch || (documents.value && documents.value.length === 0)) {
        search()
      }
    }
  } catch (err) {
    $q.notify({ type: 'negative', message: String(err) })
  }
}

async function search() {
  try {
    await documentsApi.search()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao buscar documentos: ' + err })
  }
}

async function exportData(format: 'csv' | 'xlsx' | 'zip') {
  try {
    const result = await documentsApi.exportDocuments(format)
    if (!result) return
    $q.notify({ type: 'positive', message: `Exportado com sucesso para ${result.OutPath}` })
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao exportar: ' + String(err) })
  }
}

async function exportDanfse(chaveAcesso: string) {
  try {
    const result = await documentsApi.exportDANFSe(chaveAcesso)
    if (!result) return
    $q.notify({ type: 'positive', message: `DANFSe exportado com sucesso para ${result.OutPath}` })
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao exportar DANFSe: ' + String(err) })
  }
}

async function exportXML(chaveAcesso: string) {
  try {
    const result = await documentsApi.exportXML(chaveAcesso)
    if (!result) return
    $q.notify({ type: 'positive', message: `XML exportado com sucesso para ${result.OutPath}` })
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao exportar XML: ' + String(err) })
  }
}

async function exportDanfseZip() {
  try {
    const result = await documentsApi.exportDANFSeZIP()
    if (!result) return
    $q.notify({ type: 'positive', message: `DANFSes exportados com sucesso para ${result.OutPath}` })
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao exportar DANFSes: ' + String(err) })
  }
}

onMounted(() => {
  loadCompanies()
})
</script>

<style scoped>
:deep(.q-table) {
  table-layout: fixed;
}
:deep(.q-table td) {
  white-space: normal !important;
  word-break: break-word;
}
</style>
