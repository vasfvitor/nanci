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

      <template #body-cell-credencial="props">
        <q-td :props="props" style="width: 250px">
          <q-select
            v-model="selectedCredentials[props.row.CNPJ]"
            :options="credentialOptions"
            emit-value
            map-options
            label="Credencial"
            outlined
            dense
            options-dense
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
import { onMounted, ref } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import {
  AssignCredentialToCompany,
  ListCompanies,
  ListCredentials,
  Pull,
} from '../../wailsjs/go/main/App'
import { nfse } from '../../wailsjs/go/models'
import AddCompanyDialog from '../components/AddCompanyDialog.vue'
import EditCompanyDialog from '../components/EditCompanyDialog.vue'

const $q = useQuasar()
const companies = ref<nfse.Company[]>([])
const showAddDialog = ref(false)
const syncing = ref<string | null>(null)
const credentialOptions = ref<{ label: string; value: string }[]>([])
const selectedCredentials = ref<Record<string, string>>({})
const showEditDialog = ref(false)
const selectedCompanyToEdit = ref<nfse.Company | null>(null)

const columns: QTableColumn[] = [
  { name: 'nome', label: 'Nome', field: 'Name', align: 'left', sortable: true },
  { name: 'cnpj', label: 'CNPJ', field: 'CNPJ', align: 'left', sortable: true },
  { name: 'ambiente', label: 'Ambiente', field: 'Environment', align: 'left', sortable: true },
  { name: 'nsu', label: 'Último NSU', field: 'LastNSU', align: 'left', sortable: true },
  { name: 'credencial', label: 'Credencial', field: () => '', align: 'left' },
  { name: 'acoes', label: 'Ações', field: () => '', align: 'right' },
]

function openEditDialog(company: nfse.Company) {
  selectedCompanyToEdit.value = company
  showEditDialog.value = true
}

async function loadCredentials() {
  const list = (await ListCredentials()) || []
  credentialOptions.value = list.map((credential) => ({
    label: `${credential.Label} (${credential.Environment})`,
    value: credential.ID,
  }))
}

async function loadCompanies() {
  const list = (await ListCompanies()) || []
  companies.value = list
  selectedCredentials.value = Object.fromEntries(
    list.map((company) => [company.CNPJ, company.CredentialID])
  )
}

async function reloadData() {
  try {
    await Promise.all([loadCredentials(), loadCompanies()])
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar empresas: ' + String(err) })
  }
}

async function assignCredential(cnpj: string) {
  try {
    const credId = selectedCredentials.value[cnpj]
    if (!credId) return

    await AssignCredentialToCompany({
      CompanyCNPJ: cnpj,
      CredentialID: credId,
    })
    $q.notify({ type: 'positive', message: 'Credencial atribuída com sucesso.' })
    await loadCompanies()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao atribuir credencial: ' + String(err) })
  }
}

async function syncCompany(cnpj: string) {
  syncing.value = cnpj
  try {
    const result = await Pull({ CNPJ: cnpj })
    const credentialCNPJ = result.CredentialCNPJ || 'pendente'
    $q.notify({
      type: 'positive',
      message: `Sincronização concluída! Docs: ${result.DocumentsFound}, Base: ${result.ConsultationBasis}, Credencial: ${credentialCNPJ}`,
    })
    await loadCompanies()
  } catch (err) {
    if (String(err).includes('operação cancelada')) {
      $q.notify({ type: 'warning', message: 'Sincronização cancelada.' })
    } else {
      $q.notify({ type: 'negative', message: 'Erro na sincronização: ' + String(err) })
    }
  } finally {
    syncing.value = null
  }
}

onMounted(() => {
  reloadData()
})
</script>
