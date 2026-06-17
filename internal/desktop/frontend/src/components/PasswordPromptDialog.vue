<template>
  <q-dialog v-model="isOpen" persistent>
    <q-card style="min-width: 350px">
      <q-card-section>
        <div class="text-h6">Senha do Certificado</div>
        <div class="text-caption text-app-muted">Empresa: {{ requestData?.CompanyName }}</div>
        <div class="text-caption text-app-muted">CNPJ consultado: {{ requestData?.TargetCNPJ }}</div>
        <div class="text-caption text-app-muted">Credencial: {{ requestData?.CredentialLabel }}</div>
        <div class="text-caption text-app-muted">Arquivo: {{ requestData?.CertPath }}</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input v-model="password" dense autofocus type="password" @keyup.enter="onSubmit" />
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn flat label="Cancelar" @click="onCancel" />
        <q-btn flat label="Confirmar" @click="onSubmit" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { desktopClient } from '@/platform/wails/client'
import { onWailsEvent, type Unsubscribe } from '@/platform/wails/events'

interface CertPasswordRequest {
  RequestID: string
  CompanyName: string
  TargetCNPJ: string
  CredentialLabel: string
  CertPath: string
}

const requests = ref<CertPasswordRequest[]>([])
const password = ref('')
const isOpen = computed(() => requests.value.length > 0)
const requestData = computed(() => requests.value[0])
const unsubscribe = ref<Unsubscribe | null>(null)

function handleRequest(req: CertPasswordRequest) {
  requests.value.push(req)
}

async function onSubmit() {
  if (requests.value.length > 0) {
    const req = requests.value[0]
    if (req) {
      await desktopClient.submitCertPassword(req.RequestID, password.value)
    }
    password.value = ''
    requests.value.shift()
  }
}

async function onCancel() {
  if (requests.value.length > 0) {
    const req = requests.value[0]
    if (req) {
      await desktopClient.cancelCertPassword(req.RequestID)
    }
    password.value = ''
    requests.value.shift()
  }
}

onMounted(() => {
  unsubscribe.value = onWailsEvent<CertPasswordRequest>('request-cert-password', handleRequest)
})

onUnmounted(() => {
  unsubscribe.value?.()
  unsubscribe.value = null
})
</script>
