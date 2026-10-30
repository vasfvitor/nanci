import { afterEach, beforeEach, expect, vi } from 'vitest'
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
  ExportCTeDocuments,
  ExportCTeXML,
  ExportDACTE,
  ExportDANFE,
  ExportDANFSe,
  ExportDocuments,
  ExportNFeDocuments,
  ExportNFeXML,
  FetchNFeByChave,
  GetBuildInfo,
  ImportCTeXML,
  ImportNFeXML,
  ListCTe,
  ListCTeEvents,
  ListCompanies,
  ListCredentials,
  ListDocuments,
  ListEventsForDocument,
  ListNFe,
  ListNFeEvents,
  ListNFePendingManifestacoes,
  MarkCTeViewed,
  MarkDocumentsViewed,
  MarkNFeViewed,
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
  SelectXMLFiles,
  StatusCTe,
  StatusNFe,
  TestCTeConnection,
} from '../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app'
import type * as desktopapi from '../../../bindings/github.com/vasfvitor/nanci/internal/desktop/desktopapi/models'

vi.mock('../../../bindings/github.com/vasfvitor/nanci/internal/desktop/app', () => ({
  AddCompany: vi.fn(),
  AddCredential: vi.fn(),
  AssignCredentialToCompany: vi.fn(),
  CancelCertPassword: vi.fn(),
  ConfirmQuit: vi.fn(),
  ExportCTeDocuments: vi.fn(),
  ExportCTeXML: vi.fn(),
  ExportDACTE: vi.fn(),
  ExportDANFE: vi.fn(),
  ExportDANFSe: vi.fn(),
  ExportDocuments: vi.fn(),
  ExportLogs: vi.fn(),
  ExportNFeDocuments: vi.fn(),
  ExportNFeXML: vi.fn(),
  ExportXML: vi.fn(),
  FetchNFeByChave: vi.fn(),
  GetBuildInfo: vi.fn(),
  GetDataDirectory: vi.fn(),
  ImportCTeXML: vi.fn(),
  ImportNFeXML: vi.fn(),
  ListCTe: vi.fn(),
  ListCTeEvents: vi.fn(),
  ListCompanies: vi.fn(),
  ListCredentials: vi.fn(),
  ListDocuments: vi.fn(),
  ListEventsForDocument: vi.fn(),
  ListNFe: vi.fn(),
  ListNFeEvents: vi.fn(),
  ListNFePendingManifestacoes: vi.fn(),
  MarkCTeViewed: vi.fn(),
  MarkDocumentsViewed: vi.fn(),
  MarkNFeViewed: vi.fn(),
  OpenDataDirectory: vi.fn(),
  OpenLogsDirectory: vi.fn(),
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
  SelectSaveFile: vi.fn(),
  SelectXMLFiles: vi.fn(),
  SetLogLevel: vi.fn(),
  SetWindowTheme: vi.fn(),
  ShowInFolder: vi.fn(),
  Status: vi.fn(),
  StatusCTe: vi.fn(),
  StatusNFe: vi.fn(),
  SubmitCertPassword: vi.fn(),
  TestCTeConnection: vi.fn(),
  TestConnection: vi.fn(),
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
    } as unknown as desktopapi.CompanySummary)

    expect(company.ID).toBe('company-1')
    expect(company.UF).toBe('SP')
    expect(
      mapCompanySummary({ ID: 'company-2' } as unknown as desktopapi.CompanySummary).LastSyncAt
    ).toBeNull()
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
    } as unknown as desktopapi.CredentialSummary)

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
    } as unknown as desktopapi.DocumentRow)

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
    expect(
      mapDocumentEvent({ ID: 'evt-1', Type: 'cancelamento' } as unknown as desktopapi.DocumentEvent)
    ).toMatchObject({
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
    expect(ListDocuments).toHaveBeenCalledWith({
      CNPJ: '123',
      Competence: '',
      Direction: '',
      OnlyUnread: false,
    })
  })

  it('marks NFS-e viewed by company and chaves', async () => {
    vi.mocked(MarkDocumentsViewed).mockResolvedValue(2 as never)

    await expect(desktopClient.markDocumentsViewed('123', ['chave-1'])).resolves.toBe(2)

    expect(MarkDocumentsViewed).toHaveBeenCalledWith({ CNPJ: '123', ChavesAcesso: ['chave-1'] })
  })

  it('maps the build info', async () => {
    vi.mocked(GetBuildInfo).mockResolvedValue({
      version: 'v1.2.3',
      commit: 'abc1234',
      date: '2026-10-17',
      webview2Version: '141.0.3537.71',
    })

    await expect(desktopClient.getBuildInfo()).resolves.toEqual({
      version: 'v1.2.3',
      commit: 'abc1234',
      date: '2026-10-17',
      webview2Version: '141.0.3537.71',
    })
  })

  it('normalizes cancelled dialogs to null', async () => {
    vi.mocked(SelectCertificate).mockResolvedValue('')
    await expect(desktopClient.selectCertificate()).resolves.toBeNull()
  })

  it('maps DANFSe export requests through Wails DTOs', async () => {
    vi.mocked(ExportDANFSe).mockResolvedValue({
      OutPath: 'C:\\exports\\danfse.pdf',
      Format: 'danfse',
      Incremental: false,
      ExportedCount: 0,
    })
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

    expect(ExportDANFSe).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: 'chave-1',
      OutPath: 'C:\\mock\\save\\path.ext',
    })
  })

  describe('exportDocuments', () => {
    beforeEach(() => {
      vi.useFakeTimers({ toFake: ['Date'] })
      vi.setSystemTime(new Date(2026, 9, 17, 9, 5, 7))
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it.each([
      {
        format: 'csv',
        title: 'Exportar Documentos',
        name: 'nanci_exportacao_123_2026_10_17_090507.csv',
        pattern: '*.csv',
      },
      {
        format: 'xlsx',
        title: 'Exportar Documentos',
        name: 'nanci_exportacao_123_2026_10_17_090507.xlsx',
        pattern: '*.xlsx',
      },
      {
        format: 'zip',
        title: 'Exportar Documentos',
        name: 'nanci_exportacao_123_2026_10_17_090507.zip',
        pattern: '*.zip',
      },
      {
        format: 'danfse-zip',
        title: 'Salvar ZIP de DANFSes',
        name: 'danfses_123_2026_10_17_090507.zip',
        pattern: '*.zip',
      },
    ] as const)('saves $format through the dialog "$title"', async (want) => {
      const { format, title, name, pattern } = want
      vi.mocked(SelectSaveFile).mockResolvedValue('C:\\exports\\saida')
      vi.mocked(ExportDocuments).mockResolvedValue({
        OutPath: 'C:\\exports\\saida',
        Format: 'csv',
        Incremental: false,
        ExportedCount: 2,
      })

      await expect(
        desktopClient.exportDocuments({
          CNPJ: '123',
          Competence: '',
          Direction: '',
          Format: format,
          Incremental: true,
          ChavesAcesso: ['chave-1'],
        })
      ).resolves.toEqual({
        OutPath: 'C:\\exports\\saida',
        Format: 'csv',
        Incremental: false,
        ExportedCount: 2,
      })

      expect(SelectSaveFile).toHaveBeenCalledWith(title, name, pattern)
      expect(ExportDocuments).toHaveBeenCalledWith({
        CNPJ: '123',
        Competence: '',
        Direction: '',
        Format: format,
        Incremental: true,
        ChavesAcesso: ['chave-1'],
        OutPath: 'C:\\exports\\saida',
      })
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
      desktopClient.exportDocuments({
        CNPJ: '123',
        Competence: '2026-06',
        Direction: 'tomada',
        Format: 'danfse-zip',
        Incremental: false,
        ChavesAcesso: [],
      })
    ).resolves.toBeNull()

    expect(ExportDocuments).not.toHaveBeenCalled()
    expect(ExportDANFSe).not.toHaveBeenCalled()
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
      ViewedAt: '2024-09-02T08:00:00Z',
      EventCount: 2,
      FirstSeenNSU: 321,
      TpAmb: '1',
      NatOp: 'Venda de mercadoria',
      EmitenteUF: 'SP',
      ICMSValue: 20400,
      IPIValue: 5050,
      LayoutVersion: '4.00',
      ParseWarnings: ['aviso'],
    } as unknown as desktopapi.NFeRow)

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
      ViewedAt: '2024-09-02T08:00:00Z',
      EventCount: 2,
      FirstSeenNSU: 321,
      TpAmb: '1',
      NatOp: 'Venda de mercadoria',
      EmitenteUF: 'SP',
      ICMSValue: 20400,
      IPIValue: 5050,
      LayoutVersion: '4.00',
      ParseWarnings: ['aviso'],
    })
    expect(row.ManifestacaoAt).toBeNull()
    expect(row.CienciaDue).toBeNull()
    expect(
      mapNFeRow({ ViewedAt: null, ParseWarnings: null } as unknown as desktopapi.NFeRow)
    ).toMatchObject({
      ViewedAt: null,
      FirstSeenNSU: null,
      ParseWarnings: [],
    })
  })

  it('maps unknown enum values to an empty string', () => {
    const row = mapNFeRow({
      Situacao: 'suspensa',
      Completeness: 42,
      Manifestacao: 'ciente',
      CompanyRole: 'tomador',
    } as unknown as desktopapi.NFeRow)

    expect(row.Situacao).toBe('')
    expect(row.Completeness).toBe('')
    expect(row.Manifestacao).toBe('')
    expect(row.CompanyRole).toBe('')
    expect(
      mapNFeStatus({ BlockedReason: 'sem_documentos' } as unknown as desktopapi.NFeStatusResult)
        .BlockedReason
    ).toBe('')
    expect(
      mapNFeEventBatchResult({
        Results: [{ Status: 'registered' }],
      } as unknown as desktopapi.NFeEventBatchResult).Results[0]?.Status
    ).toBe('')
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
    } as unknown as desktopapi.NFePendingRow)

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
    expect(mapNFePendingRow({ Kind: 'outro' } as unknown as desktopapi.NFePendingRow).Kind).toBe('')
    expect(mapNFePendingRow({} as unknown as desktopapi.NFePendingRow).DaysLeft).toBeNull()
    expect(mapNFePendingRow({} as unknown as desktopapi.NFePendingRow).CienciaDaysLeft).toBeNull()
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
      } as unknown as desktopapi.NFeEvent)
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
    expect(
      mapNFeEventBatchResult({
        Results: null,
        Skipped: null,
      } as unknown as desktopapi.NFeEventBatchResult)
    ).toMatchObject({
      Results: [],
      Skipped: [],
    })

    const plan = mapNFeCienciaPlan({
      Eligible: [{ ChaveAcesso: chave, TotalValue: 100, Numero: '12345' }],
      Skipped: [{ ChaveAcesso: 'y', Reason: 'já possui manifestação' }],
    } as unknown as desktopapi.NFeCienciaPlan)
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
      IdleDays: 47,
    } as unknown as desktopapi.NFeStatusResult)
    expect(status).toMatchObject({
      TpAmb: '2',
      LastNSU: 10,
      MaxNSU: null,
      NextAllowedAt: '2026-09-23T15:00:00Z',
      BlockedReason: 'caught_up',
      PendingCiencia: 4,
      PendingConclusiva: 2,
      IdleDays: 47,
    })
    expect(mapNFeStatus({ MaxNSU: 99 } as unknown as desktopapi.NFeStatusResult).MaxNSU).toBe(99)

    expect(
      mapPullNFeResult({
        LastNSU: 5,
        MaxNSU: 9,
        ResumosSaved: 3,
        NextAllowedAt: null,
      } as unknown as desktopapi.PullNFeResult)
    ).toMatchObject({ LastNSU: 5, MaxNSU: 9, ResumosSaved: 3, NextAllowedAt: null })
    expect(
      mapPullNFeResult({ MaxNSU: null } as unknown as desktopapi.PullNFeResult).MaxNSU
    ).toBeNull()
  })
})

