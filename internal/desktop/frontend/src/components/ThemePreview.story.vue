<template>
  <Story title="Design System/Theme Palette">
    <Variant title="Light Theme">
      <div class="body--light preview-container q-pa-lg">
        <div class="text-h4 q-mb-md">Light Theme Tokens</div>
        
        <div class="q-pa-md bg-app-bg q-mb-md border-radius border-std">
          <div class="text-subtitle1 q-mb-sm">Background Area (--app-bg)</div>
          <div class="q-pa-md bg-app-surface border-radius border-std">
            <div class="text-h6">Surface Area (--app-surface)</div>
            <p>Main text color over surface (--app-text).</p>
            <p class="text-app-muted">Muted text color over surface (--app-text-muted).</p>
            
            <q-btn color="primary" class="q-mt-sm" style="color: var(--app-primary-text)">
              Primary Button Text (--app-primary-text)
            </q-btn>
          </div>
        </div>
      </div>
    </Variant>

    <Variant title="Dark Theme">
      <div class="body--dark preview-container q-pa-lg">
        <div class="text-h4 q-mb-md">Dark Theme Tokens</div>
        
        <div class="q-pa-md bg-app-bg q-mb-md border-radius border-std">
          <div class="text-subtitle1 q-mb-sm">Background Area (--app-bg)</div>
          <div class="q-pa-md bg-app-surface border-radius border-std">
            <div class="text-h6">Surface Area (--app-surface)</div>
            <p>Main text color over surface (--app-text).</p>
            <p class="text-app-muted">Muted text color over surface (--app-text-muted).</p>
            
            <q-btn color="primary" class="q-mt-sm" style="color: var(--app-primary-text)">
              Primary Button Text (--app-primary-text)
            </q-btn>
          </div>
        </div>
      </div>
    </Variant>
  </Story>
</template>

<style scoped>
.preview-container {
  min-height: 400px;
  background: var(--app-bg);
  color: var(--app-text);
}
.border-radius {
  border-radius: 8px;
}
.border-std {
  border: 1px solid var(--app-border);
}
.bg-app-bg {
  background: var(--app-bg);
}
</style>
