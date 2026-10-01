import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import EventsDialogFrame, { eventDateColumn } from './EventsDialogFrame.vue'

const notify = vi.fn()

vi.mock('quasar', () => ({ useQuasar: () => ({ notify }) }))

const chave = '35260811222333000181570010000012341000012345'
const rows = [{ ID: 'evt-1', Type: 'cancelamento' }]
const columns = [{ name: 'tipo', label: 'Tipo', field: 'ID' as const }]

function mountFrame(load = vi.fn(async () => {}), modelValue = false) {
  return shallowMount(EventsDialogFrame, {
    props: { modelValue, title: 'Eventos do CT-e', chaveAcesso: chave, rows, columns, loading: true, load },
    slots: { 'body-cell-tipo': '<template #body-cell-tipo="cell"><b class="tipo">{{ cell.row.Type }}</b></template>' },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QDialog: {
          name: 'QDialog',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<div class="q-dialog-stub"><slot /></div>',
        },
        QTable: {
          name: 'QTable',
          props: { rows: Array, columns: Array, loading: Boolean, hidePagination: Boolean },
          template: '<div class="table"><slot name="body-cell-tipo" :row="rows[0]" /></div>',
        },
        QBtn: { name: 'QBtn', props: ['label', 'icon', 'ariaLabel'], template: '<button />' },
      },
      directives: { closePopup: {} },
    },
  })
}

describe('EventsDialogFrame', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows the title, the grouped key and the events table', () => {
    const wrapper = mountFrame()
    expect(wrapper.text()).toContain('Eventos do CT-e')
    expect(wrapper.text()).toContain('3526 0811 2223 3300 0181 5700 1000 0012 3410 0001 2345')
    expect(wrapper.find('.events-dialog').exists()).toBe(true)

    const table = wrapper.getComponent({ name: 'QTable' })
    expect(table.props()).toMatchObject({ rows, columns, loading: true, hidePagination: true })
  })

  it('passes the custom cells to the table', () => {
    expect(mountFrame().find('.table .tipo').text()).toBe('cancelamento')
  })

  it('loads the events each time it opens', async () => {
    const load = vi.fn(async () => {})
    const wrapper = mountFrame(load)
    expect(load).not.toHaveBeenCalled()

    await wrapper.setProps({ modelValue: true })
    await wrapper.setProps({ modelValue: false })
    await wrapper.setProps({ modelValue: true })

    expect(load).toHaveBeenCalledTimes(2)
  })

  it('reports a failed load', async () => {
    const wrapper = mountFrame(vi.fn(async () => Promise.reject(new Error('boom'))))

    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(notify).toHaveBeenCalledWith({ type: 'negative', message: 'Erro ao carregar eventos: boom' })
  })

  it('binds the dialog to v-model and offers a "Fechar" button', () => {
    const wrapper = mountFrame()
    const dialog = wrapper.getComponent({ name: 'QDialog' })
    expect(dialog.props('modelValue')).toBe(false)

    dialog.vm.$emit('update:modelValue', true)
    expect(wrapper.emitted('update:modelValue')).toEqual([[true]])

    const labels = wrapper.findAllComponents({ name: 'QBtn' }).map((btn) => btn.props('label'))
    expect(labels).toContain('Fechar')
  })
})

describe('eventDateColumn', () => {
  it('shows the event date, or a dash without one', () => {
    expect(eventDateColumn.field({ EventAt: '2026-07-26T12:00:00Z' })).toBe('2026-07-26T12:00:00Z')
    expect(eventDateColumn.format(null)).toBe('—')
  })
})