describe('NF-e client calls', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('passes the Wails DTOs for NF-e calls', async () => {
    vi.mocked(PullNFe).mockResolvedValue({ CNPJ: '123', ResumosSaved: 2 } as never)
    vi.mocked(StatusNFe).mockResolvedValue({ CNPJ: '123' } as never)
    vi.mocked(ResetNFe).mockResolvedValue({
      CompanyName: '',
      CNPJ: '123',
      CompanyDocuments: 3,
      Documents: 0,
      Events: 0,
      ExportMarks: 0,
      ManifestacoesKept: 0,
    })
    vi.mocked(ListNFe).mockResolvedValue([{ ID: 'rel-1', Situacao: 'autorizada' }] as never)
    vi.mocked(ListNFeEvents).mockResolvedValue(null as never)
    vi.mocked(ListNFePendingManifestacoes).mockResolvedValue([{ ChaveAcesso: chave }] as never)
    vi.mocked(PlanNFeCiencia).mockResolvedValue({ Eligible: [], Skipped: [] } as never)
    vi.mocked(RegisterNFeCiencia).mockResolvedValue({ Interrupted: 'timeout' } as never)
    vi.mocked(RegisterNFeManifestacao).mockResolvedValue({ Status: 'registrada' } as never)
    vi.mocked(MarkNFeViewed).mockResolvedValue(1 as never)

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
    await desktopClient.listNFe({ ...listInput, OnlyUnread: true })
    await expect(desktopClient.markNFeViewed('123', [chave])).resolves.toBe(1)
    await expect(desktopClient.listNFeEvents('123', chave)).resolves.toEqual([])
    await expect(desktopClient.listNFePendingManifestacoes('123')).resolves.toMatchObject([
      { ChaveAcesso: chave },
    ])
    await desktopClient.listNFePendingManifestacoes('123', 10)
    await expect(desktopClient.planNFeCiencia('123', [chave])).resolves.toEqual({
      Eligible: [],
      Skipped: [],
    })
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
    expect(ListNFe).toHaveBeenNthCalledWith(1, {
      ...listInput,
      ChavesAcesso: [],
      OnlyUnread: false,
    })
    expect(ListNFe).toHaveBeenNthCalledWith(2, { ...listInput, ChavesAcesso: [], OnlyUnread: true })
    expect(MarkNFeViewed).toHaveBeenCalledWith({ CNPJ: '123', ChavesAcesso: [chave] })
    expect(ListNFeEvents).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: chave })
    expect(ListNFePendingManifestacoes).toHaveBeenNthCalledWith(1, {
      CNPJ: '123',
      DueWithinDays: 0,
    })
    expect(ListNFePendingManifestacoes).toHaveBeenNthCalledWith(2, {
      CNPJ: '123',
      DueWithinDays: 10,
    })
    expect(PlanNFeCiencia).toHaveBeenCalledWith({ CNPJ: '123', ChavesAcesso: [chave] })
    expect(RegisterNFeCiencia).toHaveBeenCalledWith({ CNPJ: '123', ChavesAcesso: [chave] })
    expect(RegisterNFeManifestacao).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: chave,
      Tipo: '210240',
      Justificativa: 'mercadoria não recebida',
    })
  })

  it('exports the NF-e XML and DANFE of a row to the path chosen in the save dialog', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('C:\\out\\file')
    vi.mocked(ExportNFeXML).mockResolvedValue({
      OutPath: 'C:\\out\\file',
      Format: 'xml',
      Incremental: false,
      ExportedCount: 0,
    })
    vi.mocked(ExportDANFE).mockResolvedValue({
      OutPath: 'C:\\out\\file',
      Format: 'danfe',
      Incremental: false,
      ExportedCount: 1,
    })

    const xml = await desktopClient.exportNFeXML({ CNPJ: '123', ChaveAcesso: chave })
    const danfe = await desktopClient.exportDANFE({ CNPJ: '123', ChaveAcesso: chave })

    expect(SelectSaveFile).toHaveBeenNthCalledWith(
      1,
      'Salvar XML da NF-e',
      `nfe_${chave}.xml`,
      '*.xml'
    )
    expect(SelectSaveFile).toHaveBeenNthCalledWith(2, 'Salvar DANFE', `danfe_${chave}.pdf`, '*.pdf')
    expect(ExportNFeXML).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: chave,
      OutPath: 'C:\\out\\file',
    })
    expect(ExportDANFE).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: chave,
      OutPath: 'C:\\out\\file',
    })
    expect(xml).toEqual({
      OutPath: 'C:\\out\\file',
      Format: 'xml',
      Incremental: false,
      ExportedCount: 0,
    })
    expect(danfe).toEqual({
      OutPath: 'C:\\out\\file',
      Format: 'danfe',
      Incremental: false,
      ExportedCount: 1,
    })
  })

  describe('exportNFeDocuments', () => {
    beforeEach(() => {
      vi.useFakeTimers({ toFake: ['Date'] })
      vi.setSystemTime(new Date(2026, 9, 17, 9, 5, 7))
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it.each([
      {
        format: 'csv',
        title: 'Exportar NF-e',
        name: 'nfe_123_2026_10_17_090507.csv',
        pattern: '*.csv',
      },
      {
        format: 'xlsx',
        title: 'Exportar NF-e',
        name: 'nfe_123_2026_10_17_090507.xlsx',
        pattern: '*.xlsx',
      },
      {
        format: 'zip',
        title: 'Salvar XMLs de NF-e (ZIP)',
        name: 'nfe_123_2026_10_17_090507.zip',
        pattern: '*.zip',
      },
      {
        format: 'danfe-zip',
        title: 'Salvar ZIP de DANFEs',
        name: 'danfes_123_2026_10_17_090507.zip',
        pattern: '*.zip',
      },
    ] as const)('saves $format through the dialog "$title"', async (want) => {
      const { format, title, name, pattern } = want
      vi.mocked(SelectSaveFile).mockResolvedValue('C:\\out\\file')
      vi.mocked(ExportNFeDocuments).mockResolvedValue({
        OutPath: 'C:\\out\\file',
        Format: 'xml',
        Incremental: false,
        ExportedCount: 3,
        SkippedResumos: 2,
      })

      await expect(
        desktopClient.exportNFeDocuments({
          CNPJ: '123',
          Competence: '2024-09',
          Role: '',
          Format: format,
          ChavesAcesso: [chave],
          IncludeResumos: true,
          Incremental: false,
        })
      ).resolves.toEqual({
        OutPath: 'C:\\out\\file',
        Format: 'xml',
        Incremental: false,
        ExportedCount: 3,
        SkippedResumos: 2,
      })

      expect(SelectSaveFile).toHaveBeenCalledWith(title, name, pattern)
      expect(ExportNFeDocuments).toHaveBeenCalledWith({
        CNPJ: '123',
        Competence: '2024-09',
        Role: '',
        Format: format,
        ChavesAcesso: [chave],
        IncludeResumos: true,
        Incremental: false,
        OutPath: 'C:\\out\\file',
      })
    })
  })

  it('returns null and skips NF-e exports when the save dialog is cancelled', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('')

    await expect(
      desktopClient.exportNFeXML({ CNPJ: '123', ChaveAcesso: chave })
    ).resolves.toBeNull()
    await expect(desktopClient.exportDANFE({ CNPJ: '123', ChaveAcesso: chave })).resolves.toBeNull()
    await expect(
      desktopClient.exportNFeDocuments({
        CNPJ: '123',
        Competence: '',
        Role: '',
        Format: 'danfe-zip',
        ChavesAcesso: [],
        IncludeResumos: false,
        Incremental: true,
      })
    ).resolves.toBeNull()

    expect(ExportNFeXML).not.toHaveBeenCalled()
    expect(ExportDANFE).not.toHaveBeenCalled()
    expect(ExportNFeDocuments).not.toHaveBeenCalled()
  })
})

