import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import CTePage from './CTePage.vue'
import ImportResultDialog from '@/components/ImportResultDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CTeRow, CTeStatusResult, ImportXMLResult } from '@/types/desktop'
import { DOCUMENT_COLUMN_NAMES } from '@/utils/documentColumns'
import { CTE_EXPORT_FORMATS } from '@/utils/exportFormats'
import { makeCTeRow } from '@/test/fixtures'

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
    exportDACTE: vi.fn(),
    exportCTeDocuments: vi.fn(),
    markCTeViewed: vi.fn(),
    selectXMLFiles: vi.fn(),
    importCTeXML: vi.fn(),
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

const tomada = makeCTeRow({
  ChaveAcesso: '35260811222333000181570010000012341000012345',
  Modelo: '57',
  TipoDocumento: 'cte',
  HasDACTE: true,
  Numero: '1234',
  EmitenteName: 'Transportadora Fictícia',
  TomadorName: 'Empresa Um',
  CompanyRole: 'tomador',
  Papeis: ['tomador', 'remetente'],
  Situacao: 'autorizada',
  NFeChaves: [nfeChave],
})
const servico = makeCTeRow({
  ChaveAcesso: '35260811222333000181670010000000561000000567',
  Modelo: '67',
  TipoDocumento: 'cte_os',
  HasDACTE: false,
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
    IdleDays: 0,
    TotalTomador: 2,
    TotalDestinatario: 0,
    TotalRemetente: 0,
    TotalOutros: 0,
    ...overrides,
  }
}

