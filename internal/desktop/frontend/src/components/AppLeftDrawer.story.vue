<script setup lang="ts">
import { ref } from 'vue'
import AppLeftDrawer from './AppLeftDrawer.vue'
import { useWorkspaceStore } from '@/stores/workspace'
import type { CompanySummary } from '@/types/desktop'

const open = ref(true)

function company(CNPJ: string, Name: string): CompanySummary {
  return {
    ID: CNPJ,
    CNPJ,
    CNPJRoot: CNPJ.slice(0, 8),
    Name,
    CredentialID: '',
    CredentialLabel: '',
    CredentialCertPath: '',
    Environment: 'producao',
    UF: 'SP',
    LastFoundNSU: null,
    SyncStartPolicy: 'all',
    LastRunStatus: '',
    LastRunStopReason: '',
  }
}

// Histoire has Pinia but no Wails, so the story fills the workspace itself.
const workspace = useWorkspaceStore()
workspace.companies = [
  company('11222333000181', 'ACME Comércio Ltda.'),
  company('44555666000199', 'Wayne Empreendimentos S.A.'),
]
workspace.cnpj = '11222333000181'
workspace.competence = '2026-09'
workspace.loaded = true
</script>

<template>
  <Story title="Layout/AppLeftDrawer">
    <Variant title="Light Theme">
      <div class="body--light preview-container">
        <q-layout view="hHh Lpr lFf" container class="fit" style="height: 400px">
          <AppLeftDrawer v-model="open" />
          <q-page-container>
            <q-page class="q-pa-md">
              <q-btn color="primary" @click="open = !open">Toggle Drawer</q-btn>
            </q-page>
          </q-page-container>
        </q-layout>
      </div>
    </Variant>

    <Variant title="Dark Theme">
      <div class="body--dark preview-container">
        <q-layout view="hHh Lpr lFf" container class="fit" style="height: 400px">
          <AppLeftDrawer v-model="open" />
          <q-page-container>
            <q-page class="q-pa-md">
              <q-btn color="primary" @click="open = !open">Toggle Drawer</q-btn>
            </q-page>
          </q-page-container>
        </q-layout>
      </div>
    </Variant>
  </Story>
</template>

<style scoped>
.preview-container {
  height: 450px;
  position: relative;
  background: var(--app-bg);
  color: var(--app-text);
}
</style>