describe('CT-e mappers', () => {
  const cteChave = '35240911111111000111570010000000011000000011'
  const noMunicipio = { Codigo: '', Nome: '', UF: '' }

  it('keeps cents, arrays, municípios, nullable fields, and known enum values of a row', () => {
    const row = mapCTeRow({
      ID: 'rel-1',
      DocumentID: 'doc-1',
      ChaveAcesso: cteChave,
      TpAmb: '2',
      Modelo: '57',
      TipoDocumento: 'cte_simplificado',
      HasDACTE: true,
      TpServ: '2',
      Modal: '01',
      MunIni: { Codigo: '3550308', Nome: 'São Paulo', UF: 'SP' },
      MunFim: noMunicipio,
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
      ViewedAt: '2024-09-02T08:00:00Z',
      EventCount: 3,
    } as unknown as desktopapi.CTeRow)

    expect(row).toMatchObject({
      ID: 'rel-1',
      DocumentID: 'doc-1',
      ChaveAcesso: cteChave,
      TpAmb: '2',
      Modelo: '57',
      TipoDocumento: 'cte_simplificado',
      HasDACTE: true,
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
      NFeChaves: [chave],
      Situacao: 'cancelada',
      CompanyRole: 'tomador',
      Papeis: ['tomador', 'remetente'],
      AuthorizedAt: '2024-09-01T10:00:00Z',
      FirstSeenNSU: 42,
      LastSeenNSU: null,
      ViewedAt: '2024-09-02T08:00:00Z',
      EventCount: 3,
    })
    expect(
      mapCTeRow({
        MunIni: noMunicipio,
        MunFim: noMunicipio,
        NFeChaves: null,
        Papeis: null,
        ParseWarnings: null,
        ViewedAt: null,
      } as unknown as desktopapi.CTeRow)
    ).toMatchObject({
      NFeChaves: [],
      Papeis: [],
      ParseWarnings: [],
      ViewedAt: null,
    })
  })

  it('maps unknown enum values to an empty string', () => {
    const row = mapCTeRow({
      Modelo: '55',
      TipoDocumento: 'nfe',
      Situacao: 'suspensa',
      CompanyRole: 'transportador',
      Papeis: ['tomador', 'transportador', 42, 'recebedor'],
      MunIni: noMunicipio,
      MunFim: noMunicipio,
    } as unknown as desktopapi.CTeRow)

    expect(row.Modelo).toBe('')
    expect(row.TipoDocumento).toBe('')
    expect(row.Situacao).toBe('')
    expect(row.CompanyRole).toBe('')
    expect(row.Papeis).toEqual(['tomador', 'recebedor'])
    expect(mapCTeEvent({ Type: 'manifestacao' } as unknown as desktopapi.CTeEvent).Type).toBe('')
    expect(
      mapCTeStatus({ BlockedReason: 'sem_documentos' } as unknown as desktopapi.CTeStatusResult)
        .BlockedReason
    ).toBe('')
  })

  it('maps events, status, pull, and reset results', () => {
    expect(
      mapCTeEvent({
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
        IdleDays: 45,
      } as unknown as desktopapi.CTeStatusResult)
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
      IdleDays: 45,
    })

    expect(
      mapPullCTeResult({
        LastNSU: 5,
        MaxNSU: 9,
        DocumentsSaved: 3,
        EventsSaved: 1,
      } as unknown as desktopapi.PullCTeResult)
    ).toMatchObject({
      LastNSU: 5,
      MaxNSU: 9,
      DocumentsSaved: 3,
      EventsSaved: 1,
      NextAllowedAt: null,
    })
    expect(
      mapPullCTeResult({ MaxNSU: null } as unknown as desktopapi.PullCTeResult).MaxNSU
    ).toBeNull()

    expect(
      mapCTeResetResult({
        CompanyName: '',
        CNPJ: '123',
        CompanyDocuments: 5,
        Documents: 0,
        Events: 7,
        ExportMarks: 0,
      })
    ).toEqual({
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
    const noMunicipio = { Codigo: '', Nome: '', UF: '' }
    vi.mocked(ListCTe).mockResolvedValue([
      { ID: 'rel-1', Situacao: 'autorizada', MunIni: noMunicipio, MunFim: noMunicipio },
    ] as never)
    vi.mocked(ListCTeEvents).mockResolvedValue(null as never)
    vi.mocked(PreviewResetCTe).mockResolvedValue({ CNPJ: '123', CompanyDocuments: 3 } as never)
    vi.mocked(ResetCTe).mockResolvedValue({ CNPJ: '123', CompanyDocuments: 3 } as never)
    vi.mocked(MarkCTeViewed).mockResolvedValue(2 as never)

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
    await desktopClient.listCTe({
      ...listInput,
      ChavesAcesso: [cteChave],
      Limit: 1,
      OnlyUnread: true,
    })
    await expect(desktopClient.markCTeViewed('123', [cteChave])).resolves.toBe(2)
    await expect(desktopClient.listCTeEvents('123', cteChave)).resolves.toEqual([])
    await expect(desktopClient.previewResetCTe('123')).resolves.toMatchObject({
      CompanyDocuments: 3,
    })
    await expect(desktopClient.resetCTe('123')).resolves.toMatchObject({ CompanyDocuments: 3 })

    expect(PullCTe).toHaveBeenCalledWith({ CNPJ: '123' })
    expect(StatusCTe).toHaveBeenCalledWith('123')
    expect(ListCTe).toHaveBeenNthCalledWith(1, {
      ...listInput,
      ChavesAcesso: [],
      OnlyUnread: false,
      Limit: 0,
    })
    expect(ListCTe).toHaveBeenNthCalledWith(2, {
      ...listInput,
      ChavesAcesso: [cteChave],
      OnlyUnread: true,
      Limit: 1,
    })
    expect(MarkCTeViewed).toHaveBeenCalledWith({ CNPJ: '123', ChavesAcesso: [cteChave] })
    expect(ListCTeEvents).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: cteChave })
    expect(PreviewResetCTe).toHaveBeenCalledWith('123')
    expect(ResetCTe).toHaveBeenCalledWith('123')
  })

  it('tests the CT-e connection and maps the result', async () => {
    vi.mocked(TestCTeConnection).mockResolvedValue({
      certLoaded: true,
      certSubject: 'ACME',
      certExpiration: '',
      mtlsAccepted: true,
      endpointReached: true,
      responseCode: '',
      responseDetail: '',
      statusExplanation: 'Conexão TLS com a SEFAZ (CT-e) estabelecida.',
    })

    await expect(desktopClient.testCTeConnection('123')).resolves.toEqual({
      certLoaded: true,
      certSubject: 'ACME',
      certExpiration: '',
      mtlsAccepted: true,
      endpointReached: true,
      responseCode: '',
      responseDetail: '',
      statusExplanation: 'Conexão TLS com a SEFAZ (CT-e) estabelecida.',
    })
    expect(TestCTeConnection).toHaveBeenCalledWith('123')
  })

  it('exports the CT-e XML and DACTE of a row to the path chosen in the save dialog', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('C:\\out\\file')
    vi.mocked(ExportCTeXML).mockResolvedValue({
      OutPath: 'C:\\out\\file',
      Format: 'xml',
      Incremental: false,
      ExportedCount: 0,
    })
    vi.mocked(ExportDACTE).mockResolvedValue({
      OutPath: 'C:\\out\\file',
      Format: 'dacte',
      Incremental: false,
      ExportedCount: 1,
    })

    const xml = await desktopClient.exportCTeXML({ CNPJ: '123', ChaveAcesso: cteChave })
    const dacte = await desktopClient.exportDACTE({ CNPJ: '123', ChaveAcesso: cteChave })

    expect(SelectSaveFile).toHaveBeenNthCalledWith(
      1,
      'Salvar XML do CT-e',
      `cte_${cteChave}.xml`,
      '*.xml'
    )
    expect(SelectSaveFile).toHaveBeenNthCalledWith(
      2,
      'Salvar DACTE',
      `dacte_${cteChave}.pdf`,
      '*.pdf'
    )
    expect(ExportCTeXML).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: cteChave,
      OutPath: 'C:\\out\\file',
    })
    expect(ExportDACTE).toHaveBeenCalledWith({
      CNPJ: '123',
      ChaveAcesso: cteChave,
      OutPath: 'C:\\out\\file',
    })
    expect(xml).toEqual({
      OutPath: 'C:\\out\\file',
      Format: 'xml',
      Incremental: false,
      ExportedCount: 0,
    })
    expect(dacte).toEqual({
      OutPath: 'C:\\out\\file',
      Format: 'dacte',
      Incremental: false,
      ExportedCount: 1,
    })
  })

  describe('exportCTeDocuments', () => {
    beforeEach(() => {
      vi.useFakeTimers({ toFake: ['Date'] })
      vi.setSystemTime(new Date(2026, 9, 17, 9, 5, 7))
    })

    afterEach(() => {
      vi.useRealTimers()
    })

    it.each([
      {
        format: 'csv',
        title: 'Exportar CT-e',
        name: 'cte_123_2026_10_17_090507.csv',
        pattern: '*.csv',
      },
      {
        format: 'xlsx',
        title: 'Exportar CT-e',
        name: 'cte_123_2026_10_17_090507.xlsx',
        pattern: '*.xlsx',
      },
      {
        format: 'zip',
        title: 'Salvar XMLs de CT-e (ZIP)',
        name: 'cte_123_2026_10_17_090507.zip',
        pattern: '*.zip',
      },
      {
        format: 'dacte-zip',
        title: 'Salvar ZIP de DACTEs',
        name: 'dactes_123_2026_10_17_090507.zip',
        pattern: '*.zip',
      },
    ] as const)('saves $format through the dialog "$title"', async (want) => {
      const { format, title, name, pattern } = want
      vi.mocked(SelectSaveFile).mockResolvedValue('C:\\out\\file')
      vi.mocked(ExportCTeDocuments).mockResolvedValue({
        OutPath: 'C:\\out\\file',
        Format: 'dacte',
        Incremental: true,
        ExportedCount: 3,
        SkippedSemDACTE: 1,
      } as never)

      await expect(
        desktopClient.exportCTeDocuments({
          CNPJ: '123',
          Competence: '2024-09',
          Role: 'tomador',
          Format: format,
          ChavesAcesso: [cteChave],
          Incremental: true,
        })
      ).resolves.toEqual({
        OutPath: 'C:\\out\\file',
        Format: 'dacte',
        Incremental: true,
        ExportedCount: 3,
        SkippedSemDACTE: 1,
      })

      expect(SelectSaveFile).toHaveBeenCalledWith(title, name, pattern)
      expect(ExportCTeDocuments).toHaveBeenCalledWith({
        CNPJ: '123',
        Competence: '2024-09',
        Role: 'tomador',
        Format: format,
        ChavesAcesso: [cteChave],
        Incremental: true,
        OutPath: 'C:\\out\\file',
      })
    })
  })

  it('returns null and skips CT-e exports when the save dialog is cancelled', async () => {
    vi.mocked(SelectSaveFile).mockResolvedValue('')

    await expect(
      desktopClient.exportCTeXML({ CNPJ: '123', ChaveAcesso: cteChave })
    ).resolves.toBeNull()
    await expect(
      desktopClient.exportDACTE({ CNPJ: '123', ChaveAcesso: cteChave })
    ).resolves.toBeNull()
    await expect(
      desktopClient.exportCTeDocuments({
        CNPJ: '123',
        Competence: '',
        Role: '',
        Format: 'dacte-zip',
        ChavesAcesso: [],
        Incremental: false,
      })
    ).resolves.toBeNull()

    expect(ExportCTeXML).not.toHaveBeenCalled()
    expect(ExportDACTE).not.toHaveBeenCalled()
    expect(ExportCTeDocuments).not.toHaveBeenCalled()
  })
})

