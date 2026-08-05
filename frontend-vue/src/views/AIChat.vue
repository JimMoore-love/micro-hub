<script setup lang="ts">
import { ref, reactive, nextTick, onMounted, watch } from 'vue'
import { ElMessage } from 'element-plus'
import { aiApi } from '../api/ai'
import type { ChatMessage, AIUsage } from '../api/ai'
import { marked } from 'marked'

// 对话历史列表
interface Conversation {
  id: string
  title: string
  messages: ChatMessage[]
  createdAt: string
}

const conversations = ref<Conversation[]>([])
const currentConversationId = ref<string>('')
const currentMessages = ref<ChatMessage[]>([])
const inputText = ref('')
const selectedModel = ref('gpt-4')
const isStreaming = ref(false)
const streamingContent = ref('')
const usage = ref<AIUsage>({ total_tokens: 0, total_cost: 0, request_count: 0 })
const chatContainerRef = ref<HTMLElement>()

const modelOptions = [
  { label: 'GPT-4', value: 'gpt-4' },
  { label: 'GPT-4o', value: 'gpt-4o' },
  { label: 'GPT-3.5 Turbo', value: 'gpt-3.5-turbo' },
  { label: 'Claude 3', value: 'claude-3' },
]

// 初始化：创建默认对话
function initConversations() {
  if (conversations.value.length === 0) {
    const defaultConv: Conversation = {
      id: generateId(),
      title: '新的对话',
      messages: [
        {
          role: 'assistant',
          content: '你好！我是 AI 助手，有什么可以帮助你的吗？',
        },
      ],
      createdAt: new Date().toISOString(),
    }
    conversations.value.push(defaultConv)
    selectConversation(defaultConv.id)
  }
}

function generateId(): string {
  return Date.now().toString(36) + Math.random().toString(36).slice(2, 8)
}

function selectConversation(id: string) {
  currentConversationId.value = id
  const conv = conversations.value.find(c => c.id === id)
  if (conv) {
    currentMessages.value = [...conv.messages]
  }
  nextTick(() => scrollToBottom())
}

function createNewConversation() {
  const newConv: Conversation = {
    id: generateId(),
    title: '新的对话',
    messages: [
      {
        role: 'assistant',
        content: '你好！我是 AI 助手，有什么可以帮助你的吗？',
      },
    ],
    createdAt: new Date().toISOString(),
  }
  conversations.value.unshift(newConv)
  selectConversation(newConv.id)
}

function saveCurrentMessages() {
  const conv = conversations.value.find(c => c.id === currentConversationId.value)
  if (conv) {
    conv.messages = [...currentMessages.value]
    // 用第一条用户消息作为对话标题
    const firstUserMsg = currentMessages.value.find(m => m.role === 'user')
    if (firstUserMsg) {
      conv.title = firstUserMsg.content.slice(0, 30) + (firstUserMsg.content.length > 30 ? '...' : '')
    }
  }
}

function deleteConversation(id: string) {
  conversations.value = conversations.value.filter(c => c.id !== id)
  if (currentConversationId.value === id) {
    if (conversations.value.length > 0) {
      selectConversation(conversations.value[0].id)
    } else {
      initConversations()
    }
  }
}

async function sendMessage() {
  const text = inputText.value.trim()
  if (!text || isStreaming.value) return

  // 添加用户消息
  const userMsg: ChatMessage = { role: 'user', content: text }
  currentMessages.value.push(userMsg)
  inputText.value = ''
  saveCurrentMessages()

  nextTick(() => scrollToBottom())

  // 创建空的 assistant 消息占位
  const assistantMsg: ChatMessage = { role: 'assistant', content: '' }
  currentMessages.value.push(assistantMsg)
  streamingContent.value = ''
  isStreaming.value = true

  try {
    await aiApi.chatStream(
      {
        messages: currentMessages.value.filter(m => m.content !== '流式响应中'),
        model: selectedModel.value,
      },
      // onToken
      (token: string) => {
        streamingContent.value += token
        assistantMsg.content = streamingContent.value
        nextTick(() => scrollToBottom())
      },
      // onDone
      (usageData: AIUsage) => {
        usage.value = usageData
        isStreaming.value = false
        saveCurrentMessages()
      },
      // onError
      (err: Error) => {
        if (assistantMsg.content === '') {
          assistantMsg.content = `抱歉，发生了错误：${err.message}`
        }
        isStreaming.value = false
        saveCurrentMessages()
      }
    )
  } catch (err: any) {
    assistantMsg.content = `请求失败：${err.message}`
    isStreaming.value = false
    saveCurrentMessages()
  }
}

function handleKeyDown(e: KeyboardEvent) {
  if (e.key === 'Enter' && !e.shiftKey) {
    e.preventDefault()
    sendMessage()
  }
}

function scrollToBottom() {
  nextTick(() => {
    if (chatContainerRef.value) {
      chatContainerRef.value.scrollTop = chatContainerRef.value.scrollHeight
    }
  })
}

