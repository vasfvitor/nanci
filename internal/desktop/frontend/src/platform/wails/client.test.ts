import { beforeEach, expect, vi } from 'vitest'
import {
  desktopClient,
  errorMessage,
  mapCompanySummary,
  mapCredentialSummary,
  mapCTeEvent,
  mapCTeResetResult,
  mapCTeRow,
  mapCTeStatus,
  mapDocumentEvent,
  mapDocumentRow,
  mapNFeCienciaPlan,
  mapNFeEvent,
  mapNFeEventBatchResult,
  mapNFePendingRow,
  mapNFeRow,
  mapNFeStatus,
  mapPullCTeResult,
  mapPullNFeResult,
  wailsErrorCode,
  WailsClientError,
} from './client'
import {
  ExportCTeXML,
  ExportCTeZIP,
  ExportDANFSe,
  ExportDANFSeZIP,
  ExportDocuments,
  ExportNFeXML,
  ExportNFeZIP,
  ListCTe,
  ListCTeEvents,
  ListCompanies,
  ListCredentials,
  ListDocuments,
  ListEventsForDocument,
  ListNFe,
  ListNFeEvents,
  ListNFePendingManifestacoes,
  PlanNFeCiencia,
  PreviewResetCTe,
  PullCTe,
  PullNFe,
  RegisterNFeCiencia,
  RegisterNFeManifestacao,
  ResetCTe,
  ResetNFe,
  SelectCertificate,
  SelectSaveFile,
  StatusCTe,
  StatusNFe,
  TestCTeConnection,
} from '../../../wailsjs/go/main/App'

vi.mock('../../../wailsjs/go/main/App', () => ({
  AddCompany: vi.fn(),
  AddCredential: vi.fn(),
  AssignCredentialToCompany: vi.fn(),
  CancelCertPassword: vi.fn(),
  ExportCTeXML: vi.fn(),
  ExportCTeZIP: vi.fn(),
  ExportDANFSe: vi.fn(),
  ExportDANFSeZIP: vi.fn(),
  ExportDocuments: vi.fn(),
  ExportNFeXML: vi.fn(),
  ExportNFeZIP: vi.fn(),
  ListCTe: vi.fn(),
  ListCTeEvents: vi.fn(),
  ListCompanies: vi.fn(),
  ListCredentials: vi.fn(),
  ListDocuments: vi.fn(),
  ListEventsForDocument: vi.fn(),
  ListNFe: vi.fn(),
  ListNFeEvents: vi.fn(),
  ListNFePendingManifestacoes: vi.fn(),
  PlanNFeCiencia: vi.fn(),
  PreviewResetCTe: vi.fn(),
  Pull: vi.fn(),
  PullCTe: vi.fn(),
  PullNFe: vi.fn(),
  QueryNFSeEvents: vi.fn(),
  RegisterNFeCiencia: vi.fn(),
  RegisterNFeManifestacao: vi.fn(),
  ResetCTe: vi.fn(),
  ResetNFe: vi.fn(),
  ResetSyncState: vi.fn(),
  SelectCertificate: vi.fn(),
  SelectExportDirectory: vi.fn(),
  SelectSaveFile: vi.fn(),
  SetLogLevel: vi.fn(),
  StatusCTe: vi.fn(),
  StatusNFe: vi.fn(),
  SubmitCertPassword: vi.fn(),
  TestCTeConnection: vi.fn(),
  UpdateCompany: vi.fn(),
  UpdateCredentialData: vi.fn(),
  UpdateCredentialPath: vi.fn(),
}))

