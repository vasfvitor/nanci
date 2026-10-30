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
import type * as desktopapi from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/desktopapi/models'
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
import { callWails, asNullableNumber, asDate } from './base'

export function mapCompanySummary(item: desktopapi.CompanySummary): CompanySummary {
  return {
    ID: item.ID,
    CNPJ: item.CNPJ,
    CNPJRoot: item.CNPJRoot,
    Name: item.Name,
    CredentialID: item.CredentialID,
    CredentialLabel: item.CredentialLabel,
    CredentialCertPath: item.CredentialCertPath,
    Environment: item.Environment,
    UF: item.UF,
    LastFoundNSU: asNullableNumber(item.LastFoundNSU),
    LastSyncAt: asDate(item.LastSyncAt),
    SyncStartPolicy: item.SyncStartPolicy as CompanySummary['SyncStartPolicy'],
    SyncStartDate: asDate(item.SyncStartDate),
    InitialSyncDoneAt: asDate(item.InitialSyncDoneAt),
    LastRunStatus: item.LastRunStatus,
    LastRunStopReason: item.LastRunStopReason,
    CreatedAt: item.CreatedAt,
    UpdatedAt: item.UpdatedAt,
  }
}

export function mapCredentialSummary(item: desktopapi.CredentialSummary): CredentialSummary {
  return {
    ID: item.ID,
    Label: item.Label,
    CertPath: item.CertPath,
    OwnerCNPJ: item.OwnerCNPJ,
    OwnerCNPJRoot: item.OwnerCNPJRoot,
    FingerprintSHA256: item.FingerprintSHA256,
    SubjectName: item.SubjectName,
    NotBefore: asDate(item.NotBefore),
    NotAfter: asDate(item.NotAfter),
    InspectedAt: asDate(item.InspectedAt),
    CreatedAt: item.CreatedAt,
    UpdatedAt: item.UpdatedAt,
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
