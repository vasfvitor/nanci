// NF-e (modelo 55)

import {
  ExportDANFE,
  ExportNFeDocuments,
  ExportNFeXML,
  FetchNFeByChave,
  ImportNFeXML,
  ListNFe,
  ListNFeEvents,
  ListNFePendingManifestacoes,
  MarkNFeViewed,
  PlanNFeCiencia,
  PullNFe,
  RegisterNFeCiencia,
  RegisterNFeManifestacao,
  ResetNFe,
  StatusNFe,
} from '../../../../wailsjs/go/main/App'
import type {
  ExportDANFEInput,
  ExportResult,
  ExportNFeDocumentsInput,
  ExportNFeXMLInput,
  FetchByChaveResult,
  FetchOutcome,
  ImportXMLResult,
  ListNFeInput,
  NFeBlockedReason,
  NFeCienciaPlan,
  NFeCompleteness,
  NFeEvent,
  NFeEventBatchResult,
  NFeEventOutcome,
  NFeEventResult,
  NFeExportFormat,
  NFeExportResult,
  NFeManifestacao,
  NFePendingKind,
  NFePendingRow,
  NFeResetResult,
  NFeRole,
  NFeRow,
  NFeSituacao,
  NFeSkipped,
  NFeStatusResult,
  PullNFeResult,
  RegisterNFeManifestacaoInput,
} from '@/types/desktop'
import {
  callWails,
  asString,
  asNumber,
  asBoolean,
  asNullableNumber,
  asRawRecord,
  asStringArray,
  asArray,
  asDate,
  asEnum,
  type SaveDialog,
  fileTimestamp,
  mapExportResult,
  mapImportXMLResult,
} from './base'
import { systemClient } from './system'

const nfeSituacoes: readonly NFeSituacao[] = ['autorizada', 'denegada', 'cancelada']
const nfeCompletenesses: readonly NFeCompleteness[] = ['resumo', 'completa']
const nfeManifestacoes: readonly NFeManifestacao[] = [
  'nenhuma',
  'ciencia',
  'confirmada',
  'desconhecida',
  'nao_realizada',
]
const nfeRoles: readonly NFeRole[] = [
  'destinatario',
  'emitente',
  'transportador',
  'autorizado',
  'none',
]
const nfeOutcomes: readonly NFeEventOutcome[] = [
  'registrada',
  'ja_registrada',
  'rejeitada',
  'nao_enviada',
]
const nfePendingKinds: readonly NFePendingKind[] = ['sem_ciencia', 'sem_conclusiva']
const nfeBlockedReasons: readonly NFeBlockedReason[] = [
  'caught_up',
  'consumo_indevido',
  'rate_budget',
]
const fetchOutcomes: readonly FetchOutcome[] = ['completa', 'resumo', 'not_found']

function nfeExportDialog(format: NFeExportFormat, cnpj: string): SaveDialog {
  switch (format) {
    case 'danfe-zip':
      return {
        title: 'Salvar ZIP de DANFEs',
        defaultName: `danfes_${cnpj}_${fileTimestamp()}.zip`,
        pattern: '*.zip',
      }
    case 'zip':
      return {
        title: 'Salvar XMLs de NF-e (ZIP)',
        defaultName: `nfe_${cnpj}_${fileTimestamp()}.zip`,
        pattern: '*.zip',
      }
    default:
      return {
        title: 'Exportar NF-e',
        defaultName: `nfe_${cnpj}_${fileTimestamp()}.${format}`,
        pattern: `*.${format}`,
      }
  }
}

// The Go inputs have no optional fields; fill the ones the frontend may omit.
function nfeListRequest(input: ListNFeInput) {
  return { ...input, ChavesAcesso: input.ChavesAcesso ?? [], OnlyUnread: input.OnlyUnread ?? false }
}

