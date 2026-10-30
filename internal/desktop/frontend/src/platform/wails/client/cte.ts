// CT-e (modelos 57, 64 and 67)

import {
  ExportCTeDocuments,
  ExportCTeXML,
  ExportDACTE,
  ImportCTeXML,
  ListCTe,
  ListCTeEvents,
  MarkCTeViewed,
  PreviewResetCTe,
  PullCTe,
  ResetCTe,
  StatusCTe,
  TestCTeConnection,
} from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app'
import type * as desktopapi from '../../../../bindings/github.com/vasfvitor/nanci/internal/desktop/desktopapi/models'
import type {
  ConnectionTestResult,
  CTeBlockedReason,
  CTeEvent,
  CTeEventType,
  CTeModelo,
  CTeMunicipio,
  CTePapel,
  CTeResetResult,
  CTeRow,
  CTeSituacao,
  CTeStatusResult,
  CTeTipoDocumento,
  CTeExportFormat,
  CTeExportResult,
  ExportCTeDocumentsInput,
  ExportCTeXMLInput,
  ExportDACTEInput,
  ExportResult,
  ImportXMLResult,
  ListCTeInput,
  PullCTeResult,
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
  mapConnectionTestResult,
  mapImportXMLResult,
} from './base'
import { systemClient } from './system'

const cteSituacoes: readonly CTeSituacao[] = ['autorizada', 'denegada', 'cancelada']
const ctePapeis: readonly CTePapel[] = [
  'tomador',
  'destinatario',
  'remetente',
  'expedidor',
  'recebedor',
  'emitente',
  'autorizado',
  'none',
]
const cteModelos: readonly CTeModelo[] = ['57', '64', '67']
const cteTiposDocumento: readonly CTeTipoDocumento[] = ['cte', 'cte_os', 'gtve', 'cte_simplificado']
const cteEventTypes: readonly CTeEventType[] = [
  'cancelamento',
  'carta_correcao',
  'epec',
  'registro_multimodal',
  'gtv',
  'comprovante_entrega',
  'cancelamento_comprovante_entrega',
  'insucesso_entrega',
  'cancelamento_insucesso_entrega',
  'prestacao_desacordo',
  'cancelamento_desacordo',
  'mdfe_autorizado',
  'mdfe_cancelado',
  'unknown',
]
const cteBlockedReasons: readonly CTeBlockedReason[] = [
  'caught_up',
  'consumo_indevido',
  'rate_budget',
]

function cteExportDialog(format: CTeExportFormat, cnpj: string): SaveDialog {
  switch (format) {
    case 'dacte-zip':
      return {
        title: 'Salvar ZIP de DACTEs',
        defaultName: `dactes_${cnpj}_${fileTimestamp()}.zip`,
        pattern: '*.zip',
      }
    case 'zip':
      return {
        title: 'Salvar XMLs de CT-e (ZIP)',
        defaultName: `cte_${cnpj}_${fileTimestamp()}.zip`,
        pattern: '*.zip',
      }
    default:
      return {
        title: 'Exportar CT-e',
        defaultName: `cte_${cnpj}_${fileTimestamp()}.${format}`,
        pattern: `*.${format}`,
      }
  }
}

// The Go inputs have no optional fields; fill the ones the frontend may omit.
function cteListRequest(input: ListCTeInput) {
  return {
    ...input,
    ChavesAcesso: input.ChavesAcesso ?? [],
    OnlyUnread: input.OnlyUnread ?? false,
    Limit: input.Limit ?? 0,
  }
}

function mapCTeExportResult(item: desktopapi.CTeExportResult): CTeExportResult {
  return {
    ...mapExportResult(item),
    SkippedSemDACTE: item.SkippedSemDACTE,
  }
}

function mapCTeMunicipio(item: desktopapi.CTeMunicipio): CTeMunicipio {
  return {
    Codigo: item.Codigo,
    Nome: item.Nome,
    UF: item.UF,
  }
}

// mapCTePapeis keeps the known roles in their order and drops the rest.
function mapCTePapeis(value: string[] | null): CTePapel[] {
  return asArray(value)
    .map((papel) => asEnum(papel, ctePapeis))
    .filter((papel): papel is CTePapel => papel !== '')
}

