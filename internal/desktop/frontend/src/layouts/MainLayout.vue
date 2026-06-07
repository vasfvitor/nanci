<template>
  <q-layout view="lHh Lpr lFf">
    <q-header elevated>
      <q-toolbar>
        <q-btn flat dense round icon="menu" aria-label="Menu" @click="toggleLeftDrawer" />
        <q-toolbar-title> Nanci Desktop </q-toolbar-title>
        <q-btn flat dense icon="terminal" label="Console" @click="toggleConsole" />
      </q-toolbar>
    </q-header>

    <q-drawer v-model="leftDrawerOpen" show-if-above bordered>
      <q-list>
        <q-item-label header> Menu </q-item-label>

        <q-item clickable v-ripple to="/" exact>
          <q-item-section avatar>
            <q-icon name="business" />
          </q-item-section>
          <q-item-section>
            <q-item-label>Empresas</q-item-label>
          </q-item-section>
        </q-item>

        <q-item clickable v-ripple to="/documents" exact>
          <q-item-section avatar>
            <q-icon name="description" />
          </q-item-section>
          <q-item-section>
            <q-item-label>Documentos</q-item-label>
          </q-item-section>
        </q-item>

        <q-item clickable v-ripple to="/credentials" exact>
          <q-item-section avatar>
            <q-icon name="vpn_key" />
          </q-item-section>
          <q-item-section>
            <q-item-label>Credenciais</q-item-label>
          </q-item-section>
        </q-item>
      </q-list>
    </q-drawer>

    <q-drawer v-model="consoleOpen" side="right" bordered :width="500" overlay class="bg-grey-10 text-white">
      <div class="column full-height">
        <q-toolbar class="bg-grey-9 text-white">
          <q-toolbar-title class="text-subtitle1">Console</q-toolbar-title>
          <q-btn flat round dense icon="content_copy" @click="copyLogs" title="Copiar logs">
            <q-tooltip>Copiar logs</q-tooltip>
          </q-btn>
          <q-btn flat round dense icon="delete" @click="clearLogs" title="Limpar logs">
            <q-tooltip>Limpar logs</q-tooltip>
          </q-btn>
          <q-btn flat round dense icon="close" @click="consoleOpen = false" />
        </q-toolbar>
        <q-scroll-area class="col q-pa-sm" ref="logScrollArea">
          <pre style="white-space: pre-wrap; font-size: 12px; font-family: monospace; margin: 0; word-break: break-all;">{{ logs.join('') }}</pre>
        </q-scroll-area>
      </div>
    </q-drawer>

    <q-page-container>
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useQuasar } from 'quasar'
import { EventsOn } from '../../wailsjs/runtime/runtime'

const $q = useQuasar()
const leftDrawerOpen = ref(false)
const consoleOpen = ref(false)
const logs = ref<string[]>([])
const logScrollArea = ref<HTMLElement | null>(null)

function toggleLeftDrawer() {
  leftDrawerOpen.value = !leftDrawerOpen.value
}

function toggleConsole() {
  consoleOpen.value = !consoleOpen.value
}

function clearLogs() {
  logs.value = []
}

async function copyLogs() {
  try {
    await navigator.clipboard.writeText(logs.value.join(''))
    $q.notify({ type: 'positive', message: 'Logs copiados para a área de transferência.' })
  } catch {
    $q.notify({ type: 'negative', message: 'Erro ao copiar logs.' })
  }
}

onMounted(() => {
  EventsOn('notify-success', (msg: string) => {
    $q.notify({ type: 'positive', message: msg })
  })
  
  EventsOn('notify-error', (msg: string) => {
    $q.notify({ type: 'negative', message: msg })
  })

  EventsOn('backend-log', (msg: string) => {
    logs.value.push(msg)
    // Limit log size to prevent memory issues (e.g. max 1000 lines/chunks)
    if (logs.value.length > 2000) {
      logs.value.splice(0, logs.value.length - 2000)
    }
    
    // Scroll to bottom
    if (consoleOpen.value) {
      nextTick(() => {
        if (logScrollArea.value) {
          // eslint-disable-next-line @typescript-eslint/no-explicit-any
          ;(logScrollArea.value as any).setScrollPercentage('vertical', 1.0)
        }
      })
    }
  })
})
</script>
