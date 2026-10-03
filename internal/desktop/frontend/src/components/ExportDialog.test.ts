import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import ExportDialog, { exportScopeText } from './ExportDialog.vue'
import type { DocumentSource } from '@/utils/documentSources'

const { onDialogOK } = vi.hoisted(() => ({ onDialogOK: vi.fn() }))

vi.mock('quasar', () => {
  const useDialogPluginComponent = Object.assign(
    () => ({
      dialogRef: ref(null),
      onDialogHide: vi.fn(),
      onDialogOK,
      onDialogCancel: vi.fn(),
    }),
    { emits: ['ok', 'hide'] }
  )
  return { useDialogPluginComponent }
})

const xmlZip = [{ label: 'XMLs (ZIP)', value: 'zip' }]
const nfseFormats = [
  { label: 'Planilha CSV', value: 'csv' },
  { label: 'Planilha Excel (XLSX)', value: 'xlsx' },
]

type Props = {
  source: DocumentSource
  count: number
  scope: 'selected' | 'listed'
  formats: { label: string; value: string }[]
  showIncludeResumos?: boolean
}

function mountDialog(props: Partial<Props> = {}) {
  return shallowMount(ExportDialog, {
    props: { source: 'cte', count: 3, scope: 'listed', formats: xmlZip, ...props },
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QOptionGroup: {
          name: 'QOptionGroup',
          props: ['modelValue', 'options'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
        QCheckbox: {
          name: 'QCheckbox',
          props: ['modelValue', 'label'],
          emits: ['update:modelValue'],
          template: '<div />',
        },
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

type Wrapper = ReturnType<typeof mountDialog>

function checkbox(wrapper: Wrapper, label: string) {
  return wrapper.findAllComponents({ name: 'QCheckbox' }).find((box) => box.props('label') === label)
}

async function clickExport(wrapper: Wrapper) {
  const found = wrapper
    .findAllComponents({ name: 'QBtn' })
    .find((btn) => btn.props('label') === 'Exportar')
  if (!found) throw new Error('Exportar button not found')
  await found.trigger('click')
}

describe('exportScopeText', () => {
  it('agrees with the source and the count', () => {
    expect(exportScopeText('nfse', 12, 'listed')).toBe('Serão exportadas as 12 NFS-e da lista.')
    expect(exportScopeText('nfe', 12, 'listed')).toBe('Serão exportadas as 12 NF-e da lista.')
    expect(exportScopeText('cte', 12, 'listed')).toBe('Serão exportados os 12 CT-e da lista.')
    expect(exportScopeText('nfe', 3, 'selected')).toBe('Serão exportadas 3 NF-e selecionadas.')
    expect(exportScopeText('cte', 3, 'selected')).toBe('Serão exportados 3 CT-e selecionados.')
    expect(exportScopeText('cte', 1, 'selected')).toBe('Será exportado 1 CT-e selecionado.')
    expect(exportScopeText('nfse', 1, 'listed')).toBe('Será exportada a NFS-e da lista.')
    expect(exportScopeText('cte', 1, 'listed')).toBe('Será exportado o CT-e da lista.')
  })
})

describe('ExportDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('titles the dialog with the source noun and says what it exports', () => {
    const wrapper = mountDialog({ source: 'nfe', count: 7, scope: 'listed' })
    expect(wrapper.text()).toContain('Exportar NF-e')
    expect(wrapper.text()).toContain('Serão exportadas as 7 NF-e da lista.')
  })

  it('shows a single format as text, without an option group', async () => {
    const wrapper = mountDialog()
    expect(wrapper.findComponent({ name: 'QOptionGroup' }).exists()).toBe(false)
    expect(wrapper.text()).toContain('Formato: XMLs (ZIP)')

    await clickExport(wrapper)
    expect(onDialogOK).toHaveBeenCalledWith({ format: 'zip', incremental: false, includeResumos: false })
  })

  it('offers several formats and returns the one picked', async () => {
    const wrapper = mountDialog({ source: 'nfse', formats: nfseFormats })
    const group = wrapper.getComponent({ name: 'QOptionGroup' })
    expect(group.props('options')).toEqual(nfseFormats)
    expect(group.props('modelValue')).toBe('csv')

    group.vm.$emit('update:modelValue', 'xlsx')
    await clickExport(wrapper)
    expect(onDialogOK).toHaveBeenCalledWith(expect.objectContaining({ format: 'xlsx' }))
  })

  it('always offers "Somente não exportados", unchecked', async () => {
    for (const scope of ['selected', 'listed'] as const) {
      expect(checkbox(mountDialog({ scope }), 'Somente não exportados')).toBeDefined()
    }

    const wrapper = mountDialog()
    expect(checkbox(wrapper, 'Somente não exportados')?.props('modelValue')).toBe(false)
    await clickExport(wrapper)
    expect(onDialogOK).toHaveBeenCalledWith(expect.objectContaining({ incremental: false }))
  })

  it('offers "Incluir resumos" only when asked', async () => {
    expect(checkbox(mountDialog(), 'Incluir resumos')).toBeUndefined()

    const wrapper = mountDialog({ source: 'nfe', showIncludeResumos: true })
    checkbox(wrapper, 'Incluir resumos')?.vm.$emit('update:modelValue', true)
    await clickExport(wrapper)
    expect(onDialogOK).toHaveBeenCalledWith({ format: 'zip', incremental: false, includeResumos: true })
  })
})
