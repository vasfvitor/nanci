import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia, type Pinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CTePage from './CTePage.vue'
import { desktopClient, mapCTeRow } from '@/platform/wails/client'
import type { CTeRow, CTeStatusResult } from '@/types/desktop'
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
  date: { formatDate: vi.fn(() => '2026-08') },
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  wailsErrorCode: () => '',
  desktopClient: {
    listCompanies: vi.fn(),
    listCTe: vi.fn(),
    statusCTe: vi.fn(),
    pullCTe: vi.fn(),
    previewResetCTe: vi.fn(),
    resetCTe: vi.fn(),
    exportCTeXML: vi.fn(),
    exportCTeZIP: vi.fn(),
    markCTeViewed: vi.fn(),
  },
}))

const company = {
  ID: 'company-1',
  CNPJ: '98765432000199',
  CNPJRoot: '98765432',
  Name: 'Empresa Um',
  CredentialID: '',
  CredentialLabel: '',
  CredentialCertPath: '',
  Environment: 'producao',
  UF: 'SP',
  LastFoundNSU: null,
  SyncStartPolicy: 'from_now' as const,
  LastRunStatus: '',
  LastRunStopReason: '',
}

const nfeChave = '35260911222333000181550010000045121418273651'
const placeholder = 'Filtrar por chave, número, nome ou CNPJ...'

const tomada = mapCTeRow({
  ChaveAcesso: '35260811222333000181570010000012341000012345',
  Modelo: '57',
  TipoDocumento: 'cte',
  Numero: '1234',
  EmitenteName: 'Transportadora Fictícia',
  TomadorName: 'Empresa Um',
  CompanyRole: 'tomador',
  Papeis: ['tomador', 'remetente'],
  Situacao: 'autorizada',
  NFeChaves: [nfeChave],
})
const servico = mapCTeRow({
  ChaveAcesso: '35260811222333000181670010000000561000000567',
  Modelo: '67',
  TipoDocumento: 'cte_os',
  Numero: '56',
  EmitenteName: 'Fretamento Serra Ltda',
  CompanyRole: 'tomador',
  Papeis: ['tomador'],
  Situacao: 'cancelada',
})

function status(overrides: Partial<CTeStatusResult> = {}): CTeStatusResult {
  return {
    CompanyName: 'Empresa Um',
    CNPJ: company.CNPJ,
    UF: 'SP',
    TpAmb: '1',
    LastNSU: 10,
    MaxNSU: 10,
    LastRunStatus: 'completed',
    LastRunStopReason: '',
    NextAllowedAt: null,
    BlockedReason: '',
    RequestsLastHour: 1,
    RequestBudget: 20,
    TotalTomador: 2,
    TotalDestinatario: 0,
    TotalRemetente: 0,
    TotalOutros: 0,
    ...overrides,
  }
}

