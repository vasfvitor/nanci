<template>
  <q-drawer
    v-model="model"
    side="right"
    bordered
    :width="500"
    overlay
  >
    <div class="column full-height">
      <q-toolbar class="dense">
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
          class="q-mr-sm log-level-select"
          @update:model-value="onLogLevelChange"
        />
        <q-btn flat round dense icon="content_copy" aria-label="Copiar logs" @click="copyLogs">
          <q-tooltip>Copiar logs</q-tooltip>
        </q-btn>
        <q-btn flat round dense icon="delete" aria-label="Limpar logs" @click="clearLogs">
          <q-tooltip>Limpar logs</q-tooltip>
        </q-btn>
        <q-btn flat round dense icon="close" aria-label="Fechar console" @click="model = false" />
      </q-toolbar>
      <q-scroll-area ref="logScrollArea" class="col q-pa-sm app-console">
        <div v-if="filteredLogEntries.length === 0" class="text-grey-6 text-center q-mt-md">Nenhum log disponível</div>
        <div
          v-for="(log, idx) in filteredLogEntries"
          :key="idx"
          class="q-mb-xs log-line"
        >
          <span class="text-grey-6 q-mr-sm">[{{ formatTime(log.time) }}]</span>
          <span :class="getLevelColor(log.level)" class="text-weight-bold q-mr-sm">{{ log.level }}</span>
          <span class="log-message q-mr-sm">{{ log.msg }}</span>
          <span v-if="log.attrs" class="text-grey-5">{{ log.attrs }}</span>
        </div>
      </q-scroll-area>
    </div>
  </q-drawer>
</template>

<script setup lang="ts">
import { storeToRefs } from 'pinia'
import { ref, nextTick, watch } from 'vue'
import { useQuasar } from 'quasar'
import { useConsoleStore } from '@/stores/console'

interface QScrollAreaRef {
  setScrollPercentage: (axis: 'vertical' | 'horizontal', percentage: number) => void
}

const model = defineModel<boolean>({ default: false })

const $q = useQuasar()
const consoleStore = useConsoleStore()
const { logFilterLevel, filteredLogEntries, filteredLogsText } = storeToRefs(consoleStore)
const logScrollArea = ref<QScrollAreaRef | null>(null)

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

async function onLogLevelChange(level: 'info' | 'warn' | 'debug' | 'trace') {
  await consoleStore.setLogFilter(level)
}

function clearLogs() {
  consoleStore.clearLogs()
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
  if (!model.value) return
  nextTick(() => {
    logScrollArea.value?.setScrollPercentage('vertical', 1.0)
  })
}

watch(() => filteredLogEntries.value.length, scrollLogsToBottom)
watch(model, (open) => {
  if (open) {
    scrollLogsToBottom()
  }
})
</script>

<style scoped>
.log-level-select {
  min-width: 150px;
}

.log-line {
  font-size: 13px;
  font-family: 'Fira Code', monospace;
  word-break: break-all;
}

.log-message {
  color: var(--app-console-text);
}
</style>
