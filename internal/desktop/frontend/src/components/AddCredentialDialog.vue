<template>
  <q-dialog v-model="isOpen" :no-backdrop-dismiss="dirty" :persistent="loading">
    <q-card style="min-width: 420px">
      <q-form @submit="submit">
        <q-card-section>
          <div class="text-h6">Adicionar Credencial</div>
        </q-card-section>

        <q-card-section class="q-pt-none q-gutter-y-md">
          <DraftBanner v-if="restored" @clear="clearDraft" />
          <q-input v-model="form.Label" label="Rótulo" outlined dense autofocus />

          <div class="row items-center q-gutter-x-sm">
            <q-input
              v-model="form.CertPath"
              label="Caminho do Certificado (.pfx/.p12)"
              outlined
              dense
              readonly
              class="col"
              :error="submitted && !!errors.CertPath"
              :error-message="errors.CertPath"
            />
            <q-btn icon="folder" color="primary" @click="selectCert" />
          </div>
        </q-card-section>

        <q-card-actions align="right">
          <q-btn v-close-popup flat label="Cancelar" color="primary" :disable="loading" />
          <q-btn flat type="submit" label="Salvar" color="primary" :loading="loading" />
        </q-card-actions>
      </q-form>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
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
// submitted turns the inline errors on after the first save attempt.
const submitted = ref(false)

const { form, dirty, restored, open, keep, clear, done } = useFormDraft('credential:add', () => ({
  Label: '',
  CertPath: '',
}))

// The label is optional: the backend falls back to the certificate path.
const errors = computed(() => {
  const found: Partial<Record<'CertPath', string>> = {}
  if (!form.value.CertPath) found.CertPath = 'Selecione um certificado.'
  return found
})

watch(
  () => props.modelValue,
  (val) => {
    isOpen.value = val
    if (val) {
      submitted.value = false
      open()
    }
  }
)

watch(isOpen, (val) => {
  if (!val) keep()
  emit('update:modelValue', val)
})

function clearDraft() {
  submitted.value = false
  clear()
}

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
  if (loading.value) return
  submitted.value = true
  if (Object.keys(errors.value).length > 0) return

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
