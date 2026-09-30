import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import DetailList from './DetailList.vue'

describe('DetailList', () => {
  it('lists labels and values, with a dash for empty values', () => {
    const wrapper = mount(DetailList, {
      props: {
        title: 'Prestação',
        items: [
          { label: 'CFOP', value: '5353' },
          { label: 'Protocolo', value: '', mono: true },
          { label: 'Carga', value: 0, mono: true },
          { label: 'Produto', value: null },
          { label: 'Tomador', value: 'Cliente', caption: '12.345.678/0001-99' },
        ],
      },
    })

    expect(wrapper.find('.text-subtitle2').text()).toBe('Prestação')
    expect(wrapper.findAll('dt').map((el) => el.text())).toEqual([
      'CFOP',
      'Protocolo',
      'Carga',
      'Produto',
      'Tomador',
    ])
    const values = wrapper.findAll('dd')
    expect(values.map((el) => el.find('span').text())).toEqual(['5353', '—', '0', '—', 'Cliente'])
    expect(values[1]?.find('span').classes()).toContain('text-mono')
    expect(values[0]?.find('span').classes()).not.toContain('text-mono')
    expect(values[4]?.findAll('span')[1]?.text()).toBe('12.345.678/0001-99')
  })
})
