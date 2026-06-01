import { createApp } from 'vue'
import { createPinia } from 'pinia'
import App from './App.vue'
import router from './router'
import './style.css'
import { useConfigStore } from './stores/config'

const app = createApp(App)

const pinia = createPinia()
app.use(pinia)
app.use(router)

// ── Global Error Boundary ──────────────────────────────────────────────────────
app.config.errorHandler = (err, instance, info) => {
    console.error('[Vue Error Boundary]', err, info, instance)
    if (window.NProgress) window.NProgress.done()
}

// ── v-click-outside directive ─────────────────────────────────────────────────
app.directive('click-outside', {
    mounted(el, binding) {
        el.__clickOutsideHandler = (e) => {
            if (!el.contains(e.target)) binding.value(e)
        }
        document.addEventListener('mousedown', el.__clickOutsideHandler)
    },
    unmounted(el) {
        document.removeEventListener('mousedown', el.__clickOutsideHandler)
    },
})

// Fetch deployment feature flags before mounting so the router guard and all
// components see the correct values on first render.
const configStore = useConfigStore()
configStore.load().finally(() => app.mount('#app'))
