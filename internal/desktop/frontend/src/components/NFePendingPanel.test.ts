import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import ChaveCell from './ChaveCell.vue'
import NFePendingPanel from './NFePendingPanel.vue'
import PartyCell from './PartyCell.vue'
import type { NFePendingRow } from '@/types/desktop'

vi.mock('quasar', () => ({ useQuasar: () => ({ dark: { isActive: false } }) }))

function daysFromNow(days: number) {
  return new Date(Date.now() + days * 24 * 60 * 60 * 1000).toISOString()
}

function row(chave: string, fields: Partial<NFePendingRow>): NFePendingRow {
  return {
    ChaveAcesso: chave,
    Kind: 'sem_conclusiva',
    Manifestacao: 'ciencia',
    DaysLeft: 0,
    CienciaDaysLeft: null,
    TacitlyConfirmed: false,
    CienciaOverdue: false,
    ...fields,
  } as NFePendingRow
}

function mountPanel(rows: NFePendingRow[]) {
  return shallowMount(NFePendingPanel, {
    props: { rows, loading: false, busy: () => false },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QTable: {
          props: ['rows'],
          template:
            '<div><div v-for="r in rows" :key="r.ChaveAcesso" :data-chave="r.ChaveAcesso"><slot name="body-cell-prazo" :row="r" /></div></div>',
        },
        QTd: { template: '<div><slot /></div>' },
        QChip: { props: ['label'], template: '<span class="chip">{{ label }}</span>' },
      },
    },
  })
}

describe('NFePendingPanel', () => {
  it('labels expired rows as tacitly confirmed', () => {
    const wrapper = mountPanel([
      row('on-time', { ConclusiveDue: daysFromNow(20), DaysLeft: 20 }),
      row('due-today', { ConclusiveDue: daysFromNow(0), DaysLeft: 0 }),
      row('expired', { ConclusiveDue: daysFromNow(-3), DaysLeft: -3, TacitlyConfirmed: true }),
      row('expired-sem-ciencia', {
        Kind: 'sem_ciencia',
        Manifestacao: 'nenhuma',
        CienciaDue: daysFromNow(-83),
        CienciaDaysLeft: -83,
        CienciaOverdue: true,
        ConclusiveDue: daysFromNow(-3),
        DaysLeft: -3,
        TacitlyConfirmed: true,
      }),
    ])

    const chip = (chave: string) => wrapper.find(`[data-chave="${chave}"] .chip`).text()
    expect(chip('on-time')).toBe('20 d restantes')
    expect(chip('due-today')).toBe('Vence hoje')
    expect(chip('expired')).toBe('Confirmada tacitamente')
    expect(chip('expired-sem-ciencia')).toBe('Confirmada tacitamente')
    expect(wrapper.text()).toMatch(/até 90 dias da\s+autorização/)
    expect(wrapper.text()).toMatch(/confirmada por lei/)
  })

  it('keeps the backend order of conclusive rows', () => {
    const wrapper = mountPanel([
      row('first', { ConclusiveDue: daysFromNow(5), DaysLeft: 5 }),
      row('second', { ConclusiveDue: daysFromNow(40), DaysLeft: 40 }),
    ])

    const order = wrapper.findAll('[data-chave]').map((item) => item.attributes('data-chave'))
    expect(order).toEqual(['first', 'second'])
  })

  it('shows the ciência days left the backend counted', () => {
    const wrapper = mountPanel([
      row('ciencia-soon', {
        Kind: 'sem_ciencia',
        Manifestacao: 'nenhuma',
        CienciaDue: daysFromNow(2),
        CienciaDaysLeft: 2,
        ConclusiveDue: daysFromNow(82),
        DaysLeft: 82,
      }),
    ])

    expect(wrapper.find('[data-chave="ciencia-soon"] .chip').text()).toBe('2 d restantes')
  })

  it('shows the chave and the emitente with the document cells and the value without the currency', () => {
    const wrapper = shallowMount(NFePendingPanel, {
      props: {
        rows: [row('a', { EmitenteName: 'Metalúrgica Horizonte', EmitenteCNPJ: '27184593000140' })],
        loading: false,
        busy: () => false,
      },
      global: {
        stubs: {
          QTable: {
            name: 'QTable',
            props: ['rows', 'columns'],
            template:
              '<div><div v-for="r in rows" :key="r.ChaveAcesso"><slot name="body-cell-chave" :row="r" /><slot name="body-cell-emitente" :row="r" /></div></div>',
          },
          QTd: { template: '<div><slot /></div>' },
        },
      },
    })

    const table = wrapper.findAllComponents({ name: 'QTable' })[0]
    const columns = table?.props('columns') as { name: string; format?: (value: number) => string }[]
    expect(columns.map((column) => column.name)).toEqual([
      'numero',
      'chave',
      'emitente',
      'issueDate',
      'valor',
      'prazo',
      'acoes',
    ])
    expect(columns.find((column) => column.name === 'valor')?.format?.(123456)).toBe('1.234,56')
    expect(wrapper.findComponent(ChaveCell).props('chave')).toBe('a')
    expect(wrapper.findComponent(PartyCell).props()).toEqual({
      name: 'Metalúrgica Horizonte',
      cnpj: '27184593000140',
    })
  })
})
