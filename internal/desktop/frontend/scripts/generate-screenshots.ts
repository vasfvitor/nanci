import { createServer, type ViteDevServer } from 'vite'
import { chromium, type Browser, type BrowserContext, type Page } from 'playwright'
import { mkdir } from 'node:fs/promises'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

type Theme = 'light' | 'dark'
type EventCallback = (...args: unknown[]) => void

type ScreenshotSpec = {
  route: string
  name: string
  theme: Theme
  ready?: string
  setup?: (page: Page) => Promise<void>
}

declare global {
  interface Window {
    go: unknown
    runtime: {
      EventsOnMultiple: (eventName: string, callback: EventCallback, maxCallbacks?: number) => () => void
      EventsOnce: (eventName: string, callback: EventCallback) => void
      EventsOff: (eventName: string) => void
      EventsOffAll: () => void
      EventsEmit: (eventName: string, ...args: unknown[]) => void
      [key: string]: unknown
    }
    triggerWailsEvent?: (eventName: string, payload: unknown) => void
  }
}

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const rootDir = path.resolve(__dirname, '../../../..')
const screenshotsDir = path.resolve(rootDir, 'docs/screenshots')
const viteConfig = path.resolve(__dirname, '../vite.config.ts')
const appUrl = 'http://localhost:5555'

const mockCompanies = [
  {
    ID: '1',
    CNPJ: '12345678000100',
    CNPJRoot: '12345678',
    Name: 'ACME Tecnologia e Serviços LTDA',
    CredentialID: 'cred-1',
    CredentialLabel: 'Certificado PFX ACME 2026',
    CredentialCertPath: 'C:\\Certificados\\acme_corp_2026.pfx',
    Environment: 'producao',
    LastFoundNSU: 25,
    LastSyncAt: '2026-06-18T14:30:00-03:00',
    SyncStartPolicy: 'since_date',
    SyncStartDate: '2025-01-01T00:00:00Z',
    InitialSyncDoneAt: '2026-01-01T01:00:00Z',
    LastRunStatus: 'success',
    LastRunStopReason: 'no-more-nsu',
    CreatedAt: '2026-01-01T00:00:00Z',
    UpdatedAt: '2026-06-18T14:30:00Z',
  },
  {
    ID: '2',
    CNPJ: '98765432000199',
    CNPJRoot: '98765432',
    Name: 'Stark Indústrias do Brasil Ltda',
    CredentialID: 'cred-2',
    CredentialLabel: 'Certificado Stark Secure',
    CredentialCertPath: 'C:\\Certificados\\stark_secure.pfx',
    Environment: 'homologacao',
    LastFoundNSU: 25,
    LastSyncAt: '2026-06-18T11:15:22-03:00',
    SyncStartPolicy: 'from_now',
    SyncStartDate: '2026-02-15T00:00:00Z',
    InitialSyncDoneAt: '2026-02-15T01:00:00Z',
    LastRunStatus: 'idle',
    LastRunStopReason: '',
    CreatedAt: '2026-02-15T00:00:00Z',
    UpdatedAt: '2026-06-18T11:15:22Z',
  },
  {
    ID: '3',
    CNPJ: '55444333000122',
    CNPJRoot: '55444333',
    Name: 'Wayne Empreendimentos S.A.',
    CredentialID: '',
    CredentialLabel: '',
    CredentialCertPath: '',
    Environment: 'producao',
    LastFoundNSU: 0,
    LastSyncAt: null,
    SyncStartPolicy: 'from_now',
    SyncStartDate: '2026-05-10T00:00:00Z',
    InitialSyncDoneAt: null,
    LastRunStatus: 'error',
    LastRunStopReason: 'cert-missing',
    CreatedAt: '2026-05-10T10:00:00Z',
    UpdatedAt: '2026-05-10T10:00:00Z',
  },
  {
    ID: '4',
    CNPJ: '11222333000144',
    CNPJRoot: '11222333',
    Name: 'TechCorp Soluções Digitais',
    CredentialID: 'cred-3',
    CredentialLabel: 'Certificado TechCorp (Vencido)',
    CredentialCertPath: 'C:\\Certificados\\techcorp_old.pfx',
    Environment: 'producao',
    LastFoundNSU: 50,
    LastSyncAt: '2023-12-30T10:00:00-03:00',
    SyncStartPolicy: 'all',
    SyncStartDate: null,
    InitialSyncDoneAt: '2023-01-10T11:00:00Z',
    LastRunStatus: 'error',
    LastRunStopReason: 'cert-expired',
    CreatedAt: '2023-01-10T10:00:00Z',
    UpdatedAt: '2023-12-30T10:00:00Z',
  },
]

