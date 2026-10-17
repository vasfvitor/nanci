<template>
  <q-dialog v-model="isOpen" :no-backdrop-dismiss="dirty" :persistent="loading">
    <q-card style="min-width: 420px">
      <q-card-section>
        <div class="text-h6">Adicionar Credencial</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <DraftBanner v-if="restored" @clear="clear" />
        <q-input v-model="form.Label" label="Rótulo" outlined dense />

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
        <q-btn v-close-popup flat label="Cancelar" color="primary" />
        <q-btn flat label="Salvar" color="primary" :loading="loading" @click="submit" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, watch } from 'vue'
import { useQuasar } from 'quasar'
import DraftBanner from '@/components/DraftBanner.vue'
import { useFormDraft } from '@/composables/useFormDraft'
import { desktopClient, errorMessage } from '@/platform/wails/client'

const props = defineProps<{
  modelValue: boolean
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'added'): void
}>()
const $q = useQuasar()
const isOpen = ref(props.modelValue)
const loading = ref(false)

const { form, dirty, restored, open, keep, clear, done } = useFormDraft('credential:add', () => ({
  Label: '',
  CertPath: '',
}))

watch(
  () => props.modelValue,
  (val) => {
    isOpen.value = val
    if (val) open()
  }
)

watch(isOpen, (val) => {
  if (!val) keep()
  emit('update:modelValue', val)
})

async function selectCert() {
  try {
    const path = await desktopClient.selectCertificate()
    if (path) {
      form.value.CertPath = path
      if (!form.value.Label) {
        form.value.Label = path.split(/[\\/]/).pop() || path
      }
    }
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao selecionar certificado: ' + errorMessage(err) })
  }
}

async function submit() {
  if (!form.value.CertPath) {
    $q.notify({ type: 'warning', message: 'Selecione um certificado.' })
    return
  }

  loading.value = true
  try {
    await desktopClient.addCredential(form.value)
    $q.notify({ type: 'positive', message: 'Credencial adicionada com sucesso!' })
    emit('added')
    done()
    isOpen.value = false
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao adicionar credencial: ' + errorMessage(err) })
  } finally {
    loading.value = false
  }
}
</script>