describe('import and fetch by chave client calls', () => {
  const nfeChave = '35260912345678000195550010000000011000000010'

  beforeEach(() => {
    vi.clearAllMocks()
  })

  it('returns the picked XML files and an empty list on cancel', async () => {
    vi.mocked(SelectXMLFiles).mockResolvedValue(['C:\\xml\\a.xml', 'C:\\xml\\b.xml'])
    await expect(desktopClient.selectXMLFiles()).resolves.toEqual([
      'C:\\xml\\a.xml',
      'C:\\xml\\b.xml',
    ])

    vi.mocked(SelectXMLFiles).mockResolvedValue(null as never)
    await expect(desktopClient.selectXMLFiles()).resolves.toEqual([])

    vi.mocked(SelectXMLFiles).mockResolvedValue([])
    await expect(desktopClient.selectXMLFiles()).resolves.toEqual([])
  })

  it('imports NF-e and CT-e XML files and maps the result', async () => {
    const raw: desktopapi.ImportXMLResult = {
      Files: [
        {
          Path: 'a.xml',
          Kind: 'completa',
          ChaveAcesso: nfeChave,
          Outcome: 'imported',
          Message: '',
        },
        {
          Path: 'b.xml',
          Kind: 'outro',
          ChaveAcesso: '',
          Outcome: 'rejected',
          Message: 'XML ilegível',
        },
      ],
      Imported: 1,
      Updated: 0,
      Rejected: 1,
    }
    vi.mocked(ImportNFeXML).mockResolvedValue(raw)
    vi.mocked(ImportCTeXML).mockResolvedValue({ Files: null, Imported: 0, Updated: 0, Rejected: 0 })

    await expect(desktopClient.importNFeXML('123', ['a.xml', 'b.xml'])).resolves.toEqual({
      Files: [
        {
          Path: 'a.xml',
          Kind: 'completa',
          ChaveAcesso: nfeChave,
          Outcome: 'imported',
          Message: '',
        },
        { Path: 'b.xml', Kind: '', ChaveAcesso: '', Outcome: 'rejected', Message: 'XML ilegível' },
      ],
      Imported: 1,
      Updated: 0,
      Rejected: 1,
    })
    await expect(desktopClient.importCTeXML('123', ['c.xml'])).resolves.toEqual({
      Files: [],
      Imported: 0,
      Updated: 0,
      Rejected: 0,
    })

    expect(ImportNFeXML).toHaveBeenCalledWith({ CNPJ: '123', Files: ['a.xml', 'b.xml'] })
    expect(ImportCTeXML).toHaveBeenCalledWith({ CNPJ: '123', Files: ['c.xml'] })
  })

  it('fetches an NF-e by chave and maps the result', async () => {
    vi.mocked(FetchNFeByChave).mockResolvedValue({
      ChaveAcesso: nfeChave,
      Outcome: 'resumo',
      Message: 'Registre a ciência.',
      RequestsLastHour: 4,
      RequestBudget: 20,
    } as never)

    await expect(desktopClient.fetchNFeByChave('123', nfeChave)).resolves.toEqual({
      ChaveAcesso: nfeChave,
      Outcome: 'resumo',
      Message: 'Registre a ciência.',
      RequestsLastHour: 4,
      RequestBudget: 20,
    })
    expect(FetchNFeByChave).toHaveBeenCalledWith({ CNPJ: '123', ChaveAcesso: nfeChave })
  })

  it('keeps the error code of a blocked fetch', async () => {
    vi.mocked(FetchNFeByChave).mockRejectedValue({ code: 'sefaz_blocked', message: 'bloqueado' })

    await expect(desktopClient.fetchNFeByChave('123', nfeChave)).rejects.toMatchObject({
      code: 'sefaz_blocked',
      message: 'bloqueado',
    })
  })
})

