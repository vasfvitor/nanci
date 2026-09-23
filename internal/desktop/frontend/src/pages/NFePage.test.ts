import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NFePage from './NFePage.vue'
import CienciaConfirmDialog from '@/components/CienciaConfirmDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import type { NFeCienciaPlan, NFeRow, NFeStatusResult } from '@/types/desktop'

type OkHandler = (payload: unknown) => void

const notify = vi.fn()
const okHandlers: OkHandler[] = []
const dialog = vi.fn(() => ({
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
    listPendingManifestations: vi.fn(),
    planCiencia: vi.fn(),
    registerCiencia: vi.fn(),
    pullNFe: vi.fn(),
    resetNFe: vi.fn(),
    exportNFeZIP: vi.fn(),
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

function nfeRow(chave: string, overrides: Partial<NFeRow> = {}): NFeRow {
  return {
    ID: `rel-${chave}`,
    DocumentID: `doc-${chave}`,
    ChaveAcesso: chave,
    Serie: '1',
    Numero: '1',
    Protocolo: '',
    TipoOperacao: '1',
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
    TacitlyConfirmed: false,
    CienciaBlockReason: '',
    ConclusiveBlockReason: '',
    ...overrides,
  }
}

function status(overrides: Partial<NFeStatusResult> = {}): NFeStatusResult {
  return {
    CompanyName: 'Empresa Um',
    CNPJ: company.CNPJ,
    UF: 'SP',
    TpAmb: '1',
    LastCheckedNSU: 10,
    MaxNSU: 10,
    LastRunStatus: 'completed',
    LastRunStopReason: '',
    NextAllowedAt: null,
    BlockedReason: '',
    RequestsLastHour: 1,
    RequestBudget: 20,
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
  CienciaBlockReason: 'a empresa não é a destinatária',
  ConclusiveBlockReason: 'a empresa não é a destinatária',
})

function mountPage() {
  return shallowMount(NFePage, {
    global: {
      stubs: {
        'q-page': { template: '<div><slot /></div>' },
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
          props: ['rows', 'selected', 'pagination'],
          emits: ['update:selected', 'update:pagination'],
          template: '<div><slot name="top" /></div>',
        },
        'q-input': {
          name: 'QInput',
          props: ['modelValue', 'placeholder'],
          emits: ['update:modelValue'],
          template: '<div><slot /></div>',
        },
        NFePendingPanel: { template: '<div />' },
        NFeEventsDialog: { template: '<div />' },
      },
      directives: {
        ClosePopup: {},
      },
    },
  })
}

function buttonStartingWith(wrapper: ReturnType<typeof mountPage>, label: string) {
  const found = wrapper
    .findAllComponents({ name: 'QBtn' })
    .find((btn) => String(btn.props('label') ?? '').startsWith(label))
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

async function selectRows(wrapper: ReturnType<typeof mountPage>, rows: NFeRow[]) {
  wrapper.getComponent({ name: 'QTable' }).vm.$emit('update:selected', rows)
  await flushPromises()
}

describe('NFePage', () => {
  beforeEach(() => {
    localStorage.clear()
    setActivePinia(createPinia())
    vi.clearAllMocks()
    okHandlers.length = 0
    vi.mocked(desktopClient.listCompanies).mockResolvedValue([company])
    vi.mocked(desktopClient.listNFe).mockResolvedValue([destinatario, emitida])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue(status())
    vi.mocked(desktopClient.listPendingManifestations).mockResolvedValue([])
  })

  it('filters the notes by accent- and case-insensitive text', async () => {
    const acentuada = nfeRow('c', { EmitenteName: 'São João Ltda' })
    vi.mocked(desktopClient.listNFe).mockResolvedValue([destinatario, emitida, acentuada])
    const wrapper = mountPage()
    await flushPromises()

    const table = () => wrapper.getComponent({ name: 'QTable' })
    const search = wrapper
      .findAllComponents({ name: 'QInput' })
      .find((input) => String(input.props('placeholder') ?? '').startsWith('Filtrar'))
    expect(table().props('rows')).toHaveLength(3)

    search?.vm.$emit('update:modelValue', 'SAO JOAO')
    await flushPromises()
    expect(table().props('rows')).toEqual([acentuada])

    search?.vm.$emit('update:modelValue', 'b')
    await flushPromises()
    expect(table().props('rows')).toEqual([emitida])
  })

  it('keeps the ciência button disabled without an eligible selection', async () => {
    const wrapper = mountPage()
    await flushPromises()

    const button = () => buttonStartingWith(wrapper, 'Registrar ciência')
    expect(button().props('disable')).toBe(true)

    await selectRows(wrapper, [emitida])
    expect(button().props('label')).toBe('Registrar ciência (0)')
    expect(button().props('disable')).toBe(true)

    await selectRows(wrapper, [emitida, destinatario])
    expect(button().props('label')).toBe('Registrar ciência (1)')
    expect(button().props('disable')).toBe(false)
  })

  it('opens the confirm dialog with the eligible chaves from planCiencia', async () => {
    const plan: NFeCienciaPlan = {
      Eligible: [destinatario],
      Skipped: [{ ChaveAcesso: 'b', Reason: 'a empresa não é a destinatária' }],
    }
    vi.mocked(desktopClient.planCiencia).mockResolvedValue(plan)
    vi.mocked(desktopClient.registerCiencia).mockResolvedValue({
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

    await buttonStartingWith(wrapper, 'Registrar ciência').trigger('click')
    await flushPromises()

    expect(desktopClient.planCiencia).toHaveBeenCalledWith(company.CNPJ, ['a', 'b'])
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        component: CienciaConfirmDialog,
        componentProps: expect.objectContaining({
          cnpj: company.CNPJ,
          tpAmb: '1',
          plan,
        }),
      })
    )

    okHandlers[0]?.(['a'])
    await flushPromises()

    expect(desktopClient.registerCiencia).toHaveBeenCalledWith(company.CNPJ, ['a'])
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({
        type: 'positive',
        actions: [expect.objectContaining({ label: 'Sincronizar agora' })],
      })
    )
  })

  it('exports the rows the grid shows when nothing is selected', async () => {
    vi.mocked(desktopClient.exportNFeZIP).mockResolvedValue(null)
    const wrapper = mountPage()
    await flushPromises()

    const search = wrapper
      .findAllComponents({ name: 'QInput' })
      .find((input) => String(input.props('placeholder') ?? '').startsWith('Filtrar'))
    if (!search) throw new Error('search input not found')
    search.vm.$emit('update:modelValue', 'b')
    await flushPromises()

    await buttonStartingWith(wrapper, 'Exportar XML (ZIP)').trigger('click')
    await flushPromises()
    expect(desktopClient.exportNFeZIP).toHaveBeenLastCalledWith(
      expect.objectContaining({ CNPJ: company.CNPJ, Competence: '', Role: '', ChavesAcesso: ['b'] })
    )

    await selectRows(wrapper, [destinatario])
    await buttonStartingWith(wrapper, 'Exportar XML (ZIP)').trigger('click')
    await flushPromises()
    expect(desktopClient.exportNFeZIP).toHaveBeenLastCalledWith(
      expect.objectContaining({ ChavesAcesso: ['a'] })
    )
  })

  it('does not open the dialog when planCiencia finds no eligible note', async () => {
    vi.mocked(desktopClient.planCiencia).mockResolvedValue({
      Eligible: [],
      Skipped: [{ ChaveAcesso: 'a', Reason: 'Já possui manifestação' }],
    })

    const wrapper = mountPage()
    await flushPromises()
    await selectRows(wrapper, [destinatario])

    await buttonStartingWith(wrapper, 'Registrar ciência').trigger('click')
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

    expect(buttonStartingWith(wrapper, 'Sincronizar NF-e').props('disable')).toBe(true)
    expect(wrapper.find('.q-banner-stub').text()).toContain('Consultas bloqueadas pela SEFAZ até')
  })

  it('enables sync when the company is not blocked', async () => {
    const wrapper = mountPage()
    await flushPromises()

    expect(buttonStartingWith(wrapper, 'Sincronizar NF-e').props('disable')).toBe(false)
    expect(wrapper.find('.q-banner-stub').exists()).toBe(false)
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
      ManifestationsKept: 1,
    })

    const wrapper = mountPage()
    await flushPromises()
    await buttonStartingWith(wrapper, 'Redefinir NF-e').trigger('click')

    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({
        title: 'Redefinir NF-e',
        message: expect.stringContaining('as manifestações registradas na SEFAZ não são afetadas'),
      })
    )
    expect(dialog).toHaveBeenCalledWith(
      expect.objectContaining({ message: expect.stringContaining('Remove as 3 NF-e de Empresa Um') })
    )
    expect(desktopClient.resetNFe).not.toHaveBeenCalled()

    vi.mocked(desktopClient.listNFe).mockClear()
    okHandlers[0]?.(undefined)
    await flushPromises()

    expect(desktopClient.resetNFe).toHaveBeenCalledWith(company.CNPJ)
    expect(desktopClient.listNFe).toHaveBeenCalled()
    expect(notify).toHaveBeenCalledWith(
      expect.objectContaining({ type: 'positive', message: expect.stringContaining('3 notas e 4 eventos') })
    )
  })
})
