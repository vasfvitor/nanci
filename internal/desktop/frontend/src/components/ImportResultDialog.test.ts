import { shallowMount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import { ref } from 'vue'
import ImportResultDialog from './ImportResultDialog.vue'
import type { ImportXMLResult } from '@/types/desktop'

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
  return { useDialogPluginComponent, useQuasar: () => ({ dark: { isActive: false } }) }
})

const chave = '35260912345678000195550010000000011000000010'

const result: ImportXMLResult = {
  Files: [
    {
      Path: 'C:\\XML\\nota.xml',
      Kind: 'completa',
      ChaveAcesso: chave,
      Outcome: 'imported',
      Message: '',
    },
    {
      Path: 'C:\\XML\\outra.xml',
      Kind: 'completa',
      ChaveAcesso: chave,
      Outcome: 'updated',
      Message: '',
    },
    {
      Path: 'C:\\XML\\quebrado.xml',
      Kind: '',
      ChaveAcesso: '',
      Outcome: 'rejected',
      Message: 'XML ilegível',
    },
  ],
  Imported: 1,
  Updated: 1,
  Rejected: 1,
}

function mountDialog(props: { result: ImportXMLResult; source: 'nfe' | 'cte' }) {
  return shallowMount(ImportResultDialog, {
    props,
    global: {
      renderStubDefaultSlot: true,
      stubs: {
        QTable: {
          name: 'QTable',
          props: ['rows', 'columns'],
          template: `<div>
            <div v-for="row in rows" :key="row.Path" class="file-row">
              <span v-for="col in columns" :key="col.name">{{ col.format ? col.format(row[col.field]) : row[col.field] }} </span>
              <slot name="body-cell-outcome" :row="row" :value="row.Outcome" />
            </div>
          </div>`,
        },
        QTd: { template: '<span><slot /></span>' },
        QBadge: { props: ['label'], template: '<span class="badge">{{ label }}</span>' },
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

describe('ImportResultDialog', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('sums up the import and lists every file with its outcome', () => {
    const wrapper = mountDialog({ result, source: 'nfe' })

    expect(wrapper.text()).toContain('Importação de XML da NF-e')
    expect(wrapper.text()).toContain('1 importado, 1 atualizado, 1 rejeitado')
    const rows = wrapper.findAll('.file-row')
    expect(rows).toHaveLength(3)
    expect(rows.map((row) => row.find('.badge').text())).toEqual([
      'Importado',
      'Atualizado',
      'Rejeitado',
    ])
    expect(rows[0]?.text()).toContain('nota.xml')
    expect(rows[0]?.text()).toContain('3526 0912 3456 7800 0195')
  })

  it('shows the reason of a rejected file', () => {
    const wrapper = mountDialog({ result, source: 'cte' })

    expect(wrapper.text()).toContain('Importação de XML do CT-e')
    const rejected = wrapper.findAll('.file-row')[2]?.text()
    expect(rejected).toContain('quebrado.xml')
    expect(rejected).toContain('Rejeitado')
    expect(rejected).toContain('XML ilegível')
    expect(rejected).toContain('—')
  })

  it('closes with Fechar', async () => {
    const wrapper = mountDialog({ result, source: 'nfe' })
    await wrapper.getComponent({ name: 'QBtn' }).trigger('click')
    expect(onDialogOK).toHaveBeenCalled()
  })
})