describe('desktop client mappers', () => {
  it('preserves company IDs, status fields, and nullable timestamps', () => {
    const company = mapCompanySummary({
      ID: 'company-1',
      CNPJ: '123',
      LastSyncAt: null,
      LastFoundNSU: 55,
      SyncStartPolicy: 'since_date',
      SyncStartDate: '2025-01-01T00:00:00Z',
      InitialSyncDoneAt: null,
      LastRunStatus: 'completed',
      LastRunStopReason: 'empty_limit',
      UF: 'SP',
    })

    expect(company.ID).toBe('company-1')
    expect(company.UF).toBe('SP')
    expect(mapCompanySummary({ ID: 'company-2' }).UF).toBe('')
    expect(mapCompanySummary({ ID: 'company-2' }).LastSyncAt).toBeNull()
    expect(company.LastSyncAt).toBeNull()
    expect(company.LastFoundNSU).toBe(55)
    expect(company.SyncStartPolicy).toBe('since_date')
    expect(company.SyncStartDate).toBe('2025-01-01T00:00:00Z')
    expect(company.InitialSyncDoneAt).toBeNull()
    expect(company.LastRunStatus).toBe('completed')
    expect(company.LastRunStopReason).toBe('empty_limit')
  })

  it('preserves credential dates and ownership fields', () => {
    const credential = mapCredentialSummary({
      ID: 'cred-1',
      OwnerCNPJ: '123',
      NotAfter: '2027-01-01T00:00:00Z',
      InspectedAt: null,
    })

    expect(credential.ID).toBe('cred-1')
    expect(credential.OwnerCNPJ).toBe('123')
    expect(credential.NotAfter).toBe('2027-01-01T00:00:00Z')
    expect(credential.InspectedAt).toBeNull()
  })

  it('preserves document money cents, IDs, role, visibility, status, and timestamps', () => {
    const row = mapDocumentRow({
      ID: 'canonical-doc',
      DocumentID: 'doc-1',
      RelationID: 'rel-1',
      ServiceValue: 12345,
      ISSValue: 67,
      TotalRetentions: 89,
      Status: 'normal',
      CompanyRole: 'tomada',
      VisibilityReason: 'exact_tomador',
      FirstSyncedAt: '2026-01-02T03:04:05Z',
      ParseWarnings: ['warn'],
    })

    expect(row.ID).toBe('canonical-doc')
    expect(row.DocumentID).toBe('doc-1')
    expect(row.RelationID).toBe('rel-1')
    expect(row.ServiceValue).toBe(12345)
    expect(row.ISSValue).toBe(67)
    expect(row.TotalRetentions).toBe(89)
    expect(row.Status).toBe('normal')
    expect(row.CompanyRole).toBe('tomada')
    expect(row.VisibilityReason).toBe('exact_tomador')
    expect(row.FirstSyncedAt).toBe('2026-01-02T03:04:05Z')
    expect(row.ParseWarnings).toEqual(['warn'])
  })

  it('maps document events', () => {
    expect(mapDocumentEvent({ ID: 'evt-1', Type: 'cancelamento' })).toMatchObject({
      ID: 'evt-1',
      Type: 'cancelamento',
    })
  })
})

describe('desktop client calls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('maps list results through frontend DTO types', async () => {
    vi.mocked(ListCompanies).mockResolvedValue([{ ID: 'c1', CNPJ: '123' }] as never)
    vi.mocked(ListCredentials).mockResolvedValue([{ ID: 'cred-1', Label: 'A1' }] as never)
    vi.mocked(ListDocuments).mockResolvedValue([
      { DocumentID: 'doc-1', ServiceValue: 100 },
    ] as never)
    vi.mocked(ListEventsForDocument).mockResolvedValue([{ ID: 'evt-1' }] as never)

    await expect(desktopClient.listCompanies()).resolves.toMatchObject([{ ID: 'c1' }])
    await expect(desktopClient.listCredentials()).resolves.toMatchObject([{ ID: 'cred-1' }])
    await expect(
      desktopClient.listDocuments({ CNPJ: '123', Competence: '', Direction: '', OnlyUnread: false })
    ).resolves.toMatchObject([{ DocumentID: 'doc-1', ServiceValue: 100 }])
    await expect(desktopClient.listEventsForDocument('doc-1')).resolves.toMatchObject([
      { ID: 'evt-1' },
    ])
  })

  it('normalizes cancelled dialogs to null', async () => {
    vi.mocked(SelectCertificate).mockResolvedValue('')
    await expect(desktopClient.selectCertificate()).resolves.toBeNull()
  })

  it('maps DANFSe export requests through Wails DTOs', async () => {
    vi.mocked(ExportDANFSe).mockResolvedValue({
      OutPath: 'C:\\exports\\danfse.pdf',
      Format: 'danfse',
    } as never)
    vi.mocked(ExportDANFSeZIP).mockResolvedValue({
      OutPath: 'C:\\exports\\danfses.zip',
      Format: 'danfse-zip',
    } as never)
    vi.mocked(SelectSaveFile).mockResolvedValue('C:\\mock\\save\\path.ext')

    await expect(
      desktopClient.exportDANFSe({
        CNPJ: '123',
        ChaveAcesso: 'chave-1',
      })
    ).resolves.toEqual({
      OutPath: 'C:\\exports\\danfse.pdf',
      Format: 'danfse',
      Incremental: false,
      ExportedCount: 0,
    })
    await desktopClient.exportDANFSeZIP({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      Format: 'zip',
      Incremental: false,
      ChavesAcesso: [],
    })

    expect(ExportDANFSe).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'chave-1',
      OutPath: 'C:\\mock\\save\\path.ext',
    })
    expect(ExportDANFSeZIP).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      Format: 'zip',
      OutPath: 'C:\\mock\\save\\path.ext',
      Incremental: false,
      ChavesAcesso: [],
    })
  })

  it('returns null and skips backend export when save-file selection is cancelled', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('')

    await expect(
      desktopClient.exportDocuments({
        CNPJ: '123',
        Competence: '2026-06',
        Direction: 'tomada',
        Format: 'csv',
        Incremental: false,
        ChavesAcesso: [],
      })
    ).resolves.toBeNull()
    await expect(
      desktopClient.exportDANFSe({
        CNPJ: '123',
        ChaveAcesso: 'chave-1',
      })
    ).resolves.toBeNull()
    await expect(
      desktopClient.exportDANFSeZIP({
        CNPJ: '123',
        Competence: '2026-06',
        Direction: 'tomada',
        Format: 'zip',
        Incremental: false,
        ChavesAcesso: [],
      })
    ).resolves.toBeNull()

    expect(ExportDocuments).not.toHaveBeenCalled()
    expect(ExportDANFSe).not.toHaveBeenCalled()
    expect(ExportDANFSeZIP).not.toHaveBeenCalled()
  })

  it('normalizes thrown Wails errors', async () => {
    vi.mocked(ListCompanies).mockRejectedValue(new Error('boom'))
    await expect(desktopClient.listCompanies()).rejects.toBeInstanceOf(WailsClientError)
    await expect(desktopClient.listCompanies()).rejects.toThrow('boom')
  })
})

