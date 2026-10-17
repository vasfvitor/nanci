<template>
  <q-dialog v-model="isOpen" :no-backdrop-dismiss="dirty" :persistent="loading">
    <q-card style="min-width: 460px">
      <q-card-section>
        <div class="text-h6">Editar Credencial</div>
      </q-card-section>

      <q-card-section class="q-pt-none q-gutter-y-md">
        <DraftBanner v-if="restored" @clear="clear" />
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
import DraftBanner from '@/components/DraftBanner.vue'
import { useFormDraft } from '@/composables/useFormDraft'
import { desktopClient, errorMessage } from '@/platform/wails/client'
import type { CredentialSummary } from '@/types/desktop'

const props = defineProps<{
  modelValue: boolean
  credentialData: CredentialSummary | null
}>()

const emit = defineEmits<{
  (e: 'update:modelValue', value: boolean): void
  (e: 'updated'): void
}>()
const $q = useQuasar()

const isOpen = ref(props.modelValue)
const loading = ref(false)

const { form, dirty, restored, open, keep, clear, done } = useFormDraft(
  () => `credential:edit:${props.credentialData?.ID ?? ''}`,
  () => ({
    CredentialID: props.credentialData?.ID || '',
    Label: props.credentialData?.Label || '',
  }),
  { dropWhenSeedChanges: true }
)

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
    done()
    isOpen.value = false
  } catch (err) {
    $q.notify({ type: 'negative', message: 'Erro ao atualizar credencial: ' + errorMessage(err) })
  } finally {
    loading.value = false
  }
}
</script>