// mountPage mounts on the active Pinia, so two mounts in a test stand for
// leaving the page and coming back. stubs adds to or replaces the default
// stubs.
function mountPage(stubs: Record<string, unknown> = {}) {
  return shallowMount(CTePage, {
    global: {
      stubs: {
        DocumentFilterBar: false,
        DocumentPageHeader: false,
        DocumentTableTop: false,
        DocumentFilterSelect: false,
        'q-page': { template: '<div><slot /></div>' },
        'q-banner': { template: '<div class="q-banner-stub"><slot /></div>' },
        'q-btn': {
          name: 'QBtn',
          props: ['label', 'disable', 'loading'],
          emits: ['click'],
          template:
            '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}<slot /></button>',
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
        ...stubs,
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

// typeFilter types text in the filter box and leaves it, which hands the
// text to the page without waiting for the debounce.
async function typeFilter(wrapper: Page, text: string) {
  const input = field(wrapper, 'QInput', placeholder)
  input.vm.$emit('update:modelValue', text)
  await input.trigger('blur')
  await flushPromises()
}

async function select(wrapper: Page, rows: CTeRow[]) {
  table(wrapper).vm.$emit('update:selected', rows)
  await flushPromises()
}

describe('CTePage', () => {
  beforeEach(async () => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listCTe).mockResolvedValue([tomada, servico])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(status())
    await useWorkspaceStore().loadCompanies()
  })

  it('lists the CT-e of the first company with its status', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: company.CNPJ })
    )
    expect(desktopClient.statusCTe).toHaveBeenCalledWith(company.CNPJ)
    expect(table(wrapper).props('rows')).toEqual([tomada, servico])
    expect(table(wrapper).props('noDataLabel')).toBe('Nenhum CT-e encontrado.')
    expect(wrapper.text()).toContain('NSU 10/10 · CT-e: 2')
  })

  it('asks for a company when there is none', async () => {
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()
    const wrapper = mountPage()
    await flushPromises()

    expect(desktopClient.listCTe).not.toHaveBeenCalled()
    expect(table(wrapper).props('noDataLabel')).toBe(
      'Nenhuma empresa cadastrada. Cadastre uma em Empresas.'
    )
  })

  it('searches when the competência changes, without reloading the status', async () => {
    mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCTe).mockClear()
    vi.mocked(desktopClient.statusCTe).mockClear()

    useWorkspaceStore().competence = '2024-08'
    await flushPromises()

    expect(desktopClient.listCTe).toHaveBeenCalledTimes(1)
    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: company.CNPJ, Competence: '2024-08' })
    )
    expect(desktopClient.statusCTe).not.toHaveBeenCalled()
  })

  it('clears the selection when the company changes', async () => {
    const outra = { ...company, ID: 'company-2', CNPJ: '12345678000100', Name: 'Outra' }
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company, outra])
    await useWorkspaceStore().loadCompanies()
    const wrapper = mountPage()
    await flushPromises()
    await select(wrapper, [tomada])

    useWorkspaceStore().cnpj = outra.CNPJ
    await flushPromises()

    expect(desktopClient.listCTe).toHaveBeenLastCalledWith(
      expect.objectContaining({ CNPJ: outra.CNPJ })
    )
    expect(table(wrapper).props('selected')).toEqual([])
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

    await typeFilter(wrapper, 'FICTICIA')
    expect(table(wrapper).props('rows')).toEqual([tomada])
  })

  it('keeps the text filter and the selection after leaving the page and coming back', async () => {
    const first = mountPage()
    await flushPromises()
    field(first, 'QInput', placeholder).vm.$emit('update:modelValue', 'FICTICIA')
    await select(first, [tomada])
    first.unmount()

    const second = mountPage()
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

    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ OnlyUnread: true })
    )
  })

  it('does not search with an NF-e key that is not 44 characters', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCTe).mockClear()

    field(wrapper, 'QInput', 'Chave de NF-e transportada').vm.$emit(
      'update:modelValue',
      '3526 0911'
    )
    await flushPromises()

    const chave = field(wrapper, 'QInput', 'Chave de NF-e transportada')
    expect(chave.props('error')).toBe(true)
    expect(chave.props('errorMessage')).toBe('A chave de NF-e tem 44 caracteres')
    expect(button(wrapper, 'Buscar').props('disable')).toBe(true)
    await chave.trigger('keyup', { key: 'Enter' })
    expect(desktopClient.listCTe).not.toHaveBeenCalled()
  })

  it('searches on Enter in the CNPJ do tomador', async () => {
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCTe).mockClear()

    const tomador = field(wrapper, 'QInput', 'CNPJ do tomador')
    tomador.vm.$emit('update:modelValue', '12345678000199')
    await tomador.trigger('keyup', { key: 'Enter' })
    await flushPromises()

    expect(desktopClient.listCTe).toHaveBeenCalledWith(
      expect.objectContaining({ TomadorCNPJ: '12345678000199' })
    )
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

    expect(desktopClient.pullCTe).toHaveBeenCalledWith(company.CNPJ, expect.any(AbortSignal))
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'positive',
        message: expect.stringContaining('3 CT-e e 1 eventos'),
      })
    )
  })

  it.each([
    { format: 'csv', noun: 'documento' },
    { format: 'xlsx', noun: 'documento' },
    { format: 'zip', noun: 'XML' },
    { format: 'dacte-zip', noun: 'DACTE' },
  ])(
    'exports the filtered rows as $format with the choice of the dialog',
    async ({ format, noun }) => {
      vi.mocked(desktopClient.exportCTeDocuments).mockResolvedValue({
        OutPath: 'C:\\exports\\cte',
        Format: 'xml',
        Incremental: true,
        ExportedCount: 1,
        SkippedSemDACTE: 0,
      })
      const wrapper = mountPage()
      await flushPromises()
      await typeFilter(wrapper, 'serra')

      await button(wrapper, 'Exportar').trigger('click')
      expect(dialog).toHaveBeenCalledWith(
        expect.objectContaining({
          componentProps: {
            source: 'cte',
            count: 1,
            scope: 'listed',
            formats: CTE_EXPORT_FORMATS,
            initialFormat: '',
          },
        })
      )
      expect(desktopClient.exportCTeDocuments).not.toHaveBeenCalled()

      okHandlers[0]?.({ format, incremental: true, includeResumos: false })
      await flushPromises()

      expect(desktopClient.exportCTeDocuments).toHaveBeenCalledWith({
        CNPJ: company.CNPJ,
        Competence: '',
        Role: '',
        Format: format,
        ChavesAcesso: [servico.ChaveAcesso],
        Incremental: true,
      })
      expect(notify).toHaveBeenCalledTimes(1)
      expect(notify).toHaveBeenCalledWith({
        type: 'positive',
        message: `1 ${noun} exportado para C:\\exports\\cte.`,
        actions: [expect.objectContaining({ label: 'Abrir pasta' })],
      })
    }
  )

  it.each([
    { skipped: 1, message: '1 documento ignorado: este tipo de documento não tem DACTE.' },
    { skipped: 2, message: '2 documentos ignorados: estes tipos de documento não têm DACTE.' },
  ])('reports the documents without DACTE left out ($skipped)', async ({ skipped, message }) => {
    vi.mocked(desktopClient.exportCTeDocuments).mockResolvedValue({
      OutPath: 'C:\\exports\\dactes.zip',
      Format: 'dacte',
      Incremental: false,
      ExportedCount: 1,
      SkippedSemDACTE: skipped,
    })
    const wrapper = mountPage()
    await flushPromises()

    await button(wrapper, 'Exportar').trigger('click')
    okHandlers[0]?.({ format: 'dacte-zip', incremental: false, includeResumos: false })
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: '1 DACTE exportado para C:\\exports\\dactes.zip.',
      actions: [expect.objectContaining({ label: 'Abrir pasta' })],
    })
    expect(notify).toHaveBeenCalledWith({ type: 'info', message })
  })

  it('says when an export found nothing to export', async () => {
    vi.mocked(desktopClient.exportCTeDocuments).mockResolvedValue({
      OutPath: '',
      Format: 'csv',
      Incremental: false,
      ExportedCount: 0,
      SkippedSemDACTE: 0,
    })
    const wrapper = mountPage()
    await flushPromises()

    await button(wrapper, 'Exportar').trigger('click')
    okHandlers[0]?.({ format: 'csv', incremental: false, includeResumos: false })
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({
      type: 'info',
      message: 'Nenhum documento para exportar.',
    })
    expect(notify).toHaveBeenCalledTimes(1)
  })

  it('exports the selection when there is one', async () => {
    vi.mocked(desktopClient.exportCTeDocuments).mockResolvedValue(null)
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

    expect(desktopClient.exportCTeDocuments).toHaveBeenCalledWith(
      expect.objectContaining({
        Format: 'zip',
        ChavesAcesso: [tomada.ChaveAcesso],
        Incremental: false,
      })
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
    expect(desktopClient.markCTeViewed).toHaveBeenCalledWith(company.CNPJ, [servico.ChaveAcesso])
    expect(table(wrapper).props('selected')).toEqual([])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (1)')
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: '1 documento marcado como visto.',
    })
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

    expect(desktopClient.markCTeViewed).toHaveBeenCalledWith(company.CNPJ, [
      tomada.ChaveAcesso,
      servico.ChaveAcesso,
    ])
    expect(button(wrapper, /^Marcar vistos/).props('label')).toBe('Marcar vistos (0)')
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: '2 documentos marcados como vistos.',
    })
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
      expect.objectContaining({
        type: 'positive',
        message: expect.stringContaining('2 CT-e e 3 eventos'),
      })
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

