import { flushPromises, type VueWrapper } from '@vue/test-utils'

// dialogStubs stands in for the Quasar components of the company and
// credential dialogs under shallowMount. Each stub declares the props the
// tests read, and the field stubs emit update:modelValue like the real ones.
export const dialogStubs = {
  'q-dialog': {
    name: 'QDialog',
    props: { noBackdropDismiss: Boolean, persistent: Boolean },
    template: '<div><slot /></div>',
  },
  'q-form': { name: 'QForm', emits: ['submit'], template: '<form><slot /></form>' },
  'q-card': { template: '<div><slot /></div>' },
  'q-card-section': { template: '<div><slot /></div>' },
  'q-card-actions': { template: '<div><slot /></div>' },
  'q-separator': { template: '<div />' },
  'q-banner': { template: '<div><slot /></div>' },
  'q-btn': {
    name: 'QBtn',
    props: { label: String, type: String, disable: Boolean, loading: Boolean },
    template: '<button />',
  },
  'q-input': {
    name: 'QInput',
    props: {
      modelValue: [String, Number],
      label: String,
      error: Boolean,
      errorMessage: String,
      autofocus: Boolean,
      mask: String,
    },
    emits: ['update:modelValue'],
    template: '<div />',
  },
  'q-select': {
    name: 'QSelect',
    props: { modelValue: null, label: String, error: Boolean, errorMessage: String },
    emits: ['update:modelValue'],
    template: '<div />',
  },
  'q-option-group': {
    name: 'QOptionGroup',
    props: { modelValue: null, options: Array },
    emits: ['update:modelValue'],
    template: '<div />',
  },
}

export const dialogDirectives = { ClosePopup: {} }

// submitForm submits the dialog form, as Enter or the Salvar button would,
// and waits for the save to settle.
export async function submitForm(wrapper: VueWrapper) {
  wrapper.getComponent({ name: 'QForm' }).vm.$emit('submit')
  await flushPromises()
}
