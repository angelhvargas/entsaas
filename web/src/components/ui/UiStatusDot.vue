<script setup>
defineProps({
  color: { type: String, default: 'default' }, // default | success | warning | danger | primary | accent
  size: { type: String, default: 'sm' },        // sm | md
  pulse: { type: Boolean, default: false },
})
</script>

<template>
  <span class="ui-dot" :class="[`ui-dot--${color}`, `ui-dot--${size}`, { 'ui-dot--pulse': pulse }]" />
</template>

<style scoped>
.ui-dot {
  display: inline-block;
  border-radius: 50%;
  flex-shrink: 0;
}
.ui-dot--sm { width: 8px; height: 8px; }
.ui-dot--md { width: 10px; height: 10px; }

.ui-dot--default  { background: var(--color-text-tertiary); }
.ui-dot--primary  { background: var(--color-primary); }
.ui-dot--success  { background: var(--color-success); }
.ui-dot--warning  { background: var(--color-warning); }
.ui-dot--danger   { background: var(--color-danger); }
.ui-dot--accent   { background: var(--color-accent); }

.ui-dot--pulse {
  animation: dot-pulse 1.5s ease-in-out infinite;
}
@keyframes dot-pulse {
  0%, 100% { opacity: 1; transform: scale(1); }
  50% { opacity: 0.5; transform: scale(0.85); }
}
</style>
