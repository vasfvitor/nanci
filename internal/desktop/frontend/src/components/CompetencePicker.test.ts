import { shallowMount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { nextTick } from 'vue'
import CompetencePicker from './CompetencePicker.vue'

const hide = vi.fn()

function mountPicker(modelValue: string, disable = false) {
  return shallowMount(CompetencePicker, {
    props: { modelValue, disable },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QBtn: {
          name: 'QBtn',
          props: ['label', 'icon', 'disable'],
          emits: ['click'],
          template:
            '<button :data-icon="icon" :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
        },
        QInput: {
          name: 'QInput',
          props: ['modelValue', 'disable', 'error', 'errorMessage'],
          emits: ['update:modelValue'],
          template: '<div><slot name="append" /></div>',
        },
        QPopupProxy: {
          name: 'QPopupProxy',
          methods: { hide },
          template: '<div><slot /></div>',
        },
        QDate: {
          name: 'QDate',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<div><slot /></div>',
        },
      },
    },
  })
}

type Picker = ReturnType<typeof mountPicker>

const input = (wrapper: Picker) => wrapper.getComponent({ name: 'QInput' })
const button = (wrapper: Picker, icon: string) => wrapper.get(`button[data-icon="${icon}"]`)
const emitted = (wrapper: Picker) => wrapper.emitted('update:modelValue') ?? []

async function type(wrapper: Picker, value: string | null) {
  input(wrapper).vm.$emit('update:modelValue', value)
  await nextTick()
}

describe('CompetencePicker', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    vi.setSystemTime(new Date(2026, 8, 23))
    hide.mockClear()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('flags a partial month and keeps it from the model', async () => {
    const wrapper = mountPicker('2026-08')
    expect(input(wrapper).props('error')).toBe(false)

    await type(wrapper, '2026-0')
    expect(input(wrapper).props('error')).toBe(true)
    expect(input(wrapper).props('errorMessage')).toBe('Mês incompleto')

    await type(wrapper, '2026-13')
    expect(input(wrapper).props('errorMessage')).toBe('Mês inválido')
    expect(emitted(wrapper)).toEqual([])

    await type(wrapper, '2026-09')
    expect(input(wrapper).props('error')).toBe(false)
    expect(input(wrapper).props('errorMessage')).toBe('')
    expect(emitted(wrapper)).toEqual([['2026-09']])
  })

  it('sends an empty competence when cleared', async () => {
    const wrapper = mountPicker('2026-08')

    await type(wrapper, null)

    expect(emitted(wrapper)).toEqual([['']])
    expect(input(wrapper).props('error')).toBe(false)
  })

  it('follows a competence set elsewhere', async () => {
    const wrapper = mountPicker('2026-08')
    await type(wrapper, '2026-0')

    await wrapper.setProps({ modelValue: '2026-07' })

    expect(input(wrapper).props('modelValue')).toBe('2026-07')
    expect(input(wrapper).props('error')).toBe(false)
  })

  it('sends quick steps once, after a short pause', async () => {
    const wrapper = mountPicker('2026-01')

    await button(wrapper, 'chevron_left').trigger('click')
    await button(wrapper, 'chevron_left').trigger('click')
    expect(input(wrapper).props('modelValue')).toBe('2025-11')
    expect(emitted(wrapper)).toEqual([])

    vi.advanceTimersByTime(199)
    expect(emitted(wrapper)).toEqual([])
    vi.advanceTimersByTime(1)
    expect(emitted(wrapper)).toEqual([['2025-11']])
  })

  it('steps from the last complete competence', async () => {
    const wrapper = mountPicker('2026-01')
    await type(wrapper, '2026-0')

    await button(wrapper, 'chevron_right').trigger('click')
    vi.advanceTimersByTime(200)

    expect(emitted(wrapper)).toEqual([['2026-02']])
  })

  it('keeps a step still waiting when unmounted', async () => {
    const wrapper = mountPicker('2026-01')
    await button(wrapper, 'chevron_right').trigger('click')

    wrapper.unmount()

    expect(emitted(wrapper)).toEqual([['2026-02']])
  })

  it('disables the chevrons without a competence or while disabled', () => {
    expect(button(mountPicker(''), 'chevron_right').attributes('disabled')).toBeDefined()
    expect(
      button(mountPicker('2026-01', true), 'chevron_right').attributes('disabled')
    ).toBeDefined()
    expect(button(mountPicker('2026-01'), 'chevron_right').attributes('disabled')).toBeUndefined()
  })

  it('picks the current month at once and closes the calendar', async () => {
    const wrapper = mountPicker('2024-01')
    const today = wrapper.findAll('button').find((item) => item.text() === 'Mês Atual')

    await today?.trigger('click')

    expect(emitted(wrapper)).toEqual([['2026-09']])
    expect(hide).toHaveBeenCalled()
  })

  it('closes the calendar once a month is picked and sends it after a pause', () => {
    const wrapper = mountPicker('2024-01')
    const calendar = wrapper.getComponent({ name: 'QDate' })

    calendar.vm.$emit('update:modelValue', '2025-01', 'year')
    expect(hide).not.toHaveBeenCalled()

    calendar.vm.$emit('update:modelValue', '2025-05', 'month')
    expect(hide).toHaveBeenCalled()
    expect(emitted(wrapper)).toEqual([])

    vi.advanceTimersByTime(200)
    expect(emitted(wrapper)).toEqual([['2025-05']])
  })
})
