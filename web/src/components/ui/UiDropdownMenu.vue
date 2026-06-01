<script setup>
import { ref, onMounted, onBeforeUnmount } from 'vue'

defineProps({
  items: { type: Array, default: () => [] }, // [{ label, icon?, variant?, divider? }]
  align: { type: String, default: 'left' },  // left | right
})
const emit = defineEmits(['select'])

const open = ref(false)
const root = ref(null)

function select(item) {
  if (!item.divider) {
    emit('select', item)
    open.value = false
  }
}

function onOutside(e) { if (root.value && !root.value.contains(e.target)) open.value = false }
onMounted(() => document.addEventListener('mousedown', onOutside))
onBeforeUnmount(() => document.removeEventListener('mousedown', onOutside))
</script>

<template>
  <div class="ui-dropdown" ref="root">
    <div @click="open = !open">
      <slot name="trigger" :open="open" />
    </div>

    <Transition name="ui-dropdown-pop">
      <div v-if="open" class="ui-dropdown__menu" :class="`ui-dropdown__menu--${align}`">
        <template v-for="(item, i) in items" :key="i">
          <div v-if="item.divider" class="ui-dropdown__divider" />
          <button
            v-else
            class="ui-dropdown__item"
            :class="item.variant ? `ui-dropdown__item--${item.variant}` : ''"
            @click="select(item)"
          >
            <component v-if="item.icon" :is="item.icon" :size="15" />
            {{ item.label }}
          </button>
        </template>
      </div>
    </Transition>
  </div>
</template>

<style scoped>
.ui-dropdown { position: relative; display: inline-block; }

.ui-dropdown__menu {
  position: absolute;
  top: calc(100% + 4px);
  z-index: 200;
  min-width: 180px;
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-md);
  box-shadow: var(--shadow-lg);
  padding: 4px;
}
.ui-dropdown__menu--left { left: 0; }
.ui-dropdown__menu--right { right: 0; }

.ui-dropdown__item {
  display: flex;
  align-items: center;
  gap: 8px;
  width: 100%;
  padding: 8px 12px;
  background: none;
  border: none;
  border-radius: var(--radius-sm);
  font-family: var(--font-sans);
  font-size: 0.875rem;
  color: var(--color-text);
  cursor: pointer;
  transition: background var(--transition-fast);
  text-align: left;
}
.ui-dropdown__item:hover { background: var(--color-bg-muted); }
.ui-dropdown__item--danger { color: var(--color-danger); }
.ui-dropdown__item--danger:hover { background: oklch(0.95 0.05 25); }
@media (prefers-color-scheme: dark) {
  .ui-dropdown__item--danger:hover { background: oklch(0.22 0.05 25); }
}

.ui-dropdown__divider {
  height: 1px;
  background: var(--color-border-subtle);
  margin: 4px 0;
}

/* Transition */
.ui-dropdown-pop-enter-active { animation: popIn 0.12s ease-out; }
.ui-dropdown-pop-leave-active { animation: popIn 0.1s ease-in reverse; }
@keyframes popIn {
  from { opacity: 0; transform: scale(0.95) translateY(-4px); }
  to   { opacity: 1; transform: scale(1) translateY(0); }
}
</style>
