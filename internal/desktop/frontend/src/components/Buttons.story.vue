<script setup lang="ts">
import { ref } from 'vue'

const loading = ref(false)
</script>

<template>
  <Story title="Design System/Buttons">
    <Variant title="Light Theme">
      <div class="body--light q-pa-lg preview-bg">
        <div class="row q-gutter-sm q-mb-md">
          <q-btn color="primary" label="Primary" />
          <q-btn color="secondary" label="Secondary" />
          <q-btn color="positive" label="Positive" />
          <q-btn color="negative" label="Negative" />
          <q-btn color="info" label="Info" />
          <q-btn color="warning" label="Warning" />
          <q-btn color="dark" label="Dark" />
        </div>

        <div class="row q-gutter-sm q-mb-md">
          <q-btn outline color="primary" label="Outline" />
          <q-btn flat color="primary" label="Flat" />
          <q-btn push color="primary" label="Push" />
          <q-btn glossy color="primary" label="Glossy" />
          <q-btn rounded color="primary" label="Rounded" />
        </div>

        <div class="row q-gutter-sm">
          <q-btn disable color="primary" label="Disabled" />
          <q-btn :loading="loading" color="primary" @click="loading = true" label="Loading" />
          <q-btn round color="primary" icon="shopping_cart" />
        </div>
      </div>
    </Variant>

    <Variant title="Dark Theme">
      <div class="body--dark q-pa-lg preview-bg">
        <div class="row q-gutter-sm q-mb-md">
          <q-btn color="primary" label="Primary" style="color: var(--app-primary-text)" />
          <q-btn color="secondary" label="Secondary" />
          <q-btn color="positive" label="Positive" />
          <q-btn color="negative" label="Negative" />
          <q-btn color="info" label="Info" />
          <q-btn color="warning" label="Warning" />
          <q-btn color="dark" label="Dark" />
        </div>

        <div class="row q-gutter-sm q-mb-md">
          <q-btn outline color="primary" label="Outline" />
          <q-btn flat color="primary" label="Flat" />
          <q-btn push color="primary" label="Push" />
          <q-btn glossy color="primary" label="Glossy" />
          <q-btn rounded color="primary" label="Rounded" />
        </div>

        <div class="row q-gutter-sm">
          <q-btn disable color="primary" label="Disabled" />
          <q-btn :loading="loading" color="primary" @click="loading = true" label="Loading" />
          <q-btn round color="primary" icon="shopping_cart" style="color: var(--app-primary-text)" />
        </div>
      </div>
    </Variant>
  </Story>
</template>

<style scoped>
.preview-bg {
  background: var(--app-bg);
  color: var(--app-text);
  min-height: 200px;
}
</style>
