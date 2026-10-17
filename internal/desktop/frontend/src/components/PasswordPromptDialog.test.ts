import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PasswordPromptDialog from './PasswordPromptDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import { deferred } from '@/test/fixtures'

type Handler = (payload: unknown) => void

const handlers: Record<string, Handler> = {}

vi.mock('@/platform/wails/events', () => ({
  onWailsEvent: (eventName: string, callback: Handler) => {
    handlers[eventName] = callback
    return vi.fn()
  },
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    submitCertPassword: vi.fn(),
    cancelCertPassword: vi.fn(),
  },
}))

function mountDialog() {
  return shallowMount(PasswordPromptDialog, {
    global: {
      stubs: {
        'q-dialog': { props: ['modelValue'], template: '<div v-if="modelValue"><slot /></div>' },
        'q-card': { template: '<div><slot /></div>' },
        'q-card-section': { template: '<div><slot /></div>' },
        'q-card-actions': { template: '<div><slot /></div>' },
        'q-form': {
          emits: ['submit'],
          template: `<form @submit.prevent="$emit('submit')"><slot /></form>`,
        },
        'q-input': {
          props: ['type', 'disable'],
          template: '<div><input :type="type" :disabled="disable" /><slot name="append" /></div>',
        },
        'q-icon': { template: '<i />' },
        'q-btn': {
          props: ['label', 'type', 'disable', 'loading'],
          template: `<button :type="type || 'button'" :disabled="disable">{{ label }}</button>`,
        },
      },
    },
  })
}

const request = {
  RequestID: 'req-1',
  CompanyName: 'Empresa Um',
  TargetCNPJ: '12345678000199',
  CredentialLabel: 'Certificado A',
  CertPath: 'C:\\a.pfx',
}

describe('PasswordPromptDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('shows the purpose of the password request', async () => {
    const wrapper = mountDialog()
    handlers['request-cert-password']?.({ ...request, Purpose: 'Sincronização NF-e' })
    await flushPromises()

    expect(wrapper.text()).toContain('Finalidade: Sincronização NF-e')
    expect(wrapper.text()).toContain('CNPJ: 12345678000199')
    expect(wrapper.text()).not.toContain('CNPJ consultado')
  })

  it('omits the purpose line when the request has none', async () => {
    const wrapper = mountDialog()
    handlers['request-cert-password']?.(request)
    await flushPromises()

    expect(wrapper.text()).toContain('Empresa: Empresa Um')
    expect(wrapper.text()).not.toContain('Finalidade')
  })

  it('toggles password visibility', async () => {
    const wrapper = mountDialog()
    handlers['request-cert-password']?.(request)
    await flushPromises()

    expect(wrapper.find('input').attributes('type')).toBe('password')
    await wrapper.find('[aria-label="Mostrar senha"]').trigger('click')
    expect(wrapper.find('input').attributes('type')).toBe('text')
    await wrapper.find('[aria-label="Ocultar senha"]').trigger('click')
    expect(wrapper.find('input').attributes('type')).toBe('password')

    await wrapper.find('[aria-label="Mostrar senha"]').trigger('click')
    handlers['request-cert-password']?.({ ...request, RequestID: 'req-2' })
    await wrapper.find('form').trigger('submit')
    await flushPromises()
    expect(wrapper.find('input').attributes('type')).toBe('password')
  })

  it('disables the actions while submitting', async () => {
    const pending = deferred<undefined>()
    vi.mocked(desktopClient.submitCertPassword).mockReturnValue(pending.promise)
    const wrapper = mountDialog()
    handlers['request-cert-password']?.(request)
    await flushPromises()

    await wrapper.find('form').trigger('submit')
    const buttons = () => wrapper.findAll('button')
    expect(buttons().map((b) => [b.text(), b.attributes('disabled')])).toEqual([
      ['Cancelar', ''],
      ['Confirmar', ''],
    ])
    await wrapper.find('form').trigger('submit')
    expect(desktopClient.submitCertPassword).toHaveBeenCalledTimes(1)

    pending.resolve(undefined)
    await flushPromises()
    expect(wrapper.find('form').exists()).toBe(false)
  })
})
