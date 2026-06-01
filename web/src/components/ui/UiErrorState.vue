<script setup>
import { AlertCircle } from 'lucide-vue-next'
defineProps({
  title: { type: String, default: 'Something went wrong' },
  message: { type: String, default: 'An unexpected error occurred. Please try again.' },
  retryLabel: { type: String, default: 'Retry' },
})
const emit = defineEmits(['retry'])
</script>

<template>
  <div class="ui-error">
    <div class="ui-error__icon">
      <AlertCircle :size="40" />
    </div>
    <h3 class="ui-error__title">{{ title }}</h3>
    <p class="ui-error__msg">{{ message }}</p>
    <slot>
      <button class="btn btn-secondary" @click="emit('retry')">{{ retryLabel }}</button>
    </slot>
  </div>
</template>

<style scoped>
.ui-error {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 2rem;
  text-align: center;
  gap: 0.5rem;
}
.ui-error__icon { color: var(--color-danger); margin-bottom: 0.75rem; }
.ui-error__title { font-size: 1rem; font-weight: 600; }
.ui-error__msg {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  max-width: 36ch;
  line-height: 1.5;
  margin-bottom: 0.75rem;
}
</style>
