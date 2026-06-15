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
import { ref, onMounted } from 'vue'
import { useQuasar } from 'quasar'
import { storeToRefs } from 'pinia'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { useAppStore } from '../stores/app'
import AppTitleBar from '../components/AppTitleBar.vue'
import AppLeftDrawer from '../components/AppLeftDrawer.vue'
import AppConsoleDrawer from '../components/AppConsoleDrawer.vue'

const $q = useQuasar()
const appStore = useAppStore()
const leftDrawerOpen = ref(false)
const { consoleOpen } = storeToRefs(appStore)

onMounted(() => {
  EventsOn('notify-success', (msg: string) => {
    $q.notify({ type: 'positive', message: msg })
  })

  EventsOn('notify-error', (msg: string) => {
    $q.notify({ type: 'negative', message: msg })
  })
})
</script>
