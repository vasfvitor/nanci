import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NFePage from './NFePage.vue'
import NFeCienciaConfirmDialog from '@/components/NFeCienciaConfirmDialog.vue'
import NFeEventResultsDialog from '@/components/NFeEventResultsDialog.vue'
import NFeManifestacaoDialog from '@/components/NFeManifestacaoDialog.vue'
import { desktopClient, mapNFeRow } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type {
  NFeCienciaPlan,
  NFeEventBatchResult,
  NFeEventResult,
  NFeRow,
  NFeStatusResult,
} from '@/types/desktop'
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
  date: { formatDate: vi.fn(() => '2024-09') },
}))

vi.mock('@/platform/wails/client', async (importOriginal) => ({
  ...(await importOriginal<typeof import('@/platform/wails/client')>()),
  wailsErrorCode: () => '',
  desktopClient: {
    listCompanies: vi.fn(),
    listNFe: vi.fn(),
    statusNFe: vi.fn(),
    listNFePendingManifestacoes: vi.fn(),
    planNFeCiencia: vi.fn(),
    registerNFeCiencia: vi.fn(),
    registerNFeManifestacao: vi.fn(),
    pullNFe: vi.fn(),
    resetNFe: vi.fn(),
    exportNFeXML: vi.fn(),
    exportNFeZIP: vi.fn(),
    markNFeViewed: vi.fn(),
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

const placeholder = 'Filtrar por chave, número, nome ou CNPJ...'

function nfeRow(chave: string, overrides: Partial<NFeRow> = {}): NFeRow {
  return mapNFeRow({
    ID: `rel-${chave}`,
    DocumentID: `doc-${chave}`,
    ChaveAcesso: chave,
    Serie: '1',
    Numero: '1',
    Protocolo: '',
    TpNF: '1',
    EmitenteCNPJ: '12345678000199',
    EmitenteName: 'Fornecedor',
    EmitenteIE: '',
    DestinatarioCNPJ: '',
    DestinatarioName: '',
    TotalValue: 100,
    Situacao: 'autorizada',
    Completeness: 'resumo',
    Manifestacao: 'nenhuma',
    CompanyRole: 'destinatario',
    EventCount: 0,
    DaysLeft: null,
    CienciaDaysLeft: null,
    TacitlyConfirmed: false,
    CienciaBlockReason: '',
    ConclusiveBlockReason: '',
    ...overrides,
  })
}

function status(overrides: Partial<NFeStatusResult> = {}): NFeStatusResult {
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
    IdleDays: 0,
    TotalDestinatario: 0,
    TotalEmitente: 0,
    TotalOutros: 0,
    TotalResumos: 0,
    TotalCompletas: 0,
    PendingCiencia: 0,
    PendingConclusiva: 0,
    CienciaOverdue: 0,
    ...overrides,
  }
}

const destinatario = nfeRow('a')
const emitida = nfeRow('b', {
  CompanyRole: 'emitente',
  DestinatarioName: 'Agropecuária Campo Verde',
  DestinatarioCNPJ: '45091726000115',
  CienciaBlockReason: 'a empresa não é a destinatária',
  ConclusiveBlockReason: 'a empresa não é a destinatária',
})

// mountPage mounts on the active Pinia, so two mounts in a test stand for
// leaving the page and coming back.
function mountPage() {
  return shallowMount(NFePage, {
    global: {
      stubs: {
        DocumentFilterBar: false,
        DocumentPageHeader: false,
        DocumentTableTop: false,
        DocumentFilterSelect: false,
        'q-page': { template: '<div><slot /></div>' },
        'q-tabs': {
          name: 'QTabs',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<div><slot /></div>',
        },
        'q-tab-panels': { template: '<div><slot /></div>' },
        'q-tab-panel': { template: '<div><slot /></div>' },
        'q-banner': { template: '<div class="q-banner-stub"><slot /></div>' },
        'q-btn': {
          name: 'QBtn',
          props: ['label', 'disable', 'loading'],
          emits: ['click'],
          template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}<slot /></button>',
        },
        'q-table': {
          name: 'QTable',
          props: ['rows', 'selected', 'pagination', 'columns', 'noDataLabel'],
          emits: ['update:selected', 'update:pagination'],
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
          props: ['modelValue', 'placeholder'],
          emits: ['update:modelValue'],
          template: '<div><slot /></div>',
        },
        'q-toggle': {
          name: 'QToggle',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
        NFePendingPanel: { name: 'NFePendingPanel', template: '<div />' },
        NFeEventsDialog: { template: '<div />' },
      },
      directives: {
        ClosePopup: {},
      },
    },
  })
}

type Page = ReturnType<typeof mountPage>

function button(wrapper: Page, label: string | RegExp) {
  const found = wrapper.findAllComponents({ name: 'QBtn' }).find((btn) => {
    const text = String(btn.props('label') ?? '')
    return typeof label === 'string' ? text === label : label.test(text)
  })
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

function searchInput(wrapper: Page) {
  const found = wrapper
    .findAllComponents({ name: 'QInput' })
    .find((input) => input.props('placeholder') === placeholder)
  if (!found) throw new Error('search input not found')
  return found
}

function select(wrapper: Page, label: string) {
  const found = wrapper.findAllComponents({ name: 'QSelect' }).find((input) => input.props('label') === label)
  if (!found) throw new Error(`select ${label} not found`)
  return found
}

function table(wrapper: Page) {
  return wrapper.getComponent({ name: 'QTable' })
}

async function selectRows(wrapper: Page, rows: NFeRow[]) {
  table(wrapper).vm.$emit('update:selected', rows)
  await flushPromises()
}

async function filterRows(wrapper: Page, text: string) {
  searchInput(wrapper).vm.$emit('update:modelValue', text)
  await flushPromises()
}

describe('NFePage', () => {
  beforeEach(async () => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listNFe).mockResolvedValue([destinatario, emitida])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue(status())
    vi.mocked(desktopClient.listNFePendingManifestacoes).mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()
  })

  it('lists the NF-e of the first company with its status', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(desktopClient.listNFe).toHaveBeenCalledWith(expect.objectContaining({ CNPJ: company.CNPJ }))
    expect(desktopClient.statusNFe).toHaveBeenCalledWith(company.CNPJ)
    expect(table(wrapper).props('rows')).toEqual([destinatario, emitida])
    expect(table(wrapper).props('noDataLabel')).toBe('Nenhuma NF-e encontrada.')
    expect(wrapper.find('h5').text()).toBe('NF-e')
    expect(wrapper.text()).toContain('NSU 10/10 · Pendências: 0')
  })

  it('asks for a company when there is none', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()
    const wrapper = mountPage()
    await flushPromises()

    expect(desktopClient.listNFe).not.toHaveBeenCalled()
    expect(table(wrapper).props('noDataLabel')).toBe(
      'Nenhuma empresa cadastrada. Cadastre uma em Empresas.'
    )
  })

  it('searches when the competência changes, without reloading the status', async () => {
    mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listNFe).mockClear()
    vi.mocked(desktopClient.statusNFe).mockClear()
    vi.mocked(desktopClient.listNFePendingManifestacoes).mockClear()

    useWorkspaceStore().competence = '2024-08'
    await flushPromises()

    expect(desktopClient.listNFe).toHaveBeenCalledTimes(1)
    expect(desktopClient.listNFe).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: company.CNPJ, Competence: '2024-08' })
    )
    expect(desktopClient.statusNFe).not.toHaveBeenCalled()
    expect(desktopClient.listNFePendingManifestacoes).not.toHaveBeenCalled()
  })

  it('shows the standard document columns in order, sorted by issue date', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const columns = table(wrapper).props('columns') as { name: string; label: string; sortable?: boolean }[]
    expect(columns.map((column) => column.name)).toEqual([...DOCUMENT_COLUMN_NAMES])
    expect(columns.find((column) => column.name === 'destinatario')?.label).toBe('Destinatário')
    expect(columns.find((column) => column.name === 'issueDate')?.sortable).toBe(true)
    expect(table(wrapper).props('pagination')).toMatchObject({ sortBy: 'issueDate', descending: true })
  })

  it('filters the notes by accent- and case-insensitive text, destinatário included', async () => {
    const acentuada = nfeRow('c', { EmitenteName: 'São João Ltda' })
    vi.mocked(desktopClient.listNFe).mockResolvedValue([destinatario, emitida, acentuada])
    const wrapper = mountPage()
    await flushPromises()
    expect(table(wrapper).props('rows')).toHaveLength(3)

    await filterRows(wrapper, 'SAO JOAO')
    expect(table(wrapper).props('rows')).toEqual([acentuada])

    await filterRows(wrapper, 'campo verde')
    expect(table(wrapper).props('rows')).toEqual([emitida])
  })

  it('keeps the text filter, the selection and the tab after leaving the page and coming back', async () => {
    const first = mountPage()
    await flushPromises()
    await filterRows(first, 'campo verde')
    await selectRows(first, [emitida])
    first.getComponent({ name: 'QTabs' }).vm.$emit('update:modelValue', 'pendencias')
    await flushPromises()
    first.unmount()

    const second = mountPage()
    await flushPromises()
    expect(searchInput(second).props('modelValue')).toBe('campo verde')
    expect(table(second).props('rows')).toEqual([emitida])
    expect(table(second).props('selected')).toEqual([emitida])
    expect(second.getComponent({ name: 'QTabs' }).props('modelValue')).toBe('pendencias')
  })

  it('searches with the filters the user picked', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listNFe).mockClear()

    select(wrapper, 'Situação').vm.$emit('update:modelValue', 'cancelada')
    select(wrapper, 'Completude').vm.$emit('update:modelValue', 'completa')
    select(wrapper, 'Manifestação').vm.$emit('update:modelValue', 'ciencia')
    select(wrapper, 'Papel').vm.$emit('update:modelValue', 'emitente')
    await flushPromises()

    await button(wrapper, 'Buscar').trigger('click')
    await flushPromises()

    expect(desktopClient.listNFe).toHaveBeenCalledWith({
      CNPJ: company.CNPJ,
      Competence: '',
      Situacao: 'cancelada',
      Completeness: 'completa',
      Manifestacao: 'ciencia',
      Role: 'emitente',
      EmitenteCNPJ: '',
      OnlyUnread: false,
    })
  })

  it('searches the unviewed NF-e when "Somente não vistos" is turned on', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listNFe).mockClear()

    wrapper.getComponent({ name: 'QToggle' }).vm.$emit('update:modelValue', true)
    await flushPromises()

    expect(desktopClient.listNFe).toHaveBeenCalledWith(expect.objectContaining({ OnlyUnread: true }))
  })

  it('keeps the ciência button disabled without an eligible selection', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const ciencia = () => button(wrapper, /^Registrar ciência/)
    expect(ciencia().props('disable')).toBe(true)

    await selectRows(wrapper, [emitida])
    expect(ciencia().props('label')).toBe('Registrar ciência (0)')
    expect(ciencia().props('disable')).toBe(true)

    await selectRows(wrapper, [emitida, destinatario])
    expect(ciencia().props('label')).toBe('Registrar ciência (1)')
    expect(ciencia().props('disable')).toBe(false)
  })

  it('keeps the ciência button busy while a plan from an earlier mount is in flight', async () => {
    const wrapper = mountPage()
    await flushPromises()
    await selectRows(wrapper, [destinatario])

    useNFeDocumentsStore().planningCiencia = true
    await flushPromises()
    const ciencia = button(wrapper, /^Registrar ciência/)
    expect(ciencia.props('loading')).toBe(true)
    expect(ciencia.props('disable')).toBe(true)
  })

  it('opens the confirm dialog with the eligible chaves from planNFeCiencia', async () => {
    const plan: NFeCienciaPlan = {
      Eligible: [destinatario],
      Skipped: [{ ChaveAcesso: 'b', Reason: 'a empresa não é a destinatária' }],
    }
    vi.mocked(desktopClient.planNFeCiencia).mockResolvedValue(plan)
    vi.mocked(desktopClient.registerNFeCiencia).mockResolvedValue({
      Results: [
        {
          ChaveAcesso: 'a',
          TpEvento: '210210',
          Status: 'registrada',
          CStat: '135',
          XMotivo: '',
          Protocolo: '1',
        },
      ],
      Skipped: [],
      Interrupted: '',
    })

    const wrapper = mountPage()
    await flushPromises()
    await selectRows(wrapper, [destinatario, emitida])

    await button(wrapper, /^Registrar ciência/).trigger('click')
    await flushPromises()

    expect(desktopClient.planNFeCiencia).toHaveBeenCalledWith(company.CNPJ, ['a', 'b'])
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        component: NFeCienciaConfirmDialog,
        componentProps: expect.objectContaining({
          cnpj: company.CNPJ,
          tpAmb: '1',
          plan,
        }),
      })
    )

    okHandlers[0]?.(['a'])
    await flushPromises()

    expect(desktopClient.registerNFeCiencia).toHaveBeenCalledWith(company.CNPJ, ['a'])
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'positive',
        actions: [expect.objectContaining({ label: 'Sincronizar agora' })],
      })
    )
  })

  it('exports the filtered rows as a ZIP with the choice of the dialog', async () => {
    vi.mocked(desktopClient.exportNFeZIP).mockResolvedValue({
      OutPath: 'C:\\exports\\nfe.zip',
      Format: 'zip',
      Incremental: true,
      ExportedCount: 1,
      SkippedResumos: 2,
    })
    const wrapper = mountPage()
    await flushPromises()
    await filterRows(wrapper, 'campo verde')

    await button(wrapper, 'Exportar').trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        componentProps: {
          source: 'nfe',
          count: 1,
          scope: 'listed',
          formats: [{ label: 'XMLs (ZIP)', value: 'zip' }],
          showIncludeResumos: true,
        },
      })
    )
    expect(desktopClient.exportNFeZIP).not.toHaveBeenCalled()

    okHandlers[0]?.({ format: 'zip', incremental: true, includeResumos: false })
    await flushPromises()

    expect(desktopClient.exportNFeZIP).toHaveBeenCalledWith({
      CNPJ: company.CNPJ,
      Competence: '',
      Role: '',
      ChavesAcesso: [emitida.ChaveAcesso],
      IncludeResumos: false,
      Incremental: true,
    })
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: '1 XML exportado para C:\\exports\\nfe.zip.',
    })
    expect(notify).toHaveBeenCalledWith({
      type: 'info',
      message: '2 resumos ignorados: o XML completo ainda não foi baixado.',
    })
  })

  it('exports the selection, with the resumos when asked', async () => {
    vi.mocked(desktopClient.exportNFeZIP).mockResolvedValue(null)
    const wrapper = mountPage()
    await flushPromises()
    await selectRows(wrapper, [destinatario])

    await button(wrapper, 'Exportar').trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        componentProps: expect.objectContaining({ count: 1, scope: 'selected' }),
      })
    )
    okHandlers[0]?.({ format: 'zip', incremental: false, includeResumos: true })
    await flushPromises()

    expect(desktopClient.exportNFeZIP).toHaveBeenCalledWith(
      expect.objectContaining({
        ChavesAcesso: [destinatario.ChaveAcesso],
        IncludeResumos: true,
        Incremental: false,
      })
    )
  })

  it('marks the selected NF-e viewed without asking', async () => {
    vi.mocked(desktopClient.markNFeViewed).mockResolvedValue(1)
    const wrapper = mountPage()
    await flushPromises()
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (2)')

    await selectRows(wrapper, [emitida])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (1)')
    await button(wrapper, /^Marcar vistos/).trigger('click')
    await flushPromises()

    expect(dialog).not.toHaveBeenCalled()
    expect(desktopClient.markNFeViewed).toHaveBeenCalledWith(company.CNPJ, [emitida.ChaveAcesso])
    expect(table(wrapper).props('selected')).toEqual([])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (1)')
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: '1 documento marcado como visto.' })
  })

  it('asks before marking the whole list viewed', async () => {
    vi.mocked(desktopClient.markNFeViewed).mockResolvedValue(2)
    const wrapper = mountPage()
    await flushPromises()

    await button(wrapper, /^Marcar vistos/).trigger('click')
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({ message: 'Marcar como vistas as 2 NF-e novas da lista?' })
    )
    expect(desktopClient.markNFeViewed).not.toHaveBeenCalled()

    okHandlers[0]?.(undefined)
    await flushPromises()

    expect(desktopClient.markNFeViewed).toHaveBeenCalledWith(company.CNPJ, [
      destinatario.ChaveAcesso,
      emitida.ChaveAcesso,
    ])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (0)')
    expect(notify).toHaveBeenCalledWith({ type: 'positive', message: '2 documentos marcados como vistos.' })
  })

  it('does not open the dialog when planNFeCiencia finds no eligible note', async () => {
    vi.mocked(desktopClient.planNFeCiencia).mockResolvedValue({
      Eligible: [],
      Skipped: [{ ChaveAcesso: 'a', Reason: 'Já possui manifestação' }],
    })

    const wrapper = mountPage()
    await flushPromises()
    await selectRows(wrapper, [destinatario])

    await button(wrapper, /^Registrar ciência/).trigger('click')
    await flushPromises()

    expect(dialog).not.toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith(expect.objectContaining({ type: 'warning' }))
  })

  it('disables sync and explains the block while SEFAZ blocks the company', async () => {
    const until = new Date(Date.now() + 60 * 60 * 1000).toISOString()
    vi.mocked(desktopClient.statusNFe).mockResolvedValue(
      status({ NextAllowedAt: until, BlockedReason: 'consumo_indevido' })
    )

    const wrapper = mountPage()
    await flushPromises()

    expect(button(wrapper, 'Sincronizar NF-e').props('disable')).toBe(true)
    expect(wrapper.find('.q-banner-stub').text()).toContain('Consultas bloqueadas pela SEFAZ até')
  })

  it('syncs when the company is not blocked', async () => {
    vi.mocked(desktopClient.pullNFe).mockResolvedValue({
      Status: 'success',
      CompletasSaved: 2,
      ResumosSaved: 1,
      EventsSaved: 3,
      LastNSU: 12,
      MaxNSU: 12,
    } as Awaited<ReturnType<typeof desktopClient.pullNFe>>)
    const wrapper = mountPage()
    await flushPromises()

    expect(wrapper.find('.q-banner-stub').exists()).toBe(false)
    const sync = button(wrapper, 'Sincronizar NF-e')
    expect(sync.props('disable')).toBe(false)
    await sync.trigger('click')
    await flushPromises()

    expect(desktopClient.pullNFe).toHaveBeenCalledWith(company.CNPJ)
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'positive',
        message: expect.stringContaining('2 completas, 1 resumos, 3 eventos'),
      })
    )
  })

  it('resets the company NF-e after the confirmation', async () => {
    vi.mocked(desktopClient.statusNFe).mockResolvedValue(status({ TotalDestinatario: 2, TotalEmitente: 1 }))
    vi.mocked(desktopClient.resetNFe).mockResolvedValue({
      CompanyName: 'Empresa Um',
      CNPJ: company.CNPJ,
      CompanyDocuments: 3,
      Documents: 3,
      Events: 4,
      ExportMarks: 0,
      ManifestacoesKept: 1,
    })

    const wrapper = mountPage()
    await flushPromises()
    await button(wrapper, 'Redefinir NF-e').trigger('click')

    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Redefinir NF-e',
        message: expect.stringContaining('as manifestações registradas na SEFAZ não são afetadas'),
      })
    )
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        message: expect.stringContaining('Remove as NF-e de Empresa Um nos dois ambientes (3 no ambiente atual)'),
      })
    )
    expect(desktopClient.resetNFe).not.toHaveBeenCalled()

    vi.mocked(desktopClient.listNFe).mockClear()
    okHandlers[0]?.(undefined)
    await flushPromises()

    expect(desktopClient.resetNFe).toHaveBeenCalledWith(company.CNPJ)
    expect(desktopClient.listNFe).toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: 'NF-e redefinidas: 3 notas e 4 eventos removidos.',
      caption: '1 manifestações enviadas mantidas no histórico.',
    })
  })

  it('shows the per-note results when a ciência has problems', async () => {
    vi.mocked(desktopClient.planNFeCiencia).mockResolvedValue({ Eligible: [destinatario], Skipped: [] })
    const result: NFeEventBatchResult = {
      Results: [
        {
          ChaveAcesso: 'a',
          TpEvento: '210210',
          Status: 'rejeitada',
          CStat: '573',
          XMotivo: 'Duplicidade',
          Protocolo: '',
        },
      ],
      Skipped: [],
      Interrupted: '',
    }
    vi.mocked(desktopClient.registerNFeCiencia).mockResolvedValue(result)

    const wrapper = mountPage()
    await flushPromises()
    await selectRows(wrapper, [destinatario])
    await button(wrapper, /^Registrar ciência/).trigger('click')
    await flushPromises()
    okHandlers[0]?.(['a'])
    await flushPromises()

    const notice = notify.mock.calls.map(([options]) => options).find((options) => options.actions)
    expect(notice).toMatchObject({ type: 'warning', message: expect.stringContaining('1 rejeitadas') })
    expect(notice.actions[0]).not.toHaveProperty('color')
    expect(dialog).toHaveBeenLastCalledWith({
      component: NFeEventResultsDialog,
      componentProps: { result },
    })
  })

  it.each([
    {
      Status: 'registrada',
      CStat: '135',
      XMotivo: '',
      Protocolo: '135',
      expected: { type: 'positive', message: 'Manifestação registrada. Protocolo 135.' },
    },
    {
      Status: 'ja_registrada',
      CStat: '',
      XMotivo: '',
      Protocolo: '',
      expected: { type: 'info', message: 'A manifestação já estava registrada na SEFAZ.' },
    },
    {
      Status: 'rejeitada',
      CStat: '596',
      XMotivo: 'Prazo vencido',
      Protocolo: '',
      expected: { type: 'negative', message: 'Manifestação rejeitada pela SEFAZ: 596 - Prazo vencido' },
    },
    {
      Status: 'nao_enviada',
      CStat: '',
      XMotivo: 'Sem resposta',
      Protocolo: '',
      expected: { type: 'warning', message: 'Manifestação não enviada. Sem resposta' },
    },
  ] as const)('notifies a $Status manifestação', async ({ expected, ...fields }) => {
    const outcome: NFeEventResult = { ChaveAcesso: 'a', TpEvento: '210200', ...fields }
    vi.mocked(desktopClient.registerNFeManifestacao).mockResolvedValue(outcome)

    const wrapper = mountPage()
    await flushPromises()
    wrapper.getComponent({ name: 'NFePendingPanel' }).vm.$emit('manifest', destinatario)
    expect(dialog).toHaveBeenCalledWith({
      component: NFeManifestacaoDialog,
      componentProps: { note: destinatario, tpAmb: '1' },
    })

    okHandlers[0]?.({ tipo: '210200', justificativa: '' })
    await flushPromises()

    expect(desktopClient.registerNFeManifestacao).toHaveBeenCalledWith({
      CNPJ: company.CNPJ,
      ChaveAcesso: 'a',
      Tipo: '210200',
      Justificativa: '',
    })
    expect(notify).toHaveBeenCalledWith(expected)
  })
})
