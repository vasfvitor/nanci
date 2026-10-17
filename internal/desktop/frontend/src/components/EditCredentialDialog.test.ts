import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DraftBanner from './DraftBanner.vue'
import EditCredentialDialog from './EditCredentialDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import { dialogDirectives, dialogStubs, submitForm } from '@/test/dialogStubs'
import type { CredentialSummary } from '@/types/desktop'

const notify = vi.hoisted(() => vi.fn())

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify }),
}))

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    updateCredentialData: vi.fn(),
  },
}))

const credential: CredentialSummary = {
  ID: 'cred-1',
  Label: 'Certificado A',
  CertPath: 'a.pfx',
  OwnerCNPJ: '12345678000195',
  OwnerCNPJRoot: '12345678',
  FingerprintSHA256: '',
  SubjectName: '',
}

function mountDialog() {
  return shallowMount(EditCredentialDialog, {
    props: { modelValue: false, credentialData: credential },
    global: { stubs: dialogStubs, directives: dialogDirectives },
  })
}

function labelInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.getComponent({ name: 'QInput' })
}

async function openDialog(wrapper: ReturnType<typeof mountDialog>) {
  await wrapper.setProps({ modelValue: true })
  await flushPromises()
}

describe('EditCredentialDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('sends the new label', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    expect(labelInput(wrapper).props('modelValue')).toBe('Certificado A')

    labelInput(wrapper).vm.$emit('update:modelValue', 'Certificado B')
    await flushPromises()
    await submitForm(wrapper)

    expect(desktopClient.updateCredentialData).toHaveBeenCalledWith({
      CredentialID: 'cred-1',
      Label: 'Certificado B',
    })
    expect(wrapper.emitted('updated')).toHaveLength(1)
  })

  it('keeps edits as a draft', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    labelInput(wrapper).vm.$emit('update:modelValue', 'Rótulo em edição')

    await wrapper.setProps({ modelValue: false })
    await openDialog(wrapper)

    expect(labelInput(wrapper).props('modelValue')).toBe('Rótulo em edição')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(true)
  })

  it('shows an empty label inline', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    expect(labelInput(wrapper).props('autofocus')).toBe(true)

    labelInput(wrapper).vm.$emit('update:modelValue', '')
    await flushPromises()
    await submitForm(wrapper)

    expect(desktopClient.updateCredentialData).not.toHaveBeenCalled()
    expect(notify).not.toHaveBeenCalled()
    expect(labelInput(wrapper).props('error')).toBe(true)
    expect(labelInput(wrapper).props('errorMessage')).toBe('Informe o rótulo da credencial.')
  })
})
