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
} from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app'
import type * as desktopapi from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/desktopapi/models'
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
  asNullableNumber,
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

export function mapNFeRow(item: desktopapi.NFeRow): NFeRow {
  return {
    ID: item.ID,
    DocumentID: item.DocumentID,
    ChaveAcesso: item.ChaveAcesso,
    Serie: item.Serie,
    Numero: item.Numero,
    IssueDate: item.IssueDate,
    AuthorizedAt: asDate(item.AuthorizedAt),
    Protocolo: item.Protocolo,
    TpNF: item.TpNF,
    TpAmb: item.TpAmb,
    NatOp: item.NatOp,
    EmitenteCNPJ: item.EmitenteCNPJ,
    EmitenteName: item.EmitenteName,
    EmitenteIE: item.EmitenteIE,
    EmitenteUF: item.EmitenteUF,
    DestinatarioCNPJ: item.DestinatarioCNPJ,
    DestinatarioName: item.DestinatarioName,
    TotalValue: item.TotalValue,
    ICMSValue: item.ICMSValue,
    IPIValue: item.IPIValue,
    Situacao: asEnum(item.Situacao, nfeSituacoes),
    Completeness: asEnum(item.Completeness, nfeCompletenesses),
    LayoutVersion: item.LayoutVersion,
    ParseWarnings: asStringArray(item.ParseWarnings),
    Manifestacao: asEnum(item.Manifestacao, nfeManifestacoes),
    ManifestacaoAt: asDate(item.ManifestacaoAt),
    CienciaDue: asDate(item.CienciaDue),
    ConclusiveDue: asDate(item.ConclusiveDue),
    CompanyRole: asEnum(item.CompanyRole, nfeRoles),
    EventCount: item.EventCount,
    FirstSeenNSU: asNullableNumber(item.FirstSeenNSU),
    FirstSyncedAt: item.FirstSyncedAt,
    LastSyncedAt: item.LastSyncedAt,
    ViewedAt: asDate(item.ViewedAt),
    DaysLeft: asNullableNumber(item.DaysLeft),
    CienciaDaysLeft: asNullableNumber(item.CienciaDaysLeft),
    TacitlyConfirmed: item.TacitlyConfirmed,
    CienciaBlockReason: item.CienciaBlockReason,
    ConclusiveBlockReason: item.ConclusiveBlockReason,
  }
}

export function mapNFePendingRow(item: desktopapi.NFePendingRow): NFePendingRow {
  return {
    ...mapNFeRow(item),
    Kind: asEnum(item.Kind, nfePendingKinds),
    CienciaOverdue: item.CienciaOverdue,
  }
}

export function mapNFeEvent(item: desktopapi.NFeEvent): NFeEvent {
  return {
    ID: item.ID,
    TpEvento: item.TpEvento,
    NSeqEvento: item.NSeqEvento,
    Description: item.Description,
    EventAt: asDate(item.EventAt),
    RegisteredAt: asDate(item.RegisteredAt),
    Protocolo: item.Protocolo,
    CStat: item.CStat,
    XMotivo: item.XMotivo,
    Justificativa: item.Justificativa,
    Correcao: item.Correcao,
    AutorCNPJ: item.AutorCNPJ,
    Completeness: asEnum(item.Completeness, nfeCompletenesses),
    Registered: item.Registered,
    SentByNanci: item.SentByNanci,
  }
}

export function mapNFeEventResult(item: desktopapi.NFeEventResult): NFeEventResult {
  return {
    ChaveAcesso: item.ChaveAcesso,
    TpEvento: item.TpEvento,
    Status: asEnum(item.Status, nfeOutcomes),
    CStat: item.CStat,
    XMotivo: item.XMotivo,
    Protocolo: item.Protocolo,
    RegisteredAt: asDate(item.RegisteredAt),
  }
}

function mapNFeSkipped(item: desktopapi.NFeSkipped): NFeSkipped {
  return {
    ChaveAcesso: item.ChaveAcesso,
    Reason: item.Reason,
  }
}