const mockCredentials = [
  {
    ID: 'cred-1',
    Label: 'Certificado PFX ACME 2026',
    CertPath: 'C:\\Certificados\\acme_corp_2026.pfx',
    OwnerCNPJ: '12345678000100',
    OwnerCNPJRoot: '12345678',
    FingerprintSHA256: 'a1b2c3d4e5f60708090a0b0c0d0e0f1011121314151617181920212223242526',
    SubjectName: 'ACME TECNOLOGIA E SERVICOS LTDA:12345678000100',
    NotBefore: '2026-01-01T00:00:00Z',
    NotAfter: '2027-01-01T23:59:59Z',
    InspectedAt: '2026-01-02T10:00:00Z',
    CreatedAt: '2026-01-02T10:00:00Z',
    UpdatedAt: '2026-01-02T10:00:00Z',
  },
  {
    ID: 'cred-2',
    Label: 'Certificado Stark Secure',
    CertPath: 'C:\\Certificados\\stark_secure.pfx',
    OwnerCNPJ: '98765432000199',
    OwnerCNPJRoot: '98765432',
    FingerprintSHA256: 'f5e4d3c2b1a0f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b7a6f5e4',
    SubjectName: 'STARK INDUSTRIAS DO BRASIL LTDA:98765432000199',
    NotBefore: '2025-06-01T00:00:00Z',
    NotAfter: '2026-06-01T23:59:59Z',
    InspectedAt: '2025-06-02T11:00:00Z',
    CreatedAt: '2025-06-02T11:00:00Z',
    UpdatedAt: '2025-06-02T11:00:00Z',
  },
  {
    ID: 'cred-3',
    Label: 'Certificado TechCorp (Vencido)',
    CertPath: 'C:\\Certificados\\techcorp_old.pfx',
    OwnerCNPJ: '11222333000144',
    OwnerCNPJRoot: '11222333',
    FingerprintSHA256: 'deadbeefd3c2b1a0f9e8d7c6b5a4f3e2d1c0b9a8f7e6d5c4b3a2f1e0d9c8b7a',
    SubjectName: 'TECHCORP SOLUCOES DIGITAIS:11222333000144',
    NotBefore: '2022-12-31T00:00:00Z',
    NotAfter: '2023-12-31T23:59:59Z',
    InspectedAt: '2023-01-02T11:00:00Z',
    CreatedAt: '2023-01-02T11:00:00Z',
    UpdatedAt: '2023-01-02T11:00:00Z',
  },
]

