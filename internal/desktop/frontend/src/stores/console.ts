import { computed, ref } from 'vue'
import { defineStore } from 'pinia'
import { desktopClient } from '@/platform/wails/client'
import { onWailsEvent, type DesktopEvents, type Unsubscribe } from '@/platform/wails/events'
import { readStorage, writeStorage } from '@/utils/storage'

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

const logLevelKey = 'nanci:logLevel'

// readLogFilterLevel returns the saved filter level, or 'info' when none is
// saved, the value is not a known level or the storage cannot be read.
function readLogFilterLevel(): LogFilterLevel {
  const saved = readStorage(logLevelKey)
  if (saved !== null && saved in minSeverityByFilter) {
    return saved as LogFilterLevel
  }
  return 'info'
}

function normalizeLogLevel(level: string): string {
  if (level.trim() === '') return 'INFO'
  return level.trim().toUpperCase()
}

// toLogEntry fills the fields a backend-log record may leave empty: no level
// reads as INFO, no time takes the current one and no raw line falls back to
// the message.
function toLogEntry(payload: DesktopEvents['backend-log']): LogEntry {
  return {
    time: payload.time || new Date().toISOString(),
    level: normalizeLogLevel(payload.level),
    msg: payload.msg,
    attrs: payload.attrs,
    raw: payload.raw || payload.msg,
  }
}

function formatLogEntry(entry: LogEntry) {
  return `[${entry.time}] ${entry.level} ${entry.msg}${entry.attrs ? ` ${entry.attrs}` : ''}`
}

export const useConsoleStore = defineStore('console', () => {
  const consoleOpen = ref(false)
  const logFilterLevel = ref<LogFilterLevel>(readLogFilterLevel())
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
    writeStorage(logLevelKey, level)

    try {
      await desktopClient.setLogLevel(level)
    } catch (error) {
      logFilterLevel.value = previousLevel
      writeStorage(logLevelKey, previousLevel)
      throw error
    }
  }

  async function syncInitialLogLevel() {
    await desktopClient.setLogLevel(logFilterLevel.value)
  }

  function initLogListeners() {
    if (logListenersInitialised.value) return
    logListenersInitialised.value = true

    logUnsubscribe.value = onWailsEvent('backend-log', (payload) => {
      pushLogEntry(toLogEntry(payload))
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
