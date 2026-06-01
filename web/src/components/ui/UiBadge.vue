<script setup>
defineProps({
  variant: { type: String, default: 'default' }, // default | primary | success | warning | danger | accent
  size: { type: String, default: 'sm' },          // sm | md
  dot: { type: Boolean, default: false },
})
</script>

<template>
  <span class="ui-badge" :class="[`ui-badge--${variant}`, `ui-badge--${size}`]">
    <span v-if="dot" class="ui-badge__dot" />
    <slot />
  </span>
</template>

<style scoped>
.ui-badge {
  display: inline-flex;
  align-items: center;
  gap: 5px;
  border-radius: 999px;
  font-weight: 500;
  white-space: nowrap;
}
.ui-badge--sm { padding: 2px 8px; font-size: 0.6875rem; }
.ui-badge--md { padding: 4px 12px; font-size: 0.8125rem; }

/* Color variants */
.ui-badge--default  { background: var(--color-bg-muted);           color: var(--color-text-secondary); }
.ui-badge--primary  { background: var(--color-primary-subtle);     color: var(--color-primary); }
.ui-badge--success  { background: oklch(0.92 0.06 155);            color: oklch(0.45 0.15 155); }
.ui-badge--warning  { background: oklch(0.92 0.06 70);             color: oklch(0.55 0.15 70); }
.ui-badge--danger   { background: oklch(0.95 0.05 25);             color: var(--color-danger); }
.ui-badge--accent   { background: oklch(0.92 0.06 160);            color: var(--color-accent); }

@media (prefers-color-scheme: dark) {
  .ui-badge--success { background: oklch(0.22 0.05 155); color: oklch(0.72 0.19 155); }
  .ui-badge--warning { background: oklch(0.22 0.05 70);  color: oklch(0.78 0.16 70); }
  .ui-badge--danger  { background: oklch(0.22 0.05 25);  color: oklch(0.72 0.22 25); }
  .ui-badge--accent  { background: oklch(0.22 0.05 160); color: oklch(0.72 0.19 160); }
}

.ui-badge__dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: currentColor;
  flex-shrink: 0;
}
</style>