// The row menu renders through the body slot of the table, which the default
// q-table stub leaves out.
describe('CTePage row menu', () => {
  const rowStubs = {
    'q-table': {
      name: 'QTable',
      props: ['rows'],
      template: `<div><div v-for="row in rows" :key="row.ChaveAcesso">
        <slot name="body" :row="row" :cols="[{ name: 'acoes' }]" :expand="false" />
      </div></div>`,
    },
    RowActionsMenu: { template: '<div><slot /></div>' },
    RowMenuItem: {
      name: 'RowMenuItem',
      props: ['label', 'caption', 'disable'],
      emits: ['click'],
      template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
    },
  }

  beforeEach(async () => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(status())
    await useWorkspaceStore().loadCompanies()
  })

  function menuItem(wrapper: Page, label: string) {
    const found = wrapper
      .findAllComponents({ name: 'RowMenuItem' })
      .find((item) => item.props('label') === label)
    if (!found) throw new Error(`menu item ${label} not found`)
    return found
  }

  it('keeps "Exportar DACTE" disabled on a kind without DACTE and says why', async () => {
    vi.mocked(desktopClient.listCTe).mockResolvedValue([servico])
    const wrapper = mountPage(rowStubs)
    await flushPromises()

    const item = menuItem(wrapper, 'Exportar DACTE')
    expect(item.props('disable')).toBe(true)
    expect(item.props('caption')).toBe('Este tipo de documento não tem DACTE')
  })

  it('exports the DACTE of a CT-e and says so', async () => {
    vi.mocked(desktopClient.listCTe).mockResolvedValue([tomada])
    vi.mocked(desktopClient.exportDACTE).mockResolvedValue({
      OutPath: 'C:\\exports\\dacte.pdf',
      Format: 'dacte',
      Incremental: false,
      ExportedCount: 1,
    })
    const wrapper = mountPage(rowStubs)
    await flushPromises()

    const item = menuItem(wrapper, 'Exportar DACTE')
    expect(item.props('disable')).toBe(false)
    expect(item.props('caption')).toBe('')
    await item.trigger('click')
    await flushPromises()

    expect(desktopClient.exportDACTE).toHaveBeenCalledWith({
      CNPJ: company.CNPJ,
      ChaveAcesso: tomada.ChaveAcesso,
    })
    expect(notify).toHaveBeenCalledWith({
      type: 'positive',
      message: '1 DACTE exportado para C:\\exports\\dacte.pdf.',
      actions: [expect.objectContaining({ label: 'Abrir pasta' })],
    })
  })
})

