<script setup>
import { ref, nextTick, watch } from 'vue'
import { useAIStore } from '@/stores/ai'
import { useConfigStore } from '@/stores/config'
import { Bot, Send, X, Trash2, Sparkles, Loader2 } from 'lucide-vue-next'

const ai = useAIStore()
const config = useConfigStore()

const isOpen = ref(false)
const input = ref('')
const messagesEl = ref(null)

function toggle() { isOpen.value = !isOpen.value }

async function send() {
    const text = input.value.trim()
    if (!text || ai.streaming) return
    input.value = ''
    await ai.sendMessage(text)
    await nextTick()
    scrollToBottom()
}

function scrollToBottom() {
    if (messagesEl.value) {
        messagesEl.value.scrollTop = messagesEl.value.scrollHeight
    }
}

function handleKeydown(e) {
    if (e.key === 'Enter' && !e.shiftKey) {
        e.preventDefault()
        send()
    }
}

watch(() => ai.messages.length, async () => {
    await nextTick()
    scrollToBottom()
})
</script>

<template>
    <div v-if="config.aiEnabled" class="ai-chat-wrapper">
        <!-- Floating trigger -->
        <button v-if="!isOpen" class="ai-trigger" @click="toggle" title="AI Assistant">
            <Sparkles :size="22" />
        </button>

        <!-- Chat panel -->
        <transition name="chat-slide">
            <div v-if="isOpen" class="ai-panel">
                <!-- Header -->
                <div class="ai-header">
                    <div class="ai-header-title">
                        <Bot :size="18" />
                        <span>AI Assistant</span>
                    </div>
                    <div class="ai-header-actions">
                        <button class="ai-header-btn" @click="ai.clearMessages()" title="Clear chat">
                            <Trash2 :size="14" />
                        </button>
                        <button class="ai-header-btn" @click="toggle" title="Close">
                            <X :size="16" />
                        </button>
                    </div>
                </div>

                <!-- Messages -->
                <div ref="messagesEl" class="ai-messages">
                    <div v-if="ai.messages.length === 0" class="ai-empty">
                        <Sparkles :size="32" class="ai-empty-icon" />
                        <p class="ai-empty-title">Ask me anything</p>
                        <p class="ai-empty-sub">I'm your AI assistant. How can I help?</p>
                    </div>

                    <div
                        v-for="msg in ai.messages"
                        :key="msg.id"
                        class="ai-message"
                        :class="'ai-message--' + msg.role"
                    >
                        <div class="ai-message-avatar" v-if="msg.role === 'assistant'">
                            <Bot :size="14" />
                        </div>
                        <div class="ai-message-content">
                            <div class="ai-message-text">{{ msg.content || '...' }}</div>
                        </div>
                    </div>

                    <div v-if="ai.streaming" class="ai-typing">
                        <Loader2 :size="14" class="spin" />
                        <span>Thinking...</span>
                    </div>
                </div>

                <!-- Error -->
                <div v-if="ai.error" class="ai-error">{{ ai.error }}</div>

                <!-- Input -->
                <div class="ai-input-area">
                    <textarea
                        v-model="input"
                        class="ai-input"
                        placeholder="Type a message..."
                        rows="1"
                        @keydown="handleKeydown"
                        :disabled="ai.streaming"
                    ></textarea>
                    <button
                        class="ai-send"
                        @click="send"
                        :disabled="!input.trim() || ai.streaming"
                    >
                        <Send :size="16" />
                    </button>
                </div>
            </div>
        </transition>
    </div>
</template>

<style scoped>
.ai-chat-wrapper {
    position: fixed;
    bottom: 24px;
    right: 24px;
    z-index: 1000;
}

/* ── Trigger button ─────────────────────────────────────────────── */
.ai-trigger {
    width: 52px;
    height: 52px;
    border-radius: 50%;
    background: var(--color-primary);
    color: var(--color-text-on-primary);
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    box-shadow: var(--shadow-lg), var(--shadow-glow);
    transition: all var(--transition-fast);
    animation: pulse-glow 3s infinite;
}
.ai-trigger:hover {
    transform: scale(1.08);
}

/* ── Panel ──────────────────────────────────────────────────────── */
.ai-panel {
    width: 380px;
    height: 520px;
    background: var(--color-bg-elevated);
    border: 1px solid var(--color-border);
    border-radius: var(--radius-xl);
    box-shadow: var(--shadow-lg);
    display: flex;
    flex-direction: column;
    overflow: hidden;
}

