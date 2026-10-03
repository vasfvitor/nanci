<template>
  <!-- container: the page scrolls inside the layout, so its scrollbar is an
       element inside the viewport and the resize edge below can cover it. -->
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

  <!-- The Wails runtime resizes the frameless window from the 6px inside each
       edge, where it watches the mouse. A mousedown there on a native
       scrollbar or on a drawer link makes the webview capture the mouse, and
       the resize the runtime asks for then fails. These strips keep the
       pointer on a plain element along both side edges and draw a 1px line
       to show where the window ends. -->
  <div class="app-resize-edge app-resize-edge--left" aria-hidden="true" />
  <div class="app-resize-edge app-resize-edge--right" aria-hidden="true" />
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

<style scoped>
.app-resize-edge {
  position: fixed;
  top: 0;
  bottom: 0;
  width: 6px;
  z-index: 3000;
}

.app-resize-edge--left {
  left: 0;
  box-shadow: inset 1px 0 0 var(--q-separator-color);
}

.app-resize-edge--right {
  right: 0;
  box-shadow: inset -1px 0 0 var(--q-separator-color);
}
</style>
