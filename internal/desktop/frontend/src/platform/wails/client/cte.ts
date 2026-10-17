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
} from '../../../../wailsjs/go/main/App'
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

function mapCTeExportResult(raw: unknown): CTeExportResult {
  return {
    ...mapExportResult(raw),
    SkippedSemDACTE: asNumber(asRawRecord(raw)['SkippedSemDACTE']),
  }
}

function mapCTeMunicipio(raw: unknown): CTeMunicipio {
  const item = asRawRecord(raw)
  return {
    Codigo: asString(item['Codigo']),
    Nome: asString(item['Nome']),
    UF: asString(item['UF']),
  }
}

// mapCTePapeis keeps the known roles in their order and drops the rest.
function mapCTePapeis(value: unknown): CTePapel[] {
  return asArray(value)
    .map((papel) => asEnum(papel, ctePapeis))
    .filter((papel): papel is CTePapel => papel !== '')
}

export function mapCTeRow(raw: unknown): CTeRow {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    DocumentID: asString(item['DocumentID']),
    ChaveAcesso: asString(item['ChaveAcesso']),
    TpAmb: asString(item['TpAmb']),
    Modelo: asEnum(item['Modelo'], cteModelos),
    TipoDocumento: asEnum(item['TipoDocumento'], cteTiposDocumento),
    HasDACTE: asBoolean(item['HasDACTE']),
    Serie: asString(item['Serie']),
    Numero: asString(item['Numero']),
    CFOP: asString(item['CFOP']),
    NatOp: asString(item['NatOp']),
    IssueDate: asDate(item['IssueDate']),
    Competence: asString(item['Competence']),
    AuthorizedAt: asDate(item['AuthorizedAt']),
    Protocolo: asString(item['Protocolo']),
    TpCTe: asString(item['TpCTe']),
    TpServ: asString(item['TpServ']),
    Modal: asString(item['Modal']),
    MunIni: mapCTeMunicipio(item['MunIni']),
    MunFim: mapCTeMunicipio(item['MunFim']),
    EmitenteCNPJ: asString(item['EmitenteCNPJ']),
    EmitenteName: asString(item['EmitenteName']),
    RemetenteCNPJ: asString(item['RemetenteCNPJ']),
    RemetenteName: asString(item['RemetenteName']),
    DestinatarioCNPJ: asString(item['DestinatarioCNPJ']),
    DestinatarioName: asString(item['DestinatarioName']),
    ExpedidorCNPJ: asString(item['ExpedidorCNPJ']),
    ExpedidorName: asString(item['ExpedidorName']),
    RecebedorCNPJ: asString(item['RecebedorCNPJ']),
    RecebedorName: asString(item['RecebedorName']),
    TomadorCNPJ: asString(item['TomadorCNPJ']),
    TomadorName: asString(item['TomadorName']),
    TomadorIE: asString(item['TomadorIE']),
    TomadorUF: asString(item['TomadorUF']),
    TomadorIndicador: asString(item['TomadorIndicador']),
    TotalValue: asNumber(item['TotalValue']),
    ReceivableValue: asNumber(item['ReceivableValue']),
    ICMSValue: asNumber(item['ICMSValue']),
    TotTribValue: asNumber(item['TotTribValue']),
    CargaValue: asNumber(item['CargaValue']),
    ProdutoPredominante: asString(item['ProdutoPredominante']),
    NFeChaves: asStringArray(item['NFeChaves']),
    Situacao: asEnum(item['Situacao'], cteSituacoes),
    CompanyRole: asEnum(item['CompanyRole'], ctePapeis),
    Papeis: mapCTePapeis(item['Papeis']),
    VisibilityReason: asString(item['VisibilityReason']),
    EventCount: asNumber(item['EventCount']),
    FirstSeenNSU: asNullableNumber(item['FirstSeenNSU']),
    LastSeenNSU: asNullableNumber(item['LastSeenNSU']),
    FirstSyncedAt: asDate(item['FirstSyncedAt']),
    LastSyncedAt: asDate(item['LastSyncedAt']),
    ViewedAt: asDate(item['ViewedAt']),
    LayoutVersion: asString(item['LayoutVersion']),
    ParseWarnings: asStringArray(item['ParseWarnings']),
  }
}

export function mapCTeEvent(raw: unknown): CTeEvent {
  const item = asRawRecord(raw)
  return {
    ID: asString(item['ID']),
    TpEvento: asString(item['TpEvento']),
    Type: asEnum(item['Type'], cteEventTypes),
    NSeqEvento: asNumber(item['NSeqEvento']),
    Description: asString(item['Description']),
    EventAt: asDate(item['EventAt']),
    RegisteredAt: asDate(item['RegisteredAt']),
    Protocolo: asString(item['Protocolo']),
    CStat: asString(item['CStat']),
    XMotivo: asString(item['XMotivo']),
    AutorCNPJ: asString(item['AutorCNPJ']),
    Justificativa: asString(item['Justificativa']),
    Observacao: asString(item['Observacao']),
    Correcao: asString(item['Correcao']),
    Registered: asBoolean(item['Registered']),
  }
}

export function mapCTeStatus(raw: unknown): CTeStatusResult {
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
    BlockedReason: asEnum(item['BlockedReason'], cteBlockedReasons),
    RequestsLastHour: asNumber(item['RequestsLastHour']),
    RequestBudget: asNumber(item['RequestBudget']),
    IdleDays: asNumber(item['IdleDays']),
    TotalTomador: asNumber(item['TotalTomador']),
    TotalDestinatario: asNumber(item['TotalDestinatario']),
    TotalRemetente: asNumber(item['TotalRemetente']),
    TotalOutros: asNumber(item['TotalOutros']),
  }
}

export function mapPullCTeResult(raw: unknown): PullCTeResult {
  const item = asRawRecord(raw)
  return {
    CompanyName: asString(item['CompanyName']),
    CNPJ: asString(item['CNPJ']),
    Status: asString(item['Status']),
    StopReason: asString(item['StopReason']),
    LastNSU: asNumber(item['LastNSU']),
    MaxNSU: asNullableNumber(item['MaxNSU']),
    DocumentsSaved: asNumber(item['DocumentsSaved']),
    EventsSaved: asNumber(item['EventsSaved']),
    Errors: asNumber(item['Errors']),
    NextAllowedAt: asDate(item['NextAllowedAt']),
    RequestsLastHour: asNumber(item['RequestsLastHour']),
    RequestBudget: asNumber(item['RequestBudget']),
    Duration: asNumber(item['Duration']),
  }
}

export function mapCTeResetResult(raw: unknown): CTeResetResult {
  const item = asRawRecord(raw)
  return {
    CompanyName: asString(item['CompanyName']),
    CNPJ: asString(item['CNPJ']),
    CompanyDocuments: asNumber(item['CompanyDocuments']),
    Documents: asNumber(item['Documents']),
    Events: asNumber(item['Events']),
    ExportMarks: asNumber(item['ExportMarks']),
  }
}

export const cteClient = {
  // testCTeConnection opens a TLS connection to the CT-e distribution host;
  // it sends no query, so it does not use the hourly budget.
  async testCTeConnection(cnpj: string): Promise<ConnectionTestResult> {
    const res = await callWails(() => TestCTeConnection(cnpj))
    return mapConnectionTestResult(res)
  },
  async pullCTe(cnpj: string): Promise<PullCTeResult> {
    const res = await callWails(() => PullCTe({ CNPJ: cnpj }))
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
