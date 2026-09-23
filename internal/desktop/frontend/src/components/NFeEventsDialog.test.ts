import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NFeEventsDialog from './NFeEventsDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import type { NFeEvent } from '@/types/desktop'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ dark: { isActive: false }, notify }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: { listNFeEvents: vi.fn() },
}))

function mountDialog() {
  return shallowMount(NFeEventsDialog, {
    props: { modelValue: false, cnpj: '123', chaveAcesso: 'chave-1' },
    global: {
      stubs: {
        QTable: { name: 'QTable', props: ['rows', 'loading'], template: '<div />' },
      },
    },
  })
}

describe('NFeEventsDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('loads the note events each time it opens', async () => {
    const event = { ID: 'evt-1', TpEvento: '210210' } as NFeEvent
    vi.mocked(desktopClient.listNFeEvents).mockResolvedValue([event])
    const wrapper = mountDialog()
    expect(desktopClient.listNFeEvents).not.toHaveBeenCalled()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(desktopClient.listNFeEvents).toHaveBeenCalledWith('123', 'chave-1')
    expect(wrapper.getComponent({ name: 'QTable' }).props('rows')).toEqual([event])
  })

  it('reports a failed load', async () => {
    vi.mocked(desktopClient.listNFeEvents).mockRejectedValue(new Error('boom'))
    const wrapper = mountDialog()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao carregar eventos: Error: boom',
    })
  })
})
