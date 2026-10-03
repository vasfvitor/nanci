import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentsPage from './DocumentsPage.vue'
import { desktopClient, mapDocumentRow } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CompanySummary, DocumentRow, PullResult } from '@/types/desktop'
import { DOCUMENT_COLUMN_NAMES } from '@/utils/documentColumns'

type OkHandler = (payload: unknown) => void

const notify = vi.fn()
const okHandlers: OkHandler[] = []
const dialog = vi.fn((_options: Record<string, unknown>) => ({
  onOk: (handler: OkHandler) => {
    okHandlers.push(handler)
  },
}))

vi.mock('quasar', () => ({
  useQuasar: () => ({
    dark: { isActive: false },
    notify,
    dialog,
  }),
  useDialogPluginComponent: vi.fn(),
  copyToClipboard: vi.fn(),
  date: { formatDate: vi.fn(() => '2026-07') },
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  wailsErrorCode: () => '',
  desktopClient: {
    listCompanies: vi.fn(),
    listDocuments: vi.fn(),
    markDocumentsViewed: vi.fn(),
    exportDocuments: vi.fn(),
    exportDANFSeZIP: vi.fn(),
    exportDANFSe: vi.fn(),
    exportXML: vi.fn(),
    pull: vi.fn(),
    resetSyncState: vi.fn(),
    listEventsForDocument: vi.fn(),
  },
}))

function company(cnpj: string, name: string): CompanySummary {
  return {
    ID: `company-${cnpj}`,
    CNPJ: cnpj,
    CNPJRoot: cnpj.slice(0, 8),
    Name: name,
    CredentialID: '',
    CredentialLabel: '',
    CredentialCertPath: '',
    Environment: 'producao',
    UF: 'SP',
    LastFoundNSU: 25,
    SyncStartPolicy: 'from_now',
    LastRunStatus: '',
    LastRunStopReason: '',
  }
}

const acme = company('12345678000100', 'ACME')
const outra = company('98765432000199', 'Outra')

function documentRow(chave: string, fields: Partial<DocumentRow> = {}): DocumentRow {
  return mapDocumentRow({
    ID: `doc-${chave}`,
    ChaveAcesso: chave,
    Competence: '2026-07',
    PrestadorCNPJ: '11222333000181',
    PrestadorName: 'Prestador',
    TomadorCNPJ: acme.CNPJ,
    TomadorName: 'ACME',
    ServiceValue: 1000,
    Status: 'normal',
    LayoutVersion: '1.00',
    NFSeNumber: '1',
    RelationID: `rel-${chave}`,
    CompanyID: acme.ID,
    DocumentID: `document-${chave}`,
    CompanyRole: 'tomada',
    VisibilityReason: 'exact_tomador',
    FirstSeenNSU: 1,
    LastSeenNSU: 1,
    ...fields,
  })
}

const placeholder = 'Filtrar por chave, número, nome ou CNPJ...'
const consultoria = documentRow('A'.repeat(50), {
  NFSeNumber: '1234',
  PrestadorName: 'Consultoria Contábil',
})
const software = documentRow('B'.repeat(50), {
  NFSeNumber: '5678',
  PrestadorName: 'Software Ltda',
  ViewedAt: new Date('2026-07-10T00:00:00Z'),
})
const treinamento = documentRow('C'.repeat(50), { NFSeNumber: '9012', PrestadorName: 'Treinamentos' })

