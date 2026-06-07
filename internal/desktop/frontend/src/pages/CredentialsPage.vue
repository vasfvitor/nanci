<template>
  <q-page padding>
    <div class="row items-center justify-between q-mb-md">
      <h5 class="q-my-none">Credenciais</h5>
      <q-btn color="primary" icon="vpn_key" label="Adicionar" @click="showAddDialog = true" />
    </div>

    <div class="row q-col-gutter-md">
      <div v-for="credential in credentials" :key="credential.ID" class="col-12 col-md-6 col-lg-4">
        <q-card>
          <q-card-section>
            <div class="row items-center justify-between q-mb-sm">
              <div class="text-h6 text-weight-bold text-dark">{{ credential.Label }}</div>
              <q-btn flat round icon="edit" color="primary" @click="openEditDialog(credential)" />
            </div>
            <div class="text-caption text-grey-8">{{ credential.CertPath }}</div>
            <q-badge
              :color="credential.Environment === 'producao' ? 'positive' : 'warning'"
              class="q-mt-sm"
            >
              {{ credential.Environment }}
            </q-badge>
          </q-card-section>

          <q-card-section class="q-gutter-y-xs">
            <div><strong>Proprietário:</strong> {{ ownerLabel(credential) }}</div>
            <div><strong>Inspeção:</strong> {{ credential.InspectedAt ? 'Concluída' : 'Pendente' }}</div>
          </q-card-section>

          <q-card-actions align="right">
            <q-btn flat color="primary" icon="folder_open" label="Trocar arquivo" @click="changePath(credential.ID)" />
          </q-card-actions>
        </q-card>
      </div>
    </div>

    <div v-if="credentials.length === 0" class="text-center q-pa-xl text-grey-6">
      <q-icon name="vpn_key" size="4rem" />
      <p class="text-h6 q-mt-md">Nenhuma credencial cadastrada</p>
    </div>

    <AddCredentialDialog v-model="showAddDialog" @added="loadCredentials" />
    <EditCredentialDialog v-model="showEditDialog" :credentialData="selectedCredentialToEdit" @updated="loadCredentials" />
  </q-page>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useQuasar } from 'quasar'
import { ListCredentials, SelectCertificate, UpdateCredentialPath } from '../../wailsjs/go/main/App'
import { nfse } from '../../wailsjs/go/models'
import AddCredentialDialog from '../components/AddCredentialDialog.vue'
import EditCredentialDialog from '../components/EditCredentialDialog.vue'

const $q = useQuasar()
const credentials = ref<nfse.Credential[]>([])
const showAddDialog = ref(false)
const showEditDialog = ref(false)
const selectedCredentialToEdit = ref<nfse.Credential | null>(null)

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