const mockDocuments = [
  {
    ID: 'doc-1',
    ChaveAcesso: '35260612345678000100560010000012341002003004',
    IssueDate: '2026-06-18T10:00:00-03:00',
    Competence: '2026-06',
    PrestadorCNPJ: '12345678000100',
    PrestadorName: 'ACME Tecnologia e Serviços LTDA',
    TomadorCNPJ: '22222222000122',
    TomadorName: 'Parceiro Comercial S/A',
    IntermediarioCNPJ: '',
    IntermediarioName: '',
    ServiceValue: 150000,
    ISSValue: 3000,
    IRRFValue: 2250,
    INSSValue: 0,
    PISValue: 975,
    COFINSValue: 4500,
    CSLLValue: 1500,
    TotalRetentions: 9225,
    Status: 'normal',
    LayoutVersion: '3.00',
    XMLPath: 'C:\\nanci\\xmls\\doc-1.xml',
    RawHash: 'sha256...',
    ParseWarnings: [],
    NFSeNumber: '1234',
    ServiceDescription: 'Serviços de desenvolvimento de software sob medida e consultoria técnica de TI.',
    CreatedAt: '2026-06-18T10:05:00Z',
    UpdatedAt: '2026-06-18T10:05:00Z',
    RelationID: 'rel-1',
    CompanyID: '1',
    DocumentID: 'doc-1',
    CompanyRole: 'prestada',
    VisibilityReason: 'exact_prestador',
    FirstSeenNSU: 10,
    LastSeenNSU: 10,
    FirstSyncedAt: '2026-06-18T10:05:00Z',
    LastSyncedAt: '2026-06-18T10:05:00Z',
  },
  {
    ID: 'doc-2',
    ChaveAcesso: '35260698765432000199560010000056781002003005',
    IssueDate: '2026-06-17T15:30:00-03:00',
    Competence: '2026-06',
    PrestadorCNPJ: '77777777000177',
    PrestadorName: 'Consultoria Contábil Silva Ltda',
    TomadorCNPJ: '12345678000100',
    TomadorName: 'ACME Tecnologia e Serviços LTDA',
    IntermediarioCNPJ: '',
    IntermediarioName: '',
    ServiceValue: 500000,
    ISSValue: 10000,
    IRRFValue: 7500,
    INSSValue: 55000,
    PISValue: 3250,
    COFINSValue: 15000,
    CSLLValue: 5000,
    TotalRetentions: 95750,
    Status: 'normal',
    LayoutVersion: '3.00',
    XMLPath: 'C:\\nanci\\xmls\\doc-2.xml',
    RawHash: 'sha256...',
    ParseWarnings: ['Aliquota ISS incomum para o municipio'],
    NFSeNumber: '5678',
    ServiceDescription: 'Serviços de assessoria contábil mensal, fechamento fiscal e folha de pagamento.',
    CreatedAt: '2026-06-17T15:45:00Z',
    UpdatedAt: '2026-06-17T15:45:00Z',
    RelationID: 'rel-2',
    CompanyID: '1',
    DocumentID: 'doc-2',
    CompanyRole: 'tomada',
    VisibilityReason: 'exact_tomador',
    FirstSeenNSU: 102,
    LastSeenNSU: 102,
    FirstSyncedAt: '2026-06-17T15:45:00Z',
    LastSyncedAt: '2026-06-17T15:45:00Z',
  },
  {
    ID: 'doc-3',
    ChaveAcesso: '35260612345678000100560010000099991002003006',
    IssueDate: '2026-06-15T09:00:00-03:00',
    Competence: '2026-06',
    PrestadorCNPJ: '12345678000100',
    PrestadorName: 'ACME Tecnologia e Serviços LTDA',
    TomadorCNPJ: '33333333000133',
    TomadorName: 'Supermercado Central Ltda',
    IntermediarioCNPJ: '',
    IntermediarioName: '',
    ServiceValue: 85000,
    ISSValue: 1700,
    IRRFValue: 0,
    INSSValue: 0,
    PISValue: 0,
    COFINSValue: 0,
    CSLLValue: 0,
    TotalRetentions: 0,
    Status: 'cancelada',
    LayoutVersion: '3.00',
    XMLPath: 'C:\\nanci\\xmls\\doc-3.xml',
    RawHash: 'sha256...',
    ParseWarnings: [],
    NFSeNumber: '1235',
    ServiceDescription: 'Treinamento presencial de equipe de vendas corporativa e fornecimento de material didático.',
    CreatedAt: '2026-06-15T09:10:00Z',
    UpdatedAt: '2026-06-15T09:10:00Z',
    RelationID: 'rel-3',
    CompanyID: '1',
    DocumentID: 'doc-3',
    CompanyRole: 'prestada',
    VisibilityReason: 'same_root_only',
    FirstSeenNSU: 45,
    LastSeenNSU: 45,
    FirstSyncedAt: '2026-06-15T09:10:00Z',
    LastSyncedAt: '2026-06-15T09:10:00Z',
  },
  {
    ID: 'doc-4',
    ChaveAcesso: '35260611222333000144560010000088881002003007',
    IssueDate: '2026-06-14T14:20:00-03:00',
    Competence: '2026-06',
    PrestadorCNPJ: '99999999000199',
    PrestadorName: 'Agência de Publicidade Marketing S/A',
    TomadorCNPJ: '88888888000188',
    TomadorName: 'Cliente Final Ltda',
    IntermediarioCNPJ: '11222333000144',
    IntermediarioName: 'TechCorp Soluções Digitais',
    ServiceValue: 200000,
    ISSValue: 4000,
    IRRFValue: 3000,
    INSSValue: 0,
    PISValue: 1300,
    COFINSValue: 6000,
    CSLLValue: 2000,
    TotalRetentions: 12300,
    Status: 'substituida',
    LayoutVersion: '3.00',
    XMLPath: 'C:\\nanci\\xmls\\doc-4.xml',
    RawHash: 'sha256...',
    ParseWarnings: ['Documento não possui assinatura válida'],
    NFSeNumber: '8888',
    ServiceDescription: 'Serviços de intermediação de negócios e agenciamento.',
    CreatedAt: '2026-06-14T14:25:00Z',
    UpdatedAt: '2026-06-14T14:25:00Z',
    RelationID: 'rel-4',
    CompanyID: '4',
    DocumentID: 'doc-4',
    CompanyRole: 'intermediario',
    VisibilityReason: 'exact_intermediario',
    FirstSeenNSU: 20,
    LastSeenNSU: 20,
    FirstSyncedAt: '2026-06-14T14:25:00Z',
    LastSyncedAt: '2026-06-14T14:25:00Z',
  },
]

