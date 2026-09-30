import { describe, expect, it } from 'vitest'
import {
  cteDocumentAbbr,
  cteDocumentColor,
  cteDocumentCount,
  cteDocumentLabel,
  cteEventColor,
  cteEventLabel,
  cteEventTitle,
  cteModalLabel,
  cteModeloColor,
  cteModeloFilterOptions,
  cteModeloLabel,
  cteModelo,
  cteMunicipioLabel,
  cteOtherPapeis,
  ctePapel,
  ctePapelColor,
  ctePapelFilterOptions,
  ctePapelLabel,
  cteParticipantes,
  ctePercurso,
  cteSituacao,
  cteSituacaoColor,
  cteSituacaoFilterOptions,
  cteSituacaoLabel,
  cteStateBadges,
  cteStatusLine,
  cteTipoDocumento,
  cteTipoDocumentoLabel,
  cteTpServLabel,
} from './cteDisplay'
import type { CTeEventType, CTeRow, CTeStatusResult } from '@/types/desktop'
import type { DisplayTable } from './sefazDisplay'

// expectShortUniqueAbbrs checks that every value of table has an
// abbreviation of one or two characters, unique within the table.
function expectShortUniqueAbbrs(table: DisplayTable) {
  const abbrs = table.values().map((value) => table.abbr(value))
  for (const abbr of abbrs) {
    expect(abbr).toMatch(/^.{1,2}$/u)
  }
  expect(new Set(abbrs).size).toBe(abbrs.length)
}