// runtimeError builds what @wailsio/runtime rejects a bound call with when the
// Go method returns an error: a RuntimeError whose cause is the JSON from
// marshalError in internal/desktop/errors.go.
function runtimeError(message: string, cause: unknown) {
  const error = new Error(message, { cause })
  error.name = 'RuntimeError'
  return error
}

describe('Wails error codes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it.each([
    [{ code: 'canceled', message: 'operação cancelada' }, 'canceled', 'operação cancelada'],
    [
      { code: 'sefaz_blocked', message: 'consultas bloqueadas até 15:00' },
      'sefaz_blocked',
      'consultas bloqueadas até 15:00',
    ],
    [
      { code: 'sync_running', message: 'sincronização em andamento' },
      'sync_running',
      'sincronização em andamento',
    ],
    [
      { code: 'startup_failed', message: 'o nanci não iniciou: abrir banco de dados' },
      'startup_failed',
      'o nanci não iniciou: abrir banco de dados',
    ],
    [{ code: '', message: 'empresa não encontrada' }, '', 'empresa não encontrada'],
    [{ code: 'unknown', message: 'algo' }, '', 'algo'],
    ['error parsing arguments: boom', '', 'error parsing arguments: boom'],
    [new Error('boom'), '', 'boom'],
    [
      runtimeError('carregar certificado: operação cancelada', {
        code: 'canceled',
        message: 'carregar certificado: operação cancelada',
      }),
      'canceled',
      'carregar certificado: operação cancelada',
    ],
    [
      runtimeError('NF-e: sincronização já em andamento', {
        code: 'sync_running',
        message: 'NF-e: sincronização já em andamento',
      }),
      'sync_running',
      'NF-e: sincronização já em andamento',
    ],
    [
      runtimeError('empresa não encontrada', { code: '', message: 'empresa não encontrada' }),
      '',
      'empresa não encontrada',
    ],
    [runtimeError('could not parse argument #0', 'not json'), '', 'could not parse argument #0'],
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
