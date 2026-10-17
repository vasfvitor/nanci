import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import AddCredentialDialog from './AddCredentialDialog.vue'
import DraftBanner from './DraftBanner.vue'
import { desktopClient } from '@/platform/wails/client'
import { dialogDirectives, dialogStubs, submitForm } from '@/test/dialogStubs'

const notify = vi.hoisted(() => vi.fn())

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    selectCertificate: vi.fn(),
    addCredential: vi.fn(),
  },
}))

function mountDialog() {
  return shallowMount(AddCredentialDialog, {
    props: { modelValue: false },
    global: { stubs: dialogStubs, directives: dialogDirectives },
  })
}

// The label input is the first q-input in the template.
function labelInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.getComponent({ name: 'QInput' })
}

function certPathInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper
    .findAllComponents({ name: 'QInput' })
    .find((input) => input.props('label') === 'Caminho do Certificado (.pfx/.p12)')
}

describe('AddCredentialDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('keeps an abandoned form as a draft when reopened after cancelling', async () => {
    const wrapper = mountDialog()

    await wrapper.setProps({ modelValue: true })
    labelInput(wrapper).vm.$emit('update:modelValue', 'rótulo abandonado')
    await flushPromises()
    expect(labelInput(wrapper).props('modelValue')).toBe('rótulo abandonado')

    // Cancel, then reopen.
    await wrapper.setProps({ modelValue: false })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(labelInput(wrapper).props('modelValue')).toBe('rótulo abandonado')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(true)
  })

  it('asks for the certificate inline', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    expect(labelInput(wrapper).props('autofocus')).toBe(true)
    expect(certPathInput(wrapper)?.props('error')).toBe(false)

    await submitForm(wrapper)

    expect(desktopClient.addCredential).not.toHaveBeenCalled()
    expect(notify).not.toHaveBeenCalled()
    expect(certPathInput(wrapper)?.props('error')).toBe(true)
    expect(certPathInput(wrapper)?.props('errorMessage')).toBe('Selecione um certificado.')

    vi.mocked(desktopClient.selectCertificate).mockResolvedValue('C:\\certs\\empresa.pfx')
    // The folder button is the one without a label.
    await wrapper
      .findAllComponents({ name: 'QBtn' })
      .find((button) => button.props('label') === undefined)
      ?.trigger('click')
    await flushPromises()
    await submitForm(wrapper)

    expect(desktopClient.addCredential).toHaveBeenCalledWith({
      Label: 'empresa.pfx',
      CertPath: 'C:\\certs\\empresa.pfx',
    })
  })
})
