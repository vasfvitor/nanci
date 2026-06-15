<script setup lang="ts">
import { ref } from 'vue'
import AppLeftDrawer from './AppLeftDrawer.vue'

const open = ref(true)
</script>

<template>
  <Story title="Layout/AppLeftDrawer">
    <Variant title="Light Theme">
      <div class="body--light preview-container">
        <q-layout view="hHh Lpr lFf" container class="fit" style="height: 400px">
          <AppLeftDrawer v-model="open" />
          <q-page-container>
            <q-page class="q-pa-md">
              <q-btn @click="open = !open" color="primary">Toggle Drawer</q-btn>
            </q-page>
          </q-page-container>
        </q-layout>
      </div>
    </Variant>

    <Variant title="Dark Theme">
      <div class="body--dark preview-container">
        <q-layout view="hHh Lpr lFf" container class="fit" style="height: 400px">
          <AppLeftDrawer v-model="open" />
          <q-page-container>
            <q-page class="q-pa-md">
              <q-btn @click="open = !open" color="primary">Toggle Drawer</q-btn>
            </q-page>
          </q-page-container>
        </q-layout>
      </div>
    </Variant>
  </Story>
</template>

<style scoped>
.preview-container {
  height: 450px;
  position: relative;
  background: var(--app-bg);
  color: var(--app-text);
}
</style>
