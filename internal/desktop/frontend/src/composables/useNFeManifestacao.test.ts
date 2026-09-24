import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFeManifestacao } from './useNFeManifestacao'
import { desktopClient } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFeEventBatchResult, NFeEventResult, NFeRow } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listNFe: vi.fn(),
    statusNFe: vi.fn(),
    listNFePendingManifestacoes: vi.fn(),
    planNFeCiencia: vi.fn(),
    registerNFeCiencia: vi.fn(),
    registerNFeManifestacao: vi.fn(),
  },
}))

function nfeRow(chave: string): NFeRow {
  return {
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
  }
}

const batch: NFeEventBatchResult = {
  Results: [],
  Skipped: [],
  Interrupted: '',
}

function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

describe('useNFeManifestacao', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listNFe).mockResolvedValue([nfeRow('a')])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue({} as never)
    vi.mocked(desktopClient.listNFePendingManifestacoes).mockResolvedValue([])
    const store = useNFeDocumentsStore()
    store.filter.CNPJ = '123'
  })

  it('loads pendências and plans ciência for the selected company', async () => {
    const manifestacao = useNFeManifestacao()

    await manifestacao.loadPending()
    await manifestacao.planCiencia(['a'])

    expect(desktopClient.listNFePendingManifestacoes).toHaveBeenCalledWith('123')
    expect(desktopClient.planNFeCiencia).toHaveBeenCalledWith('123', ['a'])
  })

  it('keeps a pending ciência visible across a route remount', async () => {
    const store = useNFeDocumentsStore()
    store.selected = [nfeRow('a'), nfeRow('b')]
    const call = deferred<NFeEventBatchResult>()
    vi.mocked(desktopClient.registerNFeCiencia).mockReturnValue(call.promise)

    const firstPage = useNFeManifestacao()
    const sending = firstPage.registerCiencia(['a', 'b'])

    const remountedPage = useNFeManifestacao()
    expect(remountedPage.cienciaInFlight.value).toEqual(['a', 'b'])
    expect(remountedPage.isChaveBusy('a')).toBe(true)
    expect(remountedPage.isChaveBusy('z')).toBe(false)

    await expect(remountedPage.registerCiencia(['a', 'b'])).resolves.toBeNull()
    await expect(remountedPage.registerCiencia(['z'])).resolves.toBeNull()
    expect(desktopClient.registerNFeCiencia).toHaveBeenCalledTimes(1)
    expect(desktopClient.registerNFeCiencia).toHaveBeenCalledWith('123', ['a', 'b'])

    call.resolve(batch)
    await expect(sending).resolves.toEqual(batch)

    expect(remountedPage.cienciaInFlight.value).toBeNull()
    expect(remountedPage.isChaveBusy('a')).toBe(false)
    expect(store.selected).toEqual([])
    expect(desktopClient.listNFe).toHaveBeenCalledWith(store.listInput)
    expect(desktopClient.listNFePendingManifestacoes).toHaveBeenCalledWith('123')
    expect(store.rows.map((row) => row.ChaveAcesso)).toEqual(['a'])
  })

  it('clears the ciência marker when the call fails', async () => {
    const store = useNFeDocumentsStore()
    store.selected = [nfeRow('a')]
    const call = deferred<NFeEventBatchResult>()
    vi.mocked(desktopClient.registerNFeCiencia).mockReturnValue(call.promise)

    const manifestacao = useNFeManifestacao()
    const sending = manifestacao.registerCiencia(['a'])
    expect(manifestacao.isChaveBusy('a')).toBe(true)

    call.reject(new Error('ERR_CANCELED: senha não informada'))
    await expect(sending).rejects.toThrow('ERR_CANCELED')

    expect(manifestacao.cienciaInFlight.value).toBeNull()
    expect(manifestacao.isChaveBusy('a')).toBe(false)
    expect(store.selected).toHaveLength(1)
  })

  it('does not fail a registered ciência when the refresh fails', async () => {
    vi.mocked(desktopClient.registerNFeCiencia).mockResolvedValue(batch)
    vi.mocked(desktopClient.listNFe).mockRejectedValue(new Error('boom'))

    const manifestacao = useNFeManifestacao()
    await expect(manifestacao.registerCiencia(['a'])).resolves.toEqual(batch)
    expect(desktopClient.listNFePendingManifestacoes).toHaveBeenCalled()
  })

  it('keeps a pending manifestação busy per chave across a route remount', async () => {
    const call = deferred<NFeEventResult>()
    vi.mocked(desktopClient.registerNFeManifestacao).mockReturnValue(call.promise)

    const firstPage = useNFeManifestacao()
    const sending = firstPage.registerManifestacao('a', '210240', '  mercadoria não entregue  ')

    const remountedPage = useNFeManifestacao()
    expect(remountedPage.manifestacaoInFlight.value).toEqual(new Set(['a']))
    expect(remountedPage.isChaveBusy('a')).toBe(true)
    await expect(remountedPage.registerManifestacao('a', '210200')).resolves.toBeNull()
    await expect(remountedPage.registerCiencia(['a'])).resolves.toBeNull()
    expect(desktopClient.registerNFeManifestacao).toHaveBeenCalledTimes(1)
    expect(desktopClient.registerNFeCiencia).not.toHaveBeenCalled()
    expect(desktopClient.registerNFeManifestacao).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'a',
      Tipo: '210240',
      Justificativa: 'mercadoria não entregue',
    })

    const outcome = { ChaveAcesso: 'a', Status: 'registrada' } as NFeEventResult
    call.resolve(outcome)
    await expect(sending).resolves.toEqual(outcome)

    expect(remountedPage.isChaveBusy('a')).toBe(false)
    expect(desktopClient.listNFe).toHaveBeenCalled()
    expect(desktopClient.listNFePendingManifestacoes).toHaveBeenCalledWith('123')
  })

  it('reloads only the manifested note when it is listed', async () => {
    const store = useNFeDocumentsStore()
    store.setRows([nfeRow('a'), nfeRow('b')])
    const fresh = { ...nfeRow('a'), Manifestacao: 'confirmada' as const }
    vi.mocked(desktopClient.listNFe).mockResolvedValue([fresh])
    vi.mocked(desktopClient.registerNFeManifestacao).mockResolvedValue({
      ChaveAcesso: 'a',
      Status: 'registrada',
    } as NFeEventResult)

    await useNFeManifestacao().registerManifestacao('a', '210200')

    expect(desktopClient.listNFe).toHaveBeenCalledTimes(1)
    expect(desktopClient.listNFe).toHaveBeenCalledWith({ ...store.listInput, ChavesAcesso: ['a'] })
    expect(store.rows).toEqual([fresh, nfeRow('b')])
    expect(desktopClient.statusNFe).toHaveBeenCalledWith('123')
    expect(desktopClient.listNFePendingManifestacoes).toHaveBeenCalledWith('123')
  })

  it('searches the whole list when the manifested note is not listed', async () => {
    const store = useNFeDocumentsStore()
    store.setRows([nfeRow('b')])
    vi.mocked(desktopClient.registerNFeManifestacao).mockResolvedValue({
      ChaveAcesso: 'a',
      Status: 'registrada',
    } as NFeEventResult)

    await useNFeManifestacao().registerManifestacao('a', '210200')

    expect(desktopClient.listNFe).toHaveBeenCalledWith(store.listInput)
    expect(store.rows.map((row) => row.ChaveAcesso)).toEqual(['a'])
  })

  it('clears the manifestação marker when the call fails and drops justificativa for other tipos', async () => {
    vi.mocked(desktopClient.registerNFeManifestacao).mockRejectedValue(new Error('boom'))

    const manifestacao = useNFeManifestacao()
    await expect(manifestacao.registerManifestacao('a', '210200', 'ignorada')).rejects.toThrow('boom')

    expect(manifestacao.isChaveBusy('a')).toBe(false)
    expect(desktopClient.registerNFeManifestacao).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'a',
      Tipo: '210200',
      Justificativa: '',
    })
  })
})
