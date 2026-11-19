import { computed, type Ref } from 'vue'
import { useSefazBlock } from '@/composables/useSefazBlock'
import type { CompanySummary, SefazSourceStatus } from '@/types/desktop'
import { idleWarning as idleWarningText, sefazAmbiente } from '@/utils/sefazDisplay'

// useSefazStatus derives what the NF-e and CT-e pages show from the shared
// part of their status: the company name (the listed company's while the
// status loads), the ambiente badge, the 60-day idle warning and the block.
export function useSefazStatus(
  status: Readonly<Ref<SefazSourceStatus | null>>,
  selectedCompany: Readonly<Ref<CompanySummary | null | undefined>>
) {
  const companyName = computed(() => status.value?.CompanyName || selectedCompany.value?.Name || '')
  const ambiente = computed(() => (status.value ? sefazAmbiente(status.value.TpAmb) : null))
  const idleWarning = computed(() => idleWarningText(status.value))
  const { syncBlockedUntil, blockedText } = useSefazBlock(status)

  return { companyName, ambiente, idleWarning, syncBlockedUntil, blockedText }
}
