import type { CompanySummary } from '@/types/desktop'

// deferred is a promise the test settles by hand, to hold a backend call in
// flight while the test acts.
export function deferred<T>() {
  let resolve!: (value: T) => void
  let reject!: (reason: unknown) => void
  const promise = new Promise<T>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

// company builds a CompanySummary: a production company in SP with no
// credential and no sync yet, changed by fields.
export function company(
  CNPJ: string,
  Name: string,
  fields: Partial<CompanySummary> = {}
): CompanySummary {
  return {
    ID: `company-${CNPJ}`,
    CNPJ,
    CNPJRoot: CNPJ.slice(0, 8),
    Name,
    CredentialID: '',
    CredentialLabel: '',
    CredentialCertPath: '',
    Environment: 'producao',
    UF: 'SP',
    LastFoundNSU: null,
    SyncStartPolicy: 'all',
    LastRunStatus: '',
    LastRunStopReason: '',
    ...fields,
  }
}