const syncLogs = [
  { level: 'INFO', msg: 'Iniciando sincronização para ACME Tecnologia e Serviços LTDA', time: '2026-06-18T14:30:01Z' },
  { level: 'DEBUG', msg: 'Usando ambiente de produção', time: '2026-06-18T14:30:02Z' },
  { level: 'TRACE', msg: 'Consultando NSU para CNPJ 12345678000100', time: '2026-06-18T14:30:02Z' },
  { level: 'INFO', msg: 'Busca concluída: 3 documentos encontrados', time: '2026-06-18T14:30:03Z' },
  { level: 'INFO', msg: 'Documento 1234 salvo no banco local', time: '2026-06-18T14:30:03Z' },
  { level: 'WARNING', msg: 'Alíquota ISS incomum no documento 5678', time: '2026-06-18T14:30:04Z' },
  { level: 'INFO', msg: 'Documento 5678 salvo no banco local', time: '2026-06-18T14:30:04Z' },
  { level: 'INFO', msg: 'Documento 1235 salvo no banco local', time: '2026-06-18T14:30:04Z' },
  { level: 'INFO', msg: 'Sincronização concluída em 1.24s', time: '2026-06-18T14:30:05Z' },
]

const mockEvents = [
  {
    ID: 'ev-1',
    Type: 'NFSE',
    EventAt: '2026-06-14T14:20:00Z',
    ReplacementChaveAcesso: '',
    Description: 'Emissão Normal',
    RawXMLPath: 'C:\\nanci\\xmls\\doc-4.xml',
  },
  {
    ID: 'ev-2',
    Type: 'CANC',
    EventAt: '2026-06-14T15:00:00Z',
    ReplacementChaveAcesso: '',
    Description: 'Cancelamento por erro de valores na prestação do serviço.',
    RawXMLPath: 'C:\\nanci\\xmls\\doc-4-canc.xml',
  },
  {
    ID: 'ev-3',
    Type: 'SUBST',
    EventAt: '2026-06-14T16:30:00Z',
    ReplacementChaveAcesso: '35260611222333000144560010000088891002003008',
    Description: 'Nota fiscal substituída pela de final 8889.',
    RawXMLPath: 'C:\\nanci\\xmls\\doc-4-subst.xml',
  },
]

