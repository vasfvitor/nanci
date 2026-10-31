import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import NFeStateBadges from './NFeStateBadges.vue'
import type { NFeRow } from '@/types/desktop'
import { makeNFeRow } from '@/test/fixtures'
import type { StateBadge } from '@/utils/sefazDisplay'

const badges: StateBadge[] = [
  { key: 'Situação:A', abbr: 'A', label: 'Autorizada', color: 'positive', kind: 'Situação' },
]

const stubs = {
  StateBadges: {
    name: 'StateBadges',
    props: ['badges'],
    template: '<div><span v-if="$slots.default" class="slot"><slot /></span></div>',
  },
  'q-spinner': { name: 'QSpinner', template: '<i class="spinner" />' },
  'q-chip': {
    name: 'QChip',
    props: ['color', 'label', 'title'],
    template: '<span class="chip">{{ label }}</span>',
  },
}

function mountCell(row: Partial<NFeRow> = {}, busy = false) {
  return shallowMount(NFeStateBadges, {
    props: { row: makeNFeRow(row), badges, busy },
    global: { stubs },
  })
}

describe('NFeStateBadges', () => {
  it('shows only the badges of a note without ciência', () => {
    const wrapper = mountCell({ Manifestacao: 'nenhuma', ConclusiveDue: '2026-11-20' })

    expect(wrapper.getComponent({ name: 'StateBadges' }).props('badges')).toEqual(badges)
    expect(wrapper.find('.slot').exists()).toBe(false)
  })

  it('shows a spinner while an event of the note is being sent', () => {
    const wrapper = mountCell({}, true)

    expect(wrapper.find('.slot .spinner').exists()).toBe(true)
    expect(wrapper.find('.chip').exists()).toBe(false)
  })

  it('shows the conclusive deadline after the ciência', () => {
    const wrapper = mountCell({
      Manifestacao: 'ciencia',
      ConclusiveDue: '2026-11-20T00:00:00Z',
      DaysLeft: 3,
    })

    const chip = wrapper.getComponent({ name: 'QChip' })
    expect(chip.props('color')).toBe('negative')
    expect(chip.props('label')).toBe('3 d restantes')
    expect(chip.props('title')).toMatch(/^Prazo da manifestação conclusiva: \d{2}\/\d{2}\/\d{4}$/)
    expect(wrapper.find('.spinner').exists()).toBe(false)
  })

  it('says the operation was confirmed tacitly once the deadline passed', () => {
    const wrapper = mountCell({
      Manifestacao: 'ciencia',
      ConclusiveDue: '2026-10-01T00:00:00Z',
      DaysLeft: -2,
    })

    expect(wrapper.getComponent({ name: 'QChip' }).props('label')).toBe('Confirmada tacitamente')
  })
})
