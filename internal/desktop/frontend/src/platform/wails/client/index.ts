import { companiesClient } from './companies'
import { cteClient } from './cte'
import { nfeClient } from './nfe'
import { nfseClient } from './nfse'
import { systemClient } from './system'

export { errorMessage, mapImportXMLResult, WailsClientError, wailsErrorCode } from './base'
export type { WailsErrorCode } from './base'
export { mapCompanySummary, mapCredentialSummary } from './companies'
export { mapDocumentEvent, mapDocumentRow } from './nfse'
export {
  mapFetchByChaveResult,
  mapNFeCienciaPlan,
  mapNFeEvent,
  mapNFeEventBatchResult,
  mapNFeEventResult,
  mapNFePendingRow,
  mapNFeResetResult,
  mapNFeRow,
  mapNFeStatus,
  mapPullNFeResult,
} from './nfe'
export { mapCTeEvent, mapCTeResetResult, mapCTeRow, mapCTeStatus, mapPullCTeResult } from './cte'
export { mapBuildInfo } from './system'

export const desktopClient = {
  ...companiesClient,
  ...nfseClient,
  ...systemClient,
  ...nfeClient,
  ...cteClient,
}
