import { defineStore } from 'pinia'
import { ref } from 'vue'

export const useAIStore = defineStore('ai', () => {
    const messages = ref([])
    const streaming = ref(false)
    const error = ref(null)

    function addMessage(role, content) {
        messages.value.push({ role, content, id: Date.now() + Math.random() })
    }

    function clearMessages() {
        messages.value = []
        error.value = null
    }

    async function sendMessage(content) {
        error.value = null
        addMessage('user', content)

        // Add a placeholder for the assistant response.
        const assistantMsg = { role: 'assistant', content: '', id: Date.now() + Math.random() }
        messages.value.push(assistantMsg)

        streaming.value = true

        try {
            const token = localStorage.getItem('access_token')
            const response = await fetch('/v1/ai/chat', {
                method: 'POST',
                headers: {
                    'Content-Type': 'application/json',
                    Authorization: `Bearer ${token}`,
                },
                body: JSON.stringify({
                    messages: messages.value
                        .filter((m) => m.role !== 'assistant' || m.content)
                        .slice(0, -1) // exclude the empty placeholder
                        .map(({ role, content }) => ({ role, content })),
                }),
            })

            if (!response.ok) {
                const data = await response.json()
                throw new Error(data.error?.message || 'AI request failed')
            }

            const reader = response.body.getReader()
            const decoder = new TextDecoder()
            let buffer = ''

            while (true) {
                const { done, value } = await reader.read()
                if (done) break

                buffer += decoder.decode(value, { stream: true })
                const lines = buffer.split('\n')
                buffer = lines.pop() || ''

                for (const line of lines) {
                    if (!line.startsWith('event: message')) continue
                    const dataLine = lines[lines.indexOf(line) + 1]
                    if (!dataLine || !dataLine.startsWith('data: ')) continue

                    const data = dataLine.slice(6)
                    if (data === '[DONE]') break
                    assistantMsg.content += data
                }
            }

            // Fallback: try simple SSE parsing if structured parsing didn't capture content
            if (!assistantMsg.content && buffer) {
                const dataMatches = buffer.match(/data: (.+)/g)
                if (dataMatches) {
                    assistantMsg.content = dataMatches
                        .map((d) => d.slice(6))
                        .filter((d) => d !== '[DONE]')
                        .join('')
                }
            }
        } catch (err) {
            error.value = err.message
            // Remove the empty assistant message on error
            const idx = messages.value.indexOf(assistantMsg)
            if (idx !== -1 && !assistantMsg.content) {
                messages.value.splice(idx, 1)
            }
        } finally {
            streaming.value = false
        }
    }

    return { messages, streaming, error, addMessage, sendMessage, clearMessages }
})