const chave = '35240912345678000199550010000123451123456789'

describe('NF-e mappers', () => {
  it('keeps cents, nullable dates, and known enum values of a row', () => {
    const row = mapNFeRow({
      ID: 'rel-1',
      DocumentID: 'doc-1',
      ChaveAcesso: chave,
      Protocolo: '135240000000001',
      TotalValue: 123456,
      Situacao: 'autorizada',
      Completeness: 'resumo',
      Manifestacao: 'ciencia',
      CompanyRole: 'destinatario',
      AuthorizedAt: '2024-09-01T10:00:00Z',
      ManifestacaoAt: null,
      EventCount: 2,
    })

    expect(row).toMatchObject({
      ID: 'rel-1',
      DocumentID: 'doc-1',
      ChaveAcesso: chave,
      Protocolo: '135240000000001',
      TotalValue: 123456,
      Situacao: 'autorizada',
      Completeness: 'resumo',
      Manifestacao: 'ciencia',
      CompanyRole: 'destinatario',
      AuthorizedAt: '2024-09-01T10:00:00Z',
      EventCount: 2,
    })
    expect(row.ManifestacaoAt).toBeNull()
    expect(row.CienciaDue).toBeNull()
  })

  it('maps unknown enum values to an empty string', () => {
    const row = mapNFeRow({
      Situacao: 'suspensa',
      Completeness: 42,
      Manifestacao: 'ciente',
      CompanyRole: 'tomador',
    })

    expect(row.Situacao).toBe('')
    expect(row.Completeness).toBe('')
    expect(row.Manifestacao).toBe('')
    expect(row.CompanyRole).toBe('')
    expect(mapNFeStatus({ BlockedReason: 'sem_documentos' }).BlockedReason).toBe('')
    expect(mapNFeEventBatchResult({ Results: [{ Status: 'registered' }] }).Results[0]?.Status).toBe('')
  })

  it('maps flat pending rows with their deadline fields', () => {
    const pending = mapNFePendingRow({
      ID: 'rel-1',
      ChaveAcesso: chave,
      Kind: 'sem_ciencia',
      ConclusiveDue: '2025-03-01T00:00:00Z',
      DaysLeft: -5,
      CienciaDaysLeft: -85,
      TacitlyConfirmed: true,
      CienciaOverdue: true,
      CienciaBlockReason: '',
      ConclusiveBlockReason: '',
    })

    expect(pending).toMatchObject({
      ID: 'rel-1',
      ChaveAcesso: chave,
      Kind: 'sem_ciencia',
      ConclusiveDue: '2025-03-01T00:00:00Z',
      DaysLeft: -5,
      CienciaDaysLeft: -85,
      TacitlyConfirmed: true,
      CienciaOverdue: true,
      CienciaBlockReason: '',
    })
    expect(mapNFePendingRow({ Kind: 'outro' }).Kind).toBe('')
    expect(mapNFePendingRow({}).DaysLeft).toBeNull()
    expect(mapNFePendingRow({}).CienciaDaysLeft).toBeNull()
  })

  it('maps events, batch results, plans, status, and pull results', () => {
    expect(
      mapNFeEvent({
        ID: 'evt-1',
        TpEvento: '210210',
        NSeqEvento: 1,
        Description: 'Ciência da Operação',
        Protocolo: '135',
        Completeness: 'completa',
        Registered: true,
        SentByNanci: true,
      })
    ).toMatchObject({
      ID: 'evt-1',
      TpEvento: '210210',
      NSeqEvento: 1,
      Description: 'Ciência da Operação',
      Protocolo: '135',
      Completeness: 'completa',
      Registered: true,
      SentByNanci: true,
    })

    expect(
      mapNFeEventBatchResult({
        Results: [{ ChaveAcesso: chave, TpEvento: '210210', Status: 'rejeitada', CStat: '596' }],
        Skipped: [{ ChaveAcesso: 'x', Reason: 'emitente' }],
        Interrupted: '',
      })
    ).toEqual({
      Results: [
        {
          ChaveAcesso: chave,
          TpEvento: '210210',
          Status: 'rejeitada',
          CStat: '596',
          XMotivo: '',
          Protocolo: '',
          RegisteredAt: null,
        },
      ],
      Skipped: [{ ChaveAcesso: 'x', Reason: 'emitente' }],
      Interrupted: '',
    })
    expect(mapNFeEventBatchResult({ Results: null, Skipped: null })).toMatchObject({
      Results: [],
      Skipped: [],
    })

    const plan = mapNFeCienciaPlan({
      Eligible: [{ ChaveAcesso: chave, TotalValue: 100, Numero: '12345' }],
      Skipped: [{ ChaveAcesso: 'y', Reason: 'já possui manifestação' }],
    })
    expect(plan.Eligible).toHaveLength(1)
    expect(plan.Eligible[0]).toMatchObject({ ChaveAcesso: chave, TotalValue: 100, Numero: '12345' })
    expect(plan.Skipped).toEqual([{ ChaveAcesso: 'y', Reason: 'já possui manifestação' }])

    const status = mapNFeStatus({
      TpAmb: '2',
      LastNSU: 10,
      MaxNSU: null,
      NextAllowedAt: '2026-09-23T15:00:00Z',
      BlockedReason: 'caught_up',
      PendingCiencia: 4,
      PendingConclusiva: 2,
    })
    expect(status).toMatchObject({
      TpAmb: '2',
      LastNSU: 10,
      MaxNSU: null,
      NextAllowedAt: '2026-09-23T15:00:00Z',
      BlockedReason: 'caught_up',
      PendingCiencia: 4,
      PendingConclusiva: 2,
    })
    expect(mapNFeStatus({ MaxNSU: 99 }).MaxNSU).toBe(99)

    expect(
      mapPullNFeResult({ LastNSU: 5, MaxNSU: 9, ResumosSaved: 3, NextAllowedAt: null })
    ).toMatchObject({ LastNSU: 5, MaxNSU: 9, ResumosSaved: 3, NextAllowedAt: null })
    expect(mapPullNFeResult({ MaxNSU: null }).MaxNSU).toBeNull()
  })
})

