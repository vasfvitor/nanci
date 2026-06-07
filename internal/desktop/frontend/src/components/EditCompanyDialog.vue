<template>
  <q-dialog v-model="isOpen">
    <q-card style="min-width: 460px">
      <q-card-section>
        <div class="text-h6">Editar Empresa</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <q-input
          v-model="form.CNPJ"
          label="CNPJ"
          outlined
          dense
          readonly
          hint="O CNPJ não pode ser alterado."
        />
        <q-input v-model="form.Name" label="Nome / Razão Social" outlined dense />

        <q-select
          v-model="form.Environment"
          :options="['producao', 'producao_restrita']"
          label="Ambiente"
          outlined
          dense
        />
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
import { UpdateCompany } from '../../wailsjs/go/main/App'
import { useQuasar } from 'quasar'

const props = defineProps<{
  modelValue: boolean
  companyData: any
}>()

const emit = defineEmits(['update:modelValue', 'updated'])
const $q = useQuasar()

const isOpen = ref(props.modelValue)
const loading = ref(false)

const form = ref({
  CNPJ: '',
  Name: '',
  Environment: 'producao',
})

watch(
  () => props.modelValue,
  (val) => {
    isOpen.value = val
    if (val && props.companyData) {
      form.value.CNPJ = props.companyData.CNPJ || ''
      form.value.Name = props.companyData.Name || ''
      form.value.Environment = props.companyData.Environment || 'producao'
    }
  }
)

watch(isOpen, (val) => {
  emit('update:modelValue', val)
})

async function submit() {
  if (!form.value.Name) {
    $q.notify({ type: 'warning', message: 'Preencha o nome da empresa.' })
    return
  }

  loading.value = true
  try {
    await UpdateCompany({
      CNPJ: form.value.CNPJ,
      Name: form.value.Name,
      Environment: form.value.Environment,
    })
    $q.notify({ type: 'positive', message: 'Empresa atualizada com sucesso!' })
    emit('updated')
    isOpen.value = false
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao atualizar empresa: ' + String(err) })
  } finally {
    loading.value = false
  }
}
</script>
