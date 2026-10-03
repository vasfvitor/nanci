import { mount } from '@vue/test-utils'
import { beforeEach, describe, expect, it, vi } from 'vitest'
import ChaveCell from './ChaveCell.vue'

const copyChave = vi.hoisted(() => vi.fn())

vi.mock('@/composables/useNotify', () => ({ copyChave }))

const nfeChave = '35240912345678000199550010000123451123456789'

function mountCell(chave: string) {
  return mount(ChaveCell, {
    props: { chave },
    global: {
      stubs: {
        QBtn: {
          props: ['disable', 'icon'],
          emits: ['click'],
          template: '<button :disabled="disable" :data-icon="icon" @click="$emit(\'click\', $event)" />',
        },
      },
    },
  })
}

describe('ChaveCell', () => {
  beforeEach(() => {
    copyChave.mockClear()
  })

  it('shows the end of the key with the grouped key as title', () => {
    const span = mountCell(nfeChave).find('span')
    expect(span.text()).toBe('...1123456789')
    expect(span.attributes('title')).toBe('3524 0912 3456 7800 0199 5500 1000 0123 4511 2345 6789')
  })

  it('titles a key that is not a DF-e key with the key itself', () => {
    const span = mountCell('NFS3524091234').find('span')
    expect(span.attributes('title')).toBe('NFS3524091234')
  })

  it('copies the whole key from the text and from the button', async () => {
    const wrapper = mountCell(nfeChave)
    await wrapper.find('span').trigger('click')
    const button = wrapper.find('button')
    expect(button.attributes('aria-label')).toBe('Copiar chave completa')
    expect(button.attributes('data-icon')).toBe('content_copy')
    await button.trigger('click')

    expect(copyChave.mock.calls).toEqual([[nfeChave], [nfeChave]])
  })

  it('shows a dash and disables copying without a key', () => {
    const wrapper = mountCell('')
    expect(wrapper.find('span').text()).toBe('—')
    expect(wrapper.find('button').attributes('disabled')).toBeDefined()
  })
})
