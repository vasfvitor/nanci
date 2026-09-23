import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { useNFeManifestation } from './useNFeManifestation'
import { desktopClient } from '@/platform/wails/client'
import { useNFeDocumentsStore } from '@/stores/nfeDocuments'
import type { NFeEventBatchResult, NFeEventResult, NFeRow } from '@/types/desktop'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listNFe: vi.fn(),
    statusNFe: vi.fn(),
    listPendingManifestations: vi.fn(),
    planCiencia: vi.fn(),
    registerCiencia: vi.fn(),
    registerManifestation: vi.fn(),
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

describe('useNFeManifestation', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listNFe).mockResolvedValue([nfeRow('a')])
    vi.mocked(desktopClient.statusNFe).mockResolvedValue({} as never)
    vi.mocked(desktopClient.listPendingManifestations).mockResolvedValue([])
    const store = useNFeDocumentsStore()
    store.filter.CNPJ = '123'
  })

  it('loads pendências and plans ciência for the selected company', async () => {
    const manifestation = useNFeManifestation()

    await manifestation.loadPending()
    await manifestation.planCiencia(['a'])

    expect(desktopClient.listPendingManifestations).toHaveBeenCalledWith('123')
    expect(desktopClient.planCiencia).toHaveBeenCalledWith('123', ['a'])
  })

  it('keeps a pending ciência visible across a route remount', async () => {
    const store = useNFeDocumentsStore()
    store.selected = [nfeRow('a'), nfeRow('b')]
    const call = deferred<NFeEventBatchResult>()
    vi.mocked(desktopClient.registerCiencia).mockReturnValue(call.promise)

    const firstPage = useNFeManifestation()
    const sending = firstPage.registerCiencia(['a', 'b'])

    const remountedPage = useNFeManifestation()
    expect(remountedPage.cienciaInFlight.value).toEqual(['a', 'b'])
    expect(remountedPage.isChaveBusy('a')).toBe(true)
    expect(remountedPage.isChaveBusy('z')).toBe(false)

    await expect(remountedPage.registerCiencia(['a', 'b'])).resolves.toBeNull()
    await expect(remountedPage.registerCiencia(['z'])).resolves.toBeNull()
    expect(desktopClient.registerCiencia).toHaveBeenCalledTimes(1)
    expect(desktopClient.registerCiencia).toHaveBeenCalledWith('123', ['a', 'b'])

    call.resolve(batch)
    await expect(sending).resolves.toEqual(batch)

    expect(remountedPage.cienciaInFlight.value).toBeNull()
    expect(remountedPage.isChaveBusy('a')).toBe(false)
    expect(store.selected).toEqual([])
    expect(desktopClient.listNFe).toHaveBeenCalledWith(store.listInput)
    expect(desktopClient.listPendingManifestations).toHaveBeenCalledWith('123')
    expect(store.rows.map((row) => row.ChaveAcesso)).toEqual(['a'])
  })

  it('clears the ciência marker when the call fails', async () => {
    const store = useNFeDocumentsStore()
    store.selected = [nfeRow('a')]
    const call = deferred<NFeEventBatchResult>()
    vi.mocked(desktopClient.registerCiencia).mockReturnValue(call.promise)

    const manifestation = useNFeManifestation()
    const sending = manifestation.registerCiencia(['a'])
    expect(manifestation.isChaveBusy('a')).toBe(true)

    call.reject(new Error('ERR_CANCELED: senha não informada'))
    await expect(sending).rejects.toThrow('ERR_CANCELED')

    expect(manifestation.cienciaInFlight.value).toBeNull()
    expect(manifestation.isChaveBusy('a')).toBe(false)
    expect(store.selected).toHaveLength(1)
  })

  it('does not fail a registered ciência when the refresh fails', async () => {
    vi.mocked(desktopClient.registerCiencia).mockResolvedValue(batch)
    vi.mocked(desktopClient.listNFe).mockRejectedValue(new Error('boom'))

    const manifestation = useNFeManifestation()
    await expect(manifestation.registerCiencia(['a'])).resolves.toEqual(batch)
    expect(desktopClient.listPendingManifestations).toHaveBeenCalled()
  })

  it('keeps a pending manifestação busy per chave across a route remount', async () => {
    const call = deferred<NFeEventResult>()
    vi.mocked(desktopClient.registerManifestation).mockReturnValue(call.promise)

    const firstPage = useNFeManifestation()
    const sending = firstPage.registerManifestation('a', '210240', '  mercadoria não entregue  ')

    const remountedPage = useNFeManifestation()
    expect(remountedPage.manifestationInFlight.value).toEqual(new Set(['a']))
    expect(remountedPage.isChaveBusy('a')).toBe(true)
    await expect(remountedPage.registerManifestation('a', '210200')).resolves.toBeNull()
    await expect(remountedPage.registerCiencia(['a'])).resolves.toBeNull()
    expect(desktopClient.registerManifestation).toHaveBeenCalledTimes(1)
    expect(desktopClient.registerCiencia).not.toHaveBeenCalled()
    expect(desktopClient.registerManifestation).toHaveBeenCalledWith({
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
    expect(desktopClient.listPendingManifestations).toHaveBeenCalledWith('123')
  })

  it('clears the manifestação marker when the call fails and drops justificativa for other tipos', async () => {
    vi.mocked(desktopClient.registerManifestation).mockRejectedValue(new Error('boom'))

    const manifestation = useNFeManifestation()
    await expect(manifestation.registerManifestation('a', '210200', 'ignorada')).rejects.toThrow('boom')

    expect(manifestation.isChaveBusy('a')).toBe(false)
    expect(desktopClient.registerManifestation).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'a',
      Tipo: '210200',
      Justificativa: '',
    })
  })
})
