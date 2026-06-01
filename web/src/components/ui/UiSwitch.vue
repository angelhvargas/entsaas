<script setup>
import { computed } from 'vue'

const props = defineProps({
  modelValue: { type: Boolean, default: false },
  disabled: { type: Boolean, default: false },
  label: { type: String, default: '' },
  size: { type: String, default: 'md' }, // sm | md
  id: { type: String, default: () => `switch-${Math.random().toString(36).slice(2)}` },
})
const emit = defineEmits(['update:modelValue'])
const toggle = () => { if (!props.disabled) emit('update:modelValue', !props.modelValue) }
</script>

<template>
  <label class="ui-switch" :class="[`ui-switch--${size}`, { 'ui-switch--disabled': disabled }]" :for="id">
    <button
      type="button"
      role="switch"
      :id="id"
      :aria-checked="modelValue"
      class="ui-switch__track"
      :class="{ 'ui-switch__track--on': modelValue }"
      :disabled="disabled"
      @click="toggle"
    >
      <span class="ui-switch__thumb" />
    </button>
    <span v-if="label" class="ui-switch__label">{{ label }}</span>
  </label>
</template>

<style scoped>
.ui-switch {
  display: inline-flex;
  align-items: center;
  gap: 10px;
  cursor: pointer;
  user-select: none;
}
.ui-switch--disabled { opacity: 0.5; cursor: not-allowed; }

/* Track */
.ui-switch__track {
  position: relative;
  border: none;
  border-radius: 999px;
  background: var(--color-border);
  cursor: pointer;
  transition: background var(--transition-fast);
  flex-shrink: 0;
  padding: 0;
}
.ui-switch--sm .ui-switch__track { width: 32px; height: 18px; }
.ui-switch--md .ui-switch__track { width: 40px; height: 22px; }

.ui-switch__track--on { background: var(--color-primary); }

/* Thumb */
.ui-switch__thumb {
  position: absolute;
  top: 2px;
  left: 2px;
  border-radius: 50%;
  background: white;
  box-shadow: var(--shadow-sm);
  transition: transform var(--transition-fast);
}
.ui-switch--sm .ui-switch__thumb { width: 14px; height: 14px; }
.ui-switch--md .ui-switch__thumb { width: 18px; height: 18px; }

.ui-switch--sm .ui-switch__track--on .ui-switch__thumb { transform: translateX(14px); }
.ui-switch--md .ui-switch__track--on .ui-switch__thumb { transform: translateX(18px); }

.ui-switch__label { font-size: 0.875rem; font-weight: 500; color: var(--color-text); }
</style>