export function mapCTeRow(item: desktopapi.CTeRow): CTeRow {
  return {
    ID: item.ID,
    DocumentID: item.DocumentID,
    ChaveAcesso: item.ChaveAcesso,
    TpAmb: item.TpAmb,
    Modelo: asEnum(item.Modelo, cteModelos),
    TipoDocumento: asEnum(item.TipoDocumento, cteTiposDocumento),
    HasDACTE: item.HasDACTE,
    Serie: item.Serie,
    Numero: item.Numero,
    CFOP: item.CFOP,
    NatOp: item.NatOp,
    IssueDate: item.IssueDate,
    Competence: item.Competence,
    AuthorizedAt: asDate(item.AuthorizedAt),
    Protocolo: item.Protocolo,
    TpCTe: item.TpCTe,
    TpServ: item.TpServ,
    Modal: item.Modal,
    MunIni: mapCTeMunicipio(item.MunIni),
    MunFim: mapCTeMunicipio(item.MunFim),
    EmitenteCNPJ: item.EmitenteCNPJ,
    EmitenteName: item.EmitenteName,
    RemetenteCNPJ: item.RemetenteCNPJ,
    RemetenteName: item.RemetenteName,
    DestinatarioCNPJ: item.DestinatarioCNPJ,
    DestinatarioName: item.DestinatarioName,
    ExpedidorCNPJ: item.ExpedidorCNPJ,
    ExpedidorName: item.ExpedidorName,
    RecebedorCNPJ: item.RecebedorCNPJ,
    RecebedorName: item.RecebedorName,
    TomadorCNPJ: item.TomadorCNPJ,
    TomadorName: item.TomadorName,
    TomadorIE: item.TomadorIE,
    TomadorUF: item.TomadorUF,
    TomadorIndicador: item.TomadorIndicador,
    TotalValue: item.TotalValue,
    ReceivableValue: item.ReceivableValue,
    ICMSValue: item.ICMSValue,
    TotTribValue: item.TotTribValue,
    CargaValue: item.CargaValue,
    ProdutoPredominante: item.ProdutoPredominante,
    NFeChaves: asStringArray(item.NFeChaves),
    Situacao: asEnum(item.Situacao, cteSituacoes),
    CompanyRole: asEnum(item.CompanyRole, ctePapeis),
    Papeis: mapCTePapeis(item.Papeis),
    VisibilityReason: item.VisibilityReason,
    EventCount: item.EventCount,
    FirstSeenNSU: asNullableNumber(item.FirstSeenNSU),
    LastSeenNSU: asNullableNumber(item.LastSeenNSU),
    FirstSyncedAt: item.FirstSyncedAt,
    LastSyncedAt: item.LastSyncedAt,
    ViewedAt: asDate(item.ViewedAt),
    LayoutVersion: item.LayoutVersion,
    ParseWarnings: asStringArray(item.ParseWarnings),
  }
}

export function mapCTeEvent(item: desktopapi.CTeEvent): CTeEvent {
  return {
    ID: item.ID,
    TpEvento: item.TpEvento,
    Type: asEnum(item.Type, cteEventTypes),
    NSeqEvento: item.NSeqEvento,
    Description: item.Description,
    EventAt: asDate(item.EventAt),
    RegisteredAt: asDate(item.RegisteredAt),
    Protocolo: item.Protocolo,
    CStat: item.CStat,
    XMotivo: item.XMotivo,
    AutorCNPJ: item.AutorCNPJ,
    Justificativa: item.Justificativa,
    Observacao: item.Observacao,
    Correcao: item.Correcao,
    Registered: item.Registered,
  }
}

export function mapCTeStatus(item: desktopapi.CTeStatusResult): CTeStatusResult {
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
    BlockedReason: asEnum(item.BlockedReason, cteBlockedReasons),
    RequestsLastHour: item.RequestsLastHour,
    RequestBudget: item.RequestBudget,
    IdleDays: item.IdleDays,
    TotalTomador: item.TotalTomador,
    TotalDestinatario: item.TotalDestinatario,
    TotalRemetente: item.TotalRemetente,
    TotalOutros: item.TotalOutros,
  }
}

export function mapPullCTeResult(item: desktopapi.PullCTeResult): PullCTeResult {
  return {
    CompanyName: item.CompanyName,
    CNPJ: item.CNPJ,
    Status: item.Status,
    StopReason: item.StopReason,
    LastNSU: item.LastNSU,
    MaxNSU: asNullableNumber(item.MaxNSU),
    DocumentsSaved: item.DocumentsSaved,
    EventsSaved: item.EventsSaved,
    Errors: item.Errors,
    NextAllowedAt: asDate(item.NextAllowedAt),
    RequestsLastHour: item.RequestsLastHour,
    RequestBudget: item.RequestBudget,
    Duration: item.Duration,
  }
}