const screenshots: ScreenshotSpec[] = [
  { route: '/', name: 'empresas', theme: 'light', ready: 'text=Empresas' },
  { route: '/', name: 'empresas', theme: 'dark', ready: 'text=Empresas' },

  {
    route: '/',
    name: 'dialogo-adicionar-empresa',
    theme: 'light',
    ready: 'text=Empresas',
    setup: async (page) => {
      await page.click('button:has-text("Adicionar")')
      await page.waitForSelector('.q-dialog', { timeout: 3000 })
    },
  },
  {
    route: '/',
    name: 'dialogo-adicionar-empresa',
    theme: 'dark',
    ready: 'text=Empresas',
    setup: async (page) => {
      await page.click('button:has-text("Adicionar")')
      await page.waitForSelector('.q-dialog', { timeout: 3000 })
    },
  },

  {
    route: '/',
    name: 'dialogo-senha-certificado',
    theme: 'light',
    ready: 'text=Empresas',
    setup: async (page) => {
      await page.evaluate(() => {
        window.triggerWailsEvent?.('request-cert-password', {
          RequestID: 'req-123',
          CompanyName: 'TechCorp Soluções Digitais',
          TargetCNPJ: '11222333000144',
          CredentialLabel: 'Certificado TechCorp (Vencido)',
          CertPath: 'C:\\Certificados\\techcorp_old.pfx',
        })
      })
      await page.waitForSelector('.q-dialog', { timeout: 3000 })
    },
  },
  {
    route: '/',
    name: 'dialogo-senha-certificado',
    theme: 'dark',
    ready: 'text=Empresas',
    setup: async (page) => {
      await page.evaluate(() => {
        window.triggerWailsEvent?.('request-cert-password', {
          RequestID: 'req-123',
          CompanyName: 'TechCorp Soluções Digitais',
          TargetCNPJ: '11222333000144',
          CredentialLabel: 'Certificado TechCorp (Vencido)',
          CertPath: 'C:\\Certificados\\techcorp_old.pfx',
        })
      })
      await page.waitForSelector('.q-dialog', { timeout: 3000 })
    },
  },

  { route: '/documents', name: 'documentos', theme: 'light', ready: 'text=Documentos' },
  { route: '/documents', name: 'documentos', theme: 'dark', ready: 'text=Documentos' },

  {
    route: '/documents',
    name: 'detalhes-documento',
    theme: 'light',
    ready: 'text=Documentos',
    setup: async (page) => {
      // Abre a expansão da primeira nota fiscal
      await page.click('button[aria-label="Ver detalhes do serviço e impostos"]')
      await page.waitForTimeout(500) // espera a animação de expansão
    },
  },
  {
    route: '/documents',
    name: 'detalhes-documento',
    theme: 'dark',
    ready: 'text=Documentos',
    setup: async (page) => {
      // Abre a expansão da primeira nota fiscal
      await page.click('button[aria-label="Ver detalhes do serviço e impostos"]')
      await page.waitForTimeout(500) // espera a animação de expansão
    },
  },

  {
    route: '/documents',
    name: 'dialogo-eventos-documento',
    theme: 'light',
    ready: 'text=Documentos',
    setup: async (page) => {
      // Clica no botão de eventos da quarta nota (a que tem eventos na mock, status 'substituida')
      // Pode ser o primeiro botão de histórico que encontrar
      const btn = page.locator('button[aria-label="Ver Eventos"]').first()
      if (await btn.isVisible()) {
        await btn.click()
        await page.waitForSelector('.q-dialog', { timeout: 3000 })
        await page.waitForSelector('text="Emissão Normal"', { timeout: 3000 })
      }
    },
  },
  {
    route: '/documents',
    name: 'dialogo-eventos-documento',
    theme: 'dark',
    ready: 'text=Documentos',
    setup: async (page) => {
      // Clica no botão de eventos da quarta nota (a que tem eventos na mock, status 'substituida')
      // Pode ser o primeiro botão de histórico que encontrar
      const btn = page.locator('button[aria-label="Ver Eventos"]').first()
      if (await btn.isVisible()) {
        await btn.click()
        await page.waitForSelector('.q-dialog', { timeout: 3000 })
        await page.waitForSelector('text="Emissão Normal"', { timeout: 3000 })
      }
    },
  },

  { route: '/credentials', name: 'credenciais', theme: 'light', ready: 'text=Credenciais' },
  { route: '/credentials', name: 'credenciais', theme: 'dark', ready: 'text=Credenciais' },

  { route: '/query', name: 'consulta-direta', theme: 'light', ready: 'text=Consulta' },
  { route: '/query', name: 'consulta-direta', theme: 'dark', ready: 'text=Consulta' },

  { route: '/settings', name: 'configuracoes', theme: 'light', ready: 'text=Configurações' },
  { route: '/settings', name: 'configuracoes', theme: 'dark', ready: 'text=Configurações' },

  {
    route: '/',
    name: 'console-logs',
    theme: 'light',
    ready: 'text=Empresas',
    setup: async (page) => {
      await page.click('button[aria-label="Abrir console"]')
      await page.waitForSelector('.app-console', { timeout: 3000 })

      await page.evaluate((logs) => {
        logs.forEach((log) => window.triggerWailsEvent?.('backend-log', log))
      }, syncLogs)
    },
  },
  {
    route: '/',
    name: 'console-logs',
    theme: 'dark',
    ready: 'text=Empresas',
    setup: async (page) => {
      await page.click('button[aria-label="Abrir console"]')
      await page.waitForSelector('.app-console', { timeout: 3000 })

      await page.evaluate((logs) => {
        logs.forEach((log) => window.triggerWailsEvent?.('backend-log', log))
      }, syncLogs)
    },
  },
]

