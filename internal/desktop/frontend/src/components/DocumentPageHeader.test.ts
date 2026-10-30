import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DocumentPageHeader from './DocumentPageHeader.vue'

const quasar = vi.hoisted(() => ({ dark: { isActive: false } }))

vi.mock('quasar', () => ({ useQuasar: () => quasar }))

const stubs = {
  QBadge: {
    props: ['label', 'color', 'textColor'],
    template:
      '<span class="badge" :data-color="color" :data-text-color="textColor">{{ label }}<slot /></span>',
  },
  QTooltip: { template: '<span class="tooltip"><slot /></span>' },
  QBtn: {
    name: 'QBtn',
    props: ['label', 'loading', 'disable', 'title'],
    emits: ['click'],
    template:
      '<button :disabled="disable" :data-loading="loading" @click="$emit(\'click\')">{{ label }}<slot /></button>',
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
    expect(wrapper.findAll('button').map((item) => item.text())).toEqual([
      'Redefinir NF-e',
      'Sincronizar NF-e',
    ])

    await button(wrapper, 'Sincronizar NF-e').trigger('click')
    expect(wrapper.emitted('sync')).toHaveLength(1)
  })

  it('offers Cancelar only while a cancellable sync runs', async () => {
    const labels = (wrapper: ReturnType<typeof mountHeader>) =>
      wrapper.findAll('button').map((item) => item.text())
    expect(labels(mountHeader({ cancelable: true }))).not.toContain('Cancelar')
    expect(labels(mountHeader({ syncing: true }))).not.toContain('Cancelar')

    const wrapper = mountHeader({ syncing: true, cancelable: true })
    await button(wrapper, 'Cancelar').trigger('click')
    expect(wrapper.emitted('cancel')).toHaveLength(1)
  })

  it('shows a disabled Cancelando… while the backend stops the sync', () => {
    const wrapper = mountHeader({ syncing: true, cancelling: true })

    expect(wrapper.findAll('button').map((item) => item.text())).not.toContain('Cancelar')
    expect(button(wrapper, 'Cancelando…').attributes('disabled')).toBeDefined()
    expect(button(wrapper, 'Sincronizar NF-e').attributes('data-loading')).toBe('true')
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

  it('warns of an idle distribution with a badge that explains the 60-day rule', () => {
    expect(mountHeader().find('.idle-warning').exists()).toBe(false)

    const wrapper = mountHeader({ idleWarning: 'Sem consulta há 45 dias' })
    const badge = wrapper.find('.idle-warning')
    expect(badge.exists()).toBe(true)
    expect(badge.attributes('data-color')).toBe('warning')
    expect(badge.text()).toContain('Sem consulta há 45 dias')
    const tooltip = badge.find('.tooltip').text()
    expect(tooltip).toContain('últimos 60 dias')
    expect(tooltip).toContain('Sincronizar agora reinicia a contagem')
  })

  it('marks the first sync on the sync button', () => {
    const wrapper = mountHeader({ firstSync: true })

    const sync = wrapper
      .findAll('button')
      .find((item) => item.text().startsWith('Sincronizar NF-e'))
    expect(sync?.classes()).toContain('first-sync')
    expect(sync?.find('.badge').attributes('data-color')).toBe('accent')
    expect(sync?.find('.badge').text()).toBe('Primeira sincronização')
    expect(sync?.find('.tooltip').text()).toBe('Busque as notas da empresa pela primeira vez.')
  })

  it('shows no first-sync badge by default', () => {
    const sync = button(mountHeader(), 'Sincronizar NF-e')

    expect(sync.classes()).not.toContain('first-sync')
    expect(sync.find('.badge').exists()).toBe(false)
    expect(sync.find('.tooltip').exists()).toBe(false)
  })

  it('colors the ambiente badge and the banner for the theme', () => {
    const light = mountHeader({
      ambiente: { label: 'Produção', color: 'negative' },
      blockedText: 'x',
    })
    expect(light.find('.badge').text()).toBe('Produção')
    expect(light.find('.badge').attributes('data-text-color')).toBe('white')

    quasar.dark.isActive = true
    const dark = mountHeader({
      ambiente: { label: 'Produção', color: 'negative' },
      blockedText: 'x',
    })
    expect(dark.find('.badge').attributes('data-text-color')).toBe('dark')
    expect(dark.find('.banner').classes()).toContain('bg-grey-9')
  })
})
