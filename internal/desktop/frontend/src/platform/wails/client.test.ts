import { beforeEach, expect, vi } from 'vitest'
import {
  desktopClient,
  mapCompanySummary,
  mapCredentialSummary,
  mapDocumentEvent,
  mapDocumentRow,
  WailsClientError,
} from './client'
import {
  ExportDANFSe,
  ExportDANFSeZIP,
  ExportDocuments,
  ListCompanies,
  ListCredentials,
  ListDocuments,
  ListEventsForDocument,
  SelectCertificate,
  SelectSaveFile,
} from '../../../wailsjs/go/main/App'

vi.mock('../../../wailsjs/go/main/App', () => ({
  AddCompany: vi.fn(),
  AddCredential: vi.fn(),
  AssignCredentialToCompany: vi.fn(),
  CancelCertPassword: vi.fn(),
  ExportDANFSe: vi.fn(),
  ExportDANFSeZIP: vi.fn(),
  ExportDocuments: vi.fn(),
  ListCompanies: vi.fn(),
  ListCredentials: vi.fn(),
  ListDocuments: vi.fn(),
  ListEventsForDocument: vi.fn(),
  Pull: vi.fn(),
  QueryNFSe: vi.fn(),
  QueryNFSeEvents: vi.fn(),
  ResetSyncState: vi.fn(),
  SelectCertificate: vi.fn(),
  SelectExportDirectory: vi.fn(),
  SelectSaveFile: vi.fn(),
  SetLogLevel: vi.fn(),
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
      LastFoundNSUValid: true,
      LastRunStatus: 'completed',
      LastRunStopReason: 'empty_limit',
    })

    expect(company.ID).toBe('company-1')
    expect(company.LastSyncAt).toBeNull()
    expect(company.LastFoundNSU).toBe(55)
    expect(company.LastFoundNSUValid).toBe(true)
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
      desktopClient.listDocuments({ CNPJ: '123', Competence: '', Direction: '' })
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

    await desktopClient.exportDANFSe({
      CNPJ: '123',
      ChaveAcesso: 'chave-1',
    })
    await desktopClient.exportDANFSeZIP({
      CNPJ: '123',
      Competence: '2026-06',
      Direction: 'tomada',
      Format: 'zip',
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
