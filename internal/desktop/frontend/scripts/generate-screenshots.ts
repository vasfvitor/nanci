import { createServer } from 'vite'
import { chromium } from 'playwright'
import fs from 'node:fs'
import path from 'node:path'
import { fileURLToPath } from 'node:url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))
const rootDir = path.resolve(__dirname, '../../../..')
const screenshotsDir = path.resolve(rootDir, 'docs/screenshots')

// Mock structures matching types/desktop.ts
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
    LastNSU: 1024,
    LastFoundNSU: 1024,
    LastFoundNSUValid: true,
    LastSyncAt: '2026-06-18T14:30:00-03:00',
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
    LastNSU: 50,
    LastFoundNSU: 45,
    LastFoundNSUValid: true,
    LastSyncAt: '2026-06-18T11:15:22-03:00',
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
    LastNSU: 0,
    LastFoundNSU: 0,
    LastFoundNSUValid: false,
    LastSyncAt: null,
    LastRunStatus: 'error',
    LastRunStopReason: 'cert-missing',
    CreatedAt: '2026-05-10T10:00:00Z',
    UpdatedAt: '2026-05-10T10:00:00Z',
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
    ServiceValue: 150000, // R$ 1.500,00 (cents)
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
    VisibilityReason: 'prestador',
    FirstSeenNSU: 101,
    LastSeenNSU: 101,
    FirstSeenNSUValid: true,
    LastSeenNSUValid: true,
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
    ServiceValue: 500000, // R$ 5.000,00
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
    VisibilityReason: 'tomador',
    FirstSeenNSU: 102,
    LastSeenNSU: 102,
    FirstSeenNSUValid: true,
    LastSeenNSUValid: true,
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
    ServiceValue: 85000, // R$ 850,00
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
    VisibilityReason: 'prestador',
    FirstSeenNSU: 95,
    LastSeenNSU: 95,
    FirstSeenNSUValid: true,
    LastSeenNSUValid: true,
    FirstSyncedAt: '2026-06-15T09:10:00Z',
    LastSyncedAt: '2026-06-15T09:10:00Z',
  },
]

