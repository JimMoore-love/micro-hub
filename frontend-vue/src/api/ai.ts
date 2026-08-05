import client from './client'
import { useUserStore } from '../stores/user'

export interface ChatMessage {
  role: 'user' | 'assistant' | 'system'
  content: string
}

export interface ChatRequest {
  messages: ChatMessage[]
  model?: string
  stream?: boolean
}

export interface AIUsage {
  total_tokens: number
  total_cost: number
  request_count: number
}

export const aiApi = {
  /** 普通对话（非流式） */
  chat(params: ChatRequest) {
    return client.post<any, { content: string; usage: AIUsage }>('/ai/chat', { ...params, stream: false })
  },

  /** SSE 流式对话 */
  async chatStream(
    params: ChatRequest,
    onToken: (token: string) => void,
    onDone: (usage: AIUsage) => void,
    onError: (err: Error) => void
  ): Promise<void> {
    const store = useUserStore()
    const tenantId = store.currentTenantId
    const token = store.token

    const headers: Record<string, string> = {
      'Content-Type': 'application/json',
    }
    if (tenantId) headers['X-Tenant-ID'] = tenantId
    if (token) headers['Authorization'] = `Bearer ${token}`

    try {
      const response = await fetch('/api/v1/ai/chat', {
        method: 'POST',
        headers,
        body: JSON.stringify({ ...params, stream: true }),
      })

      if (!response.ok) {
        throw new Error(`HTTP ${response.status}: ${response.statusText}`)
      }

      const reader = response.body?.getReader()
      if (!reader) {
        throw new Error('Response body is not readable')
      }

      const decoder = new TextDecoder()
      let buffer = ''

      while (true) {
        const { done, value } = await reader.read()
        if (done) break

        buffer += decoder.decode(value, { stream: true })
        const lines = buffer.split('\n')
        buffer = lines.pop() || ''

        for (const line of lines) {
          if (line.startsWith('data: ')) {
            const data = line.slice(6).trim()
            if (data === '[DONE]') {
              return
            }
            try {
              const parsed = JSON.parse(data)
              if (parsed.content) {
                onToken(parsed.content)
              }
              if (parsed.usage) {
                onDone(parsed.usage)
              }
            } catch {
              // 非 JSON 内容，可能是纯文本 token
              if (data) {
                onToken(data)
              }
            }
          }
        }
      }
    } catch (err) {
      onError(err instanceof Error ? err : new Error(String(err)))
    }
  },

  /** 获取 AI 用量统计 */
  getUsage() {
    return client.get<any, AIUsage>('/ai/usage')
  },
}
