import { defineStore } from 'pinia'
import { ref } from 'vue'
import client from '@/api/client'

export const usePrefsStore = defineStore('prefs', () => {
    const preferences = ref({})

    async function fetchPrefs() {
        const { data } = await client.get('/preferences')
        preferences.value = data.preferences || {}
    }

    async function savePrefs(prefs) {
        await client.put('/preferences', prefs)
        preferences.value = { ...preferences.value, ...prefs }
    }

    return { preferences, fetchPrefs, savePrefs }
})
