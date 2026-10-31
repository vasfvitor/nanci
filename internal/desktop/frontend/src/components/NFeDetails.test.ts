import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import NFeDetails from './NFeDetails.vue'
import type { DetailItem } from './DetailList.vue'
import type { NFeRow } from '@/types/desktop'
import { makeNFeRow } from '@/test/fixtures'
import { formatCurrencyCents } from '@/utils/formatters'

const note = makeNFeRow({
  Numero: '1234',
  Serie: '1',
  NatOp: 'Venda de mercadoria',
  LayoutVersion: '4.00',
  TpAmb: '2',
  Situacao: 'autorizada',
  Protocolo: '135260000000001',
  EmitenteName: 'Fornecedor',
  EmitenteCNPJ: '12345678000199',
  EmitenteIE: '123456789',
  EmitenteUF: 'SP',
  DestinatarioName: 'Empresa Um',
  DestinatarioCNPJ: '98765432000199',
  TotalValue: 150000,
  ICMSValue: 18000,
  IPIValue: 500,
  ParseWarnings: ['campo ausente'],
})

function mountDetails(row: Partial<NFeRow> = {}) {
  return shallowMount(NFeDetails, {
    props: { row: { ...note, ...row } },
    global: {
      stubs: {
        DetailList: { name: 'DetailList', props: ['title', 'items'], template: '<div />' },
        ParseWarnings: { name: 'ParseWarnings', props: ['warnings'], template: '<div />' },
      },
    },
  })
}

type Details = ReturnType<typeof mountDetails>

function items(wrapper: Details, title: string): DetailItem[] {
  const found = wrapper
    .findAllComponents({ name: 'DetailList' })
    .find((list) => list.props('title') === title)
  if (!found) throw new Error(`list ${title} not found`)
  return found.props('items') as DetailItem[]
}

function values(wrapper: Details, title: string) {
  return Object.fromEntries(items(wrapper, title).map((item) => [item.label, item.value]))
}

describe('NFeDetails', () => {
  it('lists the note, its parties and the parse warnings', () => {
    const wrapper = mountDetails()

    expect(wrapper.findAllComponents({ name: 'DetailList' }).map((l) => l.props('title'))).toEqual([
      'Nota',
      'Emitente',
      'Destinatário',
      'Valores',
    ])
    expect(values(wrapper, 'Nota')).toMatchObject({
      Série: '1',
      'Natureza da operação': 'Venda de mercadoria',
      Layout: '4.00',
      Protocolo: '135260000000001',
    })
    expect(values(wrapper, 'Emitente')).toEqual({
      Nome: 'Fornecedor',
      CNPJ: '12.345.678/0001-99',
      IE: '123456789',
      UF: 'SP',
    })
    expect(values(wrapper, 'Destinatário')).toEqual({
      Nome: 'Empresa Um',
      CNPJ: '98.765.432/0001-99',
    })
    expect(wrapper.getComponent({ name: 'ParseWarnings' }).props('warnings')).toEqual([
      'campo ausente',
    ])
  })

  it('shows the taxes of a complete note', () => {
    expect(values(mountDetails({ Completeness: 'completa' }), 'Valores')).toEqual({
      Total: formatCurrencyCents(150000),
      ICMS: formatCurrencyCents(18000),
      IPI: formatCurrencyCents(500),
    })
  })

  it('leaves the taxes of a resumo empty, since it carries only the total', () => {
    expect(values(mountDetails({ Completeness: 'resumo' }), 'Valores')).toEqual({
      Total: formatCurrencyCents(150000),
      ICMS: '',
      IPI: '',
    })
  })

  it('leaves the ambiente and the situação empty when the note has none', () => {
    const nota = values(mountDetails({ TpAmb: '', Situacao: '' }), 'Nota')

    expect(nota['Ambiente']).toBe('')
    expect(nota['Situação']).toBe('')
  })
})
