import { shallowMount } from '@vue/test-utils'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
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
          template: '<button :data-icon="icon" :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
        },
        QInput: {
          name: 'QInput',
          props: ['modelValue', 'disable'],
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

function button(wrapper: ReturnType<typeof mountPicker>, selector: string) {
  return wrapper.get(`button${selector}`)
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

  it('steps the competence a month at a time', async () => {
    const wrapper = mountPicker('2026-01')

    await button(wrapper, '[data-icon="chevron_left"]').trigger('click')
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['2025-12'])
  })

  it('disables the chevrons without a competence or while disabled', () => {
    expect(button(mountPicker(''), '[data-icon="chevron_right"]').attributes('disabled')).toBeDefined()
    expect(
      button(mountPicker('2026-01', true), '[data-icon="chevron_right"]').attributes('disabled')
    ).toBeDefined()
    expect(
      button(mountPicker('2026-01'), '[data-icon="chevron_right"]').attributes('disabled')
    ).toBeUndefined()
  })

  it('picks the current month and closes the calendar', async () => {
    const wrapper = mountPicker('2024-01')
    const today = wrapper.findAll('button').find((item) => item.text() === 'Mês Atual')

    await today?.trigger('click')

    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['2026-09'])
    expect(hide).toHaveBeenCalled()
  })

  it('closes the calendar once a month is picked', async () => {
    const wrapper = mountPicker('2024-01')
    const calendar = wrapper.getComponent({ name: 'QDate' })

    calendar.vm.$emit('update:modelValue', '2024-05', 'year')
    expect(hide).not.toHaveBeenCalled()

    calendar.vm.$emit('update:modelValue', '2024-05', 'month')
    expect(hide).toHaveBeenCalled()
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual(['2024-05'])
  })
})
