import {
  ConfirmQuit,
  ExportLogs,
  GetBuildInfo,
  GetDataDirectory,
  OpenDataDirectory,
  OpenLogsDirectory,
  SelectSaveFile,
  SelectXMLFiles,
  SetLogLevel,
  ShowInFolder,
} from '../../../../wailsjs/go/main/App'
import type { BuildInfo } from '@/types/desktop'
import { callWails, asString, asStringArray, asRawRecord } from './base'

export function mapBuildInfo(raw: unknown): BuildInfo {
  const item = asRawRecord(raw)
  return {
    version: asString(item['version']),
    commit: asString(item['commit']),
    date: asString(item['date']),
    webview2Version: asString(item['webview2Version']),
  }
}

export const systemClient = {
  async selectSaveFile(
    title: string,
    defaultFilename: string,
    pattern: string
  ): Promise<string | null> {
    const path = await callWails(() => SelectSaveFile(title, defaultFilename, pattern))
    return path || null
  },
  // selectXMLFiles opens the dialog to pick XML files to import; an empty
  // list means the user cancelled.
  async selectXMLFiles(): Promise<string[]> {
    const paths = await callWails(() => SelectXMLFiles())
    return asStringArray(paths).filter(Boolean)
  },
  setLogLevel(level: string) {
    return callWails(() => SetLogLevel(level))
  },
  // confirmQuit closes the app after the user chose to leave during a sync.
  confirmQuit(): Promise<void> {
    return callWails(() => ConfirmQuit())
  },
  async getBuildInfo(): Promise<BuildInfo> {
    return mapBuildInfo(await callWails(() => GetBuildInfo()))
  },
  getDataDirectory(): Promise<string> {
    return callWails(() => GetDataDirectory())
  },
  openDataDirectory(): Promise<void> {
    return callWails(() => OpenDataDirectory())
  },
  openLogsDirectory(): Promise<void> {
    return callWails(() => OpenLogsDirectory())
  },
  // showInFolder opens the file browser on the folder of an exported file.
  showInFolder(path: string): Promise<void> {
    return callWails(() => ShowInFolder(path))
  },
  async exportLogs(): Promise<string | null> {
    const path = await callWails(() => ExportLogs())
    return path || null
  },
}
