import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import PasswordPromptDialog from './PasswordPromptDialog.vue'

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
        'q-input': { template: '<input />' },
        'q-btn': { template: '<button />' },
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
})