// mountPage mounts on the active Pinia, so two mounts in a test stand for
// leaving the page and coming back.
function mountPage() {
  return shallowMount(DocumentsPage, {
    global: {
      stubs: {
        DocumentFilterBar: false,
        DocumentPageHeader: false,
        DocumentTableTop: false,
        DocumentFilterSelect: false,
        'q-page': { template: '<div><slot /></div>' },
        'q-btn': {
          name: 'QBtn',
          props: ['label', 'disable', 'loading'],
          emits: ['click'],
          template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}<slot /></button>',
        },
        'q-table': {
          name: 'QTable',
          props: ['rows', 'pagination', 'selected', 'columns', 'noDataLabel', 'rowKey'],
          emits: ['update:pagination', 'update:selected'],
          template: '<div><slot name="top" /></div>',
        },
        'q-select': {
          name: 'QSelect',
          props: ['modelValue', 'label', 'options'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
        'q-input': {
          name: 'QInput',
          props: ['modelValue', 'label', 'placeholder'],
          emits: ['update:modelValue'],
          template: '<div><slot /></div>',
        },
        'q-toggle': {
          name: 'QToggle',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
        DocumentEventsDialog: { template: '<div />' },
      },
    },
  })
}

type Page = ReturnType<typeof mountPage>

function button(wrapper: Page, label: string | RegExp) {
  const found = wrapper.findAllComponents({ name: 'QBtn' }).find((btn) => {
    const text = String(btn.props('label'))
    return typeof label === 'string' ? text === label : label.test(text)
  })
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

function field(wrapper: Page, name: 'QSelect' | 'QInput', label: string) {
  const found = wrapper
    .findAllComponents({ name })
    .find((input) => input.props('label') === label || input.props('placeholder') === label)
  if (!found) throw new Error(`${name} ${label} not found`)
  return found
}

function table(wrapper: Page) {
  return wrapper.getComponent({ name: 'QTable' })
}

async function select(wrapper: Page, rows: DocumentRow[]) {
  table(wrapper).vm.$emit('update:selected', rows)
  await flushPromises()
}

function deferred<T>() {
  let resolve!: (value: T) => void
  const promise = new Promise<T>((res) => {
    resolve = res
  })
  return { promise, resolve }
}

describe('DocumentsPage', () => {
  beforeEach(async () => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([acme, outra])
    vi.mocked(desktopClient.listDocuments).mockResolvedValue([consultoria, software, treinamento])
    await useWorkspaceStore().loadCompanies()
  })

  it('lists the NFS-e of the first company with its ambiente and status line', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(desktopClient.listDocuments).toHaveBeenCalledWith({
      CNPJ: acme.CNPJ,
      Competence: '',
      Direction: '',
      OnlyUnread: false,
    })
    expect(table(wrapper).props('rows')).toEqual([consultoria, software, treinamento])
    expect(table(wrapper).props('rowKey')).toBe('ChaveAcesso')
    expect(table(wrapper).props('noDataLabel')).toBe('Nenhuma NFS-e encontrada.')
    expect(wrapper.getComponent({ name: 'DocumentPageHeader' }).props('ambiente')).toEqual({
      label: 'Produção',
      color: 'negative',
    })
    expect(wrapper.getComponent({ name: 'DocumentPageHeader' }).props('contextLine')).toBe(
      'ACME · 12.345.678/0001-00 · Todas as competências'
    )
    expect(wrapper.text()).toContain('Última sincronização: nunca · NSU 25')
  })

  it('asks for a company when there is none', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()
    const wrapper = mountPage()
    await flushPromises()

    expect(desktopClient.listDocuments).not.toHaveBeenCalled()
    expect(table(wrapper).props('noDataLabel')).toBe(
      'Nenhuma empresa cadastrada. Cadastre uma em Empresas.'
    )
  })

  it('searches when the competência changes', async () => {
    mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listDocuments).mockClear()

    useWorkspaceStore().competence = '2026-05'
    await flushPromises()

    expect(desktopClient.listDocuments).toHaveBeenCalledTimes(1)
    expect(desktopClient.listDocuments).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: acme.CNPJ, Competence: '2026-05' })
    )
  })

  it('drops a list that arrives after the user picked another company', async () => {
    const first = deferred<DocumentRow[]>()
    vi.mocked(desktopClient.listDocuments)
      .mockReturnValueOnce(first.promise)
      .mockResolvedValueOnce([treinamento])
    const wrapper = mountPage()
    await flushPromises()

    useWorkspaceStore().cnpj = outra.CNPJ
    await flushPromises()
    expect(table(wrapper).props('rows')).toEqual([treinamento])

    first.resolve([consultoria])
    await flushPromises()
    expect(table(wrapper).props('rows')).toEqual([treinamento])
  })

  it('clears the selection when the company changes', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await select(wrapper, [consultoria])

    useWorkspaceStore().cnpj = outra.CNPJ
    await flushPromises()

    expect(desktopClient.listDocuments).toHaveBeenLastCalledWith(
      expect.objectContaining({ CNPJ: outra.CNPJ })
    )
    expect(table(wrapper).props('selected')).toEqual([])
  })

  it('shows the standard document columns, with "Número" for the NFS-e number', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const columns = table(wrapper).props('columns') as { name: string; label: string }[]
    expect(columns.map((column) => column.name)).toEqual([...DOCUMENT_COLUMN_NAMES])
    expect(columns.find((column) => column.name === 'numero')?.label).toBe('Número')
    expect(columns.find((column) => column.name === 'emitente')?.label).toBe('Prestador')
    expect(columns.find((column) => column.name === 'destinatario')?.label).toBe('Tomador')
  })

  it('filters by papel with the singular options and the same Direction values', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listDocuments).mockClear()

    const papel = field(wrapper, 'QSelect', 'Papel')
    expect(papel.props('options')).toEqual([
      { label: 'Todos', value: '' },
      { label: 'Tomada', value: 'tomada' },
      { label: 'Prestada', value: 'prestada' },
      { label: 'Intermediário', value: 'intermediario' },
      { label: 'Sem papel fiscal', value: 'none' },
    ])
    papel.vm.$emit('update:modelValue', 'prestada')
    await flushPromises()
    await button(wrapper, 'Buscar').trigger('click')
    await flushPromises()

    expect(desktopClient.listDocuments).toHaveBeenCalledWith(
      expect.objectContaining({ Direction: 'prestada' })
    )
  })

  it('searches the unviewed NFS-e when "Somente não vistos" is turned on', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listDocuments).mockClear()

    const toggle = wrapper.getComponent({ name: 'QToggle' })
    expect(toggle.props('label')).toBe('Somente não vistos')
    toggle.vm.$emit('update:modelValue', true)
    await flushPromises()

    expect(desktopClient.listDocuments).toHaveBeenCalledWith(
      expect.objectContaining({ OnlyUnread: true })
    )
  })

  it('narrows the rows by accent- and case-insensitive text', async () => {
    const wrapper = mountPage()
    await flushPromises()

    field(wrapper, 'QInput', placeholder).vm.$emit('update:modelValue', 'CONTABIL')
    await flushPromises()
    expect(table(wrapper).props('rows')).toEqual([consultoria])
  })

  it('keeps the text filter and the selection after leaving the page and coming back', async () => {
    const first = mountPage()
    await flushPromises()
    field(first, 'QInput', placeholder).vm.$emit('update:modelValue', 'contabil')
    await select(first, [consultoria])
    first.unmount()

    const second = mountPage()
    await flushPromises()
    expect(field(second, 'QInput', placeholder).props('modelValue')).toBe('contabil')
    expect(table(second).props('rows')).toEqual([consultoria])
    expect(table(second).props('selected')).toEqual([consultoria])
  })

  it('clears the rows of another company when it comes back', async () => {
    const first = mountPage()
    await flushPromises()
    await select(first, [consultoria])
    first.unmount()

    useWorkspaceStore().cnpj = outra.CNPJ
    const list = deferred<DocumentRow[]>()
    vi.mocked(desktopClient.listDocuments).mockReturnValue(list.promise)
    const second = mountPage()
    await flushPromises()
    expect(table(second).props('rows')).toEqual([])
    expect(table(second).props('selected')).toEqual([])

    list.resolve([treinamento])
    await flushPromises()
    expect(desktopClient.listDocuments).toHaveBeenLastCalledWith(
      expect.objectContaining({ CNPJ: outra.CNPJ })
    )
    expect(table(second).props('rows')).toEqual([treinamento])
  })

  it('drops selected rows that are absent from a new result set', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await select(wrapper, [consultoria, software])

    vi.mocked(desktopClient.listDocuments).mockResolvedValue([software])
    await button(wrapper, 'Buscar').trigger('click')
    await flushPromises()

    expect(table(wrapper).props('selected')).toEqual([software])
  })

  it('exports the filtered rows by chave with the choice of the dialog', async () => {
    vi.mocked(desktopClient.exportDocuments).mockResolvedValue({
      OutPath: 'C:\\exports\\nfse.xlsx',
      Format: 'xlsx',
      Incremental: true,
      ExportedCount: 1,
    })
    const wrapper = mountPage()
    await flushPromises()
    field(wrapper, 'QInput', placeholder).vm.$emit('update:modelValue', 'contabil')
    await flushPromises()

    await button(wrapper, 'Exportar').trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        componentProps: {
          source: 'nfse',
          count: 1,
          scope: 'listed',
          formats: [
            { label: 'Planilha CSV', value: 'csv' },
            { label: 'Planilha Excel (XLSX)', value: 'xlsx' },
            { label: 'XMLs originais (ZIP)', value: 'zip' },
            { label: 'DANFSes (ZIP)', value: 'danfse-zip' },
          ],
          showIncludeResumos: false,
        },
      })
    )
    expect(desktopClient.exportDocuments).not.toHaveBeenCalled()

    okHandlers[0]?.({ format: 'xlsx', incremental: true, includeResumos: false })
    await flushPromises()

    expect(desktopClient.exportDocuments).toHaveBeenCalledWith({
      CNPJ: acme.CNPJ,
      Competence: '',
      Direction: '',
      Format: 'xlsx',
      Incremental: true,
      ChavesAcesso: [consultoria.ChaveAcesso],
    })
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: '1 documento exportado para C:\\exports\\nfse.xlsx.',
    })
  })

  it('exports the DANFSes of the selection when there is one', async () => {
    vi.mocked(desktopClient.exportDANFSeZIP).mockResolvedValue({
      OutPath: 'C:\\exports\\danfses.zip',
      Format: 'danfse-zip',
      Incremental: false,
      ExportedCount: 2,
    })
    const wrapper = mountPage()
    await flushPromises()
    await select(wrapper, [software, treinamento])

    await button(wrapper, 'Exportar').trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        componentProps: expect.objectContaining({ count: 2, scope: 'selected' }),
      })
    )
    okHandlers[0]?.({ format: 'danfse-zip', incremental: false, includeResumos: false })
    await flushPromises()

    expect(desktopClient.exportDANFSeZIP).toHaveBeenCalledWith(
      expect.objectContaining({
        ChavesAcesso: [software.ChaveAcesso, treinamento.ChaveAcesso],
        Incremental: false,
      })
    )
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: '2 DANFSes exportados para C:\\exports\\danfses.zip.',
    })
  })

  it('says when an export found nothing to export', async () => {
    vi.mocked(desktopClient.exportDocuments).mockResolvedValue({
      OutPath: '',
      Format: 'zip',
      Incremental: false,
      ExportedCount: 0,
    })
    const wrapper = mountPage()
    await flushPromises()

    await button(wrapper, 'Exportar').trigger('click')
    okHandlers[0]?.({ format: 'zip', incremental: false, includeResumos: false })
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({ type: 'info', message: 'Nenhum documento para exportar.' })
  })

  it('marks the selected NFS-e viewed without asking', async () => {
    vi.mocked(desktopClient.markDocumentsViewed).mockResolvedValue(1)
    const wrapper = mountPage()
    await flushPromises()
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (2)')

    await select(wrapper, [treinamento])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (1)')
    await button(wrapper, /^Marcar vistos/).trigger('click')
    await flushPromises()

    expect(dialog).not.toHaveBeenCalled()
    expect(desktopClient.markDocumentsViewed).toHaveBeenCalledWith(acme.CNPJ, [treinamento.ChaveAcesso])
    expect(table(wrapper).props('selected')).toEqual([])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (1)')
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: '1 documento marcado como visto.' })
  })

  it('asks before marking the whole list viewed', async () => {
    vi.mocked(desktopClient.markDocumentsViewed).mockResolvedValue(2)
    const wrapper = mountPage()
    await flushPromises()

    await button(wrapper, /^Marcar vistos/).trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({ message: 'Marcar como vistas as 2 NFS-e novas da lista?' })
    )
    expect(desktopClient.markDocumentsViewed).not.toHaveBeenCalled()

    okHandlers[0]?.(undefined)
    await flushPromises()

    expect(desktopClient.markDocumentsViewed).toHaveBeenCalledWith(acme.CNPJ, [
      consultoria.ChaveAcesso,
      treinamento.ChaveAcesso,
    ])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (0)')
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: '2 documentos marcados como vistos.' })
  })

  it('syncs the NFS-e and refreshes the list and the status line', async () => {
    vi.mocked(desktopClient.pull).mockResolvedValue({
      Status: 'success',
      StopReason: 'no-more-nsu',
      LastProcessedNSU: 30,
      LastFoundNSU: 28,
      CredentialCNPJ: acme.CNPJ,
    } as PullResult)
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listDocuments).mockClear()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([{ ...acme, LastFoundNSU: 28 }, outra])

    await button(wrapper, 'Sincronizar NFS-e').trigger('click')
    await flushPromises()

    expect(desktopClient.pull).toHaveBeenCalledWith({ CNPJ: acme.CNPJ, Mode: '' })
    expect(desktopClient.listDocuments).toHaveBeenCalled()
    expect(wrapper.text()).toContain('NSU 28')
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'positive',
        message: expect.stringContaining('Sincronização success (no-more-nsu). Último NSU: 30'),
      })
    )
  })

  it('confirms that the reset keeps the documents, then resets the cursor', async () => {
    vi.mocked(desktopClient.resetSyncState).mockResolvedValue(undefined)
    const wrapper = mountPage()
    await flushPromises()

    await button(wrapper, 'Redefinir NFS-e').trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Redefinir NFS-e',
        message: 'Reinicia a sincronização da NFS-e a partir do NSU 0. Os documentos já baixados ficam.',
      })
    )
    expect(desktopClient.resetSyncState).not.toHaveBeenCalled()

    okHandlers[0]?.(undefined)
    await flushPromises()

    expect(desktopClient.resetSyncState).toHaveBeenCalledWith({ CompanyCNPJ: acme.CNPJ })
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: 'Sincronização da NFS-e redefinida para ACME.',
    })
  })
})
