import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CTeEventsDialog from './CTeEventsDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import type { CTeEvent } from '@/types/desktop'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ dark: { isActive: false }, notify }),
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  desktopClient: { listCTeEvents: vi.fn() },
}))

function mountDialog() {
  return shallowMount(CTeEventsDialog, {
    props: { modelValue: false, cnpj: '123', chaveAcesso: 'chave-1' },
    global: {
      stubs: {
        QTable: { name: 'QTable', props: ['rows', 'loading'], template: '<div />' },
      },
    },
  })
}

describe('CTeEventsDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the CT-e events each time it opens', async () => {
    const event = { ID: 'evt-1', TpEvento: '110111', Type: 'cancelamento' } as CTeEvent
    vi.mocked(desktopClient.listCTeEvents).mockResolvedValue([event])
    const wrapper = mountDialog()
    expect(desktopClient.listCTeEvents).not.toHaveBeenCalled()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(desktopClient.listCTeEvents).toHaveBeenCalledWith('123', 'chave-1')
    expect(wrapper.getComponent({ name: 'QTable' }).props('rows')).toEqual([event])
  })

  it('reports a failed load', async () => {
    vi.mocked(desktopClient.listCTeEvents).mockRejectedValue(new Error('boom'))
    const wrapper = mountDialog()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao carregar eventos: boom',
    })
  })
})