export function mapNFeRow(raw: unknown): NFeRow {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    DocumentID: asString(item['DocumentID']),
    ChaveAcesso: asString(item['ChaveAcesso']),
    Serie: asString(item['Serie']),
    Numero: asString(item['Numero']),
    IssueDate: asDate(item['IssueDate']),
    AuthorizedAt: asDate(item['AuthorizedAt']),
    Protocolo: asString(item['Protocolo']),
    TpNF: asString(item['TpNF']),
    TpAmb: asString(item['TpAmb']),
    NatOp: asString(item['NatOp']),
    EmitenteCNPJ: asString(item['EmitenteCNPJ']),
    EmitenteName: asString(item['EmitenteName']),
    EmitenteIE: asString(item['EmitenteIE']),
    EmitenteUF: asString(item['EmitenteUF']),
    DestinatarioCNPJ: asString(item['DestinatarioCNPJ']),
    DestinatarioName: asString(item['DestinatarioName']),
    TotalValue: asNumber(item['TotalValue']),
    ICMSValue: asNumber(item['ICMSValue']),
    IPIValue: asNumber(item['IPIValue']),
    Situacao: asEnum(item['Situacao'], nfeSituacoes),
    Completeness: asEnum(item['Completeness'], nfeCompletenesses),
    LayoutVersion: asString(item['LayoutVersion']),
    ParseWarnings: asStringArray(item['ParseWarnings']),
    Manifestacao: asEnum(item['Manifestacao'], nfeManifestacoes),
    ManifestacaoAt: asDate(item['ManifestacaoAt']),
    CienciaDue: asDate(item['CienciaDue']),
    ConclusiveDue: asDate(item['ConclusiveDue']),
    CompanyRole: asEnum(item['CompanyRole'], nfeRoles),
    EventCount: asNumber(item['EventCount']),
    FirstSeenNSU: asNullableNumber(item['FirstSeenNSU']),
    FirstSyncedAt: asDate(item['FirstSyncedAt']),
    LastSyncedAt: asDate(item['LastSyncedAt']),
    ViewedAt: asDate(item['ViewedAt']),
    DaysLeft: asNullableNumber(item['DaysLeft']),
    CienciaDaysLeft: asNullableNumber(item['CienciaDaysLeft']),
    TacitlyConfirmed: asBoolean(item['TacitlyConfirmed']),
    CienciaBlockReason: asString(item['CienciaBlockReason']),
    ConclusiveBlockReason: asString(item['ConclusiveBlockReason']),
  }
}

export function mapNFePendingRow(raw: unknown): NFePendingRow {
  const item = asRawRecord(raw)
  return {
    ...mapNFeRow(item),
    Kind: asEnum(item['Kind'], nfePendingKinds),
    CienciaOverdue: asBoolean(item['CienciaOverdue']),
  }
}

export function mapNFeEvent(raw: unknown): NFeEvent {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    TpEvento: asString(item['TpEvento']),
    NSeqEvento: asNumber(item['NSeqEvento']),
    Description: asString(item['Description']),
    EventAt: asDate(item['EventAt']),
    RegisteredAt: asDate(item['RegisteredAt']),
    Protocolo: asString(item['Protocolo']),
    CStat: asString(item['CStat']),
    XMotivo: asString(item['XMotivo']),
    Justificativa: asString(item['Justificativa']),
    Correcao: asString(item['Correcao']),
    AutorCNPJ: asString(item['AutorCNPJ']),
    Completeness: asEnum(item['Completeness'], nfeCompletenesses),
    Registered: asBoolean(item['Registered']),
    SentByNanci: asBoolean(item['SentByNanci']),
  }
}

export function mapNFeEventResult(raw: unknown): NFeEventResult {
  const item = asRawRecord(raw)
  return {
    ChaveAcesso: asString(item['ChaveAcesso']),
    TpEvento: asString(item['TpEvento']),
    Status: asEnum(item['Status'], nfeOutcomes),
    CStat: asString(item['CStat']),
    XMotivo: asString(item['XMotivo']),
    Protocolo: asString(item['Protocolo']),
    RegisteredAt: asDate(item['RegisteredAt']),
  }
}

function mapNFeSkipped(raw: unknown): NFeSkipped {
  const item = asRawRecord(raw)
  return {
    ChaveAcesso: asString(item['ChaveAcesso']),
    Reason: asString(item['Reason']),
  }
}

export function mapNFeEventBatchResult(raw: unknown): NFeEventBatchResult {
  const item = asRawRecord(raw)
  return {
    Results: asArray(item['Results']).map(mapNFeEventResult),
    Skipped: asArray(item['Skipped']).map(mapNFeSkipped),
    Interrupted: asString(item['Interrupted']),
  }
}

export function mapNFeCienciaPlan(raw: unknown): NFeCienciaPlan {
  const item = asRawRecord(raw)
  return {
    Eligible: asArray(item['Eligible']).map(mapNFeRow),
    Skipped: asArray(item['Skipped']).map(mapNFeSkipped),
  }
}

