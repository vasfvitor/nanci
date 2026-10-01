import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DocumentFilterSelect from './DocumentFilterSelect.vue'

const stubs = {
  QSelect: {
    name: 'QSelect',
    props: {
      modelValue: String,
      options: Array,
      label: String,
      disable: Boolean,
      emitValue: Boolean,
      mapOptions: Boolean,
    },
    emits: ['update:modelValue'],
    template: '<div />',
  },
}

const options = [
  { label: 'Todas', value: '' },
  { label: 'Autorizada', value: 'autorizada' },
]

describe('DocumentFilterSelect', () => {
  it('passes the options as values and its own width class', () => {
    const wrapper = mount(DocumentFilterSelect, {
      props: { modelValue: 'autorizada', options, label: 'Situação', disable: true },
      global: { stubs },
    })

    const select = wrapper.getComponent({ name: 'QSelect' })
    expect(select.props()).toMatchObject({
      modelValue: 'autorizada',
      options,
      label: 'Situação',
      disable: true,
      emitValue: true,
      mapOptions: true,
    })
    expect(select.classes()).toEqual(['document-filter-select'])
  })

  it('updates the value', async () => {
    const wrapper = mount(DocumentFilterSelect, {
      props: { modelValue: '', options, label: 'Situação', disable: false },
      global: { stubs },
    })

    await wrapper.getComponent({ name: 'QSelect' }).vm.$emit('update:modelValue', 'autorizada')

    expect(wrapper.emitted('update:modelValue')).toEqual([['autorizada']])
  })
})
