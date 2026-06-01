<script setup>
import { watch } from 'vue'
import { useAuthStore } from './stores/auth'
import { usePrefsStore } from './stores/prefs'
import AiChatWidget from './components/ai/AiChatWidget.vue'

const auth = useAuthStore()
const prefs = usePrefsStore()

// Fetch prefs once on login
watch(
    () => auth.isAuthenticated,
    (isAuth) => {
        if (isAuth) prefs.fetchPrefs().catch(() => {})
        else prefs.$reset()
    },
    { immediate: true },
)
</script>

<template>
    <router-view />
    <AiChatWidget v-if="auth.isAuthenticated" />
</template>

<style>
@import './style.css';
</style>
