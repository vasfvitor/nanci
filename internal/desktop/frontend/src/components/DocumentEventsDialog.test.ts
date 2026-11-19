import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentEventsDialog from './DocumentEventsDialog.vue'
import type { DocumentEvent } from '@/types/desktop'
import { deferred } from '@/test/fixtures'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ dark: { isActive: false }, notify }),
  copyToClipboard: vi.fn(),
}))

// listEvents stands for the page's composable call, by document.
const listEvents = vi.fn<(documentId: string) => Promise<DocumentEvent[]>>()

function documentEvent(id: string): DocumentEvent {
  return {
    ID: id,
    Type: 'cancelamento',
    EventAt: '2026-07-26T12:00:00Z',
    ReplacementChaveAcesso: '',
    Description: `event ${id}`,
    RawXMLPath: `C:\\xml\\${id}.xml`,
  }
}

function mountDialog() {
  return shallowMount(DocumentEventsDialog, {
    props: { modelValue: false, chaveAcesso: '', load: () => listEvents('') },
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

type Dialog = ReturnType<typeof mountDialog>

function rows(wrapper: Dialog) {
  return wrapper.getComponent({ name: 'QTable' }).props('rows') as DocumentEvent[]
}

async function openFor(wrapper: Dialog, documentId: string) {
  await wrapper.setProps({
    load: () => listEvents(documentId),
    chaveAcesso: `chave-${documentId}`,
    modelValue: true,
  })
  await flushPromises()
}

describe('DocumentEventsDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('frames the events table with the NFS-e title and key', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ chaveAcesso: 'chave-1' })

    const frame = wrapper.getComponent({ name: 'EventsDialogFrame' })
    expect(frame.props('title')).toBe('Eventos da NFS-e')
    expect(frame.props('chaveAcesso')).toBe('chave-1')
    expect(wrapper.getComponent({ name: 'QTable' }).props('hidePagination')).toBe(true)
  })

  it('loads the events of the document each time it opens', async () => {
    listEvents.mockResolvedValue([documentEvent('evt-a')])
    const wrapper = mountDialog()
    expect(listEvents).not.toHaveBeenCalled()

    await openFor(wrapper, 'doc-a')

    expect(listEvents).toHaveBeenCalledWith('doc-a')
    expect(rows(wrapper).map((event) => event.ID)).toEqual(['evt-a'])
  })

  it('does not keep the previous document events when the next load fails', async () => {
    listEvents.mockResolvedValueOnce([documentEvent('evt-a')])
    const wrapper = mountDialog()
    await openFor(wrapper, 'doc-a')
    expect(rows(wrapper)).toHaveLength(1)

    await wrapper.setProps({ modelValue: false })
    listEvents.mockRejectedValueOnce(new Error('boom'))
    await openFor(wrapper, 'doc-b')

    expect(rows(wrapper)).toEqual([])
    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao carregar eventos: boom',
    })
  })

  it('does not show the previous document events while the next load is pending', async () => {
    listEvents.mockResolvedValueOnce([documentEvent('evt-a')])
    const wrapper = mountDialog()
    await openFor(wrapper, 'doc-a')

    await wrapper.setProps({ modelValue: false })
    const pending = deferred<DocumentEvent[]>()
    listEvents.mockReturnValueOnce(pending.promise)

    await wrapper.setProps({ load: () => listEvents('doc-b'), modelValue: true })
    expect(rows(wrapper)).toEqual([])

    pending.resolve([documentEvent('evt-b')])
    await flushPromises()
    expect(rows(wrapper).map((event) => event.ID)).toEqual(['evt-b'])
  })
})
