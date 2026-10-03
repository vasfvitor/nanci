<template>
  <q-drawer v-model="model" show-if-above bordered>
    <!-- The workspace: the company and competência every document page lists. -->
    <section
      aria-label="Empresa e competência"
      :title="competenceHelp"
      class="q-px-md q-pt-md q-pb-sm column no-wrap q-gutter-sm"
    >
      <div v-if="loadError" class="column items-start">
        <div class="text-body2">Não foi possível carregar as empresas.</div>
        <div class="text-caption text-app-muted">{{ loadError }}</div>
        <q-btn flat dense color="primary" label="Tentar de novo" @click="retryCompanies" />
      </div>

      <div v-if="noCompanies" class="column items-start">
        <div class="text-body2">Nenhuma empresa cadastrada.</div>
        <q-btn flat dense color="primary" label="Cadastrar empresa" to="/" />
      </div>
      <q-select
        v-else-if="companies.length > 0 || !loadError"
        v-model="cnpj"
        :options="companyOptions"
        label="Empresa"
        :display-value="selectedCompany?.Name ?? ''"
        :hint="formatCpfCnpj(cnpj)"
        :loading="!loaded && !loadError"
        dense
        outlined
        options-dense
        emit-value
        map-options
      >
        <template #option="scope">
          <q-item v-bind="scope.itemProps">
            <q-item-section>
              <q-item-label>{{ scope.opt.label }}</q-item-label>
              <q-item-label caption>{{ scope.opt.caption }}</q-item-label>
            </q-item-section>
          </q-item>
        </template>
      </q-select>

      <CompetencePicker v-model="competence" />
    </section>

    <q-separator />

    <q-list class="q-py-md">
      <q-item v-ripple clickable to="/" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="business" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Empresas</q-item-label>
        </q-item-section>
      </q-item>

      <q-item v-ripple clickable to="/documents" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="description" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">NFS-e</q-item-label>
        </q-item-section>
      </q-item>

      <q-item v-ripple clickable to="/nfe" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="receipt_long" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">NF-e</q-item-label>
        </q-item-section>
      </q-item>

      <q-item v-ripple clickable to="/cte" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="local_shipping" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">CT-e</q-item-label>
        </q-item-section>
      </q-item>

      <q-item v-ripple clickable to="/credentials" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="vpn_key" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Credenciais</q-item-label>
        </q-item-section>
      </q-item>

      <q-separator class="q-my-md" />

      <q-item v-ripple clickable to="/query" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="api" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Consulta Direta API</q-item-label>
        </q-item-section>
      </q-item>

      <q-separator class="q-my-md" />

      <q-item v-ripple clickable to="/settings" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="settings" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Configurações</q-item-label>
        </q-item-section>
      </q-item>
    </q-list>
  </q-drawer>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { storeToRefs } from 'pinia'
import CompetencePicker from './CompetencePicker.vue'
import { useWorkspaceStore } from '@/stores/workspace'
import { formatCpfCnpj } from '@/utils/formatters'

// AppLeftDrawer is the app navigation, topped by the workspace company and
// competência. Changing either makes the mounted document page search.
const model = defineModel<boolean>({ default: false })

const workspace = useWorkspaceStore()
// competence is written by CompetencePicker, which only sets a complete
// competência or '' when cleared.
const { cnpj, competence, companies, loaded, loadError, selectedCompany } = storeToRefs(workspace)

const competenceHelp =
  'Competência: na NFS-e é a competência da nota (compNFSe); na NF-e e no CT-e, o mês de emissão. Vazia, lista todos os meses.'

// companyOptions are the workspace companies, with the name and the
// formatted CNPJ the option list shows. The label is the name, so typing
// over the closed select jumps to a company by name.
const companyOptions = computed(() =>
  companies.value.map((company) => ({
    label: company.Name,
    value: company.CNPJ,
    caption: formatCpfCnpj(company.CNPJ),
  }))
)

// noCompanies is true once a list arrived empty.
const noCompanies = computed(() => loaded.value && !loadError.value && companies.value.length === 0)

// retryCompanies lists the companies again. A failure shows in loadError.
function retryCompanies() {
  workspace.loadCompanies().catch(() => {})
}
</script>
