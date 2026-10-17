import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import DraftBanner from './DraftBanner.vue'

vi.mock('quasar', () => ({
  useQuasar: () => ({ dark: { isActive: false } }),
}))

describe('DraftBanner', () => {
  it('emits clear from Limpar', async () => {
    const wrapper = mount(DraftBanner, {
      global: {
        stubs: {
          'q-banner': {
            template: '<div><slot name="avatar" /><slot /><slot name="action" /></div>',
          },
          'q-icon': { template: '<i />' },
          'q-btn': {
            name: 'QBtn',
            props: ['label'],
            emits: ['click'],
            template: '<button @click="$emit(\'click\')">{{ label }}</button>',
          },
        },
      },
    })

    expect(wrapper.text()).toContain('Rascunho recuperado')
    await wrapper.getComponent({ name: 'QBtn' }).trigger('click')
    expect(wrapper.emitted('clear')).toHaveLength(1)
  })
})
