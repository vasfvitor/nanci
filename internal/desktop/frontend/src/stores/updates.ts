import { computed, shallowRef } from 'vue'
import { defineStore } from 'pinia'
import { desktopClient, errorMessage, wailsErrorCode } from '@/platform/wails/client'
import { onWailsEvent } from '@/platform/wails/events'
import type { UpdateInfo, UpdateProgress } from '@/types/desktop'

// UpdateStatus is where the update flow stands. idle follows a check that
// found nothing (or comes before any check); unavailable means this build
// does not update itself.
export type UpdateStatus =
  'idle' | 'checking' | 'available' | 'downloading' | 'installing' | 'error' | 'unavailable'

// The updates store keeps the update state for the whole session: the
// startup check runs from MainLayout, and the Sobre section of the settings
// page shows the same state and offers the same action.
export const useUpdatesStore = defineStore('updates', () => {
  const status = shallowRef<UpdateStatus>('idle')
  const info = shallowRef<UpdateInfo | null>(null)
  const progress = shallowRef<UpdateProgress | null>(null)
  const error = shallowRef('')
  let startupCheck: Promise<UpdateInfo | null> | null = null

  const busy = computed(
    () =>
      status.value === 'checking' || status.value === 'downloading' || status.value === 'installing'
  )
  // canInstall: a release was found and nothing runs. After a failed
  // install the release stays known, so the user can try again.
  const canInstall = computed(() => !busy.value && info.value?.available === true)
  // percent of the download, or null while the size is unknown.
  const percent = computed(() => {
    const p = progress.value
    if (!p || p.total <= 0) return null
    return Math.min(100, Math.round((p.written / p.total) * 100))
  })

  // check asks the backend for a newer release. It returns null without a
  // call while a check or an install runs, and rethrows a failure after
  // keeping its message in error.
  async function check(): Promise<UpdateInfo | null> {
    if (busy.value) return null
    status.value = 'checking'
    error.value = ''
    try {
      const result = await desktopClient.checkForUpdate()
      info.value = result
      if (!result.enabled) status.value = 'unavailable'
      else status.value = result.available ? 'available' : 'idle'
      return result
    } catch (e) {
      status.value = 'error'
      error.value = errorMessage(e)
      throw e
    }
  }

  // checkOnce is the startup check: one per session, so a remount of the
  // layout gets the same promise. A failure resolves null and stays in error.
  function checkOnce(): Promise<UpdateInfo | null> {
    startupCheck ??= check().catch(() => null)
    return startupCheck
  }

  // install downloads the release, verifies it and restarts into it. It
  // returns false without a call when there is nothing to install or a check
  // or install runs. The state is marked before the call, so a second
  // screen sees the download. On success the app closes soon after; a
  // refusal because a sync runs puts the release back as available.
  async function install(): Promise<boolean> {
    if (!canInstall.value) return false
    status.value = 'downloading'
    progress.value = null
    error.value = ''
    const unsubscribe = onWailsEvent('wails:updater:download-progress', (p) => {
      progress.value = p
      if (p.total > 0 && p.written >= p.total) status.value = 'installing'
    })
    try {
      await desktopClient.installUpdate()
      status.value = 'installing'
      return true
    } catch (e) {
      if (wailsErrorCode(e) === 'sync_running') {
        status.value = 'available'
      } else {
        status.value = 'error'
        error.value = errorMessage(e)
      }
      throw e
    } finally {
      unsubscribe()
    }
  }

  return {
    status,
    info,
    progress,
    error,
    busy,
    canInstall,
    percent,
    check,
    checkOnce,
    install,
  }
})
