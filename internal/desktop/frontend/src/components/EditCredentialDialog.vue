<template>
  <q-dialog v-model="isOpen">
    <q-card style="min-width: 460px">
      <q-card-section>
        <div class="text-h6">Editar Credencial</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <q-input v-model="form.Label" label="Rótulo da Credencial" outlined dense />

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
import { desktopClient } from '@/platform/wails/client'
import type { CredentialSummary } from '@/types/desktop'

const props = defineProps<{
  modelValue: boolean
  credentialData: CredentialSummary | null
}>()

const emit = defineEmits(['update:modelValue', 'updated'])
const $q = useQuasar()

const isOpen = ref(props.modelValue)
const loading = ref(false)

const form = ref({
  CredentialID: '',
  Label: '',
})

watch(
  () => props.modelValue,
  (val) => {
    isOpen.value = val
    if (val && props.credentialData) {
      form.value.CredentialID = props.credentialData.ID || ''
      form.value.Label = props.credentialData.Label || ''
    }
  }
)

watch(isOpen, (val) => {
  emit('update:modelValue', val)
})

async function submit() {
  if (!form.value.Label) {
    $q.notify({ type: 'warning', message: 'Preencha o rótulo da credencial.' })
    return
  }

  loading.value = true
  try {
    await desktopClient.updateCredentialData({
      CredentialID: form.value.CredentialID,
      Label: form.value.Label,
    })
    $q.notify({ type: 'positive', message: 'Credencial atualizada com sucesso!' })
    emit('updated')
    isOpen.value = false
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao atualizar credencial: ' + String(err) })
  } finally {
    loading.value = false
  }
}
</script>
