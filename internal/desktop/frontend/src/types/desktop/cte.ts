import type { ExportResult, ISODateValue } from './common'

// CT-e (modelos 57 and 67, GTV-e modelo 64 and CT-e Simplificado). Enum fields
// allow '' so an unknown backend value never reads as a real fiscal state.
// TpCTe, TpServ and Modal stay the codes written in the XML.

export type CTeSituacao = 'autorizada' | 'denegada' | 'cancelada'
// CTePapel lists the roles in priority order: the first one the company
// matches is its primary role.
export type CTePapel =
  | 'tomador'
  | 'destinatario'
  | 'remetente'
  | 'expedidor'
  | 'recebedor'
  | 'emitente'
  | 'autorizado'
  | 'none'
export type CTeModelo = '57' | '64' | '67'
export type CTeTipoDocumento = 'cte' | 'cte_os' | 'gtve' | 'cte_simplificado'
export type CTeEventType =
  | 'cancelamento'
  | 'carta_correcao'
  | 'epec'
  | 'registro_multimodal'
  | 'gtv'
  | 'comprovante_entrega'
  | 'cancelamento_comprovante_entrega'
  | 'insucesso_entrega'
  | 'cancelamento_insucesso_entrega'
  | 'prestacao_desacordo'
  | 'cancelamento_desacordo'
  | 'mdfe_autorizado'
  | 'mdfe_cancelado'
  | 'unknown'
export type CTeBlockedReason = 'caught_up' | 'consumo_indevido' | 'rate_budget'

export type ListCTeInput = {
  CNPJ: string
  Competence: string
  Situacao: CTeSituacao | ''
  // Role matches the primary role or any other role the company plays.
  Role: CTePapel | ''
  Modelo: CTeModelo | ''
  EmitenteCNPJ: string
  TomadorCNPJ: string
  // NFeChave keeps the CT-e that transported this NF-e.
  NFeChave: string
  // ChavesAcesso limits the list to these CT-e; empty or absent lists all.
  ChavesAcesso?: string[]
  // OnlyUnread keeps the CT-e not yet marked as viewed.
  OnlyUnread?: boolean
  // Limit 0 or absent lists every row.
  Limit?: number
}

export type CTeMunicipio = {
  Codigo: string
  Nome: string
  UF: string
}

export type CTeRow = {
  ID: string
  DocumentID: string
  ChaveAcesso: string
  TpAmb: string
  Modelo: CTeModelo | ''
  TipoDocumento: CTeTipoDocumento | ''
  // HasDACTE is false for the kinds that have no DACTE.
  HasDACTE: boolean
  Serie: string
  Numero: string
  CFOP: string
  NatOp: string
  IssueDate?: ISODateValue
  Competence: string
  AuthorizedAt?: ISODateValue
  Protocolo: string
  TpCTe: string
  TpServ: string
  Modal: string
  MunIni: CTeMunicipio
  MunFim: CTeMunicipio
  EmitenteCNPJ: string
  EmitenteName: string
  RemetenteCNPJ: string
  RemetenteName: string
  DestinatarioCNPJ: string
  DestinatarioName: string
  ExpedidorCNPJ: string
  ExpedidorName: string
  RecebedorCNPJ: string
  RecebedorName: string
  TomadorCNPJ: string
  TomadorName: string
  TomadorIE: string
  TomadorUF: string
  TomadorIndicador: string
  // Money fields are in cents.
  TotalValue: number
  ReceivableValue: number
  ICMSValue: number
  TotTribValue: number
  CargaValue: number
  ProdutoPredominante: string
  NFeChaves: string[]
  Situacao: CTeSituacao | ''
  CompanyRole: CTePapel | ''
  // Papeis are every role the company plays, primary first. Unknown values
  // are dropped.
  Papeis: CTePapel[]
  VisibilityReason: string
  EventCount: number
  FirstSeenNSU: number | null
  LastSeenNSU: number | null
  FirstSyncedAt?: ISODateValue
  LastSyncedAt?: ISODateValue
  // ViewedAt is empty while the CT-e is new.
  ViewedAt?: ISODateValue
  LayoutVersion: string
  ParseWarnings: string[]
}

export type CTeKeyInput = {
  CNPJ: string
  ChaveAcesso: string
}

export type CTeEvent = {
  ID: string
  TpEvento: string
  Type: CTeEventType | ''
  NSeqEvento: number
  Description: string
  EventAt?: ISODateValue
  RegisteredAt?: ISODateValue
  Protocolo: string
  CStat: string
  XMotivo: string
  AutorCNPJ: string
  Justificativa: string
  Observacao: string
  Correcao: string
  Registered: boolean
}

export type PullCTeInput = {
  CNPJ: string
}

export type PullCTeResult = {
  CompanyName: string
  CNPJ: string
  Status: string
  StopReason: string
  LastNSU: number
  MaxNSU: number | null
  DocumentsSaved: number
  EventsSaved: number
  Errors: number
  NextAllowedAt?: ISODateValue
  RequestsLastHour: number
  RequestBudget: number
  Duration: number
}

export type CTeStatusResult = {
  CompanyName: string
  CNPJ: string
  UF: string
  TpAmb: string
  LastNSU: number
  MaxNSU: number | null
  LastSyncAt?: ISODateValue
  LastRunStatus: string
  LastRunStopReason: string
  InitialSyncDoneAt?: ISODateValue
  NextAllowedAt?: ISODateValue
  BlockedReason: CTeBlockedReason | ''
  RequestsLastHour: number
  RequestBudget: number
  // IdleDays is the number of days since the last successful distribution
  // query, set only when it nears the 60-day NSU rule; 0 otherwise.
  IdleDays: number
  TotalTomador: number
  TotalDestinatario: number
  TotalRemetente: number
  // TotalOutros counts expedidor, recebedor, emitente, autorizado and none.
  TotalOutros: number
}

// CTeResetResult is what ResetCTe removed, or PreviewResetCTe would remove.
export type CTeResetResult = {
  CompanyName: string
  CNPJ: string
  CompanyDocuments: number
  Documents: number
  Events: number
  ExportMarks: number
}

export type ExportCTeXMLInput = {
  CNPJ: string
  ChaveAcesso: string
  OutPath: string
}

// CTeExportFormat is the CT-e list export the user picks in the dialog.
export type CTeExportFormat = 'csv' | 'xlsx' | 'zip' | 'dacte-zip'

export type ExportCTeDocumentsInput = {
  CNPJ: string
  Competence: string
  Role: CTePapel | ''
  Format: CTeExportFormat
  ChavesAcesso: string[]
  Incremental: boolean
  OutPath: string
}

export type ExportDACTEInput = {
  CNPJ: string
  ChaveAcesso: string
  OutPath: string
}

// CTeExportResult adds the CT-e a DACTE export left out because their kind
// has no DACTE.
export type CTeExportResult = ExportResult & {
  SkippedSemDACTE: number
}
