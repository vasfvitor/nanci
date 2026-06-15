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
          <q-select
            v-model="logFilterLevel"
            :options="logLevelOptions"
            emit-value
            map-options
            label="Nível"
            outlined
            dense
            options-dense
            class="q-mr-sm"
            dark
            bg-color="grey-8"
            style="min-width: 150px"
            @update:model-value="onLogLevelChange"
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
          <div v-if="filteredLogEntries.length === 0" class="text-grey-6 text-center q-mt-md">Nenhum log disponível</div>
          <div
            v-for="(log, idx) in filteredLogEntries"
            :key="idx"
            class="q-mb-xs"
            style="font-size: 13px; font-family: 'Fira Code', monospace; word-break: break-all;"
          >
            <span class="text-grey-6 q-mr-sm">[{{ formatTime(log.time) }}]</span>
            <span :class="getLevelColor(log.level)" class="text-weight-bold q-mr-sm">{{ log.level }}</span>
            <span class="text-white q-mr-sm">{{ log.msg }}</span>
            <span v-if="log.attrs" class="text-grey-5">{{ log.attrs }}</span>
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
import { ref, onMounted, nextTick, watch } from 'vue'
import { useQuasar } from 'quasar'
import { EventsOn, WindowMinimise, WindowToggleMaximise, Quit } from '../../wailsjs/runtime/runtime'
import { useAppStore } from '../stores/app'

const $q = useQuasar()
const appStore = useAppStore()
const leftDrawerOpen = ref(false)
const { consoleOpen, logFilterLevel, filteredLogEntries, filteredLogsText } = storeToRefs(appStore)
const logScrollArea = ref<HTMLElement | null>(null)
const logLevelOptions = [
  { label: 'Info', value: 'info' },
  { label: 'Warn', value: 'warn' },
  { label: 'Debug', value: 'debug' },
  { label: 'Trace', value: 'trace' },
]

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

async function onLogLevelChange(level: 'info' | 'warn' | 'debug' | 'trace') {
  await appStore.setLogFilter(level)
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
  appStore.clearLogs()
}

async function copyLogs() {
  try {
    await navigator.clipboard.writeText(filteredLogsText.value)
    $q.notify({ type: 'positive', message: 'Logs copiados para a área de transferência.' })
  } catch {
    $q.notify({ type: 'negative', message: 'Erro ao copiar logs.' })
  }
}

function formatTime(value: string) {
  const date = new Date(value)
  if (Number.isNaN(date.getTime())) return value
  return date.toLocaleTimeString('pt-BR')
}

function scrollLogsToBottom() {
  if (!consoleOpen.value) return
  nextTick(() => {
    if (logScrollArea.value) {
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      ;(logScrollArea.value as any).setScrollPercentage('vertical', 1.0)
    }
  })
}

onMounted(() => {
  EventsOn('notify-success', (msg: string) => {
    $q.notify({ type: 'positive', message: msg })
  })

  EventsOn('notify-error', (msg: string) => {
    $q.notify({ type: 'negative', message: msg })
  })
})

watch(() => filteredLogEntries.value.length, scrollLogsToBottom)
watch(consoleOpen, (open) => {
  if (open) {
    scrollLogsToBottom()
  }
})
</script>
