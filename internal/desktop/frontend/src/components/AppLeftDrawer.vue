<template>
  <q-drawer v-model="model" show-if-above bordered>
    <q-list class="q-py-md">
      <q-item v-ripple clickable to="/" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="business" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Empresas</q-item-label>
        </q-item-section>
      </q-item>

      <q-item v-ripple clickable to="/documents" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="description" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Documentos</q-item-label>
        </q-item-section>
      </q-item>

      <q-item v-ripple clickable to="/credentials" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="vpn_key" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Credenciais</q-item-label>
        </q-item-section>
      </q-item>

      <q-separator class="q-my-md" />

      <q-item v-ripple clickable to="/query" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="api" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Consulta Direta API</q-item-label>
        </q-item-section>
      </q-item>

      <q-separator class="q-my-md" />

      <q-item v-ripple clickable to="/settings" exact dense active-class="text-primary">
        <q-item-section avatar>
          <q-icon name="settings" size="sm" />
        </q-item-section>
        <q-item-section>
          <q-item-label class="text-weight-medium">Configurações</q-item-label>
        </q-item-section>
      </q-item>
    </q-list>
  </q-drawer>
</template>

<script setup lang="ts">
const model = defineModel<boolean>({ default: false })
</script>