async function installWailsMock(context: BrowserContext) {
  await context.addInitScript(
    ({ companies, credentials, documents, events }) => {
      const listeners: Record<string, EventCallback[]> = {}

      const off = (eventName: string, callback: EventCallback) => {
        listeners[eventName] = (listeners[eventName] ?? []).filter((item) => item !== callback)
      }

      window.go = {
        main: {
          App: {
            AddCompany: async (input: unknown) => ({ ...(input as object), ID: String(Date.now()) }),
            AddCredential: async (input: unknown) => ({ ...(input as object), ID: String(Date.now()) }),
            AssignCredentialToCompany: async () => undefined,
            CancelCertPassword: async () => undefined,
            ExportDocuments: async () => ({ OutPath: 'C:\\exports\\nfs-export.xlsx', Format: 'xlsx', Incremental: false, ExportedCount: 1 }),
            ExportDANFSe: async () => ({ OutPath: 'C:\\exports\\danfse.pdf', Format: 'pdf', Incremental: false, ExportedCount: 1 }),
            ExportDANFSeZIP: async () => ({ OutPath: 'C:\\exports\\danfses.zip', Format: 'zip', Incremental: false, ExportedCount: 1 }),
            ExportXML: async () => ({ OutPath: 'C:\\exports\\nfs.xml', Format: 'xml', Incremental: false, ExportedCount: 1 }),
            ExportLogs: async () => undefined,
            CountPendingExports: async () => 0,
            MarkDocumentsViewed: async () => 0,
            ListCompanies: async () => companies,
            ListCredentials: async () => credentials,
            ListDocuments: async () => documents,
            ListEventsForDocument: async () => events,
            Pull: async (input: { CNPJ?: string }) => ({
              CompanyName: 'ACME Tecnologia e Serviços LTDA',
              CNPJ: input.CNPJ,
              CredentialLabel: 'Certificado PFX ACME 2026',
              CredentialCNPJ: '12345678000100',
              ConsultationBasis: 'prestador/tomador',
              Status: 'success',
              StopReason: 'no-more-nsu',
              LastProcessedNSU: 1024,
              LastFoundNSU: 55,
              EmptyStreak: 0,
              DocumentsFound: 3,
              EventsFound: 0,
              DocumentsSaved: 3,
              EventsSaved: 0,
              DocumentsSkippedByPolicy: 0,
              EventsSkippedByPolicy: 0,
              Errors: 0,
              Duration: 1240000000,
            }),
            ResetSyncState: async () => undefined,
            SelectCertificate: async () => 'C:\\Certificados\\mock.pfx',
            SelectExportDirectory: async () => 'C:\\exports',
            SetLogLevel: async () => undefined,
            Status: async () => ({}),
            SubmitCertPassword: async () => undefined,
            ToggleDebug: async () => undefined,
            UpdateCompany: async () => undefined,
            UpdateCredentialData: async () => undefined,
            UpdateCredentialPath: async () => undefined,
            TestConnection: async () => ({
              certLoaded: true,
              certSubject: 'ACME TECNOLOGIA',
              certExpiration: '2026-01-01T23:59:59Z',
              mtlsAccepted: true,
              endpointReached: true,
              responseCode: '200',
              responseDetail: 'OK',
              statusExplanation: 'Conexão estabelecida com sucesso.',
            }),
          },
        },
      }

      window.runtime = {
        EventsOnMultiple: (eventName, callback, maxCallbacks = Number.POSITIVE_INFINITY) => {
          let calls = 0

          const wrapped: EventCallback = (...args) => {
            calls += 1
            callback(...args)

            if (calls >= maxCallbacks) {
              off(eventName, wrapped)
            }
          }

          listeners[eventName] ??= []
          listeners[eventName].push(wrapped)

          return () => off(eventName, wrapped)
        },

        EventsOnce: (eventName, callback) => {
          window.runtime.EventsOnMultiple(eventName, callback, 1)
        },

        EventsOff: (eventName) => {
          delete listeners[eventName]
        },

        EventsOffAll: () => {
          Object.keys(listeners).forEach((eventName) => delete listeners[eventName])
        },

        EventsEmit: (eventName, ...args) => {
          for (const callback of listeners[eventName] ?? []) {
            callback(...args)
          }
        },

        WindowMinimise: () => undefined,
        WindowToggleMaximise: () => undefined,
        Quit: () => undefined,

        WindowSetDarkTheme: () => undefined,
        WindowSetLightTheme: () => undefined,
        WindowSetSystemDefaultTheme: () => undefined,

        LogPrint: () => undefined,
        LogTrace: () => undefined,
        LogDebug: () => undefined,
        LogInfo: () => undefined,
        LogWarning: () => undefined,
        LogError: () => undefined,
        LogFatal: () => undefined,
      }

      window.triggerWailsEvent = (eventName, payload) => {
        window.runtime.EventsEmit(eventName, payload)
      }
    },
    {
      companies: mockCompanies,
      credentials: mockCredentials,
      documents: mockDocuments,
      events: mockEvents,
    }
  )
}

