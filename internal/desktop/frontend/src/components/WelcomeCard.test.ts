import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import WelcomeCard from './WelcomeCard.vue'

const stubs = {
  QCard: { template: '<div><slot /></div>' },
  QCardSection: { template: '<div><slot /></div>' },
  QCardActions: { template: '<div><slot /></div>' },
  QList: { template: '<ul><slot /></ul>' },
  QItem: { template: '<li class="step"><slot /></li>' },
  QItemSection: { template: '<div><slot /></div>' },
  QItemLabel: {
    props: { caption: Boolean },
    template: "<div :class=\"caption ? 'caption' : 'title'\"><slot /></div>",
  },
  QIcon: { props: ['name'], template: '<i :data-icon="name" />' },
  QBtn: {
    props: ['label'],
    emits: ['click'],
    template: '<button @click="$emit(\'click\')">{{ label }}</button>',
  },
}

function mountCard() {
  return mount(WelcomeCard, { global: { stubs } })
}

describe('WelcomeCard', () => {
  it('lists the three steps', () => {
    const wrapper = mountCard()

    expect(wrapper.text()).toContain('Bem-vindo ao Nanci')
    const steps = wrapper.findAll('.step')
    expect(steps.map((step) => step.get('.title').text())).toEqual([
      'Certificado A1',
      'Empresa',
      'Sincronizar',
    ])
    expect(steps.map((step) => step.get('.caption').text())).toEqual([
      'O arquivo .pfx ou .p12 e a senha.',
      'CNPJ, ambiente e o histórico inicial a importar.',
      'Buscar as notas na Receita e na SEFAZ.',
    ])
    expect(steps.every((step) => Boolean(step.get('i').attributes('data-icon')))).toBe(true)
  })

  it('emits add', async () => {
    const wrapper = mountCard()
    const add = wrapper.get('button')

    expect(add.text()).toBe('Cadastrar primeira empresa')
    await add.trigger('click')

    expect(wrapper.emitted('add')).toHaveLength(1)
  })
})
