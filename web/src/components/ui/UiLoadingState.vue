<script setup>
defineProps({
  lines: { type: Number, default: 3 },  // skeleton line count
  size: { type: String, default: 'md' }, // sm | md | lg (spinner size)
  mode: { type: String, default: 'skeleton' }, // skeleton | spinner | pulse
  label: { type: String, default: 'Loading…' },
})
</script>

<template>
  <!-- Skeleton mode -->
  <div v-if="mode === 'skeleton'" class="ui-loading__skeleton" :aria-label="label" aria-busy="true">
    <div v-for="i in lines" :key="i" class="ui-loading__bone" :style="{ width: i === lines ? '60%' : '100%' }" />
  </div>

  <!-- Spinner mode -->
  <div v-else-if="mode === 'spinner'" class="ui-loading__spinner-wrap" :aria-label="label" aria-busy="true">
    <div class="ui-loading__spinner" :class="`ui-loading__spinner--${size}`" />
    <span v-if="label !== 'Loading…'" class="ui-loading__label">{{ label }}</span>
  </div>

  <!-- Pulse mode (full-area overlay) -->
  <div v-else class="ui-loading__pulse" :aria-label="label" aria-busy="true">
    <div class="ui-loading__spinner ui-loading__spinner--lg" />
  </div>
</template>

<style scoped>
/* Skeleton */
.ui-loading__skeleton { display: flex; flex-direction: column; gap: 10px; }
.ui-loading__bone {
  height: 14px;
  border-radius: var(--radius-sm);
  background: linear-gradient(90deg, var(--color-bg-muted) 25%, var(--color-bg) 50%, var(--color-bg-muted) 75%);
  background-size: 200% 100%;
  animation: shimmer 1.5s infinite;
}
@keyframes shimmer { 0% { background-position: 200% 0; } 100% { background-position: -200% 0; } }

/* Spinner */
.ui-loading__spinner-wrap { display: flex; flex-direction: column; align-items: center; gap: 12px; }
.ui-loading__spinner {
  border-radius: 50%;
  border: 2px solid var(--color-border);
  border-top-color: var(--color-primary);
  animation: spin 0.65s linear infinite;
}
.ui-loading__spinner--sm { width: 18px; height: 18px; }
.ui-loading__spinner--md { width: 28px; height: 28px; }
.ui-loading__spinner--lg { width: 40px; height: 40px; border-width: 3px; }
@keyframes spin { to { transform: rotate(360deg); } }

.ui-loading__label { font-size: 0.875rem; color: var(--color-text-secondary); }

/* Pulse */
.ui-loading__pulse {
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 3rem;
}
</style>
