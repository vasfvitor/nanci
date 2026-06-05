<template>
  <q-dialog v-model="isOpen">
    <q-card style="min-width: 400px">
      <q-card-section>
        <div class="text-h6">Adicionar Empresa</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <q-input v-model="form.CNPJ" label="CNPJ" outlined dense />
        <q-input v-model="form.Name" label="Nome / Razão Social" outlined dense />

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
      </q-card-section>

      <q-card-actions align="right">
        <q-btn flat label="Cancelar" color="primary" v-close-popup />
        <q-btn flat label="Salvar" color="primary" @click="submit" :loading="loading" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { SelectCertificate, AddCompany } from '../../wailsjs/go/main/App'
import { useQuasar } from 'quasar'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits(['update:modelValue', 'added'])
const $q = useQuasar()

const isOpen = ref(props.modelValue)
const loading = ref(false)

const form = ref({
  CNPJ: '',
  Name: '',
  Environment: 'producao',
  CertPath: '',
})

// Sync isOpen with v-model
import { watch } from 'vue'
watch(
  () => props.modelValue,
  (val) => {
    isOpen.value = val
  }
)
watch(isOpen, (val) => {
  emit('update:modelValue', val)
})

async function selectCert() {
  try {
    const path = await SelectCertificate()
    if (path) {
      form.value.CertPath = path
    }
  } catch (err: any) {
    $q.notify({ type: 'negative', message: 'Erro ao selecionar certificado: ' + err })
  }
}

async function submit() {
  if (!form.value.CNPJ || !form.value.CertPath) {
    $q.notify({ type: 'warning', message: 'Preencha CNPJ e selecione um certificado.' })
    return
  }

  loading.value = true
  try {
    // Wails generated type might require mapping
    await AddCompany(form.value)
    $q.notify({ type: 'positive', message: 'Empresa adicionada com sucesso!' })
    emit('added')
    isOpen.value = false
    // Reset
    form.value = { CNPJ: '', Name: '', Environment: 'producao', CertPath: '' }
  } catch (err: any) {
    $q.notify({ type: 'negative', message: 'Erro ao adicionar empresa: ' + err })
  } finally {
    loading.value = false
  }
}
</script>
