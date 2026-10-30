import {
  AddCompany,
  AddCredential,
  AssignCredentialToCompany,
  CancelCertPassword,
  ListCompanies,
  ListCredentials,
  SelectCertificate,
  SubmitCertPassword,
  UpdateCompany,
  UpdateCredentialData,
  UpdateCredentialPath,
} from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app'
import type {
  AddCompanyInput,
  AddCredentialInput,
  AssignCredentialInput,
  CompanySummary,
  CredentialSummary,
  UpdateCompanyInput,
  UpdateCredentialDataInput,
  UpdateCredentialPathInput,
} from '@/types/desktop'
import { callWails, asString, asNullableNumber, asRawRecord, asDate } from './base'

export function mapCompanySummary(raw: unknown): CompanySummary {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    CNPJ: asString(item['CNPJ']),
    CNPJRoot: asString(item['CNPJRoot']),
    Name: asString(item['Name']),
    CredentialID: asString(item['CredentialID']),
    CredentialLabel: asString(item['CredentialLabel']),
    CredentialCertPath: asString(item['CredentialCertPath']),
    Environment: asString(item['Environment']),
    UF: asString(item['UF']),
    LastFoundNSU: asNullableNumber(item['LastFoundNSU']),
    LastSyncAt: asDate(item['LastSyncAt']),
    SyncStartPolicy: asString(item['SyncStartPolicy']) as CompanySummary['SyncStartPolicy'],
    SyncStartDate: asDate(item['SyncStartDate']),
    InitialSyncDoneAt: asDate(item['InitialSyncDoneAt']),
    LastRunStatus: asString(item['LastRunStatus']),
    LastRunStopReason: asString(item['LastRunStopReason']),
    CreatedAt: asDate(item['CreatedAt']),
    UpdatedAt: asDate(item['UpdatedAt']),
  }
}

export function mapCredentialSummary(raw: unknown): CredentialSummary {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    Label: asString(item['Label']),
    CertPath: asString(item['CertPath']),
    OwnerCNPJ: asString(item['OwnerCNPJ']),
    OwnerCNPJRoot: asString(item['OwnerCNPJRoot']),
    FingerprintSHA256: asString(item['FingerprintSHA256']),
    SubjectName: asString(item['SubjectName']),
    NotBefore: asDate(item['NotBefore']),
    NotAfter: asDate(item['NotAfter']),
    InspectedAt: asDate(item['InspectedAt']),
    CreatedAt: asDate(item['CreatedAt']),
    UpdatedAt: asDate(item['UpdatedAt']),
  }
}

export const companiesClient = {
  addCompany(input: AddCompanyInput) {
    return callWails(() => AddCompany(input))
  },
  addCredential(input: AddCredentialInput) {
    return callWails(() => AddCredential(input))
  },
  assignCredential(input: AssignCredentialInput) {
    return callWails(() => AssignCredentialToCompany(input))
  },
  async listCompanies(): Promise<CompanySummary[]> {
    const result = await callWails(() => ListCompanies())
    return (result || []).map(mapCompanySummary)
  },
  async listCredentials(): Promise<CredentialSummary[]> {
    const result = await callWails(() => ListCredentials())
    return (result || []).map(mapCredentialSummary)
  },
  async selectCertificate(): Promise<string | null> {
    const path = await callWails(() => SelectCertificate())
    return path || null
  },
  submitCertPassword(requestID: string, password: string) {
    return callWails(() => SubmitCertPassword(requestID, password))
  },
  cancelCertPassword(requestID: string) {
    return callWails(() => CancelCertPassword(requestID))
  },
  updateCompany(input: UpdateCompanyInput) {
    return callWails(() => UpdateCompany(input))
  },
  updateCredentialData(input: UpdateCredentialDataInput) {
    return callWails(() => UpdateCredentialData(input))
  },
  updateCredentialPath(input: UpdateCredentialPathInput) {
    return callWails(() => UpdateCredentialPath(input))
  },
}
