<template>
  <router-view />
  <PasswordPromptDialog />
</template>

<script setup lang="ts">
import { useQuasar } from 'quasar'
import PasswordPromptDialog from './components/PasswordPromptDialog.vue'

const $q = useQuasar()
$q.dark.set('auto')
</script>
