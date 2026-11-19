import { desktopClient } from '@/platform/wails/client'
import type { BuildInfo } from '@/types/desktop'

// useSystemInfo holds the system actions of the settings page: what the app
// is and where its data lives, and the folders and log export. The page
// shows the results and notifies failures.
export function useSystemInfo() {
  async function loadSystemInfo(): Promise<{ buildInfo: BuildInfo; dataDir: string }> {
    const [buildInfo, dataDir] = await Promise.all([
      desktopClient.getBuildInfo(),
      desktopClient.getDataDirectory(),
    ])
    return { buildInfo, dataDir }
  }

  return {
    loadSystemInfo,
    openDataDirectory: () => desktopClient.openDataDirectory(),
    openLogsDirectory: () => desktopClient.openLogsDirectory(),
    // exportLogs returns the path written, or '' when the dialog was cancelled.
    exportLogs: () => desktopClient.exportLogs(),
  }
}
