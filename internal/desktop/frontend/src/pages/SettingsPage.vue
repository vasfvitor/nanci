<template>
  <q-page padding class="column">
    <div class="row items-center q-mb-md">
      <div class="text-h5 text-weight-bold">Configurações</div>
      <q-space />
    </div>

    <q-card flat bordered class="q-pa-md">
      <q-list>
        <q-item tag="label" v-ripple>
          <q-item-section>
            <q-item-label class="text-weight-medium">Modo Escuro (Dark Mode)</q-item-label>
            <q-item-label caption>Ajuste o tema de cores do aplicativo</q-item-label>
          </q-item-section>
          <q-item-section side>
            <q-select
              v-model="darkMode"
              :options="darkModeOptions"
              emit-value
              map-options
              dense
              outlined
              style="min-width: 200px"
              @update:model-value="updateDarkMode"
            />
          </q-item-section>
        </q-item>
      </q-list>
    </q-card>
  </q-page>
</template>

<script setup lang="ts">
import { ref, onMounted } from 'vue'
import { useQuasar } from 'quasar'

const $q = useQuasar()
const darkMode = ref<'auto' | boolean>('auto')

const darkModeOptions = [
  { label: 'Automático (Sistema)', value: 'auto' },
  { label: 'Sempre Claro', value: false },
  { label: 'Sempre Escuro', value: true }
]

onMounted(() => {
  const saved = localStorage.getItem('darkMode')
  if (saved === 'true') {
    darkMode.value = true
  } else if (saved === 'false') {
    darkMode.value = false
  } else {
    darkMode.value = 'auto'
  }
})

function updateDarkMode(val: 'auto' | boolean) {
  localStorage.setItem('darkMode', String(val))
  $q.dark.set(val)
}
</script>
