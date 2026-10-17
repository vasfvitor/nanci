import { describe, expect, it } from 'vitest'
import {
  fileName,
  importKindLabel,
  importOutcomeColor,
  importOutcomeLabel,
  importSummary,
} from './importDisplay'

describe('importDisplay', () => {
  it('counts the files by outcome in singular and plural', () => {
    expect(importSummary({ Imported: 2, Updated: 1, Rejected: 0 })).toBe(
      '2 importados, 1 atualizado, 0 rejeitados'
    )
    expect(importSummary({ Imported: 1, Updated: 0, Rejected: 1 })).toBe(
      '1 importado, 0 atualizados, 1 rejeitado'
    )
  })

  it('labels and colors the outcomes and kinds', () => {
    expect(importOutcomeLabel('imported')).toBe('Importado')
    expect(importOutcomeLabel('updated')).toBe('Atualizado')
    expect(importOutcomeLabel('rejected')).toBe('Rejeitado')
    expect(importOutcomeColor('rejected')).toBe('negative')
    expect(importOutcomeLabel('')).toBe('Desconhecido')
    expect(importKindLabel('resumo')).toBe('Resumo')
    expect(importKindLabel('evento')).toBe('Evento')
    expect(importKindLabel('')).toBe('—')
  })

  it('takes the file name out of a path', () => {
    expect(fileName('C:\\XML\\nota.xml')).toBe('nota.xml')
    expect(fileName('/home/u/evento.xml')).toBe('evento.xml')
    expect(fileName('solto.xml')).toBe('solto.xml')
  })
})
