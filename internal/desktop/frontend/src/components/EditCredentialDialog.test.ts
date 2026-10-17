import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import DraftBanner from './DraftBanner.vue'
import EditCredentialDialog from './EditCredentialDialog.vue'
import { desktopClient } from '@/platform/wails/client'
import type { CredentialSummary } from '@/types/desktop'

vi.mock('quasar', () => ({
  useQuasar: () => ({ notify: vi.fn() }),
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
    global: {
      stubs: {
        'q-dialog': { template: '<div><slot /></div>' },
        'q-card': { template: '<div><slot /></div>' },
        'q-card-section': { template: '<div><slot /></div>' },
        'q-card-actions': { template: '<div><slot /></div>' },
        'q-btn': { name: 'QBtn', props: ['label'], template: '<button />' },
        'q-input': {
          name: 'QInput',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
      },
      directives: {
        ClosePopup: {},
      },
    },
  })
}

function labelInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.getComponent({ name: 'QInput' })
}

async function save(wrapper: ReturnType<typeof mountDialog>) {
  await wrapper
    .findAllComponents({ name: 'QBtn' })
    .find((button) => button.props('label') === 'Salvar')
    ?.trigger('click')
  await flushPromises()
}

describe('EditCredentialDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
  })

  it('sends the new label', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    expect(labelInput(wrapper).props('modelValue')).toBe('Certificado A')

    labelInput(wrapper).vm.$emit('update:modelValue', 'Certificado B')
    await flushPromises()
    await save(wrapper)

    expect(desktopClient.updateCredentialData).toHaveBeenCalledWith({
      CredentialID: 'cred-1',
      Label: 'Certificado B',
    })
    expect(wrapper.emitted('updated')).toHaveLength(1)
  })

  it('keeps edits as a draft', async () => {
    const wrapper = mountDialog()
    await wrapper.setProps({ modelValue: true })
    await flushPromises()
    labelInput(wrapper).vm.$emit('update:modelValue', 'Rótulo em edição')

    await wrapper.setProps({ modelValue: false })
    await wrapper.setProps({ modelValue: true })
    await flushPromises()

    expect(labelInput(wrapper).props('modelValue')).toBe('Rótulo em edição')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(true)
  })
})
