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
import { ref, onMounted, onUnmounted, computed, watch } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import { onWailsEvent, type Unsubscribe } from '@/platform/wails/events'
import type { CertPasswordRequest } from '@/types/desktop'

const requests = ref<CertPasswordRequest[]>([])
const password = ref('')
const show = ref(false)
// submitting is set while the answer to the current request is sent, so a
// second click cannot answer it twice.
const submitting = ref(false)
const isOpen = computed(() => requests.value.length > 0)
const requestData = computed(() => requests.value[0])
const unsubscribe = ref<Unsubscribe | null>(null)

// Every request starts with an empty, hidden password.
watch(
  () => requestData.value?.RequestID,
  () => {
    password.value = ''
    show.value = false
  }
)

function handleRequest(req: CertPasswordRequest) {
  requests.value.push(req)
}

// answer sends the answer to the current request and moves to the next one.
async function answer(send: (req: CertPasswordRequest) => Promise<void>) {
  const req = requests.value[0]
  if (!req || submitting.value) return
  submitting.value = true
  try {
    await send(req)
    password.value = ''
    requests.value.shift()
  } finally {
    submitting.value = false
  }
}

function onSubmit() {
  return answer((req) => desktopClient.submitCertPassword(req.RequestID, password.value))
}

function onCancel() {
  return answer((req) => desktopClient.cancelCertPassword(req.RequestID))
}

onMounted(() => {
  unsubscribe.value = onWailsEvent<CertPasswordRequest>('request-cert-password', handleRequest)
})

onUnmounted(() => {
  unsubscribe.value?.()
  unsubscribe.value = null
})
</script>
