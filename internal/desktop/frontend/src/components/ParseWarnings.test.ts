import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import ParseWarnings from './ParseWarnings.vue'

describe('ParseWarnings', () => {
  it('lists the warnings under a title', () => {
    const wrapper = mount(ParseWarnings, { props: { warnings: ['sem chNFSe', 'data inválida'] } })

    expect(wrapper.text()).toContain('Avisos da leitura do XML')
    expect(wrapper.findAll('li').map((li) => li.text())).toEqual(['sem chNFSe', 'data inválida'])
  })

  it('renders nothing without warnings', () => {
    const wrapper = mount(ParseWarnings, { props: { warnings: [] } })

    expect(wrapper.html()).toBe('<!--v-if-->')
  })
})
