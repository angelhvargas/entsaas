<script setup>
defineProps({
  padding: { type: String, default: 'md' }, // none | sm | md | lg
  hover: { type: Boolean, default: false },
  border: { type: Boolean, default: true },
})
</script>

<template>
  <div class="ui-card" :class="[`ui-card--pad-${padding}`, { 'ui-card--hover': hover, 'ui-card--no-border': !border }]">
    <div v-if="$slots.header" class="ui-card__header">
      <slot name="header" />
    </div>
    <div class="ui-card__body">
      <slot />
    </div>
    <div v-if="$slots.footer" class="ui-card__footer">
      <slot name="footer" />
    </div>
  </div>
</template>

<style scoped>
.ui-card {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border-subtle);
  border-radius: var(--radius-lg);
  box-shadow: var(--shadow-sm);
  transition: box-shadow var(--transition-normal), border-color var(--transition-fast);
}
.ui-card--no-border { border-color: transparent; }
.ui-card--hover:hover { box-shadow: var(--shadow-md); border-color: var(--color-border); }

.ui-card--pad-none .ui-card__body { padding: 0; }
.ui-card--pad-sm .ui-card__body { padding: 12px; }
.ui-card--pad-md .ui-card__body { padding: 20px; }
.ui-card--pad-lg .ui-card__body { padding: 28px; }

.ui-card__header {
  padding: 16px 20px;
  border-bottom: 1px solid var(--color-border-subtle);
  font-weight: 600;
  font-size: 0.9375rem;
}
.ui-card__footer {
  padding: 16px 20px;
  border-top: 1px solid var(--color-border-subtle);
  display: flex;
  align-items: center;
  justify-content: flex-end;
  gap: 8px;
}
</style>
