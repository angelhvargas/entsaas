<script setup>
import { ref } from 'vue'
import { AlertTriangle } from 'lucide-vue-next'

const props = defineProps({
  title: { type: String, default: 'Are you sure?' },
  message: { type: String, default: 'This action cannot be undone.' },
  confirmLabel: { type: String, default: 'Confirm' },
  cancelLabel: { type: String, default: 'Cancel' },
  variant: { type: String, default: 'danger' }, // danger | primary
  open: { type: Boolean, default: false },
})

const emit = defineEmits(['confirm', 'cancel', 'update:open'])

function confirm() { emit('confirm'); emit('update:open', false) }
function cancel()  { emit('cancel');  emit('update:open', false) }
</script>

<template>
  <Teleport to="body">
    <Transition name="ui-dialog">
      <div v-if="open" class="ui-dialog__backdrop" @click.self="cancel" role="dialog" aria-modal="true" :aria-label="title">
        <div class="ui-dialog__panel animate-fadeIn">
          <div class="ui-dialog__icon" :class="`ui-dialog__icon--${variant}`">
            <AlertTriangle :size="24" />
          </div>
          <h2 class="ui-dialog__title">{{ title }}</h2>
          <p class="ui-dialog__msg">{{ message }}</p>
          <slot />
          <div class="ui-dialog__actions">
            <button class="btn btn-secondary" @click="cancel">{{ cancelLabel }}</button>
            <button class="btn" :class="`btn-${variant}`" @click="confirm">{{ confirmLabel }}</button>
          </div>
        </div>
      </div>
    </Transition>
  </Teleport>
</template>

<style scoped>
.ui-dialog__backdrop {
  position: fixed;
  inset: 0;
  background: oklch(0 0 0 / 0.45);
  backdrop-filter: blur(4px);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 1rem;
}
.ui-dialog__panel {
  background: var(--color-bg-elevated);
  border: 1px solid var(--color-border);
  border-radius: var(--radius-xl);
  box-shadow: var(--shadow-lg);
  padding: 2rem;
  width: 100%;
  max-width: 420px;
  text-align: center;
}
.ui-dialog__icon {
  display: inline-flex;
  align-items: center;
  justify-content: center;
  width: 52px;
  height: 52px;
  border-radius: var(--radius-lg);
  margin-bottom: 1rem;
}
.ui-dialog__icon--danger { background: oklch(0.95 0.05 25); color: var(--color-danger); }
.ui-dialog__icon--primary { background: var(--color-primary-subtle); color: var(--color-primary); }
@media (prefers-color-scheme: dark) {
  .ui-dialog__icon--danger { background: oklch(0.22 0.05 25); }
}
.ui-dialog__title { font-size: 1.125rem; font-weight: 700; margin-bottom: 0.5rem; }
.ui-dialog__msg { font-size: 0.875rem; color: var(--color-text-secondary); line-height: 1.5; margin-bottom: 1.5rem; }
.ui-dialog__actions { display: flex; gap: 8px; justify-content: center; }
.ui-dialog__actions .btn { flex: 1; max-width: 160px; }

/* Transition */
.ui-dialog-enter-active, .ui-dialog-leave-active { transition: opacity 0.15s ease; }
.ui-dialog-enter-from, .ui-dialog-leave-to { opacity: 0; }
</style>
