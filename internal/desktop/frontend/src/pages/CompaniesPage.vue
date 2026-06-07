<template>
  <q-page padding>
    <div class="row items-center justify-between q-mb-md">
      <h5 class="q-my-none">Empresas</h5>
      <q-btn color="primary" icon="add" label="Adicionar" @click="showAddDialog = true" />
    </div>

    <div class="row q-col-gutter-md">
      <div v-for="company in companies" :key="company.ID" class="col-12 col-md-6 col-lg-4">
        <q-card>
          <q-card-section>
            <div class="row items-center justify-between q-mb-sm">
              <div class="text-h6 text-weight-bold text-dark">{{ company.Name }}</div>
              <q-btn flat round icon="edit" color="primary" @click="openEditDialog(company)" />
            </div>
            <div class="text-subtitle2 text-grey-8">{{ company.CNPJ }}</div>
            <q-badge
              :color="company.Environment === 'producao' ? 'positive' : 'warning'"
              class="q-mt-sm"
            >
              {{ company.Environment }}
            </q-badge>
          </q-card-section>

          <q-card-section class="q-gutter-y-sm">
            <div><strong>Último NSU:</strong> {{ company.LastNSU }}</div>
            <div><strong>Credencial ativa:</strong> {{ company.CredentialLabel || 'Sem credencial' }}</div>

            <q-select
              v-model="selectedCredentials[company.CNPJ]"
              :options="credentialOptions"
              emit-value
              map-options
              label="Trocar credencial"
              outlined
              dense
            />
          </q-card-section>

          <q-card-actions align="right">
            <q-btn
              flat
              color="secondary"
              icon="link"
              label="Salvar credencial"
              @click="assignCredential(company.CNPJ)"
              :disable="!selectedCredentials[company.CNPJ] || selectedCredentials[company.CNPJ] === company.CredentialID"
            />
            <q-btn
              flat
              color="primary"
              icon="sync"
              label="Sincronizar"
              @click="syncCompany(company.CNPJ)"
              :loading="syncing === company.CNPJ"
            />
          </q-card-actions>
        </q-card>
      </div>
    </div>

    <div v-if="companies.length === 0" class="text-center q-pa-xl text-grey-6">
      <q-icon name="business" size="4rem" />
      <p class="text-h6 q-mt-md">Nenhuma empresa cadastrada</p>
    </div>

    <AddCompanyDialog v-model="showAddDialog" @added="reloadData" />
    <EditCompanyDialog v-model="showEditDialog" :companyData="selectedCompanyToEdit" @updated="reloadData" />
  </q-page>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useQuasar } from 'quasar'
import { AssignCredentialToCompany, ListCompanies, ListCredentials, Pull } from '../../wailsjs/go/main/App'
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
  selectedCredentials.value = Object.fromEntries(list.map((company) => [company.CNPJ, company.CredentialID]))
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
    await AssignCredentialToCompany({
      CompanyCNPJ: cnpj,
      CredentialID: selectedCredentials.value[cnpj],
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
