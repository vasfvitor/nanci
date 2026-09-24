import { beforeEach, expect, vi } from 'vitest'
import {
  desktopClient,
  errorMessage,
  mapCompanySummary,
  mapCredentialSummary,
  mapDocumentEvent,
  mapDocumentRow,
  mapNFeCienciaPlan,
  mapNFeEvent,
  mapNFeEventBatchResult,
  mapNFePendingRow,
  mapNFeRow,
  mapNFeStatus,
  mapPullNFeResult,
  wailsErrorCode,
  WailsClientError,
} from './client'
import {
  ExportDANFSe,
  ExportDANFSeZIP,
  ExportDocuments,
  ExportNFeXML,
  ExportNFeZIP,
  ListCompanies,
  ListCredentials,
  ListDocuments,
  ListEventsForDocument,
  ListNFe,
  ListNFeEvents,
  ListNFePendingManifestacoes,
  PlanNFeCiencia,
  PullNFe,
  RegisterNFeCiencia,
  RegisterNFeManifestacao,
  ResetNFe,
  SelectCertificate,
  SelectSaveFile,
  StatusNFe,
} from '../../../wailsjs/go/main/App'

vi.mock('../../../wailsjs/go/main/App', () => ({
  AddCompany: vi.fn(),
  AddCredential: vi.fn(),
  AssignCredentialToCompany: vi.fn(),
  CancelCertPassword: vi.fn(),
  ExportDANFSe: vi.fn(),
  ExportDANFSeZIP: vi.fn(),
  ExportDocuments: vi.fn(),
  ExportNFeXML: vi.fn(),
  ExportNFeZIP: vi.fn(),
  ListCompanies: vi.fn(),
  ListCredentials: vi.fn(),
  ListDocuments: vi.fn(),
  ListEventsForDocument: vi.fn(),
  ListNFe: vi.fn(),
  ListNFeEvents: vi.fn(),
  ListNFePendingManifestacoes: vi.fn(),
  PlanNFeCiencia: vi.fn(),
  Pull: vi.fn(),
  PullNFe: vi.fn(),
  QueryNFSeEvents: vi.fn(),
  RegisterNFeCiencia: vi.fn(),
  RegisterNFeManifestacao: vi.fn(),
  ResetNFe: vi.fn(),
  ResetSyncState: vi.fn(),
  SelectCertificate: vi.fn(),
  SelectExportDirectory: vi.fn(),
  SelectSaveFile: vi.fn(),
  SetLogLevel: vi.fn(),
  StatusNFe: vi.fn(),
  SubmitCertPassword: vi.fn(),
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
      TacitlyConfirmed: true,
      CienciaOverdue: true,
      CienciaBlockReason: '',
    })
    expect(mapNFePendingRow({ Kind: 'outro' }).Kind).toBe('')
    expect(mapNFePendingRow({}).DaysLeft).toBeNull()
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
      LastCheckedNSU: 10,
      MaxNSU: null,
      NextAllowedAt: '2026-09-23T15:00:00Z',
      BlockedReason: 'caught_up',
      PendingCiencia: 4,
      PendingConclusiva: 2,
    })
    expect(status).toMatchObject({
      TpAmb: '2',
      LastCheckedNSU: 10,
      MaxNSU: null,
      NextAllowedAt: '2026-09-23T15:00:00Z',
      BlockedReason: 'caught_up',
      PendingCiencia: 4,
      PendingConclusiva: 2,
    })
    expect(mapNFeStatus({ MaxNSU: 99 }).MaxNSU).toBe(99)

    expect(
      mapPullNFeResult({ UltNSU: 5, MaxNSU: 9, ResumosSaved: 3, NextAllowedAt: null })
    ).toMatchObject({ UltNSU: 5, MaxNSU: 9, ResumosSaved: 3, NextAllowedAt: null })
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
    await expect(desktopClient.listPendingManifestations('123')).resolves.toMatchObject([
      { ChaveAcesso: chave },
    ])
    await desktopClient.listPendingManifestations('123', 10)
    await expect(desktopClient.planCiencia('123', [chave])).resolves.toEqual({ Eligible: [], Skipped: [] })
    await expect(desktopClient.registerCiencia('123', [chave])).resolves.toMatchObject({
      Interrupted: 'timeout',
    })
    await expect(
      desktopClient.registerManifestation({
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

describe('Wails error codes', () => {
  beforeEach(() => {
    vi.clearAllMocks()
  })

  it.each([
    ['ERR_CANCELED: operação cancelada', 'canceled'],
    ['ERR_SEFAZ_BLOCKED: consultas bloqueadas até 15:00', 'sefaz_blocked'],
    ['ERR_SYNC_RUNNING: sincronização em andamento', 'sync_running'],
    ['ERR_UNKNOWN: algo', ''],
    ['falha de rede ERR_CANCELED:', ''],
    ['boom', ''],
  ])('parses %s', async (message, code) => {
    vi.mocked(PullNFe).mockRejectedValue(message)

    const error = await desktopClient.pullNFe('123').catch((err: unknown) => err)

    expect(error).toBeInstanceOf(WailsClientError)
    expect((error as WailsClientError).code).toBe(code)
    expect((error as WailsClientError).message).toBe(message)
    expect(wailsErrorCode(error)).toBe(code)
    expect(wailsErrorCode(new Error(message))).toBe(code)
  })
})

describe('errorMessage', () => {
  it('reads the message of errors and stringifies anything else', () => {
    expect(errorMessage(new WailsClientError('ERR_CANCELED: cancelado'))).toBe('ERR_CANCELED: cancelado')
    expect(errorMessage(new Error('boom'))).toBe('boom')
    expect(errorMessage('texto')).toBe('texto')
    expect(errorMessage(undefined)).toBe('undefined')
  })
})
