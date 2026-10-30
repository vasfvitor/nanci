import type { CompanySummary, CTeRow, DocumentRow, NFeRow } from '@/types/desktop'

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

// makeDocumentRow builds an NFS-e row with every field empty, changed by
// fields.
export function makeDocumentRow(fields: Partial<DocumentRow> = {}): DocumentRow {
  return {
    ID: '',
    ChaveAcesso: '',
    IssueDate: null,
    Competence: '',
    PrestadorCNPJ: '',
    PrestadorName: '',
    TomadorCNPJ: '',
    TomadorName: '',
    IntermediarioCNPJ: '',
    IntermediarioName: '',
    ServiceValue: 0,
    ISSValue: 0,
    IRRFValue: 0,
    INSSValue: 0,
    PISValue: 0,
    COFINSValue: 0,
    CSLLValue: 0,
    TotalRetentions: 0,
    Status: '',
    LayoutVersion: '',
    XMLPath: '',
    RawHash: '',
    ParseWarnings: [],
    NFSeNumber: '',
    ServiceDescription: '',
    CreatedAt: null,
    UpdatedAt: null,
    RelationID: '',
    CompanyID: '',
    DocumentID: '',
    CompanyRole: '',
    VisibilityReason: '',
    FirstSeenNSU: null,
    LastSeenNSU: null,
    FirstSyncedAt: null,
    LastSyncedAt: null,
    ViewedAt: null,
    ...fields,
  }
}

// makeNFeRow builds an NF-e row with every field empty, changed by fields.
export function makeNFeRow(fields: Partial<NFeRow> = {}): NFeRow {
  return {
    ID: '',
    DocumentID: '',
    ChaveAcesso: '',
    Serie: '',
    Numero: '',
    IssueDate: null,
    AuthorizedAt: null,
    Protocolo: '',
    TpNF: '',
    TpAmb: '',
    NatOp: '',
    EmitenteCNPJ: '',
    EmitenteName: '',
    EmitenteIE: '',
    EmitenteUF: '',
    DestinatarioCNPJ: '',
    DestinatarioName: '',
    TotalValue: 0,
    ICMSValue: 0,
    IPIValue: 0,
    Situacao: '',
    Completeness: '',
    LayoutVersion: '',
    ParseWarnings: [],
    Manifestacao: '',
    ManifestacaoAt: null,
    CienciaDue: null,
    ConclusiveDue: null,
    CompanyRole: '',
    EventCount: 0,
    FirstSeenNSU: null,
    FirstSyncedAt: null,
    LastSyncedAt: null,
    ViewedAt: null,
    DaysLeft: null,
    CienciaDaysLeft: null,
    TacitlyConfirmed: false,
    CienciaBlockReason: '',
    ConclusiveBlockReason: '',
    ...fields,
  }
}

// makeCTeRow builds a CT-e row with every field empty, changed by fields.
export function makeCTeRow(fields: Partial<CTeRow> = {}): CTeRow {
  return {
    ID: '',
    DocumentID: '',
    ChaveAcesso: '',
    TpAmb: '',
    Modelo: '',
    TipoDocumento: '',
    HasDACTE: false,
    Serie: '',
    Numero: '',
    CFOP: '',
    NatOp: '',
    IssueDate: null,
    Competence: '',
    AuthorizedAt: null,
    Protocolo: '',
    TpCTe: '',
    TpServ: '',
    Modal: '',
    MunIni: { Codigo: '', Nome: '', UF: '' },
    MunFim: { Codigo: '', Nome: '', UF: '' },
    EmitenteCNPJ: '',
    EmitenteName: '',
    RemetenteCNPJ: '',
    RemetenteName: '',
    DestinatarioCNPJ: '',
    DestinatarioName: '',
    ExpedidorCNPJ: '',
    ExpedidorName: '',
    RecebedorCNPJ: '',
    RecebedorName: '',
    TomadorCNPJ: '',
    TomadorName: '',
    TomadorIE: '',
    TomadorUF: '',
    TomadorIndicador: '',
    TotalValue: 0,
    ReceivableValue: 0,
    ICMSValue: 0,
    TotTribValue: 0,
    CargaValue: 0,
    ProdutoPredominante: '',
    NFeChaves: [],
    Situacao: '',
    CompanyRole: '',
    Papeis: [],
    VisibilityReason: '',
    EventCount: 0,
    FirstSeenNSU: null,
    LastSeenNSU: null,
    FirstSyncedAt: null,
    LastSyncedAt: null,
    ViewedAt: null,
    LayoutVersion: '',
    ParseWarnings: [],
    ...fields,
  }
}
