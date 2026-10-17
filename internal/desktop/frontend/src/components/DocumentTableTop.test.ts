import { mount } from '@vue/test-utils'
import { Quasar, QInput, QIcon } from 'quasar/dist/quasar.client.js'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentTableTop from './DocumentTableTop.vue'

const stubs = {
  QInput: {
    name: 'QInput',
    props: ['modelValue', 'placeholder'],
    emits: ['update:modelValue'],
    template: '<div class="search"><slot name="append" /></div>',
  },
  QIcon: { template: '<i />' },
}

// mountTop mounts the box with a v-model that follows its emits, so the
// test reads the final model from the props.
function mountTop(modelValue: string) {
  const wrapper = mount(DocumentTableTop, {
    props: {
      title: 'CT-e',
      modelValue,
      'onUpdate:modelValue': (value: string) => wrapper.setProps({ modelValue: value }),
    },
    global: { stubs },
  })
  return wrapper
}

describe('DocumentTableTop', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('shows the title and the standard search box', () => {
    const wrapper = mount(DocumentTableTop, {
      props: { title: 'Notas fiscais eletrônicas', modelValue: 'acme' },
      global: { stubs },
    })

    expect(wrapper.text()).toContain('Notas fiscais eletrônicas')
    const input = wrapper.getComponent({ name: 'QInput' })
    expect(input.props('modelValue')).toBe('acme')
    expect(input.props('placeholder')).toBe('Filtrar por chave, número, nome ou CNPJ...')
    expect(input.classes()).toContain('document-search-input')
  })

  it('updates the filter text 300 ms after the last keystroke', async () => {
    const wrapper = mountTop('')
    const input = wrapper.getComponent({ name: 'QInput' })

    await input.vm.$emit('update:modelValue', 'ser')
    await vi.advanceTimersByTimeAsync(200)
    await input.vm.$emit('update:modelValue', 'serra')
    expect(input.props('modelValue')).toBe('serra')
    await vi.advanceTimersByTimeAsync(299)
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    await vi.advanceTimersByTimeAsync(1)
    expect(wrapper.emitted('update:modelValue')).toEqual([['serra']])
  })

  it('updates the filter text at once on blur', async () => {
    const wrapper = mountTop('')
    const input = wrapper.getComponent({ name: 'QInput' })

    await input.vm.$emit('update:modelValue', 'serra')
    await input.trigger('blur')

    expect(wrapper.emitted('update:modelValue')).toEqual([['serra']])
    await vi.advanceTimersByTimeAsync(1000)
    expect(wrapper.emitted('update:modelValue')).toEqual([['serra']])
  })

  it('clears the filter with Escape', async () => {
    const wrapper = mountTop('serra')
    const input = wrapper.getComponent({ name: 'QInput' })

    await input.trigger('keyup', { key: 'Enter' })
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()

    await input.trigger('keyup', { key: 'Escape' })
    expect(wrapper.emitted('update:modelValue')).toEqual([['']])
    expect(input.props('modelValue')).toBe('')
  })

  it('ends with an empty filter when Escape comes within the debounce', async () => {
    const wrapper = mountTop('')
    const input = wrapper.getComponent({ name: 'QInput' })

    await input.vm.$emit('update:modelValue', 'serra')
    await vi.advanceTimersByTimeAsync(100)
    await input.trigger('keyup', { key: 'Escape' })
    await vi.advanceTimersByTimeAsync(1000)

    expect(wrapper.props('modelValue')).toBe('')
    expect(input.props('modelValue')).toBe('')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('drops the pending text when Escape clears a filter already applied', async () => {
    const wrapper = mountTop('serra')
    const input = wrapper.getComponent({ name: 'QInput' })

    await input.vm.$emit('update:modelValue', 'serra verde')
    await vi.advanceTimersByTimeAsync(100)
    await input.trigger('keyup', { key: 'Escape' })
    await vi.advanceTimersByTimeAsync(1000)

    expect(wrapper.props('modelValue')).toBe('')
    expect(input.props('modelValue')).toBe('')
    expect(wrapper.emitted('update:modelValue')).toEqual([['']])
  })

  it('clears the filter at once with the clear button', async () => {
    const wrapper = mountTop('serra')
    const input = wrapper.getComponent({ name: 'QInput' })

    await input.vm.$emit('update:modelValue', 'serra verde')
    await input.vm.$emit('update:modelValue', null)
    await vi.advanceTimersByTimeAsync(1000)

    expect(wrapper.props('modelValue')).toBe('')
    expect(wrapper.emitted('update:modelValue')).toEqual([['']])
  })

  it('shows a model set from outside and drops the pending text', async () => {
    const wrapper = mountTop('')
    const input = wrapper.getComponent({ name: 'QInput' })

    await input.vm.$emit('update:modelValue', 'serra')
    await wrapper.setProps({ modelValue: 'campo' })
    await vi.advanceTimersByTimeAsync(1000)

    expect(input.props('modelValue')).toBe('campo')
    expect(wrapper.emitted('update:modelValue')).toBeUndefined()
  })

  it('hands the pending text to the model on unmount', async () => {
    const wrapper = mountTop('')
    const input = wrapper.getComponent({ name: 'QInput' })

    await input.vm.$emit('update:modelValue', 'serra')
    wrapper.unmount()

    expect(wrapper.emitted('update:modelValue')).toEqual([['serra']])
  })
})

// The stubs above do not debounce; this one runs the real q-input, where the
// Escape bug lived.
describe('DocumentTableTop with the real q-input', () => {
  beforeEach(() => {
    if (!window.screen.orientation) {
      // happy-dom has no screen.orientation, which the Quasar Screen plugin reads.
      const orientation = Object.assign(new EventTarget(), { type: 'landscape-primary', angle: 0 })
      Object.defineProperty(window.screen, 'orientation', {
        value: orientation,
        configurable: true,
      })
    }
    vi.useFakeTimers()
  })
  afterEach(() => vi.useRealTimers())

  it('ends with an empty filter when Escape comes within the debounce', async () => {
    const wrapper = mount(DocumentTableTop, {
      props: {
        title: 'CT-e',
        modelValue: '',
        'onUpdate:modelValue': (value: string) => wrapper.setProps({ modelValue: value }),
      },
      global: { plugins: [Quasar], components: { QInput, QIcon } },
    })
    const input = wrapper.get('input')
    const element = input.element as HTMLInputElement

    // A keystroke fires input only; setValue would also fire change, which
    // makes q-input emit at once.
    element.value = 'serra'
    await input.trigger('input')
    await vi.advanceTimersByTimeAsync(100)
    await input.trigger('keyup', { key: 'Escape' })
    await vi.advanceTimersByTimeAsync(1000)

    expect(wrapper.props('modelValue')).toBe('')
    expect(element.value).toBe('')
  })
})
