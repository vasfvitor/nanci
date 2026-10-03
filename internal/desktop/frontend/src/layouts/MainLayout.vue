<template>
  <!-- container: the page scrolls inside the layout, not the window. The
       root scrollbar of the window swallows mouse events, so the Wails
       runtime never saw the right edge and the frameless window could not be
       resized sideways. -->
  <q-layout view="hHh Lpr lFf" container style="height: 100vh">
    <q-header bordered class="bg-app-header text-white">
      <AppTitleBar
        @toggle-menu="leftDrawerOpen = !leftDrawerOpen"
        @toggle-console="consoleOpen = !consoleOpen"
      />
    </q-header>

    <AppLeftDrawer v-model="leftDrawerOpen" />

    <AppConsoleDrawer v-model="consoleOpen" />

    <q-page-container class="bg-transparent">
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { ref, onMounted, onUnmounted } from 'vue'
import { useQuasar } from 'quasar'
import { storeToRefs } from 'pinia'
import { onWailsEvent, type Unsubscribe } from '@/platform/wails/events'
import { useNotify } from '@/composables/useNotify'
import { useConsoleStore } from '@/stores/console'
import { useWorkspaceStore } from '@/stores/workspace'
import AppTitleBar from '../components/AppTitleBar.vue'
import AppLeftDrawer from '../components/AppLeftDrawer.vue'
import AppConsoleDrawer from '../components/AppConsoleDrawer.vue'

const $q = useQuasar()
const consoleStore = useConsoleStore()
const workspace = useWorkspaceStore()
const { notifyError } = useNotify()
const leftDrawerOpen = ref(false)
const { consoleOpen } = storeToRefs(consoleStore)
const unsubscribers: Unsubscribe[] = []

onMounted(() => {
  // The layout loads the company list once for every page; a page that
  // mounted first may have started the load already.
  workspace.ensureCompanies().catch((e: unknown) => notifyError('Erro ao carregar empresas', e))
  unsubscribers.push(
    onWailsEvent<string>('notify-success', (msg) => {
      $q.notify({ type: 'positive', message: msg })
    }),
    onWailsEvent<string>('notify-error', (msg) => {
      $q.notify({ type: 'negative', message: msg })
    })
  )
})

onUnmounted(() => {
  for (const unsubscribe of unsubscribers.splice(0)) {
    unsubscribe()
  }
})
</script>