// 简单 Markdown 渲染
function renderMarkdown(text: string): string {
  if (!text) return ''
  try {
    return marked.parse(text) as string
  } catch {
    return text.replace(/\n/g, '<br>')
  }
}

// 获取 AI 用量
async function fetchUsage() {
  try {
    const res = await aiApi.getUsage()
    usage.value = res
  } catch {}
}

onMounted(() => {
  initConversations()
  fetchUsage()
})
</script>

<template>
  <div class="ai-chat-page">
    <!-- 左侧对话列表 -->
    <aside class="chat-sidebar">
      <div class="sidebar-top">
        <el-button type="primary" class="btn-gradient" style="width: 100%" @click="createNewConversation">
          <el-icon><Plus /></el-icon> 新建对话
        </el-button>
      </div>

      <div class="conversation-list">
        <div
          v-for="conv in conversations"
          :key="conv.id"
          class="conv-item"
          :class="{ active: conv.id === currentConversationId }"
          @click="selectConversation(conv.id)"
        >
          <div class="conv-title">{{ conv.title }}</div>
          <el-button
            link
            class="conv-delete"
            @click.stop="deleteConversation(conv.id)"
          >
            <el-icon><Delete /></el-icon>
          </el-button>
        </div>
      </div>

      <!-- 用量统计 -->
      <div class="usage-card">
        <div class="usage-title">用量统计</div>
        <div class="usage-stats">
          <div class="usage-item">
            <span class="usage-label">Tokens</span>
            <span class="usage-value">{{ usage.total_tokens.toLocaleString() }}</span>
          </div>
          <div class="usage-item">
            <span class="usage-label">请求数</span>
            <span class="usage-value">{{ usage.request_count }}</span>
          </div>
          <div class="usage-item">
            <span class="usage-label">费用</span>
            <span class="usage-value">${{ usage.total_cost.toFixed(4) }}</span>
          </div>
        </div>
      </div>
    </aside>

    <!-- 右侧对话区 -->
    <div class="chat-main">
      <!-- 模型选择栏 -->
      <div class="chat-header">
        <div class="model-selector">
          <el-select v-model="selectedModel" size="small" style="width: 180px">
            <el-option
              v-for="m in modelOptions"
              :key="m.value"
              :label="m.label"
              :value="m.value"
            />
          </el-select>
        </div>
        <el-tag size="small" type="info">{{ isStreaming ? '响应中...' : '就绪' }}</el-tag>
      </div>

      <!-- 消息区域 -->
      <div ref="chatContainerRef" class="chat-messages">
        <template v-if="currentMessages.length === 0">
          <div class="empty-chat">
            <div class="empty-icon">
              <el-icon :size="64"><ChatDotRound /></el-icon>
            </div>
            <h3>开始对话</h3>
            <p>选择一个模型，输入你的问题，AI 将实时回复</p>
          </div>
        </template>

        <div
          v-for="(msg, index) in currentMessages"
          :key="index"
          class="message-row"
          :class="msg.role"
        >
          <div v-if="msg.role === 'assistant'" class="msg-avatar">
            <div class="ai-avatar">AI</div>
          </div>

          <div class="msg-bubble" :class="msg.role">
            <div
              v-if="msg.role === 'assistant'"
              class="msg-content markdown-body"
              v-html="renderMarkdown(msg.content)"
            />
            <div v-else class="msg-content">
              {{ msg.content }}
            </div>
          </div>

          <div v-if="msg.role === 'user'" class="msg-avatar">
            <el-avatar :size="34" icon="UserFilled" />
          </div>
        </div>
      </div>

      <!-- 输入区域 -->
      <div class="chat-input-area">
        <div class="input-wrapper">
          <textarea
            v-model="inputText"
            class="chat-textarea"
            placeholder="输入消息，Enter 发送，Shift+Enter 换行..."
            :disabled="isStreaming"
            rows="2"
            @keydown="handleKeyDown"
          ></textarea>
          <el-button
            type="primary"
            class="btn-gradient send-btn"
            :disabled="!inputText.trim() || isStreaming"
            @click="sendMessage"
          >
            <el-icon><Promotion /></el-icon>
          </el-button>
        </div>
      </div>
    </div>
  </div>
</template>

<style scoped>
.ai-chat-page {
  display: flex;
  height: calc(100vh - 64px - 48px);
  gap: 0;
  margin: -24px;
  overflow: hidden;
}

/* -------- 左侧栏 -------- */
.chat-sidebar {
  width: 260px;
  background: var(--bg-sidebar);
  display: flex;
  flex-direction: column;
  border-right: 1px solid var(--border-color);
  flex-shrink: 0;
}

.sidebar-top {
  padding: 16px;
  border-bottom: 1px solid var(--border-color);
}

.conversation-list {
  flex: 1;
  overflow-y: auto;
  padding: 8px;
}

