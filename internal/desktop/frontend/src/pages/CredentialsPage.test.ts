import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CredentialsPage from './CredentialsPage.vue'
import { desktopClient } from '@/platform/wails/client'

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify: vi.fn() }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCredentials: vi.fn(),
    selectCertificate: vi.fn(),
    updateCredentialPath: vi.fn(),
  },
}))

function mountPage() {
  return shallowMount(CredentialsPage, {
    global: {
      stubs: {
        'q-page': { template: '<div><slot /></div>' },
        'q-btn': { template: '<button />' },
        'q-table': {
          name: 'QTable',
          props: ['rows', 'loading'],
          template: '<div />',
        },
        AddCredentialDialog: { template: '<div />' },
        EditCredentialDialog: { template: '<div />' },
      },
    },
  })
}

describe('CredentialsPage', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('drives the table loading indicator while credentials load', async () => {
    let resolveList!: (value: never[]) => void
    vi.mocked(desktopClient.listCredentials).mockReturnValue(
      new Promise((resolve) => {
        resolveList = resolve
      }) as ReturnType<typeof desktopClient.listCredentials>
    )

    const wrapper = mountPage()
    await flushPromises()

    const table = wrapper.getComponent({ name: 'QTable' })
    expect(table.props('loading')).toBe(true)

    resolveList([])
    await flushPromises()

    expect(table.props('loading')).toBe(false)
  })
})
