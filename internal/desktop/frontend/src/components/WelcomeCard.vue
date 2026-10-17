<template>
  <q-card flat bordered class="welcome-card">
    <q-card-section>
      <div class="text-h6">Bem-vindo ao Nanci</div>
      <div class="text-body2 text-app-muted">Três passos até as primeiras notas.</div>
    </q-card-section>

    <q-list>
      <q-item v-for="step in steps" :key="step.title">
        <q-item-section avatar>
          <q-icon :name="step.icon" color="primary" />
        </q-item-section>
        <q-item-section>
          <q-item-label>{{ step.title }}</q-item-label>
          <q-item-label caption>{{ step.caption }}</q-item-label>
        </q-item-section>
      </q-item>
    </q-list>

    <q-card-actions class="q-pa-md">
      <q-btn
        unelevated
        color="primary"
        icon="add"
        label="Cadastrar primeira empresa"
        @click="emit('add')"
      />
    </q-card-actions>
  </q-card>
</template>

<script setup lang="ts">
// WelcomeCard stands in for the company table while no company exists: the
// three steps to the first notas and the button that opens the add dialog.
const emit = defineEmits<{
  add: []
}>()

const steps = [
  { icon: 'key', title: 'Certificado A1', caption: 'O arquivo .pfx ou .p12 e a senha.' },
  {
    icon: 'business',
    title: 'Empresa',
    caption: 'CNPJ, ambiente e o histórico inicial a importar.',
  },
  { icon: 'sync', title: 'Sincronizar', caption: 'Buscar as notas na Receita e na SEFAZ.' },
]
</script>

<style scoped>
.welcome-card {
  max-width: 560px;
}
</style>
