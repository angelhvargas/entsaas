<script setup>
defineProps({
  label: { type: String, default: '' },
  error: { type: String, default: '' },
  hint: { type: String, default: '' },
  required: { type: Boolean, default: false },
  htmlFor: { type: String, default: '' },
})
</script>

<template>
  <div class="ui-field" :class="{ 'ui-field--error': error }">
    <label v-if="label" class="ui-field__label" :for="htmlFor">
      {{ label }}
      <span v-if="required" class="ui-field__required" aria-label="required">*</span>
    </label>
    <slot />
    <p v-if="hint && !error" class="ui-field__hint">{{ hint }}</p>
    <p v-if="error" class="ui-field__error" role="alert">{{ error }}</p>
  </div>
</template>

<style scoped>
.ui-field { margin-bottom: 1.25rem; }
.ui-field__label {
  display: block;
  font-size: 0.8125rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  margin-bottom: 6px;
}
.ui-field__required { color: var(--color-danger); margin-left: 2px; }
.ui-field__hint { margin-top: 5px; font-size: 0.75rem; color: var(--color-text-tertiary); }
.ui-field__error { margin-top: 5px; font-size: 0.75rem; color: var(--color-danger); }
</style>
