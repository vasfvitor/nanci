import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentEventsDialog from './DocumentEventsDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import type { DocumentEvent } from '@/types/desktop'
import { deferred } from '@/test/fixtures'

const notify = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ dark: { isActive: false }, notify }),
  copyToClipboard: vi.fn(),
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  desktopClient: { listEventsForDocument: vi.fn() },
}))

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
    props: { modelValue: false, documentId: '', chaveAcesso: '' },
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
  await wrapper.setProps({ documentId, chaveAcesso: `chave-${documentId}`, modelValue: true })
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
    vi.mocked(desktopClient.listEventsForDocument).mockResolvedValue([documentEvent('evt-a')])
    const wrapper = mountDialog()
    expect(desktopClient.listEventsForDocument).not.toHaveBeenCalled()

    await openFor(wrapper, 'doc-a')

    expect(desktopClient.listEventsForDocument).toHaveBeenCalledWith('doc-a')
    expect(rows(wrapper).map((event) => event.ID)).toEqual(['evt-a'])
  })

  it('does not keep the previous document events when the next load fails', async () => {
    vi.mocked(desktopClient.listEventsForDocument).mockResolvedValueOnce([documentEvent('evt-a')])
    const wrapper = mountDialog()
    await openFor(wrapper, 'doc-a')
    expect(rows(wrapper)).toHaveLength(1)

    await wrapper.setProps({ modelValue: false })
    vi.mocked(desktopClient.listEventsForDocument).mockRejectedValueOnce(new Error('boom'))
    await openFor(wrapper, 'doc-b')

    expect(rows(wrapper)).toEqual([])
    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao carregar eventos: boom',
    })
  })

  it('does not show the previous document events while the next load is pending', async () => {
    vi.mocked(desktopClient.listEventsForDocument).mockResolvedValueOnce([documentEvent('evt-a')])
    const wrapper = mountDialog()
    await openFor(wrapper, 'doc-a')

    await wrapper.setProps({ modelValue: false })
    const pending = deferred<DocumentEvent[]>()
    vi.mocked(desktopClient.listEventsForDocument).mockReturnValueOnce(pending.promise)

    await wrapper.setProps({ documentId: 'doc-b', modelValue: true })
    expect(rows(wrapper)).toEqual([])

    pending.resolve([documentEvent('evt-b')])
    await flushPromises()
    expect(rows(wrapper).map((event) => event.ID)).toEqual(['evt-b'])
  })
})
