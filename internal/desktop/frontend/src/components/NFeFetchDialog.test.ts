import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import NFeFetchDialog from './NFeFetchDialog.vue'
import { dialogStubs, submitForm } from '@/test/dialogStubs'

const { onDialogOK, onDialogCancel } = vi.hoisted(() => ({
  onDialogOK: vi.fn(),
  onDialogCancel: vi.fn(),
}))

vi.mock('quasar', () => {
  const useDialogPluginComponent = Object.assign(
    () => ({
      dialogRef: ref(null),
      onDialogHide: vi.fn(),
      onDialogOK,
      onDialogCancel,
    }),
    { emits: ['ok', 'hide'] }
  )
  return { useDialogPluginComponent }
})

const chave = '35260912345678000195550010000000011000000010'

function mountDialog() {
  return shallowMount(NFeFetchDialog, {
    props: { requestsLastHour: 3, requestBudget: 20 },
    global: {
      stubs: {
        ...dialogStubs,
        'q-btn': {
          name: 'QBtn',
          props: { label: String, type: String, disable: Boolean },
          emits: ['click'],
          template: '<button @click="$emit(\'click\')">{{ label }}</button>',
        },
      },
    },
  })
}

function chaveInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.getComponent({ name: 'QInput' })
}

function button(wrapper: ReturnType<typeof mountDialog>, label: string) {
  const found = wrapper
    .findAllComponents({ name: 'QBtn' })
    .find((btn) => btn.props('label') === label)
  if (!found) throw new Error(`button ${label} not found`)
  return found
}

function searchButton(wrapper: ReturnType<typeof mountDialog>) {
  return button(wrapper, 'Buscar')
}

describe('NFeFetchDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('says how much of the hourly budget the fetch spends', () => {
    const wrapper = mountDialog()
    expect(wrapper.text()).toContain('Gasta 1 das 20 consultas da hora (3 usadas)')
  })

  it('shows no error on an empty field and refuses to submit', async () => {
    const wrapper = mountDialog()

    expect(chaveInput(wrapper).props('error')).toBe(false)
    expect(searchButton(wrapper).props('disable')).toBe(true)
    await submitForm(wrapper)
    expect(onDialogOK).not.toHaveBeenCalled()
  })

  it('asks for 44 characters until the chave is complete', async () => {
    const wrapper = mountDialog()

    chaveInput(wrapper).vm.$emit('update:modelValue', chave.slice(0, 40))
    await wrapper.vm.$nextTick()
    expect(chaveInput(wrapper).props('error')).toBe(true)
    expect(chaveInput(wrapper).props('errorMessage')).toBe('A chave tem 44 caracteres')
    expect(searchButton(wrapper).props('disable')).toBe(true)
    await submitForm(wrapper)
    expect(onDialogOK).not.toHaveBeenCalled()
  })

  it('compacts a pasted chave and submits it on Enter', async () => {
    const wrapper = mountDialog()

    const grouped = (chave.match(/.{4}/g) ?? []).join(' ')
    chaveInput(wrapper).vm.$emit('update:modelValue', grouped)
    await wrapper.vm.$nextTick()
    expect(chaveInput(wrapper).props('modelValue')).toBe(chave)
    expect(chaveInput(wrapper).props('error')).toBe(false)
    expect(searchButton(wrapper).props('disable')).toBe(false)

    await submitForm(wrapper)
    expect(onDialogOK).toHaveBeenCalledWith(chave)
  })

  it('cancels', async () => {
    const wrapper = mountDialog()
    await button(wrapper, 'Cancelar').trigger('click')
    expect(onDialogCancel).toHaveBeenCalled()
  })
})
