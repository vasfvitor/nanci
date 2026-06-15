<script setup lang="ts">
import AppTitleBar from './AppTitleBar.vue'
</script>

<template>
  <Story title="Layout/AppTitleBar">
    <Variant title="Light Theme">
      <div class="body--light preview-container">
        <q-layout view="hHh Lpr lFf">
          <q-header bordered class="bg-app-header text-white">
            <AppTitleBar />
          </q-header>
          <q-page-container class="bg-transparent">
            <q-page class="q-pa-md">
              Window Content Placeholder
            </q-page>
          </q-page-container>
        </q-layout>
      </div>
    </Variant>

    <Variant title="Dark Theme">
      <div class="body--dark preview-container">
        <q-layout view="hHh Lpr lFf">
          <q-header bordered class="bg-app-header text-white">
            <AppTitleBar />
          </q-header>
          <q-page-container class="bg-transparent">
            <q-page class="q-pa-md">
              Window Content Placeholder
            </q-page>
          </q-page-container>
        </q-layout>
      </div>
    </Variant>
  </Story>
</template>

<style scoped>
.preview-container {
  height: 300px;
  position: relative;
  border: 1px solid var(--app-border);
  background: var(--app-bg);
  color: var(--app-text);
  overflow: hidden;
}
</style>