describe('NF-e client calls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('passes the Wails DTOs for NF-e calls', async () => {
    vi.mocked(PullNFe).mockResolvedValue({ CNPJ: '123', ResumosSaved: 2 } as never)
    vi.mocked(StatusNFe).mockResolvedValue({ CNPJ: '123' } as never)
    vi.mocked(ResetNFe).mockResolvedValue({ CNPJ: '123', CompanyDocuments: 3 } as never)
    vi.mocked(ListNFe).mockResolvedValue([{ ID: 'rel-1', Situacao: 'autorizada' }] as never)
    vi.mocked(ListNFeEvents).mockResolvedValue(null as never)
    vi.mocked(ListNFePendingManifestacoes).mockResolvedValue([{ ChaveAcesso: chave }] as never)
    vi.mocked(PlanNFeCiencia).mockResolvedValue({ Eligible: [], Skipped: [] } as never)
    vi.mocked(RegisterNFeCiencia).mockResolvedValue({ Interrupted: 'timeout' } as never)
    vi.mocked(RegisterNFeManifestacao).mockResolvedValue({ Status: 'registrada' } as never)

    const listInput = {
      CNPJ: '123',
      Competence: '2024-09',
      Situacao: '' as const,
      Completeness: '' as const,
      Manifestacao: '' as const,
      Role: 'destinatario' as const,
      EmitenteCNPJ: '',
    }

    await expect(desktopClient.pullNFe('123')).resolves.toMatchObject({ ResumosSaved: 2 })
    await expect(desktopClient.statusNFe('123')).resolves.toMatchObject({ CNPJ: '123' })
    await expect(desktopClient.resetNFe('123')).resolves.toEqual({
      CompanyName: '',
      CNPJ: '123',
      CompanyDocuments: 3,
      Documents: 0,
      Events: 0,
      ExportMarks: 0,
      ManifestacoesKept: 0,
    })
    expect(ResetNFe).toHaveBeenCalledWith('123')
    await expect(desktopClient.listNFe(listInput)).resolves.toMatchObject([
      { ID: 'rel-1', Situacao: 'autorizada' },
    ])
    await expect(desktopClient.listNFeEvents('123', chave)).resolves.toEqual([])
    await expect(desktopClient.listNFePendingManifestacoes('123')).resolves.toMatchObject([
      { ChaveAcesso: chave },
    ])
    await desktopClient.listNFePendingManifestacoes('123', 10)
    await expect(desktopClient.planNFeCiencia('123', [chave])).resolves.toEqual({ Eligible: [], Skipped: [] })
    await expect(desktopClient.registerNFeCiencia('123', [chave])).resolves.toMatchObject({
      Interrupted: 'timeout',
    })
    await expect(
      desktopClient.registerNFeManifestacao({
        CNPJ: '123',
        ChaveAcesso: chave,
        Tipo: '210240',
        Justificativa: 'mercadoria não recebida',
      })
    ).resolves.toMatchObject({ Status: 'registrada' })

    expect(PullNFe).toHaveBeenCalledWith({ CNPJ: '123' })
    expect(StatusNFe).toHaveBeenCalledWith('123')
    expect(ListNFe).toHaveBeenCalledWith({ ...listInput, ChavesAcesso: [] })
    expect(ListNFeEvents).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: chave })
    expect(ListNFePendingManifestacoes).toHaveBeenNthCalledWith(1, { CNPJ: '123', DueWithinDays: 0 })
    expect(ListNFePendingManifestacoes).toHaveBeenNthCalledWith(2, { CNPJ: '123', DueWithinDays: 10 })
    expect(PlanNFeCiencia).toHaveBeenCalledWith({ CNPJ: '123', ChavesAcesso: [chave] })
    expect(RegisterNFeCiencia).toHaveBeenCalledWith({ CNPJ: '123', ChavesAcesso: [chave] })
    expect(RegisterNFeManifestacao).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: chave,
      Tipo: '210240',
      Justificativa: 'mercadoria não recebida',
    })
  })

  it('exports NF-e XML and ZIP to the path chosen in the save dialog', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('C:\\out\\file')
    vi.mocked(ExportNFeXML).mockResolvedValue({ OutPath: 'C:\\out\\file', Format: 'xml' } as never)
    vi.mocked(ExportNFeZIP).mockResolvedValue({
      OutPath: 'C:\\out\\file',
      Format: 'zip',
      ExportedCount: 3,
      SkippedResumos: 2,
    } as never)

    const xml = await desktopClient.exportNFeXML({ CNPJ: '123', ChaveAcesso: chave })
    const zip = await desktopClient.exportNFeZIP({
      CNPJ: '123',
      Competence: '2024-09',
      Role: '',
      ChavesAcesso: [chave],
      IncludeResumos: false,
      Incremental: false,
    })

    expect(SelectSaveFile).toHaveBeenNthCalledWith(1, 'Salvar XML da NF-e', `nfe_${chave}.xml`, '*.xml')
    expect(vi.mocked(SelectSaveFile).mock.calls[1]?.[1]).toMatch(/^nfe_123_\d{4}_\d{2}_\d{2}_\d{6}\.zip$/)
    expect(ExportNFeXML).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: chave,
      OutPath: 'C:\\out\\file',
    })
    expect(ExportNFeZIP).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '2024-09',
      Role: '',
      ChavesAcesso: [chave],
      IncludeResumos: false,
      Incremental: false,
      OutPath: 'C:\\out\\file',
    })
    expect(xml).toEqual({
      OutPath: 'C:\\out\\file',
      Format: 'xml',
      Incremental: false,
      ExportedCount: 0,
    })
    expect(zip).toEqual({
      OutPath: 'C:\\out\\file',
      Format: 'zip',
      Incremental: false,
      ExportedCount: 3,
      SkippedResumos: 2,
    })
  })

  it('returns null and skips NF-e exports when the save dialog is cancelled', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('')

    await expect(desktopClient.exportNFeXML({ CNPJ: '123', ChaveAcesso: chave })).resolves.toBeNull()
    await expect(
      desktopClient.exportNFeZIP({
        CNPJ: '123',
        Competence: '',
        Role: '',
        ChavesAcesso: [],
        IncludeResumos: false,
        Incremental: true,
      })
    ).resolves.toBeNull()

    expect(ExportNFeXML).not.toHaveBeenCalled()
    expect(ExportNFeZIP).not.toHaveBeenCalled()
  })
})