.conv-item {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 14px;
  border-radius: 8px;
  cursor: pointer;
  margin-bottom: 4px;
  transition: background 0.2s;
  color: var(--text-secondary);
  font-size: 13px;
}

.conv-item:hover {
  background: rgba(255, 255, 255, 0.05);
}

.conv-item.active {
  background: rgba(59, 130, 246, 0.12);
  color: var(--text-primary);
}

.conv-title {
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
  flex: 1;
}

.conv-delete {
  opacity: 0;
  transition: opacity 0.2s;
  color: var(--danger) !important;
  padding: 2px;
}

.conv-item:hover .conv-delete {
  opacity: 1;
}

/* 用量卡片 */
.usage-card {
  margin: 12px;
  padding: 14px;
  background: rgba(59, 130, 246, 0.06);
  border-radius: 10px;
  border: 1px solid rgba(59, 130, 246, 0.1);
}

.usage-title {
  font-size: 12px;
  font-weight: 600;
  color: var(--text-muted);
  margin-bottom: 10px;
  text-transform: uppercase;
  letter-spacing: 0.5px;
}

.usage-stats {
  display: flex;
  flex-direction: column;
  gap: 6px;
}

.usage-item {
  display: flex;
  justify-content: space-between;
  font-size: 13px;
}

.usage-label {
  color: var(--text-muted);
}

.usage-value {
  color: var(--accent);
  font-weight: 600;
  font-family: 'Fira Code', monospace;
}

/* -------- 右侧对话区 -------- */
.chat-main {
  flex: 1;
  display: flex;
  flex-direction: column;
  background: var(--bg-primary);
  min-width: 0;
}

.chat-header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 12px 24px;
  background: var(--bg-header);
  border-bottom: 1px solid var(--border-color);
  flex-shrink: 0;
}

.chat-messages {
  flex: 1;
  overflow-y: auto;
  padding: 24px;
  display: flex;
  flex-direction: column;
  gap: 20px;
}

.empty-chat {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  color: var(--text-muted);
}

.empty-icon {
  margin-bottom: 16px;
  opacity: 0.3;
}

.empty-chat h3 {
  font-size: 20px;
  color: var(--text-secondary);
  margin-bottom: 8px;
}

.empty-chat p {
  font-size: 14px;
}

/* 消息行 */
.message-row {
  display: flex;
  align-items: flex-start;
  gap: 10px;
}

.message-row.user {
  justify-content: flex-end;
}

.message-row.assistant {
  justify-content: flex-start;
}

.msg-avatar {
  flex-shrink: 0;
  padding-top: 2px;
}

.ai-avatar {
  width: 34px;
  height: 34px;
  border-radius: 10px;
  background: linear-gradient(135deg, #3b82f6, #8b5cf6);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
  font-weight: 700;
  color: #fff;
}

.msg-bubble {
  max-width: 72%;
  padding: 12px 18px;
  border-radius: 14px;
  font-size: 14px;
  line-height: 1.7;
  word-break: break-word;
}

.msg-bubble.user {
  background: linear-gradient(135deg, #3b82f6, #2563eb);
  color: #fff;
  border-bottom-right-radius: 4px;
}

.msg-bubble.assistant {
  background: rgba(255, 255, 255, 0.06);
  color: var(--text-primary);
  border: 1px solid var(--border-color);
  border-bottom-left-radius: 4px;
}

.msg-content {
  white-space: pre-wrap;
}

/* Markdown 简单样式 */
.markdown-body p {
  margin: 0 0 8px 0;
}

.markdown-body p:last-child {
  margin-bottom: 0;
}

.markdown-body code {
  background: rgba(0, 0, 0, 0.3);
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
  font-family: 'Fira Code', monospace;
}

.markdown-body pre {
  background: rgba(0, 0, 0, 0.3);
  padding: 12px 16px;
  border-radius: 8px;
  overflow-x: auto;
  margin: 8px 0;
}

.markdown-body pre code {
  background: transparent;
  padding: 0;
}

.markdown-body ul,
.markdown-body ol {
  padding-left: 20px;
  margin: 8px 0;
}

/* 输入区域 */
.chat-input-area {
  padding: 16px 24px;
  background: var(--bg-header);
  border-top: 1px solid var(--border-color);
  flex-shrink: 0;
}

.input-wrapper {
  display: flex;
  align-items: flex-end;
  gap: 10px;
}

.chat-textarea {
  flex: 1;
  resize: none;
  border: 1px solid var(--border-color);
  border-radius: 12px;
  background: var(--bg-input);
  color: var(--text-primary);
  padding: 12px 16px;
  font-size: 14px;
  font-family: inherit;
  line-height: 1.5;
  outline: none;
  transition: border-color 0.2s;
  max-height: 120px;
}

.chat-textarea:focus {
  border-color: var(--accent);
}

.chat-textarea::placeholder {
  color: var(--text-muted);
}

.send-btn {
  width: 44px;
  height: 44px;
  border-radius: 12px;
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
</style>
