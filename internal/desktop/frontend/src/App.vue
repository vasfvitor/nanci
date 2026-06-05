<template>
  <router-view />
  <PasswordPromptDialog />
</template>

<script setup lang="ts">
import PasswordPromptDialog from './components/PasswordPromptDialog.vue'
</script>

<style>
/* Global styles if needed */
body {
  font-family: 'Roboto', '-apple-system', 'Helvetica Neue', Helvetica, Arial, sans-serif;
}
</style>
