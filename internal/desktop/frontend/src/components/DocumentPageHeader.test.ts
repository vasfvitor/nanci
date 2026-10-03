import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentPageHeader from './DocumentPageHeader.vue'

const quasar = vi.hoisted(() => ({ dark: { isActive: false } }))

vi.mock('quasar', () => ({ useQuasar: () => quasar }))

const stubs = {
  QBadge: {
    props: ['label', 'color', 'textColor'],
    template: '<span class="badge" :data-color="color" :data-text-color="textColor">{{ label }}</span>',
  },
  QBtn: {
    name: 'QBtn',
    props: ['label', 'loading', 'disable', 'title'],
    emits: ['click'],
    template: '<button :disabled="disable" :data-loading="loading" @click="$emit(\'click\')">{{ label }}</button>',
  },
  QBanner: { template: '<div class="banner"><slot /></div>' },
  QSpace: { template: '<span />' },
  QIcon: { template: '<i />' },
}

type HeaderProps = InstanceType<typeof DocumentPageHeader>['$props']

function mountHeader(props: Partial<HeaderProps> = {}) {
  return mount(DocumentPageHeader, {
    props: {
      title: 'NF-e',
      ambiente: null,
      contextLine: '',
      syncing: false,
      syncDisabled: false,
      resetTitle: 'Remove as NF-e da empresa',
      resetting: false,
      resetDisabled: false,
      ...props,
    },
    global: { stubs },
  })
}

function button(wrapper: ReturnType<typeof mountHeader>, label: string) {
  const found = wrapper.findAll('button').find((item) => item.text() === label)
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

describe('DocumentPageHeader', () => {
  beforeEach(() => {
    quasar.dark.isActive = false
  })

  it('shows the title and the reset and sync buttons named after it', async () => {
    const wrapper = mountHeader()

    expect(wrapper.find('h5').text()).toBe('NF-e')
    expect(wrapper.find('.badge').exists()).toBe(false)
    expect(wrapper.find('.banner').exists()).toBe(false)
    expect(wrapper.findAll('button').map((item) => item.text())).toEqual(['Redefinir NF-e', 'Sincronizar NF-e'])

    await button(wrapper, 'Sincronizar NF-e').trigger('click')
    expect(wrapper.emitted('sync')).toHaveLength(1)
  })

  it('shows the reset in flight and asks for it on click', async () => {
    const wrapper = mountHeader({ resetting: true })

    const reset = button(wrapper, 'Redefinir NF-e')
    expect(reset.attributes('data-loading')).toBe('true')
    await reset.trigger('click')
    expect(wrapper.emitted('reset')).toHaveLength(1)
  })

  it('disables the buttons as told', () => {
    const wrapper = mountHeader({ syncDisabled: true, resetDisabled: true })

    expect(button(wrapper, 'Sincronizar NF-e').attributes('disabled')).toBeDefined()
    expect(button(wrapper, 'Redefinir NF-e').attributes('disabled')).toBeDefined()
  })

  it('names the workspace under the title when given', () => {
    const wrapper = mountHeader({ contextLine: 'ACME · 12.345.678/0001-00 · Competência 09/2026' })

    expect(wrapper.find('.text-body2').text()).toBe(
      'ACME · 12.345.678/0001-00 · Competência 09/2026'
    )
  })

  it('leaves the workspace line out when it is empty', () => {
    const wrapper = mountHeader()

    expect(wrapper.find('.text-body2').exists()).toBe(false)
  })

  it('shows the status line and the block banner', () => {
    const wrapper = mountHeader({
      statusLine: 'Última sincronização: nunca · NSU 0',
      blockedText: 'Consultas bloqueadas pela SEFAZ até 14:32 (cStat 656)',
    })

    expect(wrapper.find('.text-caption').text()).toBe('Última sincronização: nunca · NSU 0')
    expect(wrapper.find('.banner').text()).toContain('cStat 656')
    expect(wrapper.find('.banner').classes()).toContain('bg-orange-1')
  })

  it('colors the ambiente badge and the banner for the theme', () => {
    const light = mountHeader({ ambiente: { label: 'Produção', color: 'negative' }, blockedText: 'x' })
    expect(light.find('.badge').text()).toBe('Produção')
    expect(light.find('.badge').attributes('data-text-color')).toBe('white')

    quasar.dark.isActive = true
    const dark = mountHeader({ ambiente: { label: 'Produção', color: 'negative' }, blockedText: 'x' })
    expect(dark.find('.badge').attributes('data-text-color')).toBe('dark')
    expect(dark.find('.banner').classes()).toContain('bg-grey-9')
  })
})