async function createPage(browser: Browser, theme: Theme) {
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 2,
    colorScheme: theme,
  })

  await installWailsMock(context)

  await context.addInitScript((selectedTheme) => {
    localStorage.setItem('darkMode', String(selectedTheme === 'dark'))
  }, theme)

  const page = await context.newPage()

  return { context, page }
}

async function waitForApp(page: Page, spec: ScreenshotSpec) {
  await page.waitForSelector('.q-page', { timeout: 5000 })

  if (spec.ready) {
    await page.waitForSelector(spec.ready, { timeout: 5000 })
  }

  await page.evaluate(async () => {
    await document.fonts?.ready
  })

  await page.waitForTimeout(150)
}

async function capture(browser: Browser, spec: ScreenshotSpec) {
  const { context, page } = await createPage(browser, spec.theme)

  try {
    const hash = `#${spec.route}`

    await page.goto(`${appUrl}/${hash}`, { waitUntil: 'domcontentloaded' })

    await page.waitForFunction(
      (expectedHash) => window.location.hash === expectedHash,
      hash
    )

    await waitForApp(page, spec)

    if (spec.setup) {
      await spec.setup(page)
      await page.waitForTimeout(150)
    }

    const outPath = path.join(screenshotsDir, `${spec.name}-${spec.theme}.png`)

    await page.screenshot({
      path: outPath,
      animations: 'disabled',
    })

    console.log(`saved ${path.relative(rootDir, outPath)}`)
  } finally {
    await context.close()
  }
}

async function main() {
  let server: ViteDevServer | undefined
  let browser: Browser | undefined

  try {
    await mkdir(screenshotsDir, { recursive: true })

    server = await createServer({
      configFile: viteConfig,
      server: {
        port: 5555,
        strictPort: true,
      },
    })

    await server.listen()

    browser = await chromium.launch({ headless: true })

    for (const spec of screenshots) {
      await capture(browser, spec)
    }
  } finally {
    await browser?.close()
    await server?.close()
  }
}

main().catch((error) => {
  console.error(error)
  process.exitCode = 1
})
