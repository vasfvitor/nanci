<template>
  <q-page padding>
    <q-table
      title="Credenciais"
      :rows="credentials"
      :columns="columns"
      row-key="ID"
      flat
      bordered
      dense
      :loading="false"
      no-data-label="Nenhuma credencial cadastrada."
    >
      <template #top-right>
        <q-btn
          color="primary"
          icon="vpn_key"
          label="Adicionar"
          dense
          flat
          @click="showAddDialog = true"
        />
      </template>

      <template #body-cell-path="props">
        <q-td :props="props">
          <div class="ellipsis text-grey" style="max-width: 200px">
            {{ props.row.CertPath }}
            <q-tooltip>{{ props.row.CertPath }}</q-tooltip>
          </div>
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

      <template #body-cell-inspecao="props">
        <q-td :props="props">
          <q-badge :color="props.row.InspectedAt ? 'positive' : 'grey'" outline>
            {{ props.row.InspectedAt ? 'Concluída' : 'Pendente' }}
          </q-badge>
        </q-td>
      </template>

      <template #body-cell-acoes="props">
        <q-td :props="props" class="q-gutter-x-sm">
          <q-btn
            dense
            flat
            round
            color="primary"
            icon="folder_open"
            title="Trocar arquivo"
            @click="changePath(props.row.ID)"
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

    <AddCredentialDialog v-model="showAddDialog" @added="loadCredentials" />
    <EditCredentialDialog
      v-model="showEditDialog"
      :credential-data="selectedCredentialToEdit"
      @updated="loadCredentials"
    />
  </q-page>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useQuasar, type QTableColumn } from 'quasar'
import { ListCredentials, SelectCertificate, UpdateCredentialPath } from '../../wailsjs/go/main/App'
import { nfse } from '../../wailsjs/go/models'
import AddCredentialDialog from '../components/AddCredentialDialog.vue'
import EditCredentialDialog from '../components/EditCredentialDialog.vue'

const $q = useQuasar()
const credentials = ref<nfse.Credential[]>([])
const showAddDialog = ref(false)
const showEditDialog = ref(false)
const selectedCredentialToEdit = ref<nfse.Credential | null>(null)

const columns: QTableColumn[] = [
  { name: 'label', label: 'Nome', field: 'Label', align: 'left', sortable: true },
  {
    name: 'owner',
    label: 'Proprietário',
    field: (row: nfse.Credential) => ownerLabel(row),
    align: 'left',
    sortable: true,
  },
  { name: 'path', label: 'Arquivo PFX', field: 'CertPath', align: 'left' },
  { name: 'ambiente', label: 'Ambiente', field: 'Environment', align: 'left', sortable: true },
  { name: 'inspecao', label: 'Inspeção', field: () => '', align: 'center', sortable: true },
  { name: 'acoes', label: 'Ações', field: () => '', align: 'right' },
]

function openEditDialog(credential: nfse.Credential) {
  selectedCredentialToEdit.value = credential
  showEditDialog.value = true
}

function ownerLabel(credential: nfse.Credential) {
  return credential.OwnerCNPJ || 'Pendente de inspeção'
}

async function loadCredentials() {
  try {
    credentials.value = (await ListCredentials()) || []
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao listar credenciais: ' + String(err) })
  }
}

async function changePath(credentialID: string) {
  try {
    const path = await SelectCertificate()
    if (!path) return
    await UpdateCredentialPath({ CredentialID: credentialID, CertPath: path })
    $q.notify({ type: 'positive', message: 'Caminho da credencial atualizado.' })
    await loadCredentials()
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao atualizar credencial: ' + String(err) })
  }
}

onMounted(() => {
  loadCredentials()
})
</script>
