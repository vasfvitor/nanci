<template>
  <Story title="Design System/Typography">
    <Variant title="Light Theme">
      <div class="body--light q-pa-lg preview-bg">
        <div class="text-h1">Heading 1</div>
        <div class="text-h2">Heading 2</div>
        <div class="text-h3">Heading 3</div>
        <div class="text-h4">Heading 4</div>
        <div class="text-h5">Heading 5</div>
        <div class="text-h6">Heading 6</div>
        <div class="q-my-md"></div>
        <div class="text-subtitle1">Subtitle 1</div>
        <div class="text-subtitle2">Subtitle 2</div>
        <div class="text-body1">Body 1 (default)</div>
        <div class="text-body2">Body 2</div>
        <div class="text-caption">Caption text</div>
        <div class="text-overline">Overline</div>
      </div>
    </Variant>

    <Variant title="Dark Theme">
      <div class="body--dark q-pa-lg preview-bg">
        <div class="text-h1">Heading 1</div>
        <div class="text-h2">Heading 2</div>
        <div class="text-h3">Heading 3</div>
        <div class="text-h4">Heading 4</div>
        <div class="text-h5">Heading 5</div>
        <div class="text-h6">Heading 6</div>
        <div class="q-my-md"></div>
        <div class="text-subtitle1">Subtitle 1</div>
        <div class="text-subtitle2">Subtitle 2</div>
        <div class="text-body1">Body 1 (default)</div>
        <div class="text-body2">Body 2</div>
        <div class="text-caption">Caption text</div>
        <div class="text-overline">Overline</div>
      </div>
    </Variant>
  </Story>
</template>

<style scoped>
.preview-bg {
  background: var(--app-bg);
  color: var(--app-text);
}
</style>