export function mapNFeStatus(raw: unknown): NFeStatusResult {
  const item = asRawRecord(raw)
  return {
    CompanyName: asString(item['CompanyName']),
    CNPJ: asString(item['CNPJ']),
    UF: asString(item['UF']),
    TpAmb: asString(item['TpAmb']),
    LastNSU: asNumber(item['LastNSU']),
    MaxNSU: asNullableNumber(item['MaxNSU']),
    LastSyncAt: asDate(item['LastSyncAt']),
    LastRunStatus: asString(item['LastRunStatus']),
    LastRunStopReason: asString(item['LastRunStopReason']),
    InitialSyncDoneAt: asDate(item['InitialSyncDoneAt']),
    NextAllowedAt: asDate(item['NextAllowedAt']),
    BlockedReason: asEnum(item['BlockedReason'], nfeBlockedReasons),
    RequestsLastHour: asNumber(item['RequestsLastHour']),
    RequestBudget: asNumber(item['RequestBudget']),
    IdleDays: asNumber(item['IdleDays']),
    TotalDestinatario: asNumber(item['TotalDestinatario']),
    TotalEmitente: asNumber(item['TotalEmitente']),
    TotalOutros: asNumber(item['TotalOutros']),
    TotalResumos: asNumber(item['TotalResumos']),
    TotalCompletas: asNumber(item['TotalCompletas']),
    PendingCiencia: asNumber(item['PendingCiencia']),
    PendingConclusiva: asNumber(item['PendingConclusiva']),
    CienciaOverdue: asNumber(item['CienciaOverdue']),
  }
}

export function mapPullNFeResult(raw: unknown): PullNFeResult {
  const item = asRawRecord(raw)
  return {
    CompanyName: asString(item['CompanyName']),
    CNPJ: asString(item['CNPJ']),
    Status: asString(item['Status']),
    StopReason: asString(item['StopReason']),
    LastNSU: asNumber(item['LastNSU']),
    MaxNSU: asNullableNumber(item['MaxNSU']),
    CompletasSaved: asNumber(item['CompletasSaved']),
    ResumosSaved: asNumber(item['ResumosSaved']),
    EventsSaved: asNumber(item['EventsSaved']),
    Errors: asNumber(item['Errors']),
    NextAllowedAt: asDate(item['NextAllowedAt']),
    RequestsLastHour: asNumber(item['RequestsLastHour']),
    RequestBudget: asNumber(item['RequestBudget']),
    Duration: asNumber(item['Duration']),
  }
}

function mapNFeExportResult(raw: unknown): NFeExportResult {
  return {
    ...mapExportResult(raw),
    SkippedResumos: asNumber(asRawRecord(raw)['SkippedResumos']),
  }
}

export function mapNFeResetResult(raw: unknown): NFeResetResult {
  const item = asRawRecord(raw)
  return {
    CompanyName: asString(item['CompanyName']),
    CNPJ: asString(item['CNPJ']),
    CompanyDocuments: asNumber(item['CompanyDocuments']),
    Documents: asNumber(item['Documents']),
    Events: asNumber(item['Events']),
    ExportMarks: asNumber(item['ExportMarks']),
    ManifestacoesKept: asNumber(item['ManifestacoesKept']),
  }
}

export function mapFetchByChaveResult(raw: unknown): FetchByChaveResult {
  const item = asRawRecord(raw)
  return {
    ChaveAcesso: asString(item['ChaveAcesso']),
    Outcome: asEnum(item['Outcome'], fetchOutcomes),
    Message: asString(item['Message']),
    RequestsLastHour: asNumber(item['RequestsLastHour']),
    RequestBudget: asNumber(item['RequestBudget']),
  }
}

