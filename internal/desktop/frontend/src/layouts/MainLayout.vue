<template>
  <q-layout view="hHh Lpr lFf">
    <q-header bordered class="bg-transparent">
      <q-bar style="--wails-draggable: drag">
        <q-btn
          dense
          flat
          round
          icon="menu"
          aria-label="Menu"
          @click="toggleLeftDrawer"
          class="q-mr-sm"
          style="--wails-draggable: no-drag"
        />
        <div class="text-weight-bold">Nanci Desktop</div>
        <q-space />
        <q-btn
          dense
          flat
          icon="terminal"
          @click="toggleConsole"
          title="Console"
          style="--wails-draggable: no-drag"
        />
        <q-btn dense flat icon="minimize" @click="minimise" style="--wails-draggable: no-drag" />
        <q-btn
          dense
          flat
          icon="crop_square"
          @click="toggleMaximise"
          style="--wails-draggable: no-drag"
        />
        <q-btn dense flat icon="close" @click="closeApp" style="--wails-draggable: no-drag" />
      </q-bar>
    </q-header>

    <q-drawer v-model="leftDrawerOpen" show-if-above bordered>
      <q-list class="q-py-md">
        <q-item clickable v-ripple to="/" exact dense active-class="text-primary">
          <q-item-section avatar>
            <q-icon name="business" size="sm" />
          </q-item-section>
          <q-item-section>
            <q-item-label class="text-weight-medium">Empresas</q-item-label>
          </q-item-section>
        </q-item>

        <q-item clickable v-ripple to="/documents" exact dense active-class="text-primary">
          <q-item-section avatar>
            <q-icon name="description" size="sm" />
          </q-item-section>
          <q-item-section>
            <q-item-label class="text-weight-medium">Documentos</q-item-label>
          </q-item-section>
        </q-item>

        <q-item clickable v-ripple to="/credentials" exact dense active-class="text-primary">
          <q-item-section avatar>
            <q-icon name="vpn_key" size="sm" />
          </q-item-section>
          <q-item-section>
            <q-item-label class="text-weight-medium">Credenciais</q-item-label>
          </q-item-section>
        </q-item>
      </q-list>
    </q-drawer>

    <q-drawer
      v-model="consoleOpen"
      side="right"
      bordered
      :width="500"
      overlay
      class="bg-grey-10 text-white"
    >
      <div class="column full-height">
        <q-toolbar class="bg-grey-9 text-white dense">
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
          <pre
            style="
              white-space: pre-wrap;
              font-size: 12px;
              font-family: monospace;
              margin: 0;
              word-break: break-all;
            "
            >{{ logs.join('') }}</pre
          >
        </q-scroll-area>
      </div>
    </q-drawer>

    <q-page-container class="bg-transparent">
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { ref, onMounted, nextTick } from 'vue'
import { useQuasar } from 'quasar'
import { EventsOn, WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'

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

function minimise() {
  WindowMinimise()
}

function toggleMaximise() {
  WindowToggleMaximise()
}

function closeApp() {
  Quit()
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