async function run() {
  console.log('🚀 Iniciando servidor Vite para captura de prints...')
  const server = await createServer({
    configFile: path.resolve(__dirname, '../vite.config.ts'),
    server: {
      port: 5173,
    },
  })
  await server.listen()
  console.log('✅ Servidor Vite rodando em http://localhost:5173')

  // Garantir existência do diretório de prints
  if (!fs.existsSync(screenshotsDir)) {
    fs.mkdirSync(screenshotsDir, { recursive: true })
    console.log(`📁 Diretório criado: ${screenshotsDir}`)
  }

  console.log('🌐 Inicializando navegador Playwright (Chromium)...')
  const browser = await chromium.launch({ headless: true })
  const context = await browser.newContext({
    viewport: { width: 1280, height: 800 },
    deviceScaleFactor: 2, // Deixa a imagem nítida (retina quality)
  })

  // Injeta mocks globais para contornar o Wails
  await context.addInitScript(
    ({ mockCompanies, mockCredentials, mockDocuments }) => {
      const eventListeners: Record<string, Function[]> = {}

      window.go = {
        main: {
          App: {
            AddCompany: async (input: any) => ({ ...input, ID: String(Date.now()) }),
            AddCredential: async (input: any) => ({ ...input, ID: String(Date.now()) }),
            AssignCredentialToCompany: async () => {},
            CancelCertPassword: async () => {},
            ExportDocuments: async () => ({ OutPath: 'C:\\exports\\nfs-export.xlsx', Format: 'xlsx' }),
            ExportLogs: async () => {},
            ListCompanies: async () => mockCompanies,
            ListCredentials: async () => mockCredentials,
            ListDocuments: async () => mockDocuments,
            ListEventsForDocument: async () => [],
            Pull: async (input: any) => ({
              CompanyName: 'ACME Tecnologia e Serviços LTDA',
              CNPJ: input.CNPJ,
              CredentialLabel: 'Certificado PFX ACME 2026',
              CredentialCNPJ: '12345678000100',
              ConsultationBasis: 'prestador/tomador',
              Status: 'success',
              StopReason: 'no-more-nsu',
              LastCheckedNSU: 1024,
              LastFoundNSU: 1024,
              LastFoundNSUValid: true,
              EmptyStreak: 0,
              DocumentsFound: 3,
              EventsFound: 0,
              Errors: 0,
              Duration: 1240000000,
            }),
            ResetSyncState: async () => {},
            SelectCertificate: async () => 'C:\\Certificados\\mock.pfx',
            SelectExportDirectory: async () => 'C:\\exports',
            SetLogLevel: async () => {},
            Status: async () => ({}),
            SubmitCertPassword: async () => {},
            ToggleDebug: async () => {},
            UpdateCompany: async () => {},
            UpdateCredentialData: async () => {},
            UpdateCredentialPath: async () => {},
          },
        },
      }

      window.runtime = {
        EventsOnMultiple: (eventName: string, callback: Function, maxCallbacks: number) => {
          if (!eventListeners[eventName]) eventListeners[eventName] = []
          eventListeners[eventName].push(callback)
          return () => {
            eventListeners[eventName] = eventListeners[eventName].filter((cb) => cb !== callback)
          }
        },
        EventsOff: (eventName: string) => {
          delete eventListeners[eventName]
        },
        EventsOffAll: () => {
          Object.keys(eventListeners).forEach((k) => delete eventListeners[k])
        },
        EventsOnce: (eventName: string, callback: Function) => {
          window.runtime.EventsOnMultiple(eventName, callback, 1)
        },
        EventsEmit: (eventName: string, ...args: any[]) => {
          if (eventListeners[eventName]) {
            eventListeners[eventName].forEach((cb) => cb(...args))
          }
        },
        WindowMinimise: () => {},
        WindowToggleMaximise: () => {},
        Quit: () => {},
        WindowSetDarkTheme: () => {},
        WindowSetLightTheme: () => {},
        WindowSetSystemDefaultTheme: () => {},
        LogPrint: () => {},
        LogTrace: () => {},
        LogDebug: () => {},
        LogInfo: () => {},
        LogWarning: () => {},
        LogError: () => {},
        LogFatal: () => {},
      }

      // Helper global para injetar logs ou notificações via console do Playwright
      window.triggerWailsEvent = (eventName: string, payload: any) => {
        window.runtime.EventsEmit(eventName, payload)
      }
    },
    { mockCompanies, mockCredentials, mockDocuments }
  )

  const page = await context.newPage()

  async function takeScreenshot(route: string, name: string, mode: 'light' | 'dark', options?: { beforeCapture?: () => Promise<void> }) {
    console.log(`📸 Capturando ${name} (${mode})...`)
    
    // Configura o tema no localStorage do navegador
    await page.goto('http://localhost:5173/')
    await page.evaluate((m) => {
      localStorage.setItem('darkMode', String(m === 'dark'))
    }, mode)

    // Navega para a rota de fato
    await page.goto(`http://localhost:5173/#${route}`)
    
    // Espera renderizar
    await page.waitForSelector('.q-page', { timeout: 5000 })
    
    // Estabiliza transições / fontes
    await page.waitForTimeout(500)

    if (options?.beforeCapture) {
      await options.beforeCapture()
      await page.waitForTimeout(300)
    }

    const screenshotPath = path.resolve(screenshotsDir, `${name}-${mode}.png`)
    await page.screenshot({ path: screenshotPath })
    console.log(`  💾 Salvo em: ${screenshotPath}`)
  }

  // 1. Pagina de Empresas
  await takeScreenshot('/', 'empresas', 'light')
  await takeScreenshot('/', 'empresas', 'dark')

  // 2. Dialogo de Adicionar Empresa (no tema escuro para destaque)
  await takeScreenshot('/', 'dialogo-adicionar-empresa', 'dark', {
    beforeCapture: async () => {
      await page.click('button:has-text("Adicionar")')
      await page.waitForSelector('.q-dialog', { timeout: 3000 })
    }
  })

  // 3. Pagina de Documentos
  await takeScreenshot('/documents', 'documentos', 'light')
  await takeScreenshot('/documents', 'documentos', 'dark')

  // 4. Pagina de Credenciais
  await takeScreenshot('/credentials', 'credenciais', 'light')
  await takeScreenshot('/credentials', 'credenciais', 'dark')

  // 5. Pagina de Consulta Direta
  await takeScreenshot('/query', 'consulta-direta', 'light')
  await takeScreenshot('/query', 'consulta-direta', 'dark')

  // 6. Pagina de Configurações
  await takeScreenshot('/settings', 'configuracoes', 'light')
  await takeScreenshot('/settings', 'configuracoes', 'dark')

  // 7. Console de Logs do Sync (com logs reais simulados)
  await takeScreenshot('/', 'console-logs', 'dark', {
    beforeCapture: async () => {
      // Abre a gaveta do console
      await page.click('button[aria-label="Abrir console"]')
      await page.waitForSelector('.app-console', { timeout: 3000 })

      // Injeta logs via evento global
      await page.evaluate(() => {
        const trigger = (window as any).triggerWailsEvent
        if (!trigger) return

        const logs = [
          { level: 'INFO', msg: 'Iniciando sincronização para ACME Tecnologia e Serviços LTDA', time: '2026-06-18T14:30:01Z' },
          { level: 'DEBUG', msg: 'Acessando endpoint do ambiente de produção', time: '2026-06-18T14:30:02Z' },
          { level: 'TRACE', msg: 'HTTP POST: https://adn.api.gov.br/consultarNSU - CNPJ: 12345678000100', time: '2026-06-18T14:30:02Z' },
          { level: 'INFO', msg: 'Busca concluída: 3 novos documentos encontrados', time: '2026-06-18T14:30:03Z' },
          { level: 'INFO', msg: 'Salvando documento 1234 (R$ 1.500,00) no banco local...', time: '2026-06-18T14:30:03Z' },
          { level: 'WARNING', msg: 'Aliquota ISS incomum para o municipio em documento 5678', time: '2026-06-18T14:30:04Z' },
          { level: 'INFO', msg: 'Salvando documento 5678 (R$ 5.000,00) no banco local...', time: '2026-06-18T14:30:04Z' },
          { level: 'INFO', msg: 'Salvando documento 1235 (R$ 850,00) [Cancelada] no banco local...', time: '2026-06-18T14:30:04Z' },
          { level: 'INFO', msg: 'Sincronização concluída com sucesso. Tempo total: 1.24s', time: '2026-06-18T14:30:05Z' },
        ]

        logs.forEach(l => trigger('backend-log', l))
      })
    }
  })

  console.log('🧹 Finalizando navegador e servidor dev...')
  await browser.close()
  await server.close()
  console.log('🎉 Todos os prints foram capturados e salvos com sucesso em docs/screenshots!')
}

run().catch((err) => {
  console.error('❌ Erro durante a geração de prints:', err)
  process.exit(1)
})
