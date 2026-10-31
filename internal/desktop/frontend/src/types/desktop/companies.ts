import type { ISODateValue } from './common'

export type CompanySummary = {
  ID: string
  CNPJ: string
  CNPJRoot: string
  Name: string
  CredentialID: string
  CredentialLabel: string
  CredentialCertPath: string
  Environment: string
  UF: string
  LastFoundNSU: number | null
  LastSyncAt?: ISODateValue
  SyncStartPolicy: SyncStartPolicy
  SyncStartDate?: ISODateValue
  InitialSyncDoneAt?: ISODateValue
  LastRunStatus: string
  LastRunStopReason: string
  LastRunError: string // set only when LastRunStatus is 'failed'
  CreatedAt?: ISODateValue
  UpdatedAt?: ISODateValue
}

export type SyncStartPolicy = 'all' | 'since_date' | 'from_now' | ''

export type CredentialSummary = {
  ID: string
  Label: string
  CertPath: string
  OwnerCNPJ: string
  OwnerCNPJRoot: string
  FingerprintSHA256: string
  SubjectName: string
  NotBefore?: ISODateValue
  NotAfter?: ISODateValue
  InspectedAt?: ISODateValue
  CreatedAt?: ISODateValue
  UpdatedAt?: ISODateValue
}

export type AddCompanyInput = {
  CNPJ: string
  Name: string
  CredentialID: string
  CredentialLabel: string
  CertPath: string
  Environment: string
  UF: string
  SyncStartPolicy: SyncStartPolicy
  SyncStartDate: string
}

export type AddCredentialInput = {
  Label: string
  CertPath: string
}

export type UpdateCompanyInput = {
  CNPJ: string
  Name: string
  Environment: string
  UF: string
  SyncStartPolicy: SyncStartPolicy
  SyncStartDate: string
}

export type UpdateCredentialDataInput = {
  CredentialID: string
  Label: string
}

export type UpdateCredentialPathInput = {
  CredentialID: string
  CertPath: string
}

export type AssignCredentialInput = {
  CompanyCNPJ: string
  CredentialID: string
}

// CertPasswordRequest is the payload of the request-cert-password event: a
// sync or an NF-e event needs a certificate password.
export type CertPasswordRequest = {
  RequestID: string
  CompanyName: string
  TargetCNPJ: string
  CredentialLabel: string
  CertPath: string
  // Purpose tells the user what the password is for, e.g. "Sincronização NFS-e".
  Purpose?: string
}
