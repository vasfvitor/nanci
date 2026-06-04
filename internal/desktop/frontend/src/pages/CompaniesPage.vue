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
            <div class="text-h6">{{ company.Name }}</div>
            <div class="text-subtitle2 text-grey-8">{{ company.CNPJ }}</div>
            <q-badge
              :color="company.Environment === 'producao' ? 'positive' : 'warning'"
              class="q-mt-sm"
            >
              {{ company.Environment }}
            </q-badge>
          </q-card-section>

          <q-card-section> Último NSU: {{ company.LastNSU }} </q-card-section>

          <q-card-actions align="right">
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

    <!-- Empty State -->
    <div v-if="companies.length === 0" class="text-center q-pa-xl text-grey-6">
      <q-icon name="business" size="4rem" />
      <p class="text-h6 q-mt-md">Nenhuma empresa cadastrada</p>
    </div>

    <AddCompanyDialog v-model="showAddDialog" @added="loadCompanies" />
  </q-page>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { ListCompanies, Pull } from '../../wailsjs/go/main/App'
import { nfse } from '../../wailsjs/go/models'
import { useQuasar } from 'quasar'
import AddCompanyDialog from '../components/AddCompanyDialog.vue'

const $q = useQuasar()
const companies = ref<nfse.Company[]>([])
const showAddDialog = ref(false)
const syncing = ref<string | null>(null)

async function loadCompanies() {
  try {
    const list = await ListCompanies()
    companies.value = list || []
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao listar empresas: ' + err })
  }
}

async function syncCompany(cnpj: string) {
  syncing.value = cnpj
  try {
    const result = await Pull({ CNPJ: cnpj })
    $q.notify({
      type: 'positive',
      message: `Sincronização concluída! Docs: ${result.DocumentsFound}, Erros: ${result.Errors}`,
    })
    await loadCompanies() // refresh last NSU
  } catch (err: any) {
    if (String(err).includes('operação cancelada')) {
      $q.notify({ type: 'warning', message: 'Sincronização cancelada.' })
    } else {
      $q.notify({ type: 'negative', message: 'Erro na sincronização: ' + err })
    }
  } finally {
    syncing.value = null
  }
}

onMounted(() => {
  loadCompanies()
})
</script>
