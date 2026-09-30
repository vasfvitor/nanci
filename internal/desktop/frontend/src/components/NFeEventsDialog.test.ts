import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NFeEventsDialog from './NFeEventsDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import type { NFeEvent } from '@/types/desktop'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ dark: { isActive: false }, notify }),
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  desktopClient: { listNFeEvents: vi.fn() },
}))

function mountDialog() {
  return shallowMount(NFeEventsDialog, {
    props: { modelValue: false, cnpj: '123', chaveAcesso: 'chave-1' },
    global: {
      stubs: {
        EventsDialogFrame: {
          name: 'EventsDialogFrame',
          props: ['modelValue', 'title', 'chaveAcesso'],
          template: '<div><slot /></div>',
        },
        QTable: {
          name: 'QTable',
          props: { rows: Array, loading: Boolean, hidePagination: Boolean },
          template: '<div />',
        },
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

  it('frames the events table with the NF-e title and key, without pagination', () => {
    const wrapper = mountDialog()
    const frame = wrapper.getComponent({ name: 'EventsDialogFrame' })
    expect(frame.props('title')).toBe('Eventos da NF-e')
    expect(frame.props('chaveAcesso')).toBe('chave-1')
    expect(wrapper.getComponent({ name: 'QTable' }).props('hidePagination')).toBe(true)
  })

  it('reports a failed load', async () => {
    vi.mocked(desktopClient.listNFeEvents).mockRejectedValue(new Error('boom'))
    const wrapper = mountDialog()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao carregar eventos: boom',
    })
  })
})
