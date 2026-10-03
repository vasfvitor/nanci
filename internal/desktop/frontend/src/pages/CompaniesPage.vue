<template>
  <q-page padding>
    <q-table
      title="Empresas"
      :rows="companies"
      :columns="columns"
      row-key="ID"
      flat
      bordered
      dense
      :loading="loading"
      no-data-label="Nenhuma empresa cadastrada."
    >
      <template #top-right>
        <q-btn
          color="primary"
          icon="add"
          label="Adicionar"
          dense
          flat
          @click="showAddDialog = true"
        />
      </template>

      <template #body-cell-nome="props">
        <q-td :props="props" :class="{ 'text-primary text-weight-bold': props.row.CNPJ === workspace.cnpj }">
          {{ props.row.Name }}
        </q-td>
      </template>

      <template #body-cell-ambiente="props">
        <q-td :props="props">
          <q-badge
            :color="props.row.Environment === 'producao' ? 'positive' : 'warning'"
            :text-color="props.row.Environment === 'producao' ? 'white' : 'dark'"
          >
            {{ props.row.Environment }}
          </q-badge>
        </q-td>
      </template>

      <template #body-cell-lastFoundNSU="props">
        <q-td :props="props">
          {{ props.row.LastFoundNSU ?? '—' }}
        </q-td>
      </template>

      <template #body-cell-lastSyncAt="props">
        <q-td :props="props">
          {{ formatDateTime(props.row.LastSyncAt) }}
        </q-td>
      </template>

      <template #body-cell-syncStart="props">
        <q-td :props="props">
          {{ syncStartLabel(props.row) }}
        </q-td>
      </template>

      <template #body-cell-credencial="props">
        <q-td :props="props" style="max-width: 250px">
          <q-select
            v-model="selectedCredentials[props.row.CNPJ]"
            :options="credentialOptions"
            emit-value
            map-options
            label="Credencial"
            outlined
            dense
            options-dense
            class="ellipsis"
            @update:model-value="assignCredential(props.row.CNPJ)"
          />
        </q-td>
      </template>

      <template #body-cell-acoes="props">
        <q-td :props="props" class="q-gutter-x-sm">
          <q-btn
            dense
            flat
            round
            color="secondary"
            icon="description"
            title="Ver documentos"
            aria-label="Ver documentos"
            @click="openDocuments(props.row.CNPJ)"
          />
          <q-btn
            dense
            flat
            round
            color="grey-7"
            icon="edit"
            title="Editar"
            @click="openEditDialog(props.row)"
          />
        </q-td>
      </template>
    </q-table>

    <AddCompanyDialog v-model="showAddDialog" @added="reloadData" />
    <EditCompanyDialog
      v-model="showEditDialog"
      :company-data="selectedCompanyToEdit"
      @updated="reloadData"
    />
  </q-page>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import { useRouter } from 'vue-router'
import { errorMessage } from '@/platform/wails/client'
import AddCompanyDialog from '../components/AddCompanyDialog.vue'
import EditCompanyDialog from '../components/EditCompanyDialog.vue'
import { useWorkspaceStore } from '@/stores/workspace'
import { useCompanies } from '@/composables/useCompanies'
import { formatCpfCnpj, formatDate, formatDateTime } from '@/utils/formatters'
import type { CompanySummary } from '@/types/desktop'

const $q = useQuasar()
const router = useRouter()
const workspace = useWorkspaceStore()
const companiesApi = useCompanies()
const { companies, credentials, loading } = companiesApi
const showAddDialog = ref(false)
const selectedCredentials = ref<Record<string, string>>({})
const showEditDialog = ref(false)
const selectedCompanyToEdit = ref<CompanySummary | null>(null)

const credentialOptions = computed(() =>
  credentials.value.map((credential) => ({
    label: credential.Label,
    value: credential.ID,
  }))
)

const columns: QTableColumn[] = [
  { name: 'acoes', label: 'Ações', field: () => '', align: 'center' as const },
  { name: 'nome', label: 'Nome', field: 'Name', align: 'left', sortable: true },
  { name: 'cnpj', label: 'CNPJ', field: 'CNPJ', align: 'left', sortable: true, format: (val: string) => formatCpfCnpj(val) },
  { name: 'ambiente', label: 'Ambiente', field: 'Environment', align: 'left', sortable: true },
  { name: 'lastFoundNSU', label: 'Último NSU (c/ doc)', field: 'LastFoundNSU', align: 'left', sortable: true },
  { name: 'syncStart', label: 'Histórico inicial', field: 'SyncStartPolicy', align: 'left', sortable: true },
  { name: 'lastSyncAt', label: 'Última sincronização', field: 'LastSyncAt', align: 'left', sortable: true },
  { name: 'credencial', label: 'Credencial', field: () => '', align: 'left', style: 'max-width: 250px' },
]

function openEditDialog(company: CompanySummary) {
  selectedCompanyToEdit.value = company
  showEditDialog.value = true
}

// openDocuments makes the company the workspace one and opens its documents,
// starting at the NFS-e.
function openDocuments(cnpj: string) {
  workspace.cnpj = cnpj
  void router.push('/documents')
}

function syncStartLabel(company: CompanySummary) {
  switch (company.SyncStartPolicy) {
    case 'all':
      return 'Todo histórico'
    case 'since_date':
      return `Desde ${formatDate(company.SyncStartDate)}`
    case 'from_now':
      return company.SyncStartDate ? `A partir de ${formatDate(company.SyncStartDate)}` : 'A partir de hoje'
    default:
      return 'A partir de hoje'
  }
}

function setSelectedCredentials(list: CompanySummary[]) {
  selectedCredentials.value = Object.fromEntries(
    list.map((company) => [company.CNPJ, company.CredentialID])
  )
}

async function loadCompanies() {
  setSelectedCredentials(await companiesApi.loadCompanies())
}

// loadInitial fills the page when it opens. The company list comes from the
// workspace load the layout also waits for, so it is listed once. The layout
// notifies a failure of that load and the drawer keeps showing it, so the
// page only notifies a failed credentials load.
async function loadInitial() {
  const [credentialsResult, companiesResult] = await Promise.allSettled([
    companiesApi.loadCredentials(),
    companiesApi.ensureCompanies(),
  ])
  if (companiesResult.status === 'fulfilled') setSelectedCredentials(companiesResult.value)
  if (credentialsResult.status === 'rejected') {
    $q.notify({
      type: 'negative',
      message: 'Erro ao carregar credenciais: ' + errorMessage(credentialsResult.reason),
    })
  }
}

async function reloadData() {
  try {
    await companiesApi.loadCredentials()
    await loadCompanies()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar empresas: ' + errorMessage(err) })
  }
}

async function assignCredential(cnpj: string) {
  const credId = selectedCredentials.value[cnpj]
  if (!credId) return

  const previousCredId = companies.value.find((company) => company.CNPJ === cnpj)?.CredentialID ?? ''

  try {
    await companiesApi.assignCredential(cnpj, credId)
    $q.notify({ type: 'positive', message: 'Credencial atribuída com sucesso.' })
    await loadCompanies()
  } catch (err) {
    selectedCredentials.value[cnpj] = previousCredId
    $q.notify({ type: 'negative', message: 'Erro ao atribuir credencial: ' + errorMessage(err) })
  }
}

onMounted(() => {
  void loadInitial()
})

</script>
