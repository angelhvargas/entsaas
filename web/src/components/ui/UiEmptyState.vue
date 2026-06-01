<script setup>
defineProps({
  icon: { type: Object, default: null },
  title: { type: String, default: 'Nothing here' },
  description: { type: String, default: '' },
  actionLabel: { type: String, default: '' },
  actionVariant: { type: String, default: 'primary' },
})
const emit = defineEmits(['action'])
</script>

<template>
  <div class="ui-empty">
    <div v-if="icon || $slots.icon" class="ui-empty__icon">
      <slot name="icon">
        <component v-if="icon" :is="icon" :size="40" />
      </slot>
    </div>
    <h3 class="ui-empty__title">{{ title }}</h3>
    <p v-if="description" class="ui-empty__desc">{{ description }}</p>
    <slot>
      <button v-if="actionLabel" class="btn" :class="`btn-${actionVariant}`" @click="emit('action')">
        {{ actionLabel }}
      </button>
    </slot>
  </div>
</template>

<style scoped>
.ui-empty {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 3rem 2rem;
  text-align: center;
  gap: 0.5rem;
}
.ui-empty__icon {
  color: var(--color-text-tertiary);
  margin-bottom: 0.75rem;
}
.ui-empty__title {
  font-size: 1rem;
  font-weight: 600;
  color: var(--color-text);
}
.ui-empty__desc {
  font-size: 0.875rem;
  color: var(--color-text-secondary);
  max-width: 36ch;
  line-height: 1.5;
}
</style>
