<script setup>
defineProps({
  icon: { type: Object, required: true },
  size: { type: Number, default: 20 },
  color: { type: String, default: '' }, // CSS color, empty = inherit
})
</script>

<template>
  <component :is="icon" :size="size" :style="color ? { color } : {}" />
</template>
