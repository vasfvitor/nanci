import { flushPromises } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import { useNFeLinkedCTe } from './useNFeLinkedCTe'
import { desktopClient } from '@/platform/wails/client'
import type { CTeRow } from '@/types/desktop'
import { deferred, makeCTeRow } from '@/test/fixtures'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCTe: vi.fn(),
  },
}))

const frete = makeCTeRow({ ChaveAcesso: 'cte-1' })

describe('useNFeLinkedCTe', () => {
  beforeEach(() => {
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCTe).mockResolvedValue([frete])
  })

  it('lists the CT-e that transported the note, from every competência', async () => {
    const { rows, loading } = useNFeLinkedCTe(
      () => '123',
      () => 'nfe-1'
    )
    expect(loading.value).toBe(true)
    await flushPromises()

    expect(desktopClient.listCTe).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '',
      Situacao: '',
      Role: '',
      Modelo: '',
      EmitenteCNPJ: '',
      TomadorCNPJ: '',
      NFeChave: 'nfe-1',
      OnlyUnread: false,
    })
    expect(rows.value).toEqual([frete])
    expect(loading.value).toBe(false)
  })

  it('does not search without a company', async () => {
    const { rows } = useNFeLinkedCTe(
      () => '',
      () => 'nfe-1'
    )
    await flushPromises()

    expect(desktopClient.listCTe).not.toHaveBeenCalled()
    expect(rows.value).toEqual([])
  })

  it('keeps only the CT-e of the latest note when an earlier load ends last', async () => {
    const first = deferred<CTeRow[]>()
    const second = makeCTeRow({ ChaveAcesso: 'cte-2' })
    vi.mocked(desktopClient.listCTe).mockReturnValueOnce(first.promise)
    vi.mocked(desktopClient.listCTe).mockResolvedValueOnce([second])
    const chave = ref('nfe-1')

    const { rows, loading } = useNFeLinkedCTe(
      () => '123',
      () => chave.value
    )
    chave.value = 'nfe-2'
    await flushPromises()
    first.resolve([frete])
    await flushPromises()

    expect(rows.value).toEqual([second])
    expect(loading.value).toBe(false)
  })

  it('stops loading when the company goes away while a load is pending', async () => {
    const pending = deferred<CTeRow[]>()
    vi.mocked(desktopClient.listCTe).mockReturnValueOnce(pending.promise)
    const cnpj = ref('123')

    const { loading } = useNFeLinkedCTe(
      () => cnpj.value,
      () => 'nfe-1'
    )
    expect(loading.value).toBe(true)
    cnpj.value = ''
    await flushPromises()
    pending.resolve([frete])
    await flushPromises()

    expect(loading.value).toBe(false)
  })

  it('says when the CT-e could not be loaded', async () => {
    vi.mocked(desktopClient.listCTe).mockRejectedValue(new Error('banco ocupado'))
    const { rows, error, loading } = useNFeLinkedCTe(
      () => '123',
      () => 'nfe-1'
    )
    await flushPromises()

    expect(rows.value).toEqual([])
    expect(error.value).toBe('Não foi possível carregar os CT-e desta NF-e.')
    expect(loading.value).toBe(false)
  })
})
