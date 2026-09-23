import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import NFePendingPanel from './NFePendingPanel.vue'
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
    CienciaOverdue: false,
    Expired: false,
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
      row('on-time', { Deadline: daysFromNow(20) }),
      row('expired', { Deadline: daysFromNow(-3), Expired: true }),
      row('expired-sem-ciencia', {
        Kind: 'sem_ciencia',
        Manifestacao: 'nenhuma',
        CienciaDue: daysFromNow(-83),
        CienciaOverdue: true,
        Deadline: daysFromNow(-3),
        Expired: true,
      }),
    ])

    const chip = (chave: string) => wrapper.find(`[data-chave="${chave}"] .chip`).text()
    expect(chip('on-time')).toMatch(/d restantes$/)
    expect(chip('expired')).toBe('Confirmada tacitamente')
    expect(chip('expired-sem-ciencia')).toBe('Confirmada tacitamente')
    expect(wrapper.text()).toMatch(/até 90 dias da\s+autorização/)
    expect(wrapper.text()).toMatch(/confirmada por lei/)
  })

  it('keeps the backend order of conclusive rows', () => {
    const wrapper = mountPanel([
      row('first', { Deadline: daysFromNow(5) }),
      row('second', { Deadline: daysFromNow(40) }),
    ])

    const order = wrapper.findAll('[data-chave]').map((item) => item.attributes('data-chave'))
    expect(order).toEqual(['first', 'second'])
  })
})
