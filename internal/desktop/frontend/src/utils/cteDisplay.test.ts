import { describe, expect, it } from 'vitest'
import {
  cteDocumentCount,
  cteEventColor,
  cteEventTitle,
  cteModalLabel,
  cteModeloFilterOptions,
  cteModelo,
  cteMunicipioLabel,
  cteOtherPapeis,
  ctePapel,
  ctePapelFilterOptions,
  cteParticipantes,
  ctePercurso,
  cteSituacao,
  cteSituacaoFilterOptions,
  cteStateBadges,
  cteStatusLine,
  cteTipoDocumento,
  cteTpServLabel,
} from './cteDisplay'
import type { CTeEventType, CTeRow, CTeStatusResult } from '@/types/desktop'
import type { DisplayTable } from './sefazDisplay'

// documentBadge is the first state badge of a CT-e, which names its kind.
function documentBadge(row: Pick<CTeRow, 'Modelo' | 'TipoDocumento'>) {
  return cteStateBadges({ ...row, Situacao: 'autorizada', CompanyRole: 'tomador', Papeis: [] })[0]
}

// eventLabel names an event type as the events dialog shows it.
function eventLabel(type: CTeEventType | '') {
  return cteEventTitle({ Type: type, TpEvento: '', Description: '' })
}

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
    expect(cteSituacao.label('autorizada')).toBe('Autorizada')
    expect(cteSituacao.label('denegada')).toBe('Denegada')
    expect(cteSituacao.label('cancelada')).toBe('Cancelada')
    expect(cteSituacao.color('autorizada')).toBe('positive')
    expect(cteSituacao.color('cancelada')).toBe('negative')
  })

  it('maps papel values', () => {
    expect(ctePapel.label('tomador')).toBe('Tomador')
    expect(ctePapel.label('destinatario')).toBe('Destinatário')
    expect(ctePapel.label('remetente')).toBe('Remetente')
    expect(ctePapel.label('expedidor')).toBe('Expedidor')
    expect(ctePapel.label('recebedor')).toBe('Recebedor')
    expect(ctePapel.label('emitente')).toBe('Emitente')
    expect(ctePapel.label('autorizado')).toBe('Autorizado')
    expect(ctePapel.label('none')).toBe('Sem papel fiscal')
    expect(ctePapel.color('tomador')).toBe('accent')
    expect(ctePapel.color('none')).toBe('grey')
  })

  it('names modelos and tipos de documento', () => {
    expect(cteModelo.label('57')).toBe('CT-e')
    expect(cteModelo.label('64')).toBe('GTV-e')
    expect(cteModelo.label('67')).toBe('CT-e OS')
    expect(cteModelo.color('57')).toBe('primary')
    expect(cteTipoDocumento.label('cte')).toBe('CT-e')
    expect(cteTipoDocumento.label('cte_os')).toBe('CT-e OS')
    expect(cteTipoDocumento.label('gtve')).toBe('GTV-e')
    expect(cteTipoDocumento.label('cte_simplificado')).toBe('CT-e Simplificado')
  })

  it('prefers the tipo de documento over the modelo', () => {
    expect(documentBadge({ Modelo: '57', TipoDocumento: 'cte_simplificado' })).toMatchObject({
      label: 'CT-e Simplificado',
      color: 'info',
    })
    expect(documentBadge({ Modelo: '67', TipoDocumento: '' })).toMatchObject({ label: 'CT-e OS', color: 'secondary' })
    expect(documentBadge({ Modelo: '', TipoDocumento: '' })?.label).toBe('Desconhecido')
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
    const labels = types.map(eventLabel)
    expect(new Set(labels).size).toBe(types.length)
    for (const label of labels) expect(label).not.toMatch(/_/)
    expect(eventLabel('cancelamento')).toBe('Cancelamento')
    expect(eventLabel('comprovante_entrega')).toBe('Comprovante de entrega')
    expect(eventLabel('prestacao_desacordo')).toBe('Prestação de serviço em desacordo')
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
    expect(cteSituacao.label('')).toBe('Desconhecido')
    expect(cteSituacao.color('suspensa')).toBe('grey')
    expect(ctePapel.label('')).toBe('Desconhecido')
    expect(ctePapel.color('transportador')).toBe('grey')
    expect(cteModelo.label('55')).toBe('Modelo 55')
    expect(cteModelo.color('55')).toBe('grey')
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
    expect(documentBadge({ TipoDocumento: 'cte_simplificado', Modelo: '57' })?.abbr).toBe('CS')
    expect(documentBadge({ TipoDocumento: '', Modelo: '67' })?.abbr).toBe('OS')
    expect(documentBadge({ TipoDocumento: '', Modelo: '64' })?.abbr).toBe('GV')
    expect(documentBadge({ TipoDocumento: '', Modelo: '' })?.abbr).toBe('—')
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
