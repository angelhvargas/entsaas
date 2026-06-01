<script setup>
defineProps({
  icon: { type: Object, required: true },
  size: { type: String, default: 'md' },    // sm | md | lg
  variant: { type: String, default: 'ghost' }, // ghost | primary | secondary | danger
  label: { type: String, default: '' },      // aria-label
  disabled: { type: Boolean, default: false },
  type: { type: String, default: 'button' },
  loading: { type: Boolean, default: false },
})
</script>

<template>
  <button
    :type="type"
    class="ui-icon-btn"
    :class="[`ui-icon-btn--${size}`, `ui-icon-btn--${variant}`, { 'ui-icon-btn--loading': loading }]"
    :disabled="disabled || loading"
    :aria-label="label"
    :title="label"
  >
    <span v-if="loading" class="ui-icon-btn__spinner" />
    <component v-else :is="icon" :size="size === 'sm' ? 14 : size === 'lg' ? 20 : 16" />
  </button>
</template>

<style scoped>
.ui-icon-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  border: none;
  border-radius: var(--radius-md);
  cursor: pointer;
  transition: all var(--transition-fast);
  flex-shrink: 0;
}
.ui-icon-btn:disabled { opacity: 0.4; cursor: not-allowed; }

.ui-icon-btn--sm { width: 28px; height: 28px; }
.ui-icon-btn--md { width: 34px; height: 34px; }
.ui-icon-btn--lg { width: 40px; height: 40px; }

.ui-icon-btn--ghost { background: transparent; color: var(--color-text-secondary); }
.ui-icon-btn--ghost:hover:not(:disabled) { background: var(--color-bg-muted); color: var(--color-text); }

.ui-icon-btn--primary { background: var(--color-primary); color: var(--color-text-on-primary); }
.ui-icon-btn--primary:hover:not(:disabled) { background: var(--color-primary-hover); box-shadow: var(--shadow-glow); }

.ui-icon-btn--secondary { background: var(--color-bg-muted); color: var(--color-text); border: 1px solid var(--color-border); }
.ui-icon-btn--secondary:hover:not(:disabled) { border-color: var(--color-primary); color: var(--color-primary); }

.ui-icon-btn--danger { background: transparent; color: var(--color-text-secondary); }
.ui-icon-btn--danger:hover:not(:disabled) { background: oklch(0.95 0.05 25); color: var(--color-danger); }
@media (prefers-color-scheme: dark) {
  .ui-icon-btn--danger:hover:not(:disabled) { background: oklch(0.22 0.05 25); }
}

.ui-icon-btn__spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
