import { flushPromises, shallowMount } from '@vue/test-utils'
import { createPinia, setActivePinia } from 'pinia'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import AddCompanyDialog from './AddCompanyDialog.vue'
import DraftBanner from './DraftBanner.vue'
import { desktopClient } from '@/platform/wails/client'
import { useFormDraftsStore } from '@/stores/formDrafts'
import { dialogDirectives, dialogStubs, submitForm } from '@/test/dialogStubs'
import { deferred } from '@/test/fixtures'

const notify = vi.hoisted(() => vi.fn())

// Keep Quasar's real date helpers; only useQuasar needs stubbing.
vi.mock('quasar', async (importOriginal) => {
  const actual = await importOriginal<typeof import('quasar')>()
  return { ...actual, useQuasar: () => ({ notify }) }
})

vi.mock('@/platform/wails/client', () => ({
  desktopClient: {
    listCredentials: vi.fn().mockResolvedValue([]),
    selectCertificate: vi.fn(),
    addCompany: vi.fn(),
  },
}))

const CNPJ = '12345678000195'

function mountDialog() {
  return shallowMount(AddCompanyDialog, {
    props: { modelValue: false },
    global: { stubs: dialogStubs, directives: dialogDirectives },
  })
}

// The CNPJ input is the first q-input in the template.
function cnpjInput(wrapper: ReturnType<typeof mountDialog>) {
  return wrapper.getComponent({ name: 'QInput' })
}

function input(wrapper: ReturnType<typeof mountDialog>, label: string) {
  return wrapper
    .findAllComponents({ name: 'QInput' })
    .find((field) => field.props('label') === label)
}

function select(wrapper: ReturnType<typeof mountDialog>, label: string) {
  return wrapper
    .findAllComponents({ name: 'QSelect' })
    .find((field) => field.props('label') === label)
}

function button(wrapper: ReturnType<typeof mountDialog>, label: string) {
  return wrapper.findAllComponents({ name: 'QBtn' }).find((field) => field.props('label') === label)
}

async function openDialog(wrapper: ReturnType<typeof mountDialog>) {
  await wrapper.setProps({ modelValue: true })
  await flushPromises()
}

async function reopen(wrapper: ReturnType<typeof mountDialog>) {
  await wrapper.setProps({ modelValue: false })
  await openDialog(wrapper)
}

async function fillCompany(wrapper: ReturnType<typeof mountDialog>) {
  cnpjInput(wrapper).vm.$emit('update:modelValue', CNPJ)
  input(wrapper, 'Nome / Razão Social')?.vm.$emit('update:modelValue', 'Empresa Um')
  await flushPromises()
}

