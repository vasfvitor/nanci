import { shallowMount } from '@vue/test-utils'
import { describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import ExportDialog from './ExportDialog.vue'

vi.mock('quasar', () => {
// eslint-disable-next-line @typescript-eslint/no-explicit-any
  const useDialogPluginComponent: any = () => ({
    dialogRef: ref(null),
    onDialogHide: vi.fn(),
    onDialogOK: vi.fn(),
    onDialogCancel: vi.fn(),
  })
  useDialogPluginComponent.emits = ['ok', 'hide']
  return { useDialogPluginComponent }
})

const globalStubs = {
  renderStubDefaultSlot: true,
  stubs: {
    QDialog: true,
    QCard: true,
    QCardSection: true,
    QOptionGroup: true,
    QCheckbox: true,
    QBtn: true,
    QCardActions: true
  }
}

describe('ExportDialog', () => {
  it('hides incremental option when items are selected', () => {
    const wrapper = shallowMount(ExportDialog, {
      global: globalStubs,
      props: { selectedCount: 3 }
    })
    
    expect(wrapper.find('q-checkbox-stub').exists()).toBe(false)
  })

  it('shows incremental option when no items are selected', () => {
    const wrapper = shallowMount(ExportDialog, {
      global: globalStubs,
      props: { selectedCount: 0 }
    })
    
    expect(wrapper.find('q-checkbox-stub').exists()).toBe(true)
  })
})
