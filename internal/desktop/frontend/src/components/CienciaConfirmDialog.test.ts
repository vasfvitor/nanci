import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import CienciaConfirmDialog from './CienciaConfirmDialog.vue'
import type { NFeCienciaPlan } from '@/types/desktop'

const onDialogOK = vi.fn()

vi.mock('quasar', () => {
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const useDialogPluginComponent: any = () => ({
    dialogRef: ref(null),
    onDialogHide: vi.fn(),
    onDialogOK,
    onDialogCancel: vi.fn(),
  })
  useDialogPluginComponent.emits = ['ok', 'hide']
  return { useDialogPluginComponent, useQuasar: () => ({ dark: { isActive: false } }) }
})

const plan: NFeCienciaPlan = {
  Eligible: [
    {
      ChaveAcesso: '35240912345678000199550010000000011000000011',
      Serie: '1',
      Numero: '1',
      EmitenteCNPJ: '12345678000199',
      EmitenteName: 'Fornecedor A',
      TotalValue: 10000,
    },
    {
      ChaveAcesso: '35240912345678000199550010000000021000000021',
      Serie: '1',
      Numero: '2',
      EmitenteCNPJ: '12345678000199',
      EmitenteName: 'Fornecedor A',
      TotalValue: 25050,
    },
  ],
  Skipped: [{ ChaveAcesso: '35240998765432000199550010000000031000000031', Reason: 'Nota cancelada' }],
}

function mountDialog() {
  return shallowMount(CienciaConfirmDialog, {
    props: {
      companyName: 'Empresa Um',
      cnpj: '98765432000199',
      tpAmb: '1',
      plan,
    },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QBtn: {
          name: 'QBtn',
          props: ['label', 'disable'],
          emits: ['click'],
          template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
        },
        QCheckbox: {
          name: 'QCheckbox',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
      },
    },
  })
}

function okButton(wrapper: ReturnType<typeof mountDialog>) {
  const button = wrapper
    .findAllComponents({ name: 'QBtn' })
    .find((btn) => btn.props('label') === 'Registrar ciência na SEFAZ')
  if (!button) throw new Error('OK button not found')
  return button
}

describe('CienciaConfirmDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('keeps OK disabled until the acknowledgment is checked', async () => {
    const wrapper = mountDialog()
    expect(okButton(wrapper).props('disable')).toBe(true)

    await okButton(wrapper).trigger('click')
    expect(onDialogOK).not.toHaveBeenCalled()

    wrapper.getComponent({ name: 'QCheckbox' }).vm.$emit('update:modelValue', true)
    await wrapper.vm.$nextTick()

    expect(okButton(wrapper).props('disable')).toBe(false)
  })

  it('emits exactly the eligible chaves', async () => {
    const wrapper = mountDialog()
    wrapper.getComponent({ name: 'QCheckbox' }).vm.$emit('update:modelValue', true)
    await wrapper.vm.$nextTick()

    await okButton(wrapper).trigger('click')

    expect(onDialogOK).toHaveBeenCalledWith([
      '35240912345678000199550010000000011000000011',
      '35240912345678000199550010000000021000000021',
    ])
  })

  it('lists the skipped notes with their reasons and the totals', () => {
    const wrapper = mountDialog()
    expect(wrapper.text()).toContain('Nota cancelada')
    expect(wrapper.text()).toContain('2 notas')
    expect(wrapper.text()).toContain('Produção')
  })
})
