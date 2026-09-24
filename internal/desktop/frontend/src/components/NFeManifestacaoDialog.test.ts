import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import NFeManifestacaoDialog from './NFeManifestacaoDialog.vue'
import type { NFeRow } from '@/types/desktop'

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

function note(overrides: Partial<NFeRow> = {}): NFeRow {
  return {
    ID: 'rel-1',
    DocumentID: 'doc-1',
    ChaveAcesso: '35240912345678000199550010000123451123456789',
    Serie: '1',
    Numero: '12345',
    Protocolo: '',
    TpNF: '1',
    EmitenteCNPJ: '12345678000199',
    EmitenteName: 'Fornecedor',
    EmitenteIE: '',
    DestinatarioCNPJ: '98765432000199',
    DestinatarioName: 'Empresa',
    TotalValue: 10000,
    Situacao: 'autorizada',
    Completeness: 'completa',
    Manifestacao: 'ciencia',
    CompanyRole: 'destinatario',
    EventCount: 1,
    DaysLeft: null,
    TacitlyConfirmed: false,
    CienciaBlockReason: '',
    ConclusiveBlockReason: '',
    ...overrides,
  }
}

type Option = { value: string; disable: boolean }

function mountDialog(row = note()) {
  return shallowMount(NFeManifestacaoDialog, {
    props: { note: row, tpAmb: '2' },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QBtn: {
          name: 'QBtn',
          props: ['label', 'disable'],
          emits: ['click'],
          template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
        },
        QOptionGroup: {
          name: 'QOptionGroup',
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
        QInput: {
          name: 'QInput',
          props: ['modelValue'],
          emits: ['update:modelValue'],
          template: '<textarea />',
        },
      },
    },
  })
}

function button(wrapper: ReturnType<typeof mountDialog>, label: string) {
  const found = wrapper.findAllComponents({ name: 'QBtn' }).find((btn) => btn.props('label') === label)
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

describe('NFeManifestacaoDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('keeps review disabled until a valid justificativa is typed for 210240', async () => {
    const wrapper = mountDialog()
    expect(button(wrapper, 'Revisar').props('disable')).toBe(true)

    wrapper.getComponent({ name: 'QOptionGroup' }).vm.$emit('update:modelValue', '210240')
    await wrapper.vm.$nextTick()
    expect(button(wrapper, 'Revisar').props('disable')).toBe(true)

    const input = wrapper.getComponent({ name: 'QInput' })
    input.vm.$emit('update:modelValue', 'curta demais')
    await wrapper.vm.$nextTick()
    expect(button(wrapper, 'Revisar').props('disable')).toBe(true)

    input.vm.$emit('update:modelValue', '  mercadoria não foi entregue  ')
    await wrapper.vm.$nextTick()
    expect(button(wrapper, 'Revisar').props('disable')).toBe(false)
  })

  it('emits the tipo and trimmed justificativa after the review step', async () => {
    const wrapper = mountDialog()
    wrapper.getComponent({ name: 'QOptionGroup' }).vm.$emit('update:modelValue', '210240')
    await wrapper.vm.$nextTick()
    wrapper.getComponent({ name: 'QInput' }).vm.$emit('update:modelValue', '  mercadoria não foi entregue  ')
    await wrapper.vm.$nextTick()

    await button(wrapper, 'Revisar').trigger('click')
    expect(wrapper.text()).toContain('Este evento é registrado na SEFAZ e não pode ser desfeito pelo Nanci')

    await button(wrapper, 'Registrar na SEFAZ').trigger('click')
    expect(onDialogOK).toHaveBeenCalledWith({
      tipo: '210240',
      justificativa: 'mercadoria não foi entregue',
    })
  })

  it('sends no justificativa for a confirmação', async () => {
    const wrapper = mountDialog()
    wrapper.getComponent({ name: 'QOptionGroup' }).vm.$emit('update:modelValue', '210200')
    await wrapper.vm.$nextTick()

    await button(wrapper, 'Revisar').trigger('click')
    await button(wrapper, 'Registrar na SEFAZ').trigger('click')

    expect(onDialogOK).toHaveBeenCalledWith({ tipo: '210200', justificativa: '' })
  })

  it('disables blocked options and shows why', async () => {
    const wrapper = mountDialog(
      note({
        Manifestacao: 'confirmada',
        ConclusiveBlockReason: 'NF-e já possui manifestação conclusiva (Confirmada)',
      })
    )
    const options = wrapper.getComponent({ name: 'QOptionGroup' }).props('options') as Option[]

    expect(options.map((option) => [option.value, option.disable])).toEqual([
      ['210200', true],
      ['210220', true],
      ['210240', true],
    ])
    expect(wrapper.text()).toContain('NF-e já possui manifestação conclusiva (Confirmada)')

    wrapper.getComponent({ name: 'QOptionGroup' }).vm.$emit('update:modelValue', '210200')
    await wrapper.vm.$nextTick()
    expect(button(wrapper, 'Revisar').props('disable')).toBe(true)
  })

  it('enables every conclusive option after ciência', () => {
    const wrapper = mountDialog()
    const options = wrapper.getComponent({ name: 'QOptionGroup' }).props('options') as Option[]
    expect(options.every((option) => !option.disable)).toBe(true)
  })
})