function mountPage(pinia?: Pinia) {
  return shallowMount(CTePage, {
    global: {
      plugins: pinia ? [pinia] : [],
      stubs: {
        DocumentFilterBar: false,
        DocumentPageHeader: false,
        'q-page': { template: '<div><slot /></div>' },
        'q-banner': { template: '<div class="q-banner-stub"><slot /></div>' },
        'q-btn': {
          name: 'QBtn',
          props: ['label', 'disable', 'loading'],
          emits: ['click'],
          template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}<slot /></button>',
        },
        'q-table': {
          name: 'QTable',
          props: ['rows', 'pagination', 'selected', 'columns', 'noDataLabel'],
          emits: ['update:pagination', 'update:selected'],
          template: '<div><slot name="top" /></div>',
        },
        'q-select': {
          name: 'QSelect',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
        'q-input': {
          name: 'QInput',
          props: ['modelValue', 'label', 'placeholder', 'error', 'errorMessage'],
          emits: ['update:modelValue'],
          template: '<div><slot /></div>',
        },
        'q-toggle': {
          name: 'QToggle',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
        CTeEventsDialog: { template: '<div />' },
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

async function select(wrapper: Page, rows: CTeRow[]) {
  table(wrapper).vm.$emit('update:selected', rows)
  await flushPromises()
}

describe('CTePage', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listCTe).mockResolvedValue([tomada, servico])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(status())
  })

  it('lists the CT-e of the first company with its status', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(desktopClient.listCTe).toHaveBeenCalledWith(expect.objectContaining({ CNPJ: company.CNPJ }))
    expect(desktopClient.statusCTe).toHaveBeenCalledWith(company.CNPJ)
    expect(table(wrapper).props('rows')).toEqual([tomada, servico])
    expect(table(wrapper).props('noDataLabel')).toBe('Nenhum CT-e encontrado.')
    expect(wrapper.text()).toContain('NSU 10/10 · CT-e: 2')
  })

  it('asks for a company when there is none', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    const wrapper = mountPage()
    await flushPromises()

    expect(desktopClient.listCTe).not.toHaveBeenCalled()
    expect(table(wrapper).props('noDataLabel')).toBe('Selecione uma empresa.')
  })

  it('shows the standard document columns in order', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const columns = table(wrapper).props('columns') as { name: string; label: string }[]
    expect(columns.map((column) => column.name)).toEqual([...DOCUMENT_COLUMN_NAMES])
    expect(columns.find((column) => column.name === 'destinatario')?.label).toBe('Tomador')
  })

  it('narrows the rows by accent- and case-insensitive text', async () => {
    const wrapper = mountPage()
    await flushPromises()

    field(wrapper, 'QInput', placeholder).vm.$emit('update:modelValue', 'FICTICIA')
    await flushPromises()
    expect(table(wrapper).props('rows')).toEqual([tomada])
  })

  it('keeps the text filter and the selection after leaving the page and coming back', async () => {
    const pinia = createPinia()
    const first = mountPage(pinia)
    await flushPromises()
    field(first, 'QInput', placeholder).vm.$emit('update:modelValue', 'FICTICIA')
    await select(first, [tomada])
    first.unmount()

    const second = mountPage(pinia)
    await flushPromises()
    expect(field(second, 'QInput', placeholder).props('modelValue')).toBe('FICTICIA')
    expect(table(second).props('rows')).toEqual([tomada])
    expect(table(second).props('selected')).toEqual([tomada])
  })

  it('searches with the filters the user picked', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCTe).mockClear()

    field(wrapper, 'QSelect', 'Papel').vm.$emit('update:modelValue', 'remetente')
    field(wrapper, 'QSelect', 'Modelo').vm.$emit('update:modelValue', '67')
    field(wrapper, 'QSelect', 'Situação').vm.$emit('update:modelValue', 'cancelada')
    field(wrapper, 'QInput', 'CNPJ do tomador').vm.$emit('update:modelValue', '98.765.432/0001-99')
    field(wrapper, 'QInput', 'Chave de NF-e transportada').vm.$emit('update:modelValue', nfeChave)
    await flushPromises()

    await button(wrapper, 'Buscar').trigger('click')
    await flushPromises()

    expect(desktopClient.listCTe).toHaveBeenCalledWith({
      CNPJ: company.CNPJ,
      Competence: '',
      Situacao: 'cancelada',
      Role: 'remetente',
      Modelo: '67',
      EmitenteCNPJ: '',
      TomadorCNPJ: company.CNPJ,
      NFeChave: nfeChave,
      OnlyUnread: false,
    })
  })

  it('searches the unviewed CT-e when "Somente não vistos" is turned on', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCTe).mockClear()

    wrapper.getComponent({ name: 'QToggle' }).vm.$emit('update:modelValue', true)
    await flushPromises()

    expect(desktopClient.listCTe).toHaveBeenCalledWith(expect.objectContaining({ OnlyUnread: true }))
  })

  it('does not search with an NF-e key that is not 44 characters', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCTe).mockClear()

    field(wrapper, 'QInput', 'Chave de NF-e transportada').vm.$emit('update:modelValue', '3526 0911')
    await flushPromises()

    const chave = field(wrapper, 'QInput', 'Chave de NF-e transportada')
    expect(chave.props('error')).toBe(true)
    expect(chave.props('errorMessage')).toBe('A chave de NF-e tem 44 caracteres')
    expect(button(wrapper, 'Buscar').props('disable')).toBe(true)
    expect(desktopClient.listCTe).not.toHaveBeenCalled()
  })

  it('disables sync and explains the block while SEFAZ blocks the company', async () => {
    const until = new Date(Date.now() + 60 * 60 * 1000).toISOString()
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(
      status({ NextAllowedAt: until, BlockedReason: 'consumo_indevido' })
    )

    const wrapper = mountPage()
    await flushPromises()

    expect(button(wrapper, 'Sincronizar CT-e').props('disable')).toBe(true)
    expect(wrapper.find('.q-banner-stub').text()).toContain('Consultas bloqueadas pela SEFAZ até')
  })

  it('syncs when the company is not blocked', async () => {
    vi.mocked(desktopClient.pullCTe).mockResolvedValue({
      Status: 'success',
      DocumentsSaved: 3,
      EventsSaved: 1,
      LastNSU: 12,
      MaxNSU: 12,
    } as Awaited<ReturnType<typeof desktopClient.pullCTe>>)
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('.q-banner-stub').exists()).toBe(false)
    const sync = button(wrapper, 'Sincronizar CT-e')
    expect(sync.props('disable')).toBe(false)
    await sync.trigger('click')
    await flushPromises()

    expect(desktopClient.pullCTe).toHaveBeenCalledWith(company.CNPJ)
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'positive', message: expect.stringContaining('3 CT-e e 1 eventos') })
    )
  })

  it('exports the filtered rows as a ZIP with the choice of the dialog', async () => {
    vi.mocked(desktopClient.exportCTeZIP).mockResolvedValue({
      OutPath: 'C:\\exports\\cte.zip',
      Format: 'zip',
      Incremental: true,
      ExportedCount: 1,
    })
    const wrapper = mountPage()
    await flushPromises()
    field(wrapper, 'QInput', placeholder).vm.$emit('update:modelValue', 'serra')
    await flushPromises()

    await button(wrapper, 'Exportar').trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        componentProps: {
          noun: 'CT-e',
          count: 1,
          scope: 'listed',
          formats: [{ label: 'XMLs (ZIP)', value: 'zip' }],
          showIncludeResumos: false,
        },
      })
    )
    expect(desktopClient.exportCTeZIP).not.toHaveBeenCalled()

    okHandlers[0]?.({ format: 'zip', incremental: true, includeResumos: false })
    await flushPromises()

    expect(desktopClient.exportCTeZIP).toHaveBeenCalledWith({
      CNPJ: company.CNPJ,
      Competence: '',
      Role: '',
      ChavesAcesso: [servico.ChaveAcesso],
      Incremental: true,
    })
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: '1 XML exportado para C:\\exports\\cte.zip.',
    })
  })

  it('exports the selection when there is one', async () => {
    vi.mocked(desktopClient.exportCTeZIP).mockResolvedValue(null)
    const wrapper = mountPage()
    await flushPromises()
    await select(wrapper, [tomada])

    await button(wrapper, 'Exportar').trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        componentProps: expect.objectContaining({ count: 1, scope: 'selected' }),
      })
    )
    okHandlers[0]?.({ format: 'zip', incremental: false, includeResumos: false })
    await flushPromises()

    expect(desktopClient.exportCTeZIP).toHaveBeenCalledWith(
      expect.objectContaining({ ChavesAcesso: [tomada.ChaveAcesso], Incremental: false })
    )
  })

  it('marks the selected CT-e viewed without asking', async () => {
    vi.mocked(desktopClient.markCTeViewed).mockResolvedValue(1)
    const wrapper = mountPage()
    await flushPromises()
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (2)')

    await select(wrapper, [servico])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (1)')
    await button(wrapper, /^Marcar vistos/).trigger('click')
    await flushPromises()

    expect(dialog).not.toHaveBeenCalled()
    expect(desktopClient.markCTeViewed).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: company.CNPJ, ChavesAcesso: [servico.ChaveAcesso] })
    )
    expect(table(wrapper).props('selected')).toEqual([])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (1)')
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: '1 documento marcado como visto.' })
  })

  it('asks before marking the whole list viewed', async () => {
    vi.mocked(desktopClient.markCTeViewed).mockResolvedValue(2)
    const wrapper = mountPage()
    await flushPromises()

    await button(wrapper, /^Marcar vistos/).trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({ message: 'Marcar como vistos os 2 CT-e novos da lista?' })
    )
    expect(desktopClient.markCTeViewed).not.toHaveBeenCalled()

    okHandlers[0]?.(undefined)
    await flushPromises()

    expect(desktopClient.markCTeViewed).toHaveBeenCalledWith(
      expect.objectContaining({ ChavesAcesso: [tomada.ChaveAcesso, servico.ChaveAcesso] })
    )
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (0)')
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: '2 documentos marcados como vistos.' })
  })

  it('previews the reset, confirms it with the counts and then resets', async () => {
    const counts = {
      CompanyName: 'Empresa Um',
      CNPJ: company.CNPJ,
      CompanyDocuments: 2,
      Documents: 2,
      Events: 3,
      ExportMarks: 1,
    }
    vi.mocked(desktopClient.previewResetCTe).mockResolvedValue(counts)
    vi.mocked(desktopClient.resetCTe).mockResolvedValue(counts)

    const wrapper = mountPage()
    await flushPromises()
    await button(wrapper, 'Redefinir CT-e').trigger('click')
    await flushPromises()

    expect(desktopClient.previewResetCTe).toHaveBeenCalledWith(company.CNPJ)
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Redefinir CT-e',
        message: expect.stringContaining(
          'Remove 2 CT-e de Empresa Um nos dois ambientes, com 3 eventos e 1 marcas de exportação'
        ),
      })
    )
    expect(desktopClient.resetCTe).not.toHaveBeenCalled()

    vi.mocked(desktopClient.listCTe).mockClear()
    okHandlers[0]?.(undefined)
    await flushPromises()

    expect(desktopClient.resetCTe).toHaveBeenCalledWith(company.CNPJ)
    expect(desktopClient.listCTe).toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'positive', message: expect.stringContaining('2 CT-e e 3 eventos') })
    )
  })

  it('reports a failed reset preview without opening the dialog', async () => {
    vi.mocked(desktopClient.previewResetCTe).mockRejectedValue(new Error('boom'))

    const wrapper = mountPage()
    await flushPromises()
    await button(wrapper, 'Redefinir CT-e').trigger('click')
    await flushPromises()

    expect(dialog).not.toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith({
      type: 'negative',
      message: 'Erro ao preparar a redefinição do CT-e: boom',
    })
  })
})
