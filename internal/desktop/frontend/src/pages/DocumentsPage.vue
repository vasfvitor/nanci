<template>
  <q-page padding>
    <div class="row items-center justify-between q-mb-md">
      <h5 class="q-my-none">Documentos Fiscais</h5>
    </div>

    <q-card class="q-mb-md">
      <q-card-section class="row q-col-gutter-sm items-center">
        <q-select
          class="col-12 col-md-3"
          v-model="filter.CNPJ"
          :options="companyOptions"
          label="Empresa"
          emit-value
          map-options
          outlined
          dense
        />
        <q-input
          class="col-12 col-md-3"
          v-model="filter.Competence"
          label="Competência (YYYY-MM)"
          outlined
          dense
          clearable
          mask="####-##"
        >
          <template v-slot:append>
            <q-icon name="sym_r_event" class="cursor-pointer">
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
        <q-select
          class="col-12 col-md-3"
          v-model="filter.Direction"
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
        />
        <div class="col-12 col-md-3 row q-gutter-x-sm">
          <q-btn
            color="primary"
            icon="search"
            label="Buscar"
            @click="search"
            :disable="!filter.CNPJ"
          />
          <q-btn-dropdown color="secondary" label="Exportar" :disable="documents.length === 0">
            <q-list>
              <q-item clickable v-close-popup @click="exportData('csv')">
                <q-item-section avatar><q-icon name="table_view" /></q-item-section>
                <q-item-section><q-item-label>Exportar CSV</q-item-label></q-item-section>
              </q-item>
              <q-item clickable v-close-popup @click="exportData('xlsx')">
                <q-item-section avatar><q-icon name="grid_on" /></q-item-section>
                <q-item-section><q-item-label>Exportar XLSX</q-item-label></q-item-section>
              </q-item>
              <q-item clickable v-close-popup @click="exportData('zip')">
                <q-item-section avatar><q-icon name="folder_zip" /></q-item-section>
                <q-item-section><q-item-label>Exportar XMLs (ZIP)</q-item-label></q-item-section>
              </q-item>
            </q-list>
          </q-btn-dropdown>
        </div>
      </q-card-section>
    </q-card>

    <q-table
      :rows="documents"
      :columns="columns"
      row-key="RelationID"
      :loading="loading"
      no-data-label="Nenhum documento encontrado."
      flat
      bordered
    >
      <template v-slot:body-cell-status="props">
        <q-td :props="props">
          <q-badge :color="props.row.Status === 'normal' ? 'positive' : 'negative'">
            {{ props.row.Status }}
          </q-badge>
        </q-td>
      </template>
      <template v-slot:body-cell-companyRole="props">
        <q-td :props="props">
          <q-badge :color="roleColor(props.row.CompanyRole)" outline>
            {{ roleLabel(props.row.CompanyRole) }}
          </q-badge>
        </q-td>
      </template>
      <template v-slot:body-cell-visibilityReason="props">
        <q-td :props="props">
          <q-badge :color="visibilityColor(props.row.VisibilityReason)" outline>
            {{ visibilityLabel(props.row.VisibilityReason) }}
          </q-badge>
        </q-td>
      </template>
    </q-table>
  </q-page>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import {
  ListCompanies,
  ListDocuments,
  ExportCSV,
  ExportXLSX,
  ExportZIP,
  SelectExportDirectory,
} from '../../wailsjs/go/main/App'
import { nfse, app } from '../../wailsjs/go/models'
import { useQuasar, date } from 'quasar'

const $q = useQuasar()
const companyOptions = ref<{ label: string; value: string }[]>([])
const documents = ref<nfse.CompanyDocument[]>([])
const loading = ref(false)

const filter = ref({
  CNPJ: '',
  Competence: '',
  Direction: '',
})

const datePopup = ref<any>(null)

function onDateChange(_val: string, reason: string, _details: any) {
  if (reason === 'month') {
    datePopup.value?.hide()
  }
}

function setToday() {
  filter.value.Competence = date.formatDate(Date.now(), 'YYYY-MM')
  datePopup.value?.hide()
}

