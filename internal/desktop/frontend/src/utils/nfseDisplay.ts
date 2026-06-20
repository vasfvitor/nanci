export const documentStatusColor: Record<string, string> = {
  normal: 'positive',
  cancelada: 'negative',
  substituida: 'negative',
}

export const roleLabels: Record<string, string> = {
  prestada: 'Prestada',
  tomada: 'Tomada',
  intermediario: 'Intermediário',
  none: 'Sem papel fiscal',
}

export const roleColors: Record<string, string> = {
  prestada: 'primary',
  tomada: 'secondary',
  intermediario: 'accent',
  none: 'grey',
}

export const visibilityLabels: Record<string, string> = {
  exact_prestador: 'Prestador exato',
  exact_tomador: 'Tomador exato',
  exact_intermediario: 'Intermediário exato',
  same_root_only: 'Mesmo raiz apenas',
  unknown: 'Desconhecida',
}

export const visibilityColors: Record<string, string> = {
  exact_prestador: 'positive',
  exact_tomador: 'positive',
  exact_intermediario: 'positive',
  same_root_only: 'warning',
  unknown: 'grey',
}

export function statusColor(status: string) {
  return documentStatusColor[status] || 'grey'
}

export function roleLabel(role: string) {
  return roleLabels[role] || role || 'Desconhecido'
}

export function roleColor(role: string) {
  return roleColors[role] || 'grey'
}

export function visibilityLabel(reason: string) {
  return visibilityLabels[reason] || reason || 'Desconhecida'
}

export function visibilityColor(reason: string) {
  return visibilityColors[reason] || 'grey'
}

export function getRoleAbbreviation(role?: string): string {
  const abbreviations: Record<string, string> = {
    prestada: 'P',
    tomada: 'T',
    intermediario: 'I',
  }
  return abbreviations[role ?? ''] ?? '-'
}

export function getVisibilityAbbreviation(reason?: string): string {
  const abbreviations: Record<string, string> = {
    exact_prestador: 'PE',
    exact_tomador: 'TE',
    exact_intermediario: 'IE',
    same_root_only: 'MR',
  }
  return abbreviations[reason ?? ''] ?? '?'
}

export function getStatusAbbreviation(status?: string): string {
  const abbreviations: Record<string, string> = {
    normal: 'N',
    cancelada: 'C',
    substituida: 'S',
  }
  return abbreviations[status ?? ''] ?? '?'
}

export function getStatusLabel(status?: string): string {
  const labels: Record<string, string> = {
    normal: 'Normal',
    cancelada: 'Cancelada',
    substituida: 'Substituída',
  }
  return labels[status ?? ''] ?? status ?? 'Desconhecido'
}
