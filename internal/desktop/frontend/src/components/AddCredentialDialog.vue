<template>
  <q-dialog v-model="isOpen">
    <q-card style="min-width: 420px">
      <q-card-section>
        <div class="text-h6">Adicionar Credencial</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <q-input v-model="form.Label" label="Rótulo" outlined dense />
        <q-select
          v-model="form.Environment"
          :options="['producao', 'producao_restrita']"
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
import { ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import { AddCredential, SelectCertificate } from '../../wailsjs/go/main/App'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits(['update:modelValue', 'added'])
const $q = useQuasar()
const isOpen = ref(props.modelValue)
const loading = ref(false)

const form = ref({
  Label: '',
  CertPath: '',
  Environment: 'producao_restrita',
})

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
      if (!form.value.Label) {
        form.value.Label = path.split(/[\\/]/).pop() || path
      }
    }
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao selecionar certificado: ' + String(err) })
  }
}

async function submit() {
  if (!form.value.CertPath) {
    $q.notify({ type: 'warning', message: 'Selecione um certificado.' })
    return
  }

  loading.value = true
  try {
    await AddCredential(form.value)
    $q.notify({ type: 'positive', message: 'Credencial adicionada com sucesso!' })
    emit('added')
    isOpen.value = false
    form.value = { Label: '', CertPath: '', Environment: 'producao_restrita' }
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao adicionar credencial: ' + String(err) })
  } finally {
    loading.value = false
  }
}
</script>