export function mapCTeResetResult(item: desktopapi.CTeResetResult): CTeResetResult {
  return {
    CompanyName: item.CompanyName,
    CNPJ: item.CNPJ,
    CompanyDocuments: item.CompanyDocuments,
    Documents: item.Documents,
    Events: item.Events,
    ExportMarks: item.ExportMarks,
  }
}

export const cteClient = {
  // testCTeConnection opens a TLS connection to the CT-e distribution host;
  // it sends no query, so it does not use the hourly budget.
  async testCTeConnection(cnpj: string): Promise<ConnectionTestResult> {
    const res = await callWails(() => TestCTeConnection(cnpj))
    return mapConnectionTestResult(res)
  },
  async pullCTe(cnpj: string, signal?: AbortSignal): Promise<PullCTeResult> {
    const res = await callWails(() => PullCTe({ CNPJ: cnpj }), signal)
    return mapPullCTeResult(res)
  },
  async statusCTe(cnpj: string): Promise<CTeStatusResult> {
    const res = await callWails(() => StatusCTe(cnpj))
    return mapCTeStatus(res)
  },
  async listCTe(input: ListCTeInput): Promise<CTeRow[]> {
    const res = await callWails(() => ListCTe(cteListRequest(input)))
    return (res || []).map(mapCTeRow)
  },
  // markCTeViewed marks the company's CT-e with these chaves and returns how
  // many were new.
  async markCTeViewed(cnpj: string, chavesAcesso: string[]): Promise<number> {
    return callWails(() => MarkCTeViewed({ CNPJ: cnpj, ChavesAcesso: chavesAcesso }))
  },
  async listCTeEvents(cnpj: string, chaveAcesso: string): Promise<CTeEvent[]> {
    const res = await callWails(() => ListCTeEvents({ CNPJ: cnpj, ChaveAcesso: chaveAcesso }))
    return (res || []).map(mapCTeEvent)
  },
  // importCTeXML stores CT-e and CT-e event XML files without asking SEFAZ.
  async importCTeXML(cnpj: string, files: string[]): Promise<ImportXMLResult> {
    const res = await callWails(() => ImportCTeXML({ CNPJ: cnpj, Files: files }))
    return mapImportXMLResult(res)
  },
  // previewResetCTe counts what resetCTe would remove, changing nothing.
  async previewResetCTe(cnpj: string): Promise<CTeResetResult> {
    const res = await callWails(() => PreviewResetCTe(cnpj))
    return mapCTeResetResult(res)
  },
  // resetCTe removes the company's CT-e and resets its CT-e sync.
  async resetCTe(cnpj: string): Promise<CTeResetResult> {
    const res = await callWails(() => ResetCTe(cnpj))
    return mapCTeResetResult(res)
  },
  async exportCTeXML(
    input: Omit<ExportCTeXMLInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `cte_${input.ChaveAcesso}.xml`
    const outPath =
      input.OutPath ||
      (await systemClient.selectSaveFile('Salvar XML do CT-e', defaultName, '*.xml'))
    if (!outPath) return null

    const res = await callWails(() =>
      ExportCTeXML({ CNPJ: input.CNPJ, ChaveAcesso: input.ChaveAcesso, OutPath: outPath })
    )
    return mapExportResult(res)
  },
  async exportDACTE(
    input: Omit<ExportDACTEInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<ExportResult | null> {
    const defaultName = input.BaseName || `dacte_${input.ChaveAcesso}.pdf`
    const outPath =
      input.OutPath || (await systemClient.selectSaveFile('Salvar DACTE', defaultName, '*.pdf'))
    if (!outPath) return null

    const res = await callWails(() =>
      ExportDACTE({ CNPJ: input.CNPJ, ChaveAcesso: input.ChaveAcesso, OutPath: outPath })
    )
    return mapExportResult(res)
  },
  async exportCTeDocuments(
    input: Omit<ExportCTeDocumentsInput, 'OutPath'> & { BaseName?: string; OutPath?: string }
  ): Promise<CTeExportResult | null> {
    const dialog = cteExportDialog(input.Format, input.CNPJ)
    const defaultName = input.BaseName || dialog.defaultName
    const outPath =
      input.OutPath ||
      (await systemClient.selectSaveFile(dialog.title, defaultName, dialog.pattern))
    if (!outPath) return null

    const res = await callWails(() =>
      ExportCTeDocuments({
        CNPJ: input.CNPJ,
        Competence: input.Competence,
        Role: input.Role,
        Format: input.Format,
        ChavesAcesso: input.ChavesAcesso || [],
        Incremental: input.Incremental,
        OutPath: outPath,
      })
    )
    return mapCTeExportResult(res)
  },
}
