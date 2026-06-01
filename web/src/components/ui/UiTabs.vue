<script setup>
import { ref, computed, provide, watch } from 'vue'

const props = defineProps({
  modelValue: { type: String, default: '' },
  tabs: { type: Array, default: () => [] }, // [{ key, label, icon? }]
})
const emit = defineEmits(['update:modelValue'])

const active = ref(props.modelValue || props.tabs[0]?.key || '')
watch(() => props.modelValue, v => { if (v) active.value = v })

function select(key) {
  active.value = key
  emit('update:modelValue', key)
}

provide('uiTabsActive', active)
</script>

<template>
  <div class="ui-tabs">
    <div class="ui-tabs__nav" role="tablist">
      <button
        v-for="tab in tabs"
        :key="tab.key"
        class="ui-tabs__tab"
        :class="{ 'ui-tabs__tab--active': active === tab.key }"
        role="tab"
        :aria-selected="active === tab.key"
        @click="select(tab.key)"
      >
        <component v-if="tab.icon" :is="tab.icon" :size="15" />
        {{ tab.label }}
      </button>
    </div>
    <div class="ui-tabs__content">
      <slot :active="active" />
    </div>
  </div>
</template>

<style scoped>
.ui-tabs__nav {
  display: flex;
  gap: 2px;
  border-bottom: 1px solid var(--color-border-subtle);
  margin-bottom: 24px;
  padding-bottom: 0;
  overflow-x: auto;
  scrollbar-width: none;
}
.ui-tabs__nav::-webkit-scrollbar { display: none; }

.ui-tabs__tab {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 10px 16px;
  background: none;
  border: none;
  border-bottom: 2px solid transparent;
  font-family: var(--font-sans);
  font-size: 0.875rem;
  font-weight: 500;
  color: var(--color-text-secondary);
  cursor: pointer;
  transition: all var(--transition-fast);
  white-space: nowrap;
  margin-bottom: -1px;
}
.ui-tabs__tab:hover { color: var(--color-text); }
.ui-tabs__tab--active {
  color: var(--color-primary);
  border-bottom-color: var(--color-primary);
}
</style>
