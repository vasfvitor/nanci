import { mount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import NSUGapNotice from './NSUGapNotice.vue'

vi.mock('quasar', () => ({ useQuasar: () => ({ dark: { isActive: false } }) }))

function mountNotice(disabledReason: string) {
  return mount(NSUGapNotice, {
    props: { notice: '3 NSUs faltando', disabledReason },
    global: {
      stubs: {
        'q-badge': { props: ['label'], template: '<span class="badge">{{ label }}<slot /></span>' },
        'q-tooltip': { template: '<em class="tip"><slot /></em>' },
        'q-btn': {
          props: ['label', 'disable'],
          emits: ['click'],
          template: `<button :disabled="disable" @click="$emit('click', $event)">{{ label }}<slot /></button>`,
        },
      },
    },
  })
}

describe('NSUGapNotice', () => {
  it('shows the missing NSUs and asks to recover them', async () => {
    const wrapper = mountNotice('')
    expect(wrapper.get('.badge').text()).toContain('3 NSUs faltando')

    await wrapper.get('button').trigger('click')
    expect(wrapper.emitted('recover')).toHaveLength(1)
  })

  it('says why the recovery cannot run now', () => {
    const wrapper = mountNotice('Aguarde a operação em andamento terminar.')
    expect(wrapper.get('button').attributes('disabled')).toBeDefined()
    expect(wrapper.get('button').text()).toContain('Aguarde a operação em andamento terminar.')
  })
})
