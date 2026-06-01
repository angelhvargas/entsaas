<script setup>
import { computed } from 'vue'
const props = defineProps({
  modelValue: { default: '' },
  type: { type: String, default: 'text' },
  placeholder: { type: String, default: '' },
  disabled: { type: Boolean, default: false },
  error: { type: String, default: '' },
  id: { type: String, default: '' },
  autocomplete: { type: String, default: '' },
  required: { type: Boolean, default: false },
  icon: { type: Object, default: null },    // Lucide component
  iconRight: { type: Object, default: null },
})
const emit = defineEmits(['update:modelValue'])
const hasError = computed(() => !!props.error)
</script>

<template>
  <div class="ui-input__wrap" :class="{ 'ui-input__wrap--error': hasError, 'ui-input__wrap--icon-left': icon, 'ui-input__wrap--icon-right': iconRight }">
    <component v-if="icon" :is="icon" :size="16" class="ui-input__icon ui-input__icon--left" />
    <input
      class="ui-input"
      :class="{ 'ui-input--error': hasError, 'ui-input--padded-left': icon, 'ui-input--padded-right': iconRight }"
      :type="type"
      :value="modelValue"
      :placeholder="placeholder"
      :disabled="disabled"
      :id="id"
      :autocomplete="autocomplete"
      :required="required"
      @input="emit('update:modelValue', $event.target.value)"
    />
    <component v-if="iconRight" :is="iconRight" :size="16" class="ui-input__icon ui-input__icon--right" />
    <p v-if="error" class="ui-input__error" role="alert">{{ error }}</p>
  </div>
</template>

<style scoped>
.ui-input__wrap { position: relative; width: 100%; }

.ui-input {
  width: 100%;
  padding: 10px 14px;
  background: var(--color-bg);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  font-family: var(--font-sans);
  font-size: 0.875rem;
  color: var(--color-text);
  transition: border-color var(--transition-fast), box-shadow var(--transition-fast);
  outline: none;
}
.ui-input:focus {
  border-color: var(--color-primary);
  box-shadow: 0 0 0 3px oklch(0.65 0.24 265 / 0.12);
}
.ui-input::placeholder { color: var(--color-text-tertiary); }
.ui-input:disabled { opacity: 0.5; cursor: not-allowed; }
.ui-input--error { border-color: var(--color-danger); }
.ui-input--error:focus { box-shadow: 0 0 0 3px oklch(0.63 0.22 25 / 0.12); }
.ui-input--padded-left { padding-left: 38px; }
.ui-input--padded-right { padding-right: 38px; }

.ui-input__icon {
  position: absolute;
  top: 50%;
  transform: translateY(-50%);
  color: var(--color-text-tertiary);
  pointer-events: none;
}
.ui-input__icon--left { left: 12px; }
.ui-input__icon--right { right: 12px; }

.ui-input__error {
  margin-top: 5px;
  font-size: 0.75rem;
  color: var(--color-danger);
}
</style>
