import { shallowMount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import NFeRowMenu from './NFeRowMenu.vue'
import type { NFeRow } from '@/types/desktop'
import { makeNFeRow } from '@/test/fixtures'

type Props = { row: NFeRow; busy: boolean; cienciaInFlight: boolean; exporting: boolean }

function mountMenu(row: Partial<NFeRow> = {}, props: Partial<Omit<Props, 'row'>> = {}) {
  return shallowMount(NFeRowMenu, {
    props: {
      row: makeNFeRow({ CompanyRole: 'destinatario', EventCount: 2, ...row }),
      busy: false,
      cienciaInFlight: false,
      exporting: false,
      ...props,
    },
    global: {
      stubs: {
        RowActionsMenu: {
          name: 'RowActionsMenu',
          props: ['source', 'expanded'],
          emits: ['update:expanded'],
          template: '<div><slot /></div>',
        },
        RowMenuItem: {
          name: 'RowMenuItem',
          props: ['label', 'caption', 'disable'],
          emits: ['click'],
          template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
        },
      },
    },
  })
}

type Menu = ReturnType<typeof mountMenu>

function item(wrapper: Menu, label: string) {
  const found = wrapper
    .findAllComponents({ name: 'RowMenuItem' })
    .find((entry) => entry.props('label') === label)
  if (!found) throw new Error(`menu item ${label} not found`)
  return found
}

function state(wrapper: Menu, label: string) {
  const entry = item(wrapper, label)
  return { disable: entry.props('disable'), caption: entry.props('caption') }
}

describe('NFeRowMenu', () => {
  it('offers every action of a complete note received by the company', () => {
    const wrapper = mountMenu({ Completeness: 'completa' })

    expect(wrapper.getComponent({ name: 'RowActionsMenu' }).props('source')).toBe('nfe')
    expect(state(wrapper, 'Eventos (2)').disable).toBe(false)
    for (const label of ['Manifestar…', 'Registrar ciência', 'Exportar XML', 'Exportar DANFE']) {
      expect(state(wrapper, label)).toEqual({ disable: false, caption: '' })
    }
  })

  it('says why a note the company did not receive takes no event', () => {
    const wrapper = mountMenu({
      CompanyRole: 'emitente',
      CienciaBlockReason: 'a empresa não é a destinatária',
      ConclusiveBlockReason: 'a empresa não é a destinatária',
    })

    expect(state(wrapper, 'Manifestar…')).toEqual({
      disable: true,
      caption: 'A empresa não é a destinatária',
    })
    expect(state(wrapper, 'Registrar ciência')).toEqual({
      disable: true,
      caption: 'A empresa não é a destinatária',
    })
  })

  it('blocks the events of a note being sent and says so', () => {
    const wrapper = mountMenu({}, { busy: true })

    expect(state(wrapper, 'Manifestar…')).toEqual({ disable: true, caption: 'Envio em andamento' })
    expect(state(wrapper, 'Registrar ciência')).toEqual({
      disable: true,
      caption: 'Envio em andamento',
    })
  })

  it('blocks another ciência while one is being sent, but not the manifestação', () => {
    const wrapper = mountMenu({}, { cienciaInFlight: true })

    expect(state(wrapper, 'Registrar ciência').disable).toBe(true)
    expect(state(wrapper, 'Manifestar…').disable).toBe(false)
  })

  it('keeps the exports of a resumo disabled and says why', () => {
    const wrapper = mountMenu({ Completeness: 'resumo' })

    for (const label of ['Exportar XML', 'Exportar DANFE']) {
      expect(state(wrapper, label)).toEqual({
        disable: true,
        caption: 'XML completo ainda não baixado',
      })
    }
  })

  it('keeps the exports disabled while another export runs', () => {
    const wrapper = mountMenu({ Completeness: 'completa' }, { exporting: true })

    expect(state(wrapper, 'Exportar XML').disable).toBe(true)
    expect(state(wrapper, 'Exportar DANFE').disable).toBe(true)
  })

  it('disables the events entry of a note without events', () => {
    expect(state(mountMenu({ EventCount: 0 }), 'Eventos (0)').disable).toBe(true)
  })

  it('emits the action of each entry', async () => {
    const wrapper = mountMenu({ Completeness: 'completa' })

    await item(wrapper, 'Manifestar…').trigger('click')
    await item(wrapper, 'Registrar ciência').trigger('click')
    await item(wrapper, 'Eventos (2)').trigger('click')
    await item(wrapper, 'Exportar XML').trigger('click')
    await item(wrapper, 'Exportar DANFE').trigger('click')

    expect(Object.keys(wrapper.emitted())).toEqual(
      expect.arrayContaining(['manifest', 'ciencia', 'events', 'exportXml', 'exportDanfe'])
    )
  })

  it('passes the expanded model through', async () => {
    const wrapper = mountMenu()

    wrapper.getComponent({ name: 'RowActionsMenu' }).vm.$emit('update:expanded', true)

    expect(wrapper.emitted('update:expanded')).toEqual([[true]])
  })
})
