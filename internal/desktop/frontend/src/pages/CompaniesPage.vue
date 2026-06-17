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
      :loading="false"
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
          {{ props.row.LastFoundNSUValid ? props.row.LastFoundNSU : '—' }}
        </q-td>
      </template>

      <template #body-cell-lastSyncAt="props">
        <q-td :props="props">
          {{ formatDateTime(props.row.LastSyncAt) }}
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
            @click="openDocuments(props.row.CNPJ)"
          />
          <q-btn
            dense
            flat
            round
            color="primary"
            icon="sync"
            :loading="syncing === props.row.CNPJ"
            title="Sincronizar"
            @click="syncCompany(props.row.CNPJ)"
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
          <q-btn
            v-if="debugEnabled"
            dense
            flat
            round
            color="negative"
            icon="restart_alt"
            title="Resetar NSU"
            @click="confirmResetSync(props.row)"
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
import { storeToRefs } from 'pinia'
import { computed, onMounted, ref } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import { useRouter } from 'vue-router'
import AddCompanyDialog from '../components/AddCompanyDialog.vue'
import EditCompanyDialog from '../components/EditCompanyDialog.vue'
import { useConsoleStore } from '@/stores/console'
import { useCompanies } from '@/composables/useCompanies'
import { formatCpfCnpj, formatDateTime } from '@/utils/formatters'
import type { CompanySummary } from '@/types/desktop'

const $q = useQuasar()
const router = useRouter()
const consoleStore = useConsoleStore()
const { debugEnabled } = storeToRefs(consoleStore)
const companiesApi = useCompanies()
const { companies, credentials, syncing } = companiesApi
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
  { name: 'nsu', label: 'Último NSU', field: 'LastNSU', align: 'left', sortable: true },
  { name: 'lastFoundNSU', label: 'Último NSU (c/ doc)', field: 'LastFoundNSU', align: 'left', sortable: true },
  { name: 'lastSyncAt', label: 'Última sincronização', field: 'LastSyncAt', align: 'left', sortable: true },
  { name: 'credencial', label: 'Credencial', field: () => '', align: 'left', style: 'max-width: 250px' },
]

function openEditDialog(company: CompanySummary) {
  selectedCompanyToEdit.value = company
  showEditDialog.value = true
}

function openDocuments(cnpj: string) {
  router.push({ path: '/documents', query: { cnpj } })
}

async function loadCompanies() {
  const list = await companiesApi.loadCompanies()
  selectedCredentials.value = Object.fromEntries(
    list.map((company) => [company.CNPJ, company.CredentialID])
  )
}

async function reloadData() {
  try {
    await companiesApi.loadCredentials()
    await loadCompanies()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar empresas: ' + String(err) })
  }
}

async function assignCredential(cnpj: string) {
  try {
    const credId = selectedCredentials.value[cnpj]
    if (!credId) return

    await companiesApi.assignCredential(cnpj, credId)
    $q.notify({ type: 'positive', message: 'Credencial atribuída com sucesso.' })
    await loadCompanies()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao atribuir credencial: ' + String(err) })
  }
}

async function syncCompany(cnpj: string) {
  try {
    const result = await companiesApi.syncCompany(cnpj)
    const credentialCNPJ = result.CredentialCNPJ || 'pendente'
    const lastFound = result.LastFoundNSUValid ? result.LastFoundNSU : '—'
    $q.notify({
      type: 'positive',
      message: `Sincronização ${result.Status || 'completed'} (${result.StopReason || 'sem motivo'}). Último NSU: ${result.LastCheckedNSU}, último com documento: ${lastFound}, credencial: ${credentialCNPJ}`,
    })
    await loadCompanies()
  } catch (err) {
    if (String(err).includes('operação cancelada')) {
      $q.notify({ type: 'warning', message: 'Sincronização cancelada.' })
    } else {
      $q.notify({ type: 'negative', message: 'Erro na sincronização: ' + String(err) })
    }
  }
}

function confirmResetSync(company: CompanySummary) {
  $q.dialog({
    title: 'Resetar sincronização',
    message: `Isso vai zerar o cursor de sincronização de ${company.Name} (${company.CNPJ}) sem apagar documentos já baixados. A próxima sincronização poderá revisitar NSUs antigos. Continuar?`,
    cancel: true,
    persistent: true,
    ok: {
      label: 'Resetar',
      color: 'negative',
      flat: false,
    },
  }).onOk(async () => {
    try {
      await companiesApi.resetSyncState(company.CNPJ)
      $q.notify({
        type: 'warning',
        message: `Cursor de sincronização resetado para ${company.Name}.`,
      })
      await loadCompanies()
    } catch (err) {
      $q.notify({ type: 'negative', message: 'Erro ao resetar sincronização: ' + String(err) })
    }
  })
}

onMounted(() => {
  reloadData()
})

</script>