describe('CTePage import', () => {
  const withRejected: ImportXMLResult = {
    Files: [
      {
        Path: 'C:\\XML\\a.xml',
        Kind: 'completa',
        ChaveAcesso: tomada.ChaveAcesso,
        Outcome: 'updated',
        Message: '',
      },
      {
        Path: 'C:\\XML\\b.xml',
        Kind: '',
        ChaveAcesso: '',
        Outcome: 'rejected',
        Message: 'o XML não cita a empresa',
      },
    ],
    Imported: 0,
    Updated: 1,
    Rejected: 1,
  }

  beforeEach(async () => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listCTe).mockResolvedValue([tomada])
    vi.mocked(desktopClient.statusCTe).mockResolvedValue(status())
    await useWorkspaceStore().loadCompanies()
  })

  it('offers the import, disabled without a company, and no fetch by chave', async () => {
    const wrapper = mountPage()
    await flushPromises()
    expect(button(wrapper, 'Importar XML').props('disable')).toBe(false)
    expect(() => button(wrapper, 'Buscar pela chave')).toThrow()

    vi.mocked(desktopClient.listCompanies).mockResolvedValue([])
    await useWorkspaceStore().loadCompanies()
    const empty = mountPage()
    await flushPromises()
    expect(button(empty, 'Importar XML').props('disable')).toBe(true)
  })

  it('imports the picked XML and lists the files when some were rejected', async () => {
    vi.mocked(desktopClient.selectXMLFiles).mockResolvedValue(['C:\\XML\\a.xml', 'C:\\XML\\b.xml'])
    vi.mocked(desktopClient.importCTeXML).mockResolvedValue(withRejected)
    const wrapper = mountPage()
    await flushPromises()
    vi.mocked(desktopClient.listCTe).mockClear()

    await button(wrapper, 'Importar XML').trigger('click')
    await flushPromises()

    expect(desktopClient.importCTeXML).toHaveBeenCalledWith(company.CNPJ, [
      'C:\\XML\\a.xml',
      'C:\\XML\\b.xml',
    ])
    expect(desktopClient.listCTe).toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith({
      type: 'warning',
      message: 'Importação: 0 importados, 1 atualizado, 1 rejeitado.',
    })
    expect(dialog).toHaveBeenCalledWith({
      component: ImportResultDialog,
      componentProps: { result: withRejected, source: 'cte' },
    })
  })

  it('reports a failed import', async () => {
    vi.mocked(desktopClient.selectXMLFiles).mockResolvedValue(['C:\\XML\\a.xml'])
    vi.mocked(desktopClient.importCTeXML).mockRejectedValue(new Error('empresa não encontrada'))
    const wrapper = mountPage()
    await flushPromises()

    await button(wrapper, 'Importar XML').trigger('click')
    await flushPromises()

    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'negative',
        message: expect.stringContaining('Erro ao importar XML'),
      })
    )
    expect(dialog).not.toHaveBeenCalled()
  })

  it('marks an imported CT-e next to its chave', async () => {
    const imported = makeCTeRow({ ChaveAcesso: 'imp', FirstSeenNSU: null })
    const distributed = makeCTeRow({ ChaveAcesso: 'dist', FirstSeenNSU: 7 })
    vi.mocked(desktopClient.listCTe).mockResolvedValue([imported, distributed])
    const wrapper = mountPage({
      'q-table': {
        name: 'QTable',
        props: ['rows'],
        template: `<div><div v-for="row in rows" :key="row.ChaveAcesso" :class="'row-' + row.ChaveAcesso">
          <slot name="body" :row="row" :cols="[{ name: 'chave' }]" :expand="false" />
        </div></div>`,
      },
      ChaveCell: { template: '<span />' },
      'q-badge': {
        name: 'QBadge',
        props: ['label'],
        template: '<span class="badge">{{ label }}</span>',
      },
    })
    await flushPromises()

    expect(wrapper.find('.row-imp .badge').text()).toBe('Importado')
    expect(wrapper.find('.row-dist .badge').exists()).toBe(false)
  })
})
