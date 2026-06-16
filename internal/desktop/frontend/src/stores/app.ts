import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { EventsOn } from '../../wailsjs/runtime/runtime'
import { SetLogLevel } from '../../wailsjs/go/main/App'
import { nfse } from '../../wailsjs/go/models'

export type LogFilterLevel = 'info' | 'warn' | 'debug' | 'trace'

export type LogEntry = {
  time: string
  level: string
  msg: string
  attrs: string
  raw: string
}

const maxLogEntries = 1000

const severityByLevel: Record<string, number> = {
  TRACE: 0,
  DEBUG: 1,
  INFO: 2,
  WARN: 3,
  WARNING: 3,
  ERROR: 4,
}

const minSeverityByFilter: Record<LogFilterLevel, number> = {
  trace: 0,
  debug: 1,
  info: 2,
  warn: 3,
}

function normalizeLogLevel(level: unknown): string {
  if (typeof level !== 'string' || level.trim() === '') return 'INFO'
  return level.trim().toUpperCase()
}

function coerceLogEntry(payload: unknown): LogEntry | null {
  if (!payload || typeof payload !== 'object') return null
  const rawPayload = payload as Record<string, unknown>
  const level = normalizeLogLevel(rawPayload['level'])
  const raw = typeof rawPayload['raw'] === 'string' ? rawPayload['raw'] : ''

  return {
    time:
      typeof rawPayload['time'] === 'string' && rawPayload['time']
        ? rawPayload['time']
        : new Date().toISOString(),
    level,
    msg: typeof rawPayload['msg'] === 'string' ? rawPayload['msg'] : raw,
    attrs: typeof rawPayload['attrs'] === 'string' ? rawPayload['attrs'] : '',
    raw: raw || String(rawPayload['msg'] || ''),
  }
}

function formatLogEntry(entry: LogEntry) {
  return `[${entry.time}] ${entry.level} ${entry.msg}${entry.attrs ? ` ${entry.attrs}` : ''}`
}

export const useAppStore = defineStore('app', () => {
  const activeCompanyId = ref<string | null>(null)
  const consoleOpen = ref(false)
  const logFilterLevel = ref<LogFilterLevel>('info')
  const logEntries = ref<LogEntry[]>([])
  const logListenersInitialised = ref(false)

  const queryForm = ref({
    cnpj: '',
    chave: '',
  })
  const queryResult = ref('')
  const queryType = ref('nfse')

  const documentsFilter = ref({
    CNPJ: '',
    Competence: '',
    Direction: '',
  })
  const documentsList = ref<nfse.CompanyDocument[]>([])

  const debugEnabled = computed(
    () => logFilterLevel.value === 'debug' || logFilterLevel.value === 'trace'
  )

  const filteredLogEntries = computed(() => {
    const minSeverity = minSeverityByFilter[logFilterLevel.value]
    return logEntries.value.filter((entry) => {
      const severity = severityByLevel[normalizeLogLevel(entry.level)]
      if (severity === undefined) {
        return logFilterLevel.value !== 'warn'
      }
      return severity >= minSeverity
    })
  })

  const filteredLogsText = computed(() => filteredLogEntries.value.map(formatLogEntry).join('\n'))

  function pushLogEntry(entry: LogEntry) {
    logEntries.value.push(entry)
    if (logEntries.value.length > maxLogEntries) {
      logEntries.value.splice(0, logEntries.value.length - maxLogEntries)
    }
  }

  function clearLogs() {
    logEntries.value = []
  }

  async function setLogFilter(level: LogFilterLevel) {
    logFilterLevel.value = level
    await SetLogLevel(level)
  }

  function initLogListeners() {
    if (logListenersInitialised.value) return
    logListenersInitialised.value = true

    EventsOn('backend-log', (payload: unknown) => {
      const entry = coerceLogEntry(payload)
      if (!entry) return
      pushLogEntry(entry)
    })
  }

  return {
    activeCompanyId,
    consoleOpen,
    logFilterLevel,
    logEntries,
    debugEnabled,
    filteredLogEntries,
    filteredLogsText,
    queryForm,
    queryResult,
    queryType,
    documentsFilter,
    documentsList,
    clearLogs,
    initLogListeners,
    setLogFilter,
  }
})
