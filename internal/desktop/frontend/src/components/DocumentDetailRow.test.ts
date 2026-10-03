import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentDetailRow from './DocumentDetailRow.vue'

const quasar = vi.hoisted(() => ({ dark: { isActive: false } }))

vi.mock('quasar', () => ({ useQuasar: () => quasar }))

const stubs = {
  QTr: { template: '<div class="tr"><slot /></div>' },
  QTd: { props: ['colspan'], template: '<div class="td" :data-colspan="colspan"><slot /></div>' },
}

function mountRow() {
  return mount(DocumentDetailRow, {
    props: { rowProps: { cols: [{}, {}, {}], expand: true } },
    slots: { default: '<p class="content">Detalhes</p>' },
    global: { stubs },
  })
}

describe('DocumentDetailRow', () => {
  beforeEach(() => {
    quasar.dark.isActive = false
  })

  it('spans every column plus the selection one', () => {
    const td = mountRow().find('.td')
    expect(td.attributes('data-colspan')).toBe('4')
    expect(td.classes()).toContain('document-detail-cell')
    expect(td.find('.document-detail .content').text()).toBe('Detalhes')
  })

  it('follows the theme background', () => {
    expect(mountRow().find('.tr').classes()).toContain('bg-grey-1')
    quasar.dark.isActive = true
    expect(mountRow().find('.tr').classes()).toContain('bg-grey-10')
  })
})
