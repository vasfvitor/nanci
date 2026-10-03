// DocumentSource is a fiscal document source of nanci. The key also names
// its table preferences and its sync in companySync.
export type DocumentSource = 'nfse' | 'nfe' | 'cte'

type SourceInfo = {
  noun: 'NFS-e' | 'NF-e' | 'CT-e'
  // The NFS-e and the NF-e are notas (feminine), the CT-e a conhecimento
  // (masculine).
  feminine: boolean
  // article is "of the" before the noun: "da NF-e", "do CT-e".
  article: 'da' | 'do'
}

export const DOCUMENT_SOURCES: Record<DocumentSource, SourceInfo> = {
  nfse: { noun: 'NFS-e', feminine: true, article: 'da' },
  nfe: { noun: 'NF-e', feminine: true, article: 'da' },
  cte: { noun: 'CT-e', feminine: false, article: 'do' },
}

// agree inflects stem for the gender of source and the number of count:
// agree('nfe', 'vist', 2) is "vistas", agree('cte', 'nov', 1) is "novo".
export function agree(source: DocumentSource, stem: string, count: number) {
  return stem + (DOCUMENT_SOURCES[source].feminine ? 'a' : 'o') + (count === 1 ? '' : 's')
}
