import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { onWailsEvent, type Unsubscribe } from '@/platform/wails/events'

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

export function coerceLogEntry(payload: unknown): LogEntry | null {
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

export const useConsoleStore = defineStore('console', () => {
  const consoleOpen = ref(false)
  const savedLevel = (localStorage.getItem('nanci:logLevel') || 'info') as LogFilterLevel
  const logFilterLevel = ref<LogFilterLevel>(savedLevel)
  const logEntries = ref<LogEntry[]>([])
  const logListenersInitialised = ref(false)
  const logUnsubscribe = ref<Unsubscribe | null>(null)

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
    const previousLevel = logFilterLevel.value
    logFilterLevel.value = level
    localStorage.setItem('nanci:logLevel', level)

    try {
      await desktopClient.setLogLevel(level)
    } catch (error) {
      logFilterLevel.value = previousLevel
      localStorage.setItem('nanci:logLevel', previousLevel)
      throw error
    }
  }

  async function syncInitialLogLevel() {
    await desktopClient.setLogLevel(logFilterLevel.value)
  }

  function initLogListeners() {
    if (logListenersInitialised.value) return
    logListenersInitialised.value = true

    logUnsubscribe.value = onWailsEvent<unknown>('backend-log', (payload) => {
      const entry = coerceLogEntry(payload)
      if (!entry) return
      pushLogEntry(entry)
    })
  }

  function disposeLogListeners() {
    logUnsubscribe.value?.()
    logUnsubscribe.value = null
    logListenersInitialised.value = false
  }

  return {
    consoleOpen,
    logFilterLevel,
    logEntries,
    logListenersInitialised,
    logUnsubscribe,
    debugEnabled,
    filteredLogEntries,
    filteredLogsText,
    pushLogEntry,
    clearLogs,
    initLogListeners,
    disposeLogListeners,
    setLogFilter,
    syncInitialLogLevel,
  }
})
