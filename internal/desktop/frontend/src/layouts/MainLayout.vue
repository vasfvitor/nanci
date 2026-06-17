<template>
  <q-layout view="hHh Lpr lFf">
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
import { useConsoleStore } from '@/stores/console'
import AppTitleBar from '../components/AppTitleBar.vue'
import AppLeftDrawer from '../components/AppLeftDrawer.vue'
import AppConsoleDrawer from '../components/AppConsoleDrawer.vue'

const $q = useQuasar()
const consoleStore = useConsoleStore()
const leftDrawerOpen = ref(false)
const { consoleOpen } = storeToRefs(consoleStore)
const unsubscribers: Unsubscribe[] = []

onMounted(() => {
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
