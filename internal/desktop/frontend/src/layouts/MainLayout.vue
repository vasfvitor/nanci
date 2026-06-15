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
          class="q-mr-sm"
          style="--wails-draggable: no-drag"
          @click="toggleLeftDrawer"
        />
        <div class="text-weight-bold">Nanci</div>
        <q-space />
        <q-btn
          dense
          flat
          icon="terminal"
          title="Console"
          style="--wails-draggable: no-drag"
          @click="toggleConsole"
        />
        <q-btn dense flat icon="minimize" style="--wails-draggable: no-drag" @click="minimise" />
        <q-btn
          dense
          flat
          icon="crop_square"
          style="--wails-draggable: no-drag"
          @click="toggleMaximise"
        />
        <q-btn dense flat icon="close" style="--wails-draggable: no-drag" @click="closeApp" />
      </q-bar>
    </q-header>

    <q-drawer v-model="leftDrawerOpen" show-if-above bordered>
      <q-list class="q-py-md">
        <q-item v-ripple clickable to="/" exact dense active-class="text-primary">
          <q-item-section avatar>
            <q-icon name="business" size="sm" />
          </q-item-section>
          <q-item-section>
            <q-item-label class="text-weight-medium">Empresas</q-item-label>
          </q-item-section>
        </q-item>

        <q-item v-ripple clickable to="/documents" exact dense active-class="text-primary">
          <q-item-section avatar>
            <q-icon name="description" size="sm" />
          </q-item-section>
          <q-item-section>
            <q-item-label class="text-weight-medium">Documentos</q-item-label>
          </q-item-section>
        </q-item>

        <q-item v-ripple clickable to="/credentials" exact dense active-class="text-primary">
          <q-item-section avatar>
            <q-icon name="vpn_key" size="sm" />
          </q-item-section>
          <q-item-section>
            <q-item-label class="text-weight-medium">Credenciais</q-item-label>
          </q-item-section>
        </q-item>

        <q-separator class="q-my-md" />

        <q-item v-ripple clickable to="/query" exact dense active-class="text-primary">
          <q-item-section avatar>
            <q-icon name="api" size="sm" />
          </q-item-section>
          <q-item-section>
            <q-item-label class="text-weight-medium">Consulta Direta API</q-item-label>
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
          <q-toggle
            v-model="debugEnabled"
            color="secondary"
            label="Debug"
            left-label
            dense
            @update:model-value="onDebugToggle"
            class="q-mr-sm"
          />
          <q-btn flat round dense icon="content_copy" title="Copiar logs" @click="copyLogs">
            <q-tooltip>Copiar logs</q-tooltip>
          </q-btn>
          <q-btn flat round dense icon="delete" title="Limpar logs" @click="clearLogs">
            <q-tooltip>Limpar logs</q-tooltip>
          </q-btn>
          <q-btn flat round dense icon="close" @click="consoleOpen = false" />
        </q-toolbar>
        <q-scroll-area ref="logScrollArea" class="col q-pa-sm bg-black">
          <div v-if="parsedLogs.length === 0" class="text-grey-6 text-center q-mt-md">Nenhum log disponível</div>
          <div
            v-for="(log, idx) in parsedLogs"
            :key="idx"
            class="q-mb-xs"
            style="font-size: 13px; font-family: 'Fira Code', monospace; word-break: break-all;"
          >
            <span class="text-grey-6 q-mr-sm">[{{ log.time }}]</span>
            <span :class="getLevelColor(log.level)" class="text-weight-bold q-mr-sm">{{ log.level }}</span>
            <span class="text-white q-mr-sm">{{ log.msg }}</span>
            <span v-if="log.extras" class="text-grey-5">{{ log.extras }}</span>
          </div>
        </q-scroll-area>
      </div>
    </q-drawer>

    <q-page-container class="bg-transparent">
      <router-view />
    </q-page-container>
  </q-layout>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { ref, onMounted, nextTick } from 'vue'
import { useQuasar } from 'quasar'
import { EventsOn, WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'
import { useAppStore } from '../stores/app'

const $q = useQuasar()
const appStore = useAppStore()
const leftDrawerOpen = ref(false)
const consoleOpen = ref(false)
const { debugEnabled } = storeToRefs(appStore)
const logs = ref<string[]>([])
const parsedLogs = ref<Array<{time: string, level: string, msg: string, extras: string}>>([])
const logScrollArea = ref<HTMLElement | null>(null)

function getLevelColor(level: string) {
  const lvl = level.toUpperCase()
  if (lvl.includes('INFO')) return 'text-blue-4'
  if (lvl.includes('WARN')) return 'text-orange-4'
  if (lvl.includes('ERROR')) return 'text-red-4'
  if (lvl.includes('DEBUG')) return 'text-purple-4'
  return 'text-green-4'
}

function toggleLeftDrawer() {
  leftDrawerOpen.value = !leftDrawerOpen.value
}

function toggleConsole() {
  consoleOpen.value = !consoleOpen.value
}

function onDebugToggle(val: boolean) {
  // Use ts-ignore just in case Wails bindings haven't caught up, it will be mapped globally on window.go.main.App
  // @ts-ignore
  if (window.go && window.go.main && window.go.main.App && window.go.main.App.ToggleDebug) {
    // @ts-ignore
    window.go.main.App.ToggleDebug(val)
  }
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
  parsedLogs.value = []
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
    
    // Parse the slog TextHandler format: time=... level=... msg="..." extras
    const line = msg.trim()
    let timeStr = ''
    let levelStr = 'INFO'
    let msgStr = line
    let extrasStr = ''

    const timeMatch = line.match(/time=([^\s]+)/)
    const levelMatch = line.match(/level=([^\s]+)/)
    const msgMatch = line.match(/msg="([^"]+)"/)
    
    if (timeMatch && timeMatch[1]) timeStr = timeMatch[1].substring(11, 19) // Extract HH:mm:ss from ISO string
    if (levelMatch && levelMatch[1]) levelStr = levelMatch[1]
    
    if (msgMatch && msgMatch[1]) {
      msgStr = msgMatch[1]
      // Everything after the message is considered 'extras'
      const afterMsg = line.substring(msgMatch.index! + msgMatch[0].length)
      extrasStr = afterMsg.trim()
    }

    parsedLogs.value.push({
      time: timeStr || new Date().toLocaleTimeString(),
      level: levelStr,
      msg: msgStr,
      extras: extrasStr
    })

    if (parsedLogs.value.length > 500) {
      parsedLogs.value.splice(0, parsedLogs.value.length - 500)
      logs.value.splice(0, logs.value.length - 500)
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
