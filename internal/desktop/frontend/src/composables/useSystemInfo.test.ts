import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useSystemInfo } from './useSystemInfo'
import { desktopClient } from '@/platform/wails/client'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    getBuildInfo: vi.fn(),
    getDataDirectory: vi.fn(),
    openDataDirectory: vi.fn(),
    openLogsDirectory: vi.fn(),
    exportLogs: vi.fn(),
  },
}))

describe('useSystemInfo', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the build info and the data directory together', async () => {
    const buildInfo = { version: '1.0.0', commit: 'abc', date: '2026-11-19', webview2Version: '' }
    vi.mocked(desktopClient.getBuildInfo).mockResolvedValue(buildInfo)
    vi.mocked(desktopClient.getDataDirectory).mockResolvedValue('C:\\dados')

    await expect(useSystemInfo().loadSystemInfo()).resolves.toEqual({
      buildInfo,
      dataDir: 'C:\\dados',
    })
  })

  it('opens the folders and exports the logs', async () => {
    vi.mocked(desktopClient.exportLogs).mockResolvedValue('C:\\logs.zip')
    const info = useSystemInfo()

    await info.openDataDirectory()
    await info.openLogsDirectory()
    await expect(info.exportLogs()).resolves.toBe('C:\\logs.zip')

    expect(desktopClient.openDataDirectory).toHaveBeenCalledOnce()
    expect(desktopClient.openLogsDirectory).toHaveBeenCalledOnce()
  })
})
