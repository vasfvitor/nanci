<template>
  <q-dialog v-model="isOpen" :no-backdrop-dismiss="dirty" :persistent="loading">
    <q-card style="min-width: 460px">
      <q-form @submit="submit">
        <q-card-section>
          <div class="text-h6">Editar Credencial</div>
        </q-card-section>

        <q-card-section class="q-pt-none q-gutter-y-md">
          <DraftBanner v-if="restored" @clear="clearDraft" />
          <q-input
            v-model="form.Label"
            label="Rótulo da Credencial"
            outlined
            dense
            autofocus
            :error="submitted && !!errors.Label"
            :error-message="errors.Label"
          />
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
// submitted turns the inline errors on after the first save attempt.
const submitted = ref(false)

const { form, dirty, restored, open, keep, clear, done } = useFormDraft(
  () => `credential:edit:${props.credentialData?.ID ?? ''}`,
  () => ({
    CredentialID: props.credentialData?.ID || '',
    Label: props.credentialData?.Label || '',
  }),
  { dropWhenSeedChanges: true }
)

const errors = computed(() => {
  const found: Partial<Record<'Label', string>> = {}
  if (!form.value.Label) found.Label = 'Informe o rótulo da credencial.'
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

async function submit() {
  if (loading.value) return
  submitted.value = true
  if (Object.keys(errors.value).length > 0) return

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