export function mapNFeEventBatchResult(item: desktopapi.NFeEventBatchResult): NFeEventBatchResult {
  return {
    Results: asArray(item.Results).map(mapNFeEventResult),
    Skipped: asArray(item.Skipped).map(mapNFeSkipped),
    Interrupted: item.Interrupted,
  }
}

export function mapNFeCienciaPlan(item: desktopapi.NFeCienciaPlan): NFeCienciaPlan {
  return {
    Eligible: asArray(item.Eligible).map(mapNFeRow),
    Skipped: asArray(item.Skipped).map(mapNFeSkipped),
  }
}

export function mapNFeStatus(item: desktopapi.NFeStatusResult): NFeStatusResult {
  return {
    CompanyName: item.CompanyName,
    CNPJ: item.CNPJ,
    UF: item.UF,
    TpAmb: item.TpAmb,
    LastNSU: item.LastNSU,
    MaxNSU: asNullableNumber(item.MaxNSU),
    LastSyncAt: asDate(item.LastSyncAt),
    LastRunStatus: item.LastRunStatus,
    LastRunStopReason: item.LastRunStopReason,
    InitialSyncDoneAt: asDate(item.InitialSyncDoneAt),
    NextAllowedAt: asDate(item.NextAllowedAt),
    BlockedReason: asEnum(item.BlockedReason, nfeBlockedReasons),
    RequestsLastHour: item.RequestsLastHour,
    RequestBudget: item.RequestBudget,
    IdleDays: item.IdleDays,
    TotalDestinatario: item.TotalDestinatario,
    TotalEmitente: item.TotalEmitente,
    TotalOutros: item.TotalOutros,
    TotalResumos: item.TotalResumos,
    TotalCompletas: item.TotalCompletas,
    PendingCiencia: item.PendingCiencia,
    PendingConclusiva: item.PendingConclusiva,
    CienciaOverdue: item.CienciaOverdue,
  }
}

export function mapPullNFeResult(item: desktopapi.PullNFeResult): PullNFeResult {
  return {
    CompanyName: item.CompanyName,
    CNPJ: item.CNPJ,
    Status: item.Status,
    StopReason: item.StopReason,
    LastNSU: item.LastNSU,
    MaxNSU: asNullableNumber(item.MaxNSU),
    CompletasSaved: item.CompletasSaved,
    ResumosSaved: item.ResumosSaved,
    EventsSaved: item.EventsSaved,
    Errors: item.Errors,
    NextAllowedAt: asDate(item.NextAllowedAt),
    RequestsLastHour: item.RequestsLastHour,
    RequestBudget: item.RequestBudget,
    Duration: item.Duration,
  }
}

function mapNFeExportResult(item: desktopapi.NFeExportResult): NFeExportResult {
  return {
    ...mapExportResult(item),
    SkippedResumos: item.SkippedResumos,
  }
}

export function mapNFeResetResult(item: desktopapi.NFeResetResult): NFeResetResult {
  return {
    CompanyName: item.CompanyName,
    CNPJ: item.CNPJ,
    CompanyDocuments: item.CompanyDocuments,
    Documents: item.Documents,
    Events: item.Events,
    ExportMarks: item.ExportMarks,
    ManifestacoesKept: item.ManifestacoesKept,
  }
}

export function mapFetchByChaveResult(item: desktopapi.FetchByChaveResult): FetchByChaveResult {
  return {
    ChaveAcesso: item.ChaveAcesso,
    Outcome: asEnum(item.Outcome, fetchOutcomes),
    Message: item.Message,
    RequestsLastHour: item.RequestsLastHour,
    RequestBudget: item.RequestBudget,
  }
}

export const nfeClient = {
  async pullNFe(cnpj: string, signal?: AbortSignal): Promise<PullNFeResult> {
    const res = await callWails(() => PullNFe({ CNPJ: cnpj }), signal)
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
