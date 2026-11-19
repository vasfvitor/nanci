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
  </div>
</template>

<script setup lang="ts">
import { useQuasar } from 'quasar'
import { badgeProps } from '@/utils/sefazDisplay'

// NSUGapNotice tells that the distribution skipped NSUs and offers to ask
// SEFAZ for them; disabledReason, when set, says why it cannot run now.
defineProps<{
  notice: string
  disabledReason: string
}>()

const emit = defineEmits<{
  recover: []
}>()

const $q = useQuasar()
</script>
