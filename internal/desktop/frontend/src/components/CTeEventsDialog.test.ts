import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CTeEventsDialog from './CTeEventsDialog.vue'
import type { CTeEvent } from '@/types/desktop'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ dark: { isActive: false }, notify }),
}))

// load stands for the page's composable call.
const load = vi.fn<() => Promise<CTeEvent[]>>()

function mountDialog() {
  return shallowMount(CTeEventsDialog, {
    props: { modelValue: false, chaveAcesso: 'chave-1', load },
    global: {
      renderStubDefaultSlot: true,
      directives: { closePopup: {} },
      stubs: {
        EventsDialogFrame: false,
        QDialog: { template: '<div><slot /></div>' },
        QTable: {
          name: 'QTable',
          props: { rows: Array, loading: Boolean, hidePagination: Boolean },
          template: '<div />',
        },
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
    load.mockResolvedValue([event])
    const wrapper = mountDialog()
    expect(load).not.toHaveBeenCalled()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(load).toHaveBeenCalledTimes(1)
    expect(wrapper.getComponent({ name: 'QTable' }).props('rows')).toEqual([event])
  })

  it('frames the events table with the CT-e title and key', () => {
    const wrapper = mountDialog()
    const frame = wrapper.getComponent({ name: 'EventsDialogFrame' })
    expect(frame.props('title')).toBe('Eventos do CT-e')
    expect(frame.props('chaveAcesso')).toBe('chave-1')
    expect(wrapper.getComponent({ name: 'QTable' }).props('hidePagination')).toBe(true)
  })

  it('reports a failed load', async () => {
    load.mockRejectedValue(new Error('boom'))
    const wrapper = mountDialog()

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao carregar eventos: boom',
    })
  })
})