.ai-header {
    display: flex;
    align-items: center;
    justify-content: space-between;
    padding: 14px 16px;
    border-bottom: 1px solid var(--color-border-subtle);
    background: var(--color-bg-muted);
}
.ai-header-title {
    display: flex;
    align-items: center;
    gap: 8px;
    font-weight: 600;
    font-size: 0.875rem;
}
.ai-header-actions {
    display: flex;
    gap: 4px;
}
.ai-header-btn {
    padding: 6px;
    border: none;
    background: none;
    color: var(--color-text-secondary);
    cursor: pointer;
    border-radius: var(--radius-sm);
    transition: all var(--transition-fast);
}
.ai-header-btn:hover {
    background: var(--color-bg);
    color: var(--color-text);
}

/* ── Messages ───────────────────────────────────────────────────── */
.ai-messages {
    flex: 1;
    overflow-y: auto;
    padding: 16px;
    display: flex;
    flex-direction: column;
    gap: 12px;
}

.ai-empty {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: center;
    justify-content: center;
    text-align: center;
}
.ai-empty-icon { color: var(--color-primary); margin-bottom: 12px; opacity: 0.6; }
.ai-empty-title { font-weight: 600; font-size: 0.9375rem; margin-bottom: 4px; }
.ai-empty-sub { font-size: 0.8125rem; color: var(--color-text-secondary); }

.ai-message {
    display: flex;
    gap: 8px;
    animation: fadeIn 0.2s ease-out;
}
.ai-message--user {
    flex-direction: row-reverse;
}
.ai-message-avatar {
    width: 28px;
    height: 28px;
    border-radius: 50%;
    background: var(--color-primary-subtle);
    color: var(--color-primary);
    display: flex;
    align-items: center;
    justify-content: center;
    flex-shrink: 0;
}
.ai-message-content {
    max-width: 80%;
}
.ai-message-text {
    padding: 10px 14px;
    border-radius: var(--radius-lg);
    font-size: 0.8125rem;
    line-height: 1.5;
    white-space: pre-wrap;
    word-break: break-word;
}
.ai-message--user .ai-message-text {
    background: var(--color-primary);
    color: var(--color-text-on-primary);
    border-bottom-right-radius: 4px;
}
.ai-message--assistant .ai-message-text {
    background: var(--color-bg-muted);
    color: var(--color-text);
    border-bottom-left-radius: 4px;
}

.ai-typing {
    display: flex;
    align-items: center;
    gap: 6px;
    font-size: 0.75rem;
    color: var(--color-text-tertiary);
    padding: 4px 0;
}

.ai-error {
    padding: 8px 16px;
    background: oklch(0.95 0.05 25);
    color: var(--color-danger);
    font-size: 0.75rem;
}
@media (prefers-color-scheme: dark) {
    .ai-error { background: oklch(0.22 0.05 25); }
}

/* ── Input ──────────────────────────────────────────────────────── */
.ai-input-area {
    display: flex;
    align-items: flex-end;
    gap: 8px;
    padding: 12px 16px;
    border-top: 1px solid var(--color-border-subtle);
}
.ai-input {
    flex: 1;
    resize: none;
    border: 1px solid var(--color-border);
    border-radius: var(--radius-md);
    padding: 10px 12px;
    font-family: var(--font-sans);
    font-size: 0.8125rem;
    color: var(--color-text);
    background: var(--color-bg);
    max-height: 100px;
    transition: border-color var(--transition-fast);
}
.ai-input:focus {
    outline: none;
    border-color: var(--color-primary);
}
.ai-input::placeholder { color: var(--color-text-tertiary); }

.ai-send {
    width: 36px;
    height: 36px;
    border-radius: var(--radius-md);
    background: var(--color-primary);
    color: var(--color-text-on-primary);
    border: none;
    cursor: pointer;
    display: flex;
    align-items: center;
    justify-content: center;
    transition: all var(--transition-fast);
    flex-shrink: 0;
}
.ai-send:hover:not(:disabled) { background: var(--color-primary-hover); }
.ai-send:disabled { opacity: 0.4; cursor: not-allowed; }

/* ── Transitions ────────────────────────────────────────────────── */
.chat-slide-enter-active { animation: slideUp 0.25s ease-out; }
.chat-slide-leave-active { animation: slideUp 0.2s ease-in reverse; }

@keyframes slideUp {
    from { opacity: 0; transform: translateY(16px) scale(0.96); }
    to { opacity: 1; transform: translateY(0) scale(1); }
}

.spin { animation: spin 1s linear infinite; }
@keyframes spin { from { transform: rotate(0deg); } to { transform: rotate(360deg); } }

@media (max-width: 480px) {
    .ai-panel { width: calc(100vw - 48px); height: calc(100vh - 120px); }
}
</style>