describe('CT-e mappers', () => {
  const cteChave = '35240911111111000111570010000000011000000011'

  it('keeps cents, arrays, municípios, nullable fields, and known enum values of a row', () => {
    const row = mapCTeRow({
      ID: 'rel-1',
      DocumentID: 'doc-1',
      ChaveAcesso: cteChave,
      TpAmb: '2',
      Modelo: '57',
      TipoDocumento: 'cte_simplificado',
      TpServ: '2',
      Modal: '01',
      MunIni: { Codigo: '3550308', Nome: 'São Paulo', UF: 'SP' },
      TomadorCNPJ: '22222222000122',
      TomadorIE: '123',
      TomadorUF: 'SP',
      TotalValue: 123456,
      ICMSValue: 14815,
      CargaValue: 9990000,
      NFeChaves: [chave, 7, null],
      Situacao: 'cancelada',
      CompanyRole: 'tomador',
      Papeis: ['tomador', 'remetente'],
      AuthorizedAt: '2024-09-01T10:00:00Z',
      FirstSeenNSU: 42,
      EventCount: 3,
    })

    expect(row).toMatchObject({
      ID: 'rel-1',
      DocumentID: 'doc-1',
      ChaveAcesso: cteChave,
      TpAmb: '2',
      Modelo: '57',
      TipoDocumento: 'cte_simplificado',
      TpServ: '2',
      Modal: '01',
      MunIni: { Codigo: '3550308', Nome: 'São Paulo', UF: 'SP' },
      MunFim: { Codigo: '', Nome: '', UF: '' },
      TomadorCNPJ: '22222222000122',
      TomadorIE: '123',
      TomadorUF: 'SP',
      TotalValue: 123456,
      ICMSValue: 14815,
      CargaValue: 9990000,
      ReceivableValue: 0,
      NFeChaves: [chave],
      Situacao: 'cancelada',
      CompanyRole: 'tomador',
      Papeis: ['tomador', 'remetente'],
      AuthorizedAt: '2024-09-01T10:00:00Z',
      FirstSeenNSU: 42,
      LastSeenNSU: null,
      EventCount: 3,
    })
    expect(mapCTeRow({ NFeChaves: null, Papeis: null, ParseWarnings: null })).toMatchObject({
      NFeChaves: [],
      Papeis: [],
      ParseWarnings: [],
    })
  })

  it('maps unknown enum values to an empty string', () => {
    const row = mapCTeRow({
      Modelo: '55',
      TipoDocumento: 'nfe',
      Situacao: 'suspensa',
      CompanyRole: 'transportador',
      Papeis: ['tomador', 'transportador', 42, 'recebedor'],
    })

    expect(row.Modelo).toBe('')
    expect(row.TipoDocumento).toBe('')
    expect(row.Situacao).toBe('')
    expect(row.CompanyRole).toBe('')
    expect(row.Papeis).toEqual(['tomador', 'recebedor'])
    expect(mapCTeEvent({ Type: 'manifestacao' }).Type).toBe('')
    expect(mapCTeStatus({ BlockedReason: 'sem_documentos' }).BlockedReason).toBe('')
  })

  it('maps events, status, pull, and reset results', () => {
    expect(
      mapCTeEvent({
        ID: 'evt-1',
        TpEvento: '110180',
        Type: 'comprovante_entrega',
        NSeqEvento: 1,
        Protocolo: '135',
        Observacao: 'entregue',
        Registered: true,
        EventAt: null,
      })
    ).toEqual({
      ID: 'evt-1',
      TpEvento: '110180',
      Type: 'comprovante_entrega',
      NSeqEvento: 1,
      Description: '',
      EventAt: null,
      RegisteredAt: null,
      Protocolo: '135',
      CStat: '',
      XMotivo: '',
      AutorCNPJ: '',
      Justificativa: '',
      Observacao: 'entregue',
      Correcao: '',
      Registered: true,
    })

    expect(
      mapCTeStatus({
        TpAmb: '1',
        LastNSU: 10,
        MaxNSU: null,
        BlockedReason: 'rate_budget',
        RequestsLastHour: 20,
        RequestBudget: 20,
        TotalTomador: 4,
        TotalDestinatario: 3,
        TotalRemetente: 2,
        TotalOutros: 1,
      })
    ).toMatchObject({
      TpAmb: '1',
      LastNSU: 10,
      MaxNSU: null,
      BlockedReason: 'rate_budget',
      RequestsLastHour: 20,
      RequestBudget: 20,
      TotalTomador: 4,
      TotalDestinatario: 3,
      TotalRemetente: 2,
      TotalOutros: 1,
    })

    expect(mapPullCTeResult({ LastNSU: 5, MaxNSU: 9, DocumentsSaved: 3, EventsSaved: 1 })).toMatchObject({
      LastNSU: 5,
      MaxNSU: 9,
      DocumentsSaved: 3,
      EventsSaved: 1,
      NextAllowedAt: null,
    })
    expect(mapPullCTeResult({ MaxNSU: null }).MaxNSU).toBeNull()

    expect(mapCTeResetResult({ CNPJ: '123', CompanyDocuments: 5, Events: 7 })).toEqual({
      CompanyName: '',
      CNPJ: '123',
      CompanyDocuments: 5,
      Documents: 0,
      Events: 7,
      ExportMarks: 0,
    })
  })
})

