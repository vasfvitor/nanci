<template>
  <q-dialog v-model="isOpen" persistent>
    <q-card style="min-width: 350px">
      <q-card-section>
        <div class="text-h6">Senha do Certificado</div>
        <div class="text-caption text-grey-8">Empresa: {{ requestData?.CompanyName }}</div>
        <div class="text-caption text-grey-8">CNPJ consultado: {{ requestData?.TargetCNPJ }}</div>
        <div class="text-caption text-grey-8">Credencial: {{ requestData?.CredentialLabel }}</div>
        <div class="text-caption text-grey-8">Arquivo: {{ requestData?.CertPath }}</div>
      </q-card-section>

      <q-card-section class="q-pt-none">
        <q-input dense v-model="password" autofocus type="password" @keyup.enter="onSubmit" />
      </q-card-section>

      <q-card-actions align="right" class="text-primary">
        <q-btn flat label="Cancelar" @click="onCancel" />
        <q-btn flat label="Confirmar" @click="onSubmit" />
      </q-card-actions>
    </q-card>
  </q-dialog>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { SubmitCertPassword, CancelCertPassword } from '../../wailsjs/go/main/App'

const isOpen = ref(false)
const password = ref('')
const requestData = ref<any>(null)

function handleRequest(req: any) {
  requestData.value = req
  password.value = ''
  isOpen.value = true
}

async function onSubmit() {
  await SubmitCertPassword(password.value)
  isOpen.value = false
}

async function onCancel() {
  await CancelCertPassword()
  isOpen.value = false
}

onMounted(() => {
  EventsOn('request-cert-password', handleRequest)
})

onUnmounted(() => {
  EventsOff('request-cert-password')
})
</script>
