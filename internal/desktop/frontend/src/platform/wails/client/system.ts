import {
  CheckForUpdate,
  ConfirmQuit,
  ExportLogs,
  GetBuildInfo,
  GetDataDirectory,
  InstallUpdate,
  OpenDataDirectory,
  OpenLogsDirectory,
  SelectSaveFile,
  SelectXMLFiles,
  SetLogLevel,
  ShowInFolder,
} from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app'
import type * as desktopapi from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/desktopapi/models'
import type { BuildInfo, UpdateInfo } from '@/types/desktop'
import { callWails, asString, asStringArray } from './base'

export function mapBuildInfo(item: desktopapi.BuildInfo): BuildInfo {
  return {
    version: item.version,
    commit: item.commit,
    date: item.date,
    webview2Version: item.webview2Version,
  }
}

export function mapUpdateInfo(item: desktopapi.UpdateInfo): UpdateInfo {
  return {
    enabled: item.enabled === true,
    currentVersion: asString(item.currentVersion),
    available: item.available === true,
    version: asString(item.version),
    notes: asString(item.notes),
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
  async checkForUpdate(): Promise<UpdateInfo> {
    return mapUpdateInfo(await callWails(() => CheckForUpdate()))
  },
  // installUpdate downloads and verifies the release the last check found,
  // then restarts into it: on success the app closes soon after it resolves.
  installUpdate(): Promise<void> {
    return callWails(() => InstallUpdate())
  },
}
