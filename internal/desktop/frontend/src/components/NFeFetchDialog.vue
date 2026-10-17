<template>
  <q-dialog ref="dialogRef" @hide="onDialogHide">
    <q-card class="q-dialog-plugin nfe-fetch-dialog">
      <q-form @submit="onSubmit">
        <q-card-section>
          <div class="text-h6">Buscar NF-e pela chave</div>
          <div class="text-caption text-app-muted">{{ budgetText }}</div>
        </q-card-section>

        <q-card-section class="q-pt-none">
          <q-input
            :model-value="chave"
            label="Chave de acesso"
            outlined
            dense
            autofocus
            input-class="text-mono"
            :maxlength="CHAVE_INPUT_MAX_LENGTH"
            :error="chave.length > 0 && !valid"
            error-message="A chave tem 44 caracteres"
            @update:model-value="onInput"
          />
        </q-card-section>

        <q-card-actions align="right">
          <q-btn flat label="Cancelar" @click="onDialogCancel" />
          <q-btn color="primary" type="submit" label="Buscar" :disable="!valid" />
        </q-card-actions>
      </q-form>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref } from 'vue'
import { useDialogPluginComponent } from 'quasar'
import { compactCode } from '@/utils/cnpj'

const CHAVE_PATTERN = /^[0-9A-Z]{44}$/
// A pasted chave may come grouped with spaces ("3526 0912 ..."), so the
// field accepts more than 44 characters and compactCode strips the rest.
const CHAVE_INPUT_MAX_LENGTH = 60

const props = defineProps<{
  requestsLastHour: number
  requestBudget: number
}>()

defineEmits<{
  ok: [chave: string]
  hide: []
}>()

const { dialogRef, onDialogHide, onDialogOK, onDialogCancel } = useDialogPluginComponent()

const chave = ref('')
const valid = computed(() => CHAVE_PATTERN.test(chave.value))

const budgetText = computed(
  () => `Gasta 1 das ${props.requestBudget} consultas da hora (${props.requestsLastHour} usadas)`
)

function onInput(value: string | number | null) {
  chave.value = compactCode(value === null ? null : String(value))
}

function onSubmit() {
  if (!valid.value) return
  onDialogOK(chave.value)
}

defineExpose({ dialogRef })
</script>

<style scoped>
.nfe-fetch-dialog {
  width: 560px;
  max-width: 90vw;
}
</style>
