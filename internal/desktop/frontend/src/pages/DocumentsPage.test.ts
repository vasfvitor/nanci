import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentsPage from './DocumentsPage.vue'
import { useDocumentsStore } from '@/stores/documents'
import type { DocumentRow } from '@/types/desktop'

vi.mock('quasar', () => ({
  useQuasar: () => ({
    dark: { isActive: false },
    notify: vi.fn(),
    dialog: vi.fn(() => ({ onOk: vi.fn() })),
  }),
  copyToClipboard: vi.fn(),
  date: { formatDate: vi.fn(() => '2026-07') },
}))

vi.mock('vue-router', () => ({
  useRoute: () => ({ query: {} }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCompanies: vi.fn().mockResolvedValue([]),
    listDocuments: vi.fn().mockResolvedValue([]),
  },
}))

function documentRow(relationID: string, chaveAcesso: string): DocumentRow {
  return {
    ID: relationID,
    ChaveAcesso: chaveAcesso,
    Competence: '2026-07',
    PrestadorCNPJ: '',
    PrestadorName: '',
    TomadorCNPJ: '',
    TomadorName: '',
    IntermediarioCNPJ: '',
    IntermediarioName: '',
    ServiceValue: 0,
    ISSValue: 0,
    IRRFValue: 0,
    INSSValue: 0,
    PISValue: 0,
    COFINSValue: 0,
    CSLLValue: 0,
    TotalRetentions: 0,
    Status: 'normal',
    LayoutVersion: '',
    XMLPath: '',
    RawHash: '',
    ParseWarnings: [],
    NFSeNumber: '',
    ServiceDescription: '',
    RelationID: relationID,
    CompanyID: '',
    DocumentID: relationID,
    CompanyRole: 'prestada',
    VisibilityReason: '',
    FirstSeenNSU: null,
    LastSeenNSU: null,
  }
}

function mountPage() {
  return shallowMount(DocumentsPage, {
    global: {
      stubs: {
        'q-page': { template: '<div><slot /></div>' },
        'q-badge': { template: '<div />' },
        'q-btn': { template: '<button />' },
        'q-checkbox': { template: '<div />' },
        'q-date': { template: '<div />' },
        'q-icon': { template: '<i />' },
        'q-input': { template: '<div />' },
        'q-popup-proxy': { template: '<div />' },
        'q-select': { template: '<div />' },
        'q-separator': { template: '<div />' },
        'q-space': { template: '<div />' },
        'q-td': { template: '<td />' },
        'q-toggle': { template: '<div />' },
        'q-tooltip': { template: '<div />' },
        'q-tr': { template: '<tr />' },
        'q-table': {
          name: 'QTable',
          props: ['rows', 'selected', 'pagination'],
          emits: ['update:selected', 'update:pagination'],
          template: '<div />',
        },
      },
      directives: {
        ClosePopup: {},
      },
    },
  })
}

function selectedProp(wrapper: ReturnType<typeof mountPage>) {
  return wrapper.getComponent({ name: 'QTable' }).props('selected') as DocumentRow[]
}

describe('DocumentsPage selection', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('drops selected rows that are absent from a new result set', async () => {
    const store = useDocumentsStore()
    store.setDocuments([documentRow('rel-a', '1'.repeat(50))])

    const wrapper = mountPage()
    await flushPromises()

    wrapper
      .getComponent({ name: 'QTable' })
      .vm.$emit('update:selected', [documentRow('rel-a', '1'.repeat(50))])
    await flushPromises()
    expect(selectedProp(wrapper)).toHaveLength(1)

    // A different company/competence returns an entirely different result set.
    store.setDocuments([documentRow('rel-b', '2'.repeat(50))])
    await flushPromises()

    expect(selectedProp(wrapper)).toEqual([])
  })

  it('keeps selected rows that are still present after a refresh', async () => {
    const store = useDocumentsStore()
    store.setDocuments([documentRow('rel-a', '1'.repeat(50))])

    const wrapper = mountPage()
    await flushPromises()

    wrapper
      .getComponent({ name: 'QTable' })
      .vm.$emit('update:selected', [documentRow('rel-a', '1'.repeat(50))])
    await flushPromises()

    // Re-running the same search replaces the array with equivalent rows.
    store.setDocuments([
      documentRow('rel-a', '1'.repeat(50)),
      documentRow('rel-b', '2'.repeat(50)),
    ])
    await flushPromises()

    expect(selectedProp(wrapper).map((row) => row.RelationID)).toEqual(['rel-a'])
  })
})
