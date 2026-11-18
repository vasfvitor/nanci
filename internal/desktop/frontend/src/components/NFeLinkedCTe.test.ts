import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import NFeLinkedCTe from './NFeLinkedCTe.vue'
import { desktopClient } from '@/platform/wails/client'
import { makeCTeRow } from '@/test/fixtures'

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCTe: vi.fn(),
  },
}))

const frete = makeCTeRow({
  ChaveAcesso: 'cte-1',
  Numero: '1234',
  EmitenteName: 'Transportadora Fictícia',
  TotalValue: 25000,
})

function mountBlock() {
  return shallowMount(NFeLinkedCTe, {
    props: { cnpj: '123', chave: 'nfe-1' },
    global: {
      stubs: {
        'q-btn': {
          name: 'QBtn',
          props: ['label'],
          emits: ['click'],
          template: `<button @click="$emit('click', $event)">{{ label }}</button>`,
        },
      },
    },
  })
}

describe('NFeLinkedCTe', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('lists each CT-e of the note and asks to open the one clicked', async () => {
    vi.mocked(desktopClient.listCTe).mockResolvedValue([frete])
    const wrapper = mountBlock()
    await flushPromises()

    expect(wrapper.text()).toContain('CT-e 000.001.234')
    expect(wrapper.text()).toContain('Transportadora Fictícia')

    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('open')).toEqual([['cte-1']])
  })

  it('says when no stored CT-e transports the note', async () => {
    vi.mocked(desktopClient.listCTe).mockResolvedValue([])
    const wrapper = mountBlock()
    await flushPromises()

    expect(wrapper.text()).toContain('Nenhum CT-e guardado para esta empresa transporta esta NF-e.')
    expect(wrapper.find('button').exists()).toBe(false)
  })
})