describe('AddCompanyDialog', () => {
  beforeEach(() => {
    setActivePinia(createPinia())
    vi.clearAllMocks()
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([
      { ID: 'cred-1', Label: 'Certificado A' },
    ] as never)
  })

  afterEach(() => {
    vi.useRealTimers()
  })

  it('anchors a relative sync start to the local date, not the UTC one', async () => {
    // Late evening in UTC-3: the UTC calendar date is already the next day.
    // Only Date is faked, so flushPromises' timers keep working.
    vi.useFakeTimers({ toFake: ['Date'] })
    vi.setSystemTime(new Date('2026-07-26T23:30:00-03:00'))

    const wrapper = mountDialog()
    // Open so credentials load and the form is reset before filling it in.
    await openDialog(wrapper)
    await fillCompany(wrapper)
    // The first option group selects the initial-history policy.
    wrapper.getComponent({ name: 'QOptionGroup' }).vm.$emit('update:modelValue', 'last_12_months')
    await flushPromises()
    await submitForm(wrapper)

    expect(desktopClient.addCompany).toHaveBeenCalledWith(
      expect.objectContaining({
        SyncStartPolicy: 'since_date',
        SyncStartDate: '2025-07-26',
      })
    )
  })

  it('sends the selected UF, and an empty one after clearing', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    await fillCompany(wrapper)
    expect(select(wrapper, 'UF (opcional)')?.props('modelValue')).toBe('')

    select(wrapper, 'UF (opcional)')?.vm.$emit('update:modelValue', 'SP')
    await flushPromises()
    await submitForm(wrapper)
    expect(desktopClient.addCompany).toHaveBeenLastCalledWith(expect.objectContaining({ UF: 'SP' }))

    // Saving closes and resets the form; reopen and clear the select.
    await reopen(wrapper)
    await fillCompany(wrapper)
    select(wrapper, 'UF (opcional)')?.vm.$emit('update:modelValue', null)
    await flushPromises()
    await submitForm(wrapper)
    expect(desktopClient.addCompany).toHaveBeenLastCalledWith(expect.objectContaining({ UF: '' }))
  })

  it('defaults a new company to producao', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    expect(select(wrapper, 'Ambiente da Empresa')?.props('modelValue')).toBe('producao')

    await fillCompany(wrapper)
    await submitForm(wrapper)
    expect(desktopClient.addCompany).toHaveBeenLastCalledWith(
      expect.objectContaining({ Environment: 'producao' })
    )
  })

  it('keeps an abandoned form as a draft when reopened after cancelling', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(false)

    cnpjInput(wrapper).vm.$emit('update:modelValue', CNPJ)
    await flushPromises()
    expect(cnpjInput(wrapper).props('modelValue')).toBe(CNPJ)

    // Cancel, then reopen.
    await reopen(wrapper)

    expect(cnpjInput(wrapper).props('modelValue')).toBe(CNPJ)
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(true)
  })

  it('Limpar starts an empty form', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    cnpjInput(wrapper).vm.$emit('update:modelValue', CNPJ)
    await reopen(wrapper)

    wrapper.getComponent(DraftBanner).vm.$emit('clear')
    await flushPromises()

    expect(cnpjInput(wrapper).props('modelValue')).toBe('')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(false)
    await reopen(wrapper)
    expect(cnpjInput(wrapper).props('modelValue')).toBe('')
  })

  it('blocks backdrop dismiss only while dirty', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    const dialog = () => wrapper.getComponent({ name: 'QDialog' })
    expect(dialog().props('noBackdropDismiss')).toBe(false)

    cnpjInput(wrapper).vm.$emit('update:modelValue', '1')
    await flushPromises()
    expect(dialog().props('noBackdropDismiss')).toBe(true)

    cnpjInput(wrapper).vm.$emit('update:modelValue', '')
    await flushPromises()
    expect(dialog().props('noBackdropDismiss')).toBe(false)
  })

  it('falls back to the first credential when the drafted one is gone', async () => {
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([
      { ID: 'cred-1', Label: 'Certificado A' },
      { ID: 'cred-2', Label: 'Certificado B' },
    ] as never)
    const wrapper = mountDialog()
    await openDialog(wrapper)
    expect(select(wrapper, 'Credencial')?.props('modelValue')).toBe('cred-1')
    select(wrapper, 'Credencial')?.vm.$emit('update:modelValue', 'cred-2')
    cnpjInput(wrapper).vm.$emit('update:modelValue', CNPJ)
    await flushPromises()

    // cred-2 is deleted while the dialog is closed.
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([
      { ID: 'cred-3', Label: 'Certificado C' },
      { ID: 'cred-1', Label: 'Certificado A' },
    ] as never)
    await reopen(wrapper)

    expect(cnpjInput(wrapper).props('modelValue')).toBe(CNPJ)
    expect(select(wrapper, 'Credencial')?.props('modelValue')).toBe('cred-3')
  })

  it('a saved company leaves no draft and emits added with the CNPJ', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    await fillCompany(wrapper)
    await submitForm(wrapper)

    expect(wrapper.emitted('added')).toEqual([[CNPJ]])
    expect(wrapper.emitted('update:modelValue')?.at(-1)).toEqual([false])
    expect(useFormDraftsStore().get('company:add')).toBeUndefined()

    await reopen(wrapper)
    expect(cnpjInput(wrapper).props('modelValue')).toBe('')
    expect(wrapper.findComponent(DraftBanner).exists()).toBe(false)
  })

  it('stays open and disables Cancel while saving', async () => {
    const save = deferred<undefined>()
    vi.mocked(desktopClient.addCompany).mockReturnValue(save.promise)
    const wrapper = mountDialog()
    await openDialog(wrapper)
    await fillCompany(wrapper)
    await submitForm(wrapper)

    expect(wrapper.getComponent({ name: 'QDialog' }).props('persistent')).toBe(true)
    expect(button(wrapper, 'Cancelar')?.props('disable')).toBe(true)
    expect(button(wrapper, 'Salvar')?.props('loading')).toBe(true)
    // Enter again while the save is in flight does not send a second one.
    await submitForm(wrapper)
    expect(desktopClient.addCompany).toHaveBeenCalledTimes(1)

    save.resolve(undefined)
    await flushPromises()
    expect(wrapper.getComponent({ name: 'QDialog' }).props('persistent')).toBe(false)
    expect(button(wrapper, 'Cancelar')?.props('disable')).toBe(false)
  })

  it('shows missing fields inline without a toast', async () => {
    vi.mocked(desktopClient.listCredentials).mockResolvedValue([])
    const wrapper = mountDialog()
    await openDialog(wrapper)
    // Nothing is flagged before the first save attempt.
    expect(cnpjInput(wrapper).props('error')).toBe(false)

    await submitForm(wrapper)

    expect(desktopClient.addCompany).not.toHaveBeenCalled()
    expect(notify).not.toHaveBeenCalled()
    expect(cnpjInput(wrapper).props('error')).toBe(true)
    expect(cnpjInput(wrapper).props('errorMessage')).toBe('Informe o CNPJ.')
    const name = input(wrapper, 'Nome / Razão Social')
    expect(name?.props('error')).toBe(true)
    expect(name?.props('errorMessage')).toBe('Informe o nome da empresa.')
    // With no credential, the form starts on a new one and asks for its file.
    const certPath = input(wrapper, 'Caminho do Certificado (.pfx/.p12)')
    expect(certPath?.props('errorMessage')).toBe('Selecione um certificado.')

    wrapper.getComponent({ name: 'QOptionGroup' }).vm.$emit('update:modelValue', 'custom_date')
    await flushPromises()
    expect(input(wrapper, 'Importar a partir de')?.props('errorMessage')).toBe(
      'Informe a data inicial de importação.'
    )

    // The error clears as the field is filled.
    await fillCompany(wrapper)
    expect(cnpjInput(wrapper).props('error')).toBe(false)
  })

  it('rejects a CNPJ with wrong check digits', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)

    // A short CNPJ waits for a save attempt.
    cnpjInput(wrapper).vm.$emit('update:modelValue', '1234567800019')
    await flushPromises()
    expect(cnpjInput(wrapper).props('error')).toBe(false)

    // The check digits are judged as soon as the 14 characters are in.
    cnpjInput(wrapper).vm.$emit('update:modelValue', '12345678000199')
    await flushPromises()
    expect(cnpjInput(wrapper).props('error')).toBe(true)
    expect(cnpjInput(wrapper).props('errorMessage')).toBe('CNPJ inválido.')

    input(wrapper, 'Nome / Razão Social')?.vm.$emit('update:modelValue', 'Empresa Um')
    await submitForm(wrapper)
    expect(desktopClient.addCompany).not.toHaveBeenCalled()

    cnpjInput(wrapper).vm.$emit('update:modelValue', '1234567800019')
    await flushPromises()
    expect(cnpjInput(wrapper).props('errorMessage')).toBe('CNPJ deve ter 14 caracteres.')
  })

  it('normalizes a pasted formatted CNPJ', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    expect(cnpjInput(wrapper).props('mask')).toBe('XX.XXX.XXX/XXXX-##')

    cnpjInput(wrapper).vm.$emit('update:modelValue', '12.abc.345/01de-35')
    input(wrapper, 'Nome / Razão Social')?.vm.$emit('update:modelValue', 'Empresa Um')
    await flushPromises()
    expect(cnpjInput(wrapper).props('modelValue')).toBe('12ABC34501DE35')
    expect(cnpjInput(wrapper).props('error')).toBe(false)

    await submitForm(wrapper)
    expect(desktopClient.addCompany).toHaveBeenCalledWith(
      expect.objectContaining({ CNPJ: '12ABC34501DE35' })
    )
    expect(wrapper.emitted('added')).toEqual([['12ABC34501DE35']])
  })

  it('autofocuses the CNPJ', async () => {
    const wrapper = mountDialog()
    await openDialog(wrapper)
    expect(cnpjInput(wrapper).props('label')).toBe('CNPJ')
    expect(cnpjInput(wrapper).props('autofocus')).toBe(true)
    expect(input(wrapper, 'Nome / Razão Social')?.props('autofocus')).toBe(false)
  })
})