describe('cteDisplay', () => {
  it('maps situação values', () => {
    expect(cteSituacaoLabel('autorizada')).toBe('Autorizada')
    expect(cteSituacaoLabel('denegada')).toBe('Denegada')
    expect(cteSituacaoLabel('cancelada')).toBe('Cancelada')
    expect(cteSituacaoColor('autorizada')).toBe('positive')
    expect(cteSituacaoColor('cancelada')).toBe('negative')
  })

  it('maps papel values', () => {
    expect(ctePapelLabel('tomador')).toBe('Tomador')
    expect(ctePapelLabel('destinatario')).toBe('Destinatário')
    expect(ctePapelLabel('remetente')).toBe('Remetente')
    expect(ctePapelLabel('expedidor')).toBe('Expedidor')
    expect(ctePapelLabel('recebedor')).toBe('Recebedor')
    expect(ctePapelLabel('emitente')).toBe('Emitente')
    expect(ctePapelLabel('autorizado')).toBe('Autorizado')
    expect(ctePapelLabel('none')).toBe('Sem papel fiscal')
    expect(ctePapelColor('tomador')).toBe('accent')
    expect(ctePapelColor('none')).toBe('grey')
  })

  it('names modelos and tipos de documento', () => {
    expect(cteModeloLabel('57')).toBe('CT-e')
    expect(cteModeloLabel('64')).toBe('GTV-e')
    expect(cteModeloLabel('67')).toBe('CT-e OS')
    expect(cteModeloColor('57')).toBe('primary')
    expect(cteTipoDocumentoLabel('cte')).toBe('CT-e')
    expect(cteTipoDocumentoLabel('cte_os')).toBe('CT-e OS')
    expect(cteTipoDocumentoLabel('gtve')).toBe('GTV-e')
    expect(cteTipoDocumentoLabel('cte_simplificado')).toBe('CT-e Simplificado')
  })

  it('prefers the tipo de documento over the modelo', () => {
    expect(cteDocumentLabel({ Modelo: '57', TipoDocumento: 'cte_simplificado' })).toBe('CT-e Simplificado')
    expect(cteDocumentColor({ Modelo: '57', TipoDocumento: 'cte_simplificado' })).toBe('info')
    expect(cteDocumentLabel({ Modelo: '67', TipoDocumento: '' })).toBe('CT-e OS')
    expect(cteDocumentColor({ Modelo: '67', TipoDocumento: '' })).toBe('secondary')
    expect(cteDocumentLabel({ Modelo: '', TipoDocumento: '' })).toBe('Desconhecido')
  })

  it('maps tpServ and modal codes', () => {
    expect(cteTpServLabel('0')).toBe('Normal')
    expect(cteTpServLabel('1')).toBe('Subcontratação')
    expect(cteTpServLabel('2')).toBe('Redespacho')
    expect(cteTpServLabel('3')).toBe('Redespacho intermediário')
    expect(cteTpServLabel('4')).toBe('Vinculado a multimodal')
    expect(cteTpServLabel('7')).toBe('Transporte de valores')
    expect(cteTpServLabel('9')).toBe('Serviço 9')
    expect(cteModalLabel('01')).toBe('Rodoviário')
    expect(cteModalLabel('02')).toBe('Aéreo')
    expect(cteModalLabel('03')).toBe('Aquaviário')
    expect(cteModalLabel('04')).toBe('Ferroviário')
    expect(cteModalLabel('05')).toBe('Dutoviário')
    expect(cteModalLabel('06')).toBe('Multimodal')
    expect(cteModalLabel('07')).toBe('Modal 07')
  })

  it('labels every event type in Portuguese', () => {
    const types: CTeEventType[] = [
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
    const labels = types.map(cteEventLabel)
    expect(new Set(labels).size).toBe(types.length)
    for (const label of labels) expect(label).not.toMatch(/_/)
    expect(cteEventLabel('cancelamento')).toBe('Cancelamento')
    expect(cteEventLabel('comprovante_entrega')).toBe('Comprovante de entrega')
    expect(cteEventLabel('prestacao_desacordo')).toBe('Prestação de serviço em desacordo')
    expect(cteEventColor('cancelamento')).toBe('negative')
    expect(cteEventColor('comprovante_entrega')).toBe('positive')
  })

  it('titles unrecognized events from the description or the tpEvento', () => {
    expect(cteEventTitle({ Type: 'carta_correcao', TpEvento: '110110', Description: 'x' })).toBe('Carta de Correção')
    expect(cteEventTitle({ Type: 'unknown', TpEvento: '999999', Description: 'Evento novo' })).toBe('Evento novo')
    expect(cteEventTitle({ Type: '', TpEvento: '999999', Description: '' })).toBe('Evento 999999')
    expect(cteEventTitle({ Type: '', TpEvento: '', Description: '' })).toBe('Evento não reconhecido')
  })

  it('shows unknown values as unknown in grey', () => {
    expect(cteSituacaoLabel('')).toBe('Desconhecido')
    expect(cteSituacaoColor('suspensa')).toBe('grey')
    expect(ctePapelLabel('')).toBe('Desconhecido')
    expect(ctePapelColor('transportador')).toBe('grey')
    expect(cteModeloLabel('55')).toBe('Modelo 55')
    expect(cteModeloColor('55')).toBe('grey')
    expect(cteEventLabel('')).toBe('Desconhecido')
    expect(cteEventColor('other')).toBe('grey')
  })

  it('builds filter options with an "all" entry first', () => {
    expect(cteSituacaoFilterOptions[0]).toEqual({ label: 'Todas', value: '' })
    expect(cteSituacaoFilterOptions.map((o) => o.value)).toEqual(['', 'autorizada', 'denegada', 'cancelada'])
    expect(ctePapelFilterOptions[0]).toEqual({ label: 'Todos', value: '' })
    expect(ctePapelFilterOptions.map((o) => o.value)).toEqual([
      '',
      'tomador',
      'destinatario',
      'remetente',
      'expedidor',
      'recebedor',
      'emitente',
      'autorizado',
      'none',
    ])
    expect(cteModeloFilterOptions).toEqual([
      { label: 'Todos', value: '' },
      { label: 'CT-e', value: '57' },
      { label: 'GTV-e', value: '64' },
      { label: 'CT-e OS', value: '67' },
    ])
  })

  it('counts CT-e and sums up the status in one line', () => {
    const status = {
      LastSyncAt: null,
      LastNSU: 10,
      MaxNSU: null,
      TotalTomador: 4,
      TotalDestinatario: 3,
      TotalRemetente: 2,
      TotalOutros: 1,
    } as unknown as CTeStatusResult
    expect(cteDocumentCount(status)).toBe(10)
    expect(cteDocumentCount(null)).toBe(0)
    expect(cteStatusLine(status)).toBe('Última sincronização: nunca · NSU 10/— · CT-e: 10')
    expect(cteStatusLine({ ...status, MaxNSU: 12 })).toContain('NSU 10/12')
  })

  it('names municípios and the percurso', () => {
    const saoPaulo = { Codigo: '3550308', Nome: 'São Paulo', UF: 'SP' }
    expect(cteMunicipioLabel(saoPaulo)).toBe('São Paulo/SP')
    expect(cteMunicipioLabel({ Codigo: '3550308', Nome: '', UF: '' })).toBe('3550308')
    expect(cteMunicipioLabel({ Codigo: '', Nome: '', UF: '' })).toBe('—')
    expect(ctePercurso({ MunIni: saoPaulo, MunFim: { Codigo: '4106902', Nome: 'Curitiba', UF: 'PR' } })).toBe(
      'São Paulo/SP → Curitiba/PR'
    )
  })

  it('lists the parties present and the other roles of the company', () => {
    const row = {
      RemetenteCNPJ: '11222333000181',
      RemetenteName: 'Remetente',
      DestinatarioCNPJ: '',
      DestinatarioName: '',
      ExpedidorCNPJ: '',
      ExpedidorName: '',
      RecebedorCNPJ: '',
      RecebedorName: 'Só o nome',
      TomadorCNPJ: '11222333000181',
      TomadorName: 'Remetente',
      CompanyRole: 'tomador',
      Papeis: ['tomador', 'remetente'],
    } as CTeRow

    expect(cteParticipantes(row).map((party) => party.label)).toEqual(['Remetente', 'Recebedor', 'Tomador'])
    expect(cteOtherPapeis(row)).toEqual(['remetente'])
  })

  it('abbreviates every value with one or two unique characters', () => {
    for (const table of [cteSituacao, ctePapel, cteModelo, cteTipoDocumento]) {
      expectShortUniqueAbbrs(table)
    }
    expect(cteTipoDocumento.values().map(cteTipoDocumento.abbr)).toEqual(['CT', 'OS', 'GV', 'CS'])
    expect(ctePapel.values().map(ctePapel.abbr)).toEqual(['TO', 'DE', 'RE', 'EX', 'RC', 'EM', 'AU', 'SP'])
    expect(cteSituacao.values().map(cteSituacao.abbr)).toEqual(['A', 'D', 'C'])
  })

  it('abbreviates the document by its tipo, falling back to the modelo', () => {
    expect(cteDocumentAbbr({ TipoDocumento: 'cte_simplificado', Modelo: '57' })).toBe('CS')
    expect(cteDocumentAbbr({ TipoDocumento: '', Modelo: '67' })).toBe('OS')
    expect(cteDocumentAbbr({ TipoDocumento: '', Modelo: '64' })).toBe('GV')
    expect(cteDocumentAbbr({ TipoDocumento: '', Modelo: '' })).toBe('—')
  })

  it('lists the state badges with the other papéis as secondary', () => {
    const badges = cteStateBadges({
      TipoDocumento: 'cte',
      Modelo: '57',
      Situacao: 'autorizada',
      CompanyRole: 'tomador',
      Papeis: ['tomador', 'remetente', 'expedidor'],
    })
    expect(badges.map((badge) => [badge.kind, badge.abbr, badge.secondary ?? false])).toEqual([
      ['Documento', 'CT', false],
      ['Situação', 'A', false],
      ['Papel', 'TO', false],
      ['Papel', 'RE', true],
      ['Papel', 'EX', true],
    ])
    expect(new Set(badges.map((badge) => badge.key)).size).toBe(badges.length)
  })

  it('takes the document badge from the modelo when the tipo is missing', () => {
    const [documento] = cteStateBadges({
      TipoDocumento: '',
      Modelo: '67',
      Situacao: 'cancelada',
      CompanyRole: 'destinatario',
      Papeis: ['destinatario'],
    })
    expect(documento).toEqual({
      key: 'Documento:67',
      abbr: 'OS',
      label: 'CT-e OS',
      color: 'secondary',
      kind: 'Documento',
    })
  })
})