const columns = [
  {
    name: 'issueDate',
    label: 'Emissão',
    field: 'IssueDate',
    sortable: true,
    format: (val: string | Date | null) => formatDate(val),
  },
  { name: 'competence', label: 'Competência', field: 'Competence', sortable: true },
  { name: 'chaveAcesso', label: 'Chave de Acesso', field: 'ChaveAcesso', sortable: true },
  { name: 'companyRole', label: 'Direção', field: 'CompanyRole', sortable: true },
  { name: 'visibilityReason', label: 'Visibilidade', field: 'VisibilityReason', sortable: true },
  { name: 'status', label: 'Status', field: 'Status', sortable: true },
  { name: 'prestador', label: 'Prestador', field: 'PrestadorCNPJ', sortable: true },
  { name: 'tomador', label: 'Tomador', field: 'TomadorCNPJ', sortable: true },
  {
    name: 'value',
    label: 'Valor (R$)',
    field: 'ServiceValue',
    format: (val: number) => val.toFixed(2),
    sortable: true,
  },
]

function formatDate(value: string | Date | null | undefined) {
  if (!value) return ''
  const date = value instanceof Date ? value : new Date(value)
  if (Number.isNaN(date.getTime())) return ''
  return date.toLocaleDateString('pt-BR')
}

function roleLabel(role: string) {
  switch (role) {
    case 'prestada':
      return 'Prestada'
    case 'tomada':
      return 'Tomada'
    case 'intermediario':
      return 'Intermediário'
    case 'none':
      return 'Sem papel fiscal'
    default:
      return role || 'Desconhecido'
  }
}

function roleColor(role: string) {
  switch (role) {
    case 'prestada':
      return 'primary'
    case 'tomada':
      return 'secondary'
    case 'intermediario':
      return 'accent'
    case 'none':
      return 'grey-7'
    default:
      return 'grey-7'
  }
}

function visibilityLabel(reason: string) {
  switch (reason) {
    case 'exact_prestador':
      return 'Prestador exato'
    case 'exact_tomador':
      return 'Tomador exato'
    case 'exact_intermediario':
      return 'Intermediário exato'
    case 'same_root_only':
      return 'Mesmo raiz apenas'
    case 'unknown':
      return 'Desconhecida'
    default:
      return reason || 'Desconhecida'
  }
}

function visibilityColor(reason: string) {
  switch (reason) {
    case 'exact_prestador':
    case 'exact_tomador':
    case 'exact_intermediario':
      return 'positive'
    case 'same_root_only':
      return 'warning'
    default:
      return 'grey-7'
  }
}

async function loadCompanies() {
  try {
    const list = await ListCompanies()
    companyOptions.value = (list || []).map((c) => ({
      label: `${c.Name} (${c.CNPJ})`,
      value: c.CNPJ,
    }))
    if (companyOptions.value.length > 0) {
      filter.value.CNPJ = companyOptions.value[0].value
      search()
    }
  } catch (err) {
    console.error(err)
  }
}

async function search() {
  if (!filter.value.CNPJ) return
  loading.value = true
  try {
    const req = new app.ListInput({
      CNPJ: filter.value.CNPJ,
      Competence: filter.value.Competence || '',
      Direction: filter.value.Direction || '',
    })
    const res = await ListDocuments(req)
    documents.value = res || []
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao buscar documentos: ' + err })
  } finally {
    loading.value = false
  }
}

async function exportData(format: 'csv' | 'xlsx' | 'zip') {
  try {
    const outDir = await SelectExportDirectory()
    if (!outDir) return // user cancelled

    const ext = format === 'csv' ? '.csv' : format === 'xlsx' ? '.xlsx' : '.zip'
    const fileName = `export_${filter.value.CNPJ}_${Date.now()}${ext}`
    // Quick workaround for cross-platform paths
    const outPath =
      outDir.endsWith('\\') || outDir.endsWith('/')
        ? `${outDir}${fileName}`
        : `${outDir}\\${fileName}`

    const req = new app.ExportInput({
      CNPJ: filter.value.CNPJ,
      Competence: filter.value.Competence || '',
      Direction: filter.value.Direction || '',
      OutPath: outPath,
    })

    if (format === 'csv') await ExportCSV(req)
    else if (format === 'xlsx') await ExportXLSX(req)
    else if (format === 'zip') await ExportZIP(req)

    $q.notify({ type: 'positive', message: `Exportado com sucesso para ${outPath}` })
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao exportar: ' + String(err) })
  }
}

onMounted(() => {
  loadCompanies()
})
</script>
