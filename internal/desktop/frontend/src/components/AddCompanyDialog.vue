<template>
  <q-dialog v-model="isOpen">
    <q-card style="min-width: 460px">
      <q-card-section>
        <div class="text-h6">Adicionar Empresa</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <q-input v-model="form.CNPJ" label="CNPJ" outlined dense />
        <q-input v-model="form.Name" label="Nome / Razão Social" outlined dense />

        <q-option-group
          v-model="credentialMode"
          inline
          :options="[
            { label: 'Usar credencial existente', value: 'existing' },
            { label: 'Criar nova credencial', value: 'new' },
          ]"
        />

        <q-select
          v-if="credentialMode === 'existing'"
          v-model="form.CredentialID"
          :options="credentialOptions"
          label="Credencial"
          emit-value
          map-options
          outlined
          dense
        />

        <template v-else>
          <q-input v-model="form.CredentialLabel" label="Rótulo da Credencial" outlined dense />
          <q-select
            v-model="form.Environment"
            :options="['producao', 'producao_restrita', 'homologacao']"
            label="Ambiente"
            outlined
            dense
          />
          <div class="row items-center q-gutter-x-sm">
            <q-input
              v-model="form.CertPath"
              label="Caminho do Certificado (.pfx/.p12)"
              outlined
              dense
              readonly
              class="col"
            />
            <q-btn icon="folder" color="primary" @click="selectCert" />
          </div>
        </template>
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat label="Cancelar" color="primary" v-close-popup />
        <q-btn flat label="Salvar" color="primary" @click="submit" :loading="loading" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { AddCompany, ListCredentials, SelectCertificate } from '../../wailsjs/go/main/App'
import { useQuasar } from 'quasar'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits(['update:modelValue', 'added'])
const $q = useQuasar()

const isOpen = ref(props.modelValue)
const loading = ref(false)
const credentialMode = ref<'existing' | 'new'>('existing')
const credentialOptions = ref<{ label: string; value: string }[]>([])

const form = ref({
  CNPJ: '',
  Name: '',
  CredentialID: '',
  CredentialLabel: '',
  CertPath: '',
  Environment: 'producao_restrita',
})

watch(
  () => props.modelValue,
  async (val) => {
    isOpen.value = val
    if (val) {
      await loadCredentials()
    }
  }
)

watch(isOpen, (val) => {
  emit('update:modelValue', val)
})

async function loadCredentials() {
  try {
    const credentials = (await ListCredentials()) || []
    credentialOptions.value = credentials.map((credential) => ({
      label: `${credential.Label} (${credential.Environment})`,
      value: credential.ID,
    }))
    credentialMode.value = credentialOptions.value.length > 0 ? 'existing' : 'new'
    if (credentialOptions.value.length > 0 && !form.value.CredentialID) {
      form.value.CredentialID = credentialOptions.value[0].value
    }
  } catch (err: any) {
    $q.notify({ type: 'negative', message: 'Erro ao carregar credenciais: ' + err })
  }
}

async function selectCert() {
  try {
    const path = await SelectCertificate()
    if (path) {
      form.value.CertPath = path
      if (!form.value.CredentialLabel) {
        form.value.CredentialLabel = path.split(/[\\/]/).pop() || path
      }
    }
  } catch (err: any) {
    $q.notify({ type: 'negative', message: 'Erro ao selecionar certificado: ' + err })
  }
}

async function submit() {
  if (!form.value.CNPJ || !form.value.Name) {
    $q.notify({ type: 'warning', message: 'Preencha CNPJ e nome da empresa.' })
    return
  }
  if (credentialMode.value === 'existing' && !form.value.CredentialID) {
    $q.notify({ type: 'warning', message: 'Selecione uma credencial.' })
    return
  }
  if (credentialMode.value === 'new' && !form.value.CertPath) {
    $q.notify({ type: 'warning', message: 'Selecione um certificado.' })
    return
  }

  loading.value = true
  try {
    await AddCompany({
      CNPJ: form.value.CNPJ,
      Name: form.value.Name,
      CredentialID: credentialMode.value === 'existing' ? form.value.CredentialID : '',
      CredentialLabel: credentialMode.value === 'new' ? form.value.CredentialLabel : '',
      CertPath: credentialMode.value === 'new' ? form.value.CertPath : '',
      Environment: credentialMode.value === 'new' ? form.value.Environment : '',
    })
    $q.notify({ type: 'positive', message: 'Empresa adicionada com sucesso!' })
    emit('added')
    isOpen.value = false
    form.value = {
      CNPJ: '',
      Name: '',
      CredentialID: credentialOptions.value[0]?.value || '',
      CredentialLabel: '',
      CertPath: '',
      Environment: 'producao_restrita',
    }
  } catch (err: any) {
    $q.notify({ type: 'negative', message: 'Erro ao adicionar empresa: ' + err })
  } finally {
    loading.value = false
  }
}
</script>
