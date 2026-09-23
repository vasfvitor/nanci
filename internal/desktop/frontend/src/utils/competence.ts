// competenceOf returns the YYYY-MM competence of date, in local time.
export function competenceOf(date: Date): string {
  return `${date.getFullYear()}-${String(date.getMonth() + 1).padStart(2, '0')}`
}

// shiftCompetence moves a YYYY-MM competence by monthDelta months. An empty
// value starts from the current month; an invalid one is returned unchanged.
export function shiftCompetence(
  value: string | null | undefined,
  monthDelta: number,
  now: Date = new Date()
): string {
  const competence = value || competenceOf(now)
  const [yearText, monthText] = competence.split('-')
  const year = Number(yearText)
  const month = Number(monthText)
  if (!Number.isInteger(year) || !Number.isInteger(month) || month < 1 || month > 12) {
    return competence
  }
  return competenceOf(new Date(year, month - 1 + monthDelta, 1))
}