describe('CT-e client calls', () => {
  const cteChave = '35240911111111000111570010000000011000000011'

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('passes the Wails DTOs for CT-e calls', async () => {
    vi.mocked(PullCTe).mockResolvedValue({ CNPJ: '123', DocumentsSaved: 2 } as never)
    vi.mocked(StatusCTe).mockResolvedValue({ CNPJ: '123', TotalTomador: 1 } as never)
    vi.mocked(ListCTe).mockResolvedValue([{ ID: 'rel-1', Situacao: 'autorizada' }] as never)
    vi.mocked(ListCTeEvents).mockResolvedValue(null as never)
    vi.mocked(TestCTeConnection).mockResolvedValue({ certLoaded: true, endpointReached: true } as never)
    vi.mocked(PreviewResetCTe).mockResolvedValue({ CNPJ: '123', CompanyDocuments: 3 } as never)
    vi.mocked(ResetCTe).mockResolvedValue({ CNPJ: '123', CompanyDocuments: 3 } as never)

    const listInput = {
      CNPJ: '123',
      Competence: '2024-09',
      Situacao: '' as const,
      Role: 'tomador' as const,
      Modelo: '67' as const,
      EmitenteCNPJ: '',
      TomadorCNPJ: '',
      NFeChave: chave,
    }

    await expect(desktopClient.pullCTe('123')).resolves.toMatchObject({ DocumentsSaved: 2 })
    await expect(desktopClient.statusCTe('123')).resolves.toMatchObject({ TotalTomador: 1 })
    await expect(desktopClient.listCTe(listInput)).resolves.toMatchObject([
      { ID: 'rel-1', Situacao: 'autorizada', Papeis: [], NFeChaves: [] },
    ])
    await desktopClient.listCTe({ ...listInput, ChavesAcesso: [cteChave], Limit: 1 })
    await expect(desktopClient.listCTeEvents('123', cteChave)).resolves.toEqual([])
    await expect(desktopClient.testCTeConnection('123')).resolves.toMatchObject({
      certLoaded: true,
      endpointReached: true,
      mtlsAccepted: false,
    })
    await expect(desktopClient.previewResetCTe('123')).resolves.toMatchObject({ CompanyDocuments: 3 })
    await expect(desktopClient.resetCTe('123')).resolves.toMatchObject({ CompanyDocuments: 3 })

    expect(PullCTe).toHaveBeenCalledWith({ CNPJ: '123' })
    expect(StatusCTe).toHaveBeenCalledWith('123')
    expect(ListCTe).toHaveBeenNthCalledWith(1, { ...listInput, ChavesAcesso: [], Limit: 0 })
    expect(ListCTe).toHaveBeenNthCalledWith(2, { ...listInput, ChavesAcesso: [cteChave], Limit: 1 })
    expect(ListCTeEvents).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: cteChave })
    expect(TestCTeConnection).toHaveBeenCalledWith('123')
    expect(PreviewResetCTe).toHaveBeenCalledWith('123')
    expect(ResetCTe).toHaveBeenCalledWith('123')
  })

  it('exports CT-e XML and ZIP to the path chosen in the save dialog', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('C:\\out\\file')
    vi.mocked(ExportCTeXML).mockResolvedValue({ OutPath: 'C:\\out\\file', Format: 'xml' } as never)
    vi.mocked(ExportCTeZIP).mockResolvedValue({
      OutPath: 'C:\\out\\file',
      Format: 'xml',
      Incremental: true,
      ExportedCount: 3,
    } as never)

    const xml = await desktopClient.exportCTeXML({ CNPJ: '123', ChaveAcesso: cteChave })
    const zip = await desktopClient.exportCTeZIP({
      CNPJ: '123',
      Competence: '2024-09',
      Role: 'tomador',
      ChavesAcesso: [cteChave],
      Incremental: true,
    })

    expect(SelectSaveFile).toHaveBeenNthCalledWith(1, 'Salvar XML do CT-e', `cte_${cteChave}.xml`, '*.xml')
    expect(vi.mocked(SelectSaveFile).mock.calls[1]?.[1]).toMatch(/^cte_123_\d{4}_\d{2}_\d{2}_\d{6}\.zip$/)
    expect(ExportCTeXML).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: cteChave,
      OutPath: 'C:\\out\\file',
    })
    expect(ExportCTeZIP).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '2024-09',
      Role: 'tomador',
      ChavesAcesso: [cteChave],
      Incremental: true,
      OutPath: 'C:\\out\\file',
    })
    expect(xml).toEqual({ OutPath: 'C:\\out\\file', Format: 'xml', Incremental: false, ExportedCount: 0 })
    expect(zip).toEqual({ OutPath: 'C:\\out\\file', Format: 'xml', Incremental: true, ExportedCount: 3 })
  })

  it('returns null and skips CT-e exports when the save dialog is cancelled', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('')

    await expect(desktopClient.exportCTeXML({ CNPJ: '123', ChaveAcesso: cteChave })).resolves.toBeNull()
    await expect(
      desktopClient.exportCTeZIP({ CNPJ: '123', Competence: '', Role: '', ChavesAcesso: [], Incremental: false })
    ).resolves.toBeNull()

    expect(ExportCTeXML).not.toHaveBeenCalled()
    expect(ExportCTeZIP).not.toHaveBeenCalled()
  })
})

