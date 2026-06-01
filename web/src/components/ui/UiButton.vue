<script setup>
defineProps({
  variant: { type: String, default: 'primary' }, // primary | secondary | danger | ghost | outline
  size: { type: String, default: 'md' },          // sm | md | lg
  loading: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  full: { type: Boolean, default: false },
  type: { type: String, default: 'button' },
})
</script>

<template>
  <button
    :type="type"
    class="ui-btn"
    :class="[`ui-btn--${variant}`, `ui-btn--${size}`, { 'ui-btn--full': full, 'ui-btn--loading': loading }]"
    :disabled="disabled || loading"
  >
    <span v-if="loading" class="ui-btn__spinner" aria-hidden="true" />
    <slot />
  </button>
</template>

<style scoped>
.ui-btn {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  gap: 8px;
  border: none;
  border-radius: var(--radius-md);
  font-family: var(--font-sans);
  font-weight: 500;
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
  position: relative;
  text-decoration: none;
}
.ui-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.ui-btn--full { width: 100%; }

/* Sizes */
.ui-btn--sm { padding: 6px 12px; font-size: 0.8125rem; }
.ui-btn--md { padding: 10px 20px; font-size: 0.875rem; }
.ui-btn--lg { padding: 13px 28px; font-size: 0.9375rem; }

/* Variants */
.ui-btn--primary {
  background: var(--color-primary);
  color: var(--color-text-on-primary);
}
.ui-btn--primary:hover:not(:disabled) {
  background: var(--color-primary-hover);
  box-shadow: var(--shadow-glow);
  transform: translateY(-1px);
}
.ui-btn--secondary {
  background: transparent;
  color: var(--color-text);
  border: 1px solid var(--color-border);
}
.ui-btn--secondary:hover:not(:disabled) {
  background: var(--color-bg-muted);
  border-color: var(--color-primary);
  color: var(--color-primary);
}
.ui-btn--danger {
  background: var(--color-danger);
  color: var(--color-text-on-primary);
}
.ui-btn--danger:hover:not(:disabled) {
  opacity: 0.9;
  transform: translateY(-1px);
}
.ui-btn--ghost {
  background: transparent;
  color: var(--color-text-secondary);
}
.ui-btn--ghost:hover:not(:disabled) {
  background: var(--color-bg-muted);
  color: var(--color-text);
}
.ui-btn--outline {
  background: transparent;
  color: var(--color-primary);
  border: 1px solid var(--color-primary);
}
.ui-btn--outline:hover:not(:disabled) {
  background: var(--color-primary-subtle);
}

/* Loading spinner */
.ui-btn__spinner {
  width: 14px;
  height: 14px;
  border: 2px solid currentColor;
  border-top-color: transparent;
  border-radius: 50%;
  animation: spin 0.6s linear infinite;
  flex-shrink: 0;
}
@keyframes spin { to { transform: rotate(360deg); } }
</style>
