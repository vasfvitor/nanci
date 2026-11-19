<template>
  <q-dialog v-model="isOpen" persistent>
    <q-card style="min-width: 350px">
      <q-form @submit="onSubmit">
        <q-card-section>
          <div class="text-h6">Senha do Certificado</div>
          <div v-if="requestData?.Purpose" class="text-caption text-app-muted">
            Finalidade: {{ requestData.Purpose }}
          </div>
          <div class="text-caption text-app-muted">Empresa: {{ requestData?.CompanyName }}</div>
          <div class="text-caption text-app-muted">CNPJ: {{ requestData?.TargetCNPJ }}</div>
          <div class="text-caption text-app-muted">
            Credencial: {{ requestData?.CredentialLabel }}
          </div>
          <div class="text-caption text-app-muted">Arquivo: {{ requestData?.CertPath }}</div>
        </q-card-section>

        <q-card-section class="q-pt-none">
          <q-input
            v-model="password"
            dense
            autofocus
            :type="show ? 'text' : 'password'"
            :disable="submitting"
          >
            <template #append>
              <q-icon
                :name="show ? 'visibility_off' : 'visibility'"
                :aria-label="show ? 'Ocultar senha' : 'Mostrar senha'"
                role="button"
                tabindex="0"
                class="cursor-pointer"
                @click="show = !show"
                @keyup.enter.stop="show = !show"
              />
            </template>
          </q-input>
        </q-card-section>

        <q-card-actions align="right" class="text-primary">
          <q-btn flat label="Cancelar" :disable="submitting" @click="onCancel" />
          <q-btn flat label="Confirmar" type="submit" :loading="submitting" :disable="submitting" />
        </q-card-actions>
      </q-form>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { computed, ref, watch } from 'vue'
import { useCertPasswordPrompt } from '@/composables/useCertPasswordPrompt'

// The dialog shows the oldest certificate password request; the composable
// holds the queue and sends the answers.
const { current: requestData, submitting, submit, cancel } = useCertPasswordPrompt()
const password = ref('')
const show = ref(false)
const isOpen = computed(() => Boolean(requestData.value))

// Every request starts with an empty, hidden password.
watch(
  () => requestData.value?.RequestID,
  () => {
    password.value = ''
    show.value = false
  }
)

async function onSubmit() {
  if (await submit(password.value)) password.value = ''
}

function onCancel() {
  return cancel()
}
</script>