describe('Wails error codes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it.each([
    [{ code: 'canceled', message: 'operação cancelada' }, 'canceled', 'operação cancelada'],
    [{ code: 'sefaz_blocked', message: 'consultas bloqueadas até 15:00' }, 'sefaz_blocked', 'consultas bloqueadas até 15:00'],
    [{ code: 'sync_running', message: 'sincronização em andamento' }, 'sync_running', 'sincronização em andamento'],
    [{ code: '', message: 'empresa não encontrada' }, '', 'empresa não encontrada'],
    [{ code: 'unknown', message: 'algo' }, '', 'algo'],
    ['error parsing arguments: boom', '', 'error parsing arguments: boom'],
    [new Error('boom'), '', 'boom'],
  ])('normalizes %o', async (rejection, code, message) => {
    vi.mocked(PullNFe).mockRejectedValue(rejection)

    const error = await desktopClient.pullNFe('123').catch((err: unknown) => err)

    expect(error).toBeInstanceOf(WailsClientError)
    expect((error as WailsClientError).code).toBe(code)
    expect((error as WailsClientError).message).toBe(message)
    expect((error as WailsClientError).cause).toBe(rejection)
    expect(wailsErrorCode(error)).toBe(code)
    expect(wailsErrorCode(rejection)).toBe(code)
    expect(errorMessage(rejection)).toBe(message)
  })

  it('does not read codes from the message text', () => {
    expect(wailsErrorCode(new Error('ERR_CANCELED: cancelado'))).toBe('')
    expect(wailsErrorCode('canceled')).toBe('')
  })
})

describe('errorMessage', () => {
  it('reads the message of errors and stringifies anything else', () => {
    expect(errorMessage(new WailsClientError('cancelado', 'canceled'))).toBe('cancelado')
    expect(errorMessage(new Error('boom'))).toBe('boom')
    expect(errorMessage('texto')).toBe('texto')
    expect(errorMessage(undefined)).toBe('undefined')
  })
})
