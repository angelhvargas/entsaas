import { defineStore } from 'pinia'
import { ref } from 'vue'
import client from '@/api/client'

export const useConfigStore = defineStore('config', () => {
    const registrationEnabled = ref(true)
    const emailVerificationEnabled = ref(false)
    const aiEnabled = ref(false)
    const loaded = ref(false)

    async function load() {
        try {
            const { data } = await client.get('/config')
            registrationEnabled.value = data.registration_enabled ?? true
            emailVerificationEnabled.value = data.email_verification_enabled ?? false
            aiEnabled.value = data.ai_enabled ?? false
            loaded.value = true
        } catch {
            loaded.value = true
        }
    }

    return { registrationEnabled, emailVerificationEnabled, aiEnabled, loaded, load }
})
