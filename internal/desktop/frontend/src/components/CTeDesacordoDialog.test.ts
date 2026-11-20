import { flushPromises, shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import CTeDesacordoDialog from './CTeDesacordoDialog.vue'
import type { CTeDesacordoPlan } from '@/types/desktop'
import { makeCTeRow } from '@/test/fixtures'

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

const observacao = 'Carga entregue avariada no destino'

function plan(overrides: Partial<CTeDesacordoPlan> = {}): CTeDesacordoPlan {
  return {
    ChaveAcesso: '35260912345678000195570010000001011123456784',
    TpAmb: '1',
    NSeqEvento: 1,
    Observacao: observacao,
    Due: '2026-10-17T11:30:41Z',
    DaysLeft: 12,
    BlockReason: '',
    ...overrides,
  }
}

function mountDialog(load: (text: string) => Promise<CTeDesacordoPlan>) {
  const row = makeCTeRow({
    ChaveAcesso: '35260912345678000195570010000001011123456784',
    TpAmb: '1',
    Numero: '101',
    Serie: '1',
    EmitenteName: 'Transportadora',
    EmitenteCNPJ: '12345678000195',
    TotalValue: 150000,
  })
  return shallowMount(CTeDesacordoDialog, {
    props: { row, load },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QBtn: {
          name: 'QBtn',
          props: ['label', 'disable', 'loading'],
          emits: ['click'],
          template: '<button :disabled="disable" @click="$emit(\'click\')">{{ label }}</button>',
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
  const found = wrapper
    .findAllComponents({ name: 'QBtn' })
    .find((btn) => btn.props('label') === label)
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

async function typeAndReview(wrapper: ReturnType<typeof mountDialog>, text: string) {
  wrapper.getComponent({ name: 'QInput' }).vm.$emit('update:modelValue', text)
  await wrapper.vm.$nextTick()
  await button(wrapper, 'Revisar').trigger('click')
  await flushPromises()
}

describe('CTeDesacordoDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('keeps review disabled until the observação follows the schema', async () => {
    const wrapper = mountDialog(vi.fn())
    expect(button(wrapper, 'Revisar').props('disable')).toBe(true)

    const input = wrapper.getComponent({ name: 'QInput' })
    input.vm.$emit('update:modelValue', 'Carga “avariada” no destino')
    await wrapper.vm.$nextTick()
    expect(button(wrapper, 'Revisar').props('disable')).toBe(true)

    input.vm.$emit('update:modelValue', observacao)
    await wrapper.vm.$nextTick()
    expect(button(wrapper, 'Revisar').props('disable')).toBe(false)
  })

  it('reviews the plan of the backend and emits the observação', async () => {
    const load = vi.fn().mockResolvedValue(plan())
    const wrapper = mountDialog(load)
    await typeAndReview(wrapper, observacao)

    expect(load).toHaveBeenCalledWith(observacao)
    expect(wrapper.text()).toContain('faltam 12 dia(s)')
    expect(wrapper.text()).toContain('não pode ser desfeito pelo Nanci')

    await button(wrapper, 'Registrar na SEFAZ').trigger('click')
    expect(onDialogOK).toHaveBeenCalledWith(observacao)
  })

  it('warns about a passed deadline without blocking', async () => {
    const wrapper = mountDialog(vi.fn().mockResolvedValue(plan({ DaysLeft: -3 })))
    await typeAndReview(wrapper, observacao)

    expect(wrapper.text()).toContain('vencido há 3 dia(s)')
    expect(wrapper.text()).toContain('cStat 787')
    expect(button(wrapper, 'Registrar na SEFAZ').props('disable')).toBe(false)
  })

  it('shows the block reason and does not send', async () => {
    const wrapper = mountDialog(
      vi
        .fn()
        .mockResolvedValue(plan({ BlockReason: 'CT-e já tem prestação em desacordo registrada' }))
    )
    await typeAndReview(wrapper, observacao)

    expect(wrapper.text()).toContain('já tem prestação em desacordo registrada')
    expect(button(wrapper, 'Registrar na SEFAZ').props('disable')).toBe(true)
    await button(wrapper, 'Registrar na SEFAZ').trigger('click')
    expect(onDialogOK).not.toHaveBeenCalled()
  })

  it('stays on the observação with the error when the plan fails', async () => {
    const wrapper = mountDialog(vi.fn().mockRejectedValue(new Error('CT-e não encontrado')))
    await typeAndReview(wrapper, observacao)

    expect(wrapper.text()).toContain('CT-e não encontrado')
    expect(button(wrapper, 'Revisar').exists()).toBe(true)
  })
})
