<template>
  <div class="row items-center no-wrap q-gutter-x-sm q-mb-sm">
    <q-badge v-bind="badgeProps('warning', $q.dark.isActive)" :label="notice" class="cursor-help">
      <q-tooltip max-width="24rem">
        A SEFAZ numera os documentos de cada CNPJ sem lacunas. Um NSU pulado costuma ser um
        documento que outro programa consultou com o mesmo CNPJ, como o sistema do contador. A
        recuperação pede cada um à SEFAZ, gastando as consultas da hora.
      </q-tooltip>
    </q-badge>
    <q-btn
      dense
      flat
      no-caps
      size="sm"
      color="primary"
      icon="playlist_add"
      label="Recuperar NSUs"
      :disable="Boolean(disabledReason)"
      @click="emit('recover')"
    >
      <q-tooltip v-if="disabledReason">{{ disabledReason }}</q-tooltip>
    </q-btn>
    <q-btn
      dense
      flat
      no-caps
      size="sm"
      color="grey-7"
      icon="block"
      label="Descartar"
      title="Encerra estes NSUs sem pedi-los à SEFAZ"
      :disable="discardDisabled"
      @click="emit('discard')"
    />
  </div>
</template>

<script setup lang="ts">
import { useQuasar } from 'quasar'
import { badgeProps } from '@/utils/sefazDisplay'

// NSUGapNotice tells that the distribution skipped NSUs and offers to ask
// SEFAZ for them, or to discard them; disabledReason, when set, says why the
// recovery cannot run now.
withDefaults(
  defineProps<{
    notice: string
    disabledReason: string
    // discardDisabled is true while another operation of the source runs.
    discardDisabled?: boolean
  }>(),
  { discardDisabled: false }
)

const emit = defineEmits<{
  recover: []
  discard: []
}>()

const $q = useQuasar()
</script>
