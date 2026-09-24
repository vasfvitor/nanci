import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import NFeEventResultsDialog from './NFeEventResultsDialog.vue'
import type { NFeEventBatchResult, NFeEventResult } from '@/types/desktop'

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

function outcome(chave: string, status: NFeEventResult['Status']): NFeEventResult {
  return { ChaveAcesso: chave, TpEvento: '210210', Status: status, CStat: '', XMotivo: '', Protocolo: '' }
}

function mountDialog(result: NFeEventBatchResult) {
  return shallowMount(NFeEventResultsDialog, {
    props: { result },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QTable: { name: 'QTable', props: ['rows'], template: '<div />' },
        QBanner: { template: '<div class="banner"><slot /></div>' },
        QBtn: {
          name: 'QBtn',
          props: ['label'],
          emits: ['click'],
          template: '<button @click="$emit(\'click\')">{{ label }}</button>',
        },
      },
    },
  })
}

describe('NFeEventResultsDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('counts every outcome and lists only the notes with a problem', () => {
    const rejeitada = outcome('c', 'rejeitada')
    const naoEnviada = outcome('d', 'nao_enviada')
    const desconhecida = outcome('e', '')
    const wrapper = mountDialog({
      Results: [outcome('a', 'registrada'), outcome('b', 'ja_registrada'), rejeitada, naoEnviada, desconhecida],
      Skipped: [],
      Interrupted: '',
    })

    expect(wrapper.text()).toMatch(/1 registradas · 1 já registradas ·\s+1 rejeitadas · 2 não enviadas/)
    expect(wrapper.getComponent({ name: 'QTable' }).props('rows')).toEqual([rejeitada, naoEnviada, desconhecida])
    expect(wrapper.find('.banner').exists()).toBe(false)
  })

  it('explains an interrupted batch', () => {
    const wrapper = mountDialog({
      Results: [outcome('a', 'nao_enviada')],
      Skipped: [],
      Interrupted: 'senha não informada',
    })

    expect(wrapper.find('.banner').text()).toContain('O envio foi interrompido: senha não informada.')
  })

  it('closes with OK', async () => {
    const wrapper = mountDialog({ Results: [], Skipped: [], Interrupted: '' })
    await wrapper.getComponent({ name: 'QBtn' }).trigger('click')
    expect(onDialogOK).toHaveBeenCalled()
  })
})
