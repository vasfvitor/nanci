<template>
  <q-bar class="app-titlebar bg-app-header text-white" @dblclick="onDblClick">
    <q-btn
      dense
      flat
      round
      icon="menu"
      aria-label="Abrir menu"
      title="Menu"
      class="app-titlebar__action q-mr-sm"
      @click="$emit('toggle-menu')"
    />

    <div class="text-weight-bold app-titlebar__title">Nanci</div>

    <q-space />

    <q-btn
      dense
      flat
      icon="terminal"
      aria-label="Abrir console"
      title="Console"
      class="app-titlebar__action"
      @click="$emit('toggle-console')"
    />

    <q-btn
      dense
      flat
      icon="minimize"
      aria-label="Minimizar janela"
      title="Minimizar"
      class="app-titlebar__action"
      @click="minimise"
    />

    <q-btn
      dense
      flat
      icon="crop_square"
      aria-label="Maximizar ou restaurar janela"
      title="Maximizar"
      class="app-titlebar__action"
      @click="toggleMaximise"
    />

    <q-btn
      dense
      flat
      icon="close"
      aria-label="Fechar aplicativo"
      title="Fechar"
      class="app-titlebar__action"
      @click="closeApp"
    />
  </q-bar>
</template>

<script setup lang="ts">
import { desktopRuntime } from '@/platform/wails/runtime'

defineEmits<{
  'toggle-menu': []
  'toggle-console': []
}>()

function minimise() {
  desktopRuntime.minimiseWindow()
}

function toggleMaximise() {
  desktopRuntime.toggleMaximiseWindow()
}

function closeApp() {
  desktopRuntime.quit()
}

function onDblClick(event: MouseEvent) {
  const target = event.target as HTMLElement

  if (target.closest('button, input, textarea, select, [role="button"]')) {
    return
  }

  desktopRuntime.toggleMaximiseWindow()
}
</script>

<style scoped>
.app-titlebar {
  height: 44px;
  --wails-draggable: drag;
  user-select: none;
}

.app-titlebar__action {
  --wails-draggable: no-drag;
}

.app-titlebar__title {
  pointer-events: none;
}
</style>