export const nfeClient = {
  async pullNFe(cnpj: string): Promise<PullNFeResult> {
    const res = await callWails(() => PullNFe({ CNPJ: cnpj }))
    return mapPullNFeResult(res)
  },
  // resetNFe removes the company's NF-e and resets its NF-e sync.
  async resetNFe(cnpj: string): Promise<NFeResetResult> {
    const res = await callWails(() => ResetNFe(cnpj))
    return mapNFeResetResult(res)
  },
  async statusNFe(cnpj: string): Promise<NFeStatusResult> {
    const res = await callWails(() => StatusNFe(cnpj))
    return mapNFeStatus(res)
  },
  async listNFe(input: ListNFeInput): Promise<NFeRow[]> {
    const res = await callWails(() => ListNFe(nfeListRequest(input)))
    return (res || []).map(mapNFeRow)
  },
  // markNFeViewed marks the company's NF-e with these chaves and returns how
  // many were new.
  async markNFeViewed(cnpj: string, chavesAcesso: string[]): Promise<number> {
    return callWails(() => MarkNFeViewed({ CNPJ: cnpj, ChavesAcesso: chavesAcesso }))
  },
  async listNFeEvents(cnpj: string, chaveAcesso: string): Promise<NFeEvent[]> {
    const res = await callWails(() => ListNFeEvents({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
    return (res || []).map(mapNFeEvent)
  },
  // dueWithinDays 0 lists every pending manifestação.
  async listNFePendingManifestacoes(cnpj: string, dueWithinDays = 0): Promise<NFePendingRow[]> {
    const res = await callWails(() =>
      ListNFePendingManifestacoes({ CNPJ: cnpj, DueWithinDays: dueWithinDays })
    )
    return (res || []).map(mapNFePendingRow)
  },
  async planNFeCiencia(cnpj: string, chavesAcesso: string[]): Promise<NFeCienciaPlan> {
    const res = await callWails(() => PlanNFeCiencia({ CNPJ: cnpj, ChavesAcesso: chavesAcesso }))
    return mapNFeCienciaPlan(res)
  },
  async registerNFeCiencia(cnpj: string, chavesAcesso: string[]): Promise<NFeEventBatchResult> {
    const res = await callWails(() =>
      RegisterNFeCiencia({ CNPJ: cnpj, ChavesAcesso: chavesAcesso })
    )
    return mapNFeEventBatchResult(res)
  },
  // importNFeXML stores NF-e and NF-e event XML files without asking SEFAZ.
  async importNFeXML(cnpj: string, files: string[]): Promise<ImportXMLResult> {
    const res = await callWails(() => ImportNFeXML({ CNPJ: cnpj, Files: files }))
    return mapImportXMLResult(res)
  },
  // fetchNFeByChave asks SEFAZ for one NF-e; it spends one request of the
  // company's NF-e hourly budget.
  async fetchNFeByChave(cnpj: string, chaveAcesso: string): Promise<FetchByChaveResult> {
    const res = await callWails(() => FetchNFeByChave({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
    return mapFetchByChaveResult(res)
  },
  async registerNFeManifestacao(input: RegisterNFeManifestacaoInput): Promise<NFeEventResult> {
    const res = await callWails(() => RegisterNFeManifestacao(input))
    return mapNFeEventResult(res)
  },
  async exportNFeXML(
    input: Omit<ExportNFeXMLInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `nfe_${input.ChaveAcesso}.xml`
    const outPath =
      input.OutPath ||
      (await systemClient.selectSaveFile('Salvar XML da NF-e', defaultName, '*.xml'))
    if (!outPath) return null

    const res = await callWails(() =>
      ExportNFeXML({ CNPJ: input.CNPJ, ChaveAcesso: input.ChaveAcesso, OutPath: outPath })
    )
    return mapExportResult(res)
  },
  async exportDANFE(
    input: Omit<ExportDANFEInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `danfe_${input.ChaveAcesso}.pdf`
    const outPath =
      input.OutPath || (await systemClient.selectSaveFile('Salvar DANFE', defaultName, '*.pdf'))
    if (!outPath) return null

    const res = await callWails(() =>
      ExportDANFE({ CNPJ: input.CNPJ, ChaveAcesso: input.ChaveAcesso, OutPath: outPath })
    )
    return mapExportResult(res)
  },
  async exportNFeDocuments(
    input: Omit<ExportNFeDocumentsInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<NFeExportResult | null> {
    const dialog = nfeExportDialog(input.Format, input.CNPJ)
    const defaultName = input.BaseName || dialog.defaultName
    const outPath =
      input.OutPath ||
      (await systemClient.selectSaveFile(dialog.title, defaultName, dialog.pattern))
    if (!outPath) return null

    const res = await callWails(() =>
      ExportNFeDocuments({
        CNPJ: input.CNPJ,
        Competence: input.Competence,
        Role: input.Role,
        Format: input.Format,
        ChavesAcesso: input.ChavesAcesso || [],
        IncludeResumos: input.IncludeResumos,
        Incremental: input.Incremental,
        OutPath: outPath,
      })
    )
    return mapNFeExportResult(res)
  },
}
