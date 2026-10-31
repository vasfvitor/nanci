import { mount } from '@vue/test-utils'
import { describe, expect, it } from 'vitest'
import NFeListActions from './NFeListActions.vue'

const stubs = {
  QBtn: {
    name: 'QBtn',
    props: ['label', 'disable', 'loading'],
    emits: ['click'],
    template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
  },
}

type Props = InstanceType<typeof NFeListActions>['$props']

function mountActions(props: Partial<Props> = {}) {
  return mount(NFeListActions, {
    props: {
      eligibleCount: 2,
      planningCiencia: false,
      cienciaInFlight: false,
      importDisabled: false,
      ...props,
    },
    global: { stubs },
  })
}

type Actions = ReturnType<typeof mountActions>

function button(wrapper: Actions, label: string | RegExp) {
  const found = wrapper.findAllComponents({ name: 'QBtn' }).find((btn) => {
    const text = String(btn.props('label') ?? '')
    return typeof label === 'string' ? text === label : label.test(text)
  })
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

describe('NFeListActions', () => {
  it('renders the three buttons without a wrapper, in order', () => {
    const wrapper = mountActions()

    expect(wrapper.findAll('button').map((btn) => btn.text())).toEqual([
      'Registrar ciência (2)',
      'Importar XML',
      'Buscar pela chave',
    ])
    expect(wrapper.findAll('div')).toHaveLength(0)
  })

  it.each([
    { props: { eligibleCount: 0 }, disable: true, loading: false },
    { props: { planningCiencia: true }, disable: true, loading: true },
    { props: { cienciaInFlight: true }, disable: true, loading: false },
    { props: {}, disable: false, loading: false },
  ])('ciência is disabled $disable and loading $loading with $props', (c) => {
    const ciencia = button(mountActions(c.props), /^Registrar ciência/)

    expect(ciencia.props('disable')).toBe(c.disable)
    expect(ciencia.props('loading')).toBe(c.loading)
  })

  it('disables the import and the fetch together', () => {
    const wrapper = mountActions({ importDisabled: true })

    expect(button(wrapper, 'Importar XML').props('disable')).toBe(true)
    expect(button(wrapper, 'Buscar pela chave').props('disable')).toBe(true)
    expect(button(wrapper, /^Registrar ciência/).props('disable')).toBe(false)
  })

  it('emits the action of each button', async () => {
    const wrapper = mountActions()

    await button(wrapper, /^Registrar ciência/).trigger('click')
    await button(wrapper, 'Importar XML').trigger('click')
    await button(wrapper, 'Buscar pela chave').trigger('click')

    expect(wrapper.emitted('ciencia')).toHaveLength(1)
    expect(wrapper.emitted('import')).toHaveLength(1)
    expect(wrapper.emitted('fetch')).toHaveLength(1)
  })
})
