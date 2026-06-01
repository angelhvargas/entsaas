<script setup>
defineProps({
  modelValue: { default: '' },
  options: { type: Array, default: () => [] }, // [{ value, label }] or ['string']
  placeholder: { type: String, default: 'Select…' },
  disabled: { type: Boolean, default: false },
  error: { type: String, default: '' },
  id: { type: String, default: '' },
})
const emit = defineEmits(['update:modelValue'])

function label(opt) { return typeof opt === 'object' ? opt.label : opt }
function value(opt) { return typeof opt === 'object' ? opt.value : opt }
</script>

<template>
  <div class="ui-select__wrap">
    <select
      class="ui-select"
      :class="{ 'ui-select--error': error }"
      :value="modelValue"
      :disabled="disabled"
      :id="id"
      @change="emit('update:modelValue', $event.target.value)"
    >
      <option value="" disabled>{{ placeholder }}</option>
      <option v-for="opt in options" :key="value(opt)" :value="value(opt)">{{ label(opt) }}</option>
    </select>
    <svg class="ui-select__chevron" viewBox="0 0 20 20" fill="currentColor" aria-hidden="true">
      <path fill-rule="evenodd" d="M5.22 8.22a.75.75 0 0 1 1.06 0L10 11.94l3.72-3.72a.75.75 0 1 1 1.06 1.06l-4.25 4.25a.75.75 0 0 1-1.06 0L5.22 9.28a.75.75 0 0 1 0-1.06Z" clip-rule="evenodd" />
    </svg>
    <p v-if="error" class="ui-select__error" role="alert">{{ error }}</p>
  </div>
</template>

<style scoped>
.ui-select__wrap { position: relative; width: 100%; }
.ui-select {
  width: 100%;
  padding: 10px 36px 10px 14px;
  appearance: none;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-family: var(--font-sans);
  font-size: 0.875rem;
  color: var(--color-text);
  cursor: pointer;
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
  outline: none;
}
.ui-select:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px oklch(0.65 0.24 265 / 0.12);
}
.ui-select:disabled { opacity: 0.5; cursor: not-allowed; }
.ui-select--error { border-color: var(--color-danger); }

.ui-select__chevron {
  position: absolute;
  right: 12px;
  top: 50%;
  transform: translateY(-50%);
  width: 16px;
  height: 16px;
  color: var(--color-text-tertiary);
  pointer-events: none;
}
.ui-select__error {
  margin-top: 5px;
  font-size: 0.75rem;
  color: var(--color-danger);
}
</style>
