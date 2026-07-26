import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentEventsDialog from './DocumentEventsDialog.vue'
import type { DocumentEvent } from '@/types/desktop'

const loadEvents = vi.fn()

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify: vi.fn() }),
}))

vi.mock('@/composables/useDocuments', () => ({
  useDocuments: () => ({ loadEvents }),
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
    props: { modelValue: false, documentId: '' },
    global: {
      stubs: {
        'q-dialog': { template: '<div><slot /></div>' },
        'q-card': { template: '<div><slot /></div>' },
        'q-card-section': { template: '<div><slot /></div>' },
        'q-card-actions': { template: '<div><slot /></div>' },
        'q-space': { template: '<div />' },
        'q-btn': { template: '<button />' },
        'q-badge': { template: '<div />' },
        'q-td': { template: '<td />' },
        'q-table': {
          name: 'QTable',
          props: ['rows', 'loading'],
          template: '<div />',
        },
      },
      directives: {
        ClosePopup: {},
      },
    },
  })
}

function rows(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.getComponent({ name: 'QTable' }).props('rows') as DocumentEvent[]
}

describe('DocumentEventsDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('does not keep the previous document events when the next load fails', async () => {
    loadEvents.mockResolvedValueOnce([documentEvent('evt-a')])

    const wrapper = mountDialog()
    await wrapper.setProps({ documentId: 'doc-a', modelValue: true })
    await flushPromises()
    expect(rows(wrapper)).toHaveLength(1)

    await wrapper.setProps({ modelValue: false })
    loadEvents.mockRejectedValueOnce(new Error('boom'))

    await wrapper.setProps({ documentId: 'doc-b', modelValue: true })
    await flushPromises()

    expect(rows(wrapper)).toEqual([])
  })

  it('does not show the previous document events while the next load is pending', async () => {
    loadEvents.mockResolvedValueOnce([documentEvent('evt-a')])

    const wrapper = mountDialog()
    await wrapper.setProps({ documentId: 'doc-a', modelValue: true })
    await flushPromises()

    await wrapper.setProps({ modelValue: false })
    let resolvePending: (value: DocumentEvent[]) => void = () => {}
    loadEvents.mockReturnValueOnce(
      new Promise<DocumentEvent[]>((resolve) => {
        resolvePending = resolve
      })
    )

    await wrapper.setProps({ documentId: 'doc-b', modelValue: true })
    expect(rows(wrapper)).toEqual([])

    resolvePending([documentEvent('evt-b')])
    await flushPromises()
    expect(rows(wrapper).map((event) => event.ID)).toEqual(['evt-b'])
  })
})
