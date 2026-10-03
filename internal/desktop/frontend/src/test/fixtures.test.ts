import { describe, expect, it } from 'vitest'
import { company, deferred } from './fixtures'

describe('deferred', () => {
  it('settles when the test says so', async () => {
    const ok = deferred<number>()
    ok.resolve(7)
    await expect(ok.promise).resolves.toBe(7)

    const failed = deferred<number>()
    failed.reject(new Error('boom'))
    await expect(failed.promise).rejects.toThrow('boom')
  })
})

describe('company', () => {
  it('builds a full company and takes overrides', () => {
    const acme = company('12345678000100', 'ACME', { LastFoundNSU: 25 })

    expect(acme).toMatchObject({
      ID: 'company-12345678000100',
      CNPJ: '12345678000100',
      CNPJRoot: '12345678',
      Name: 'ACME',
      Environment: 'producao',
      LastFoundNSU: 25,
    })
  })
})
