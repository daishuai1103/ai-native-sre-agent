import { useState } from 'react'
import { sendAlert } from '../api/sse'
import './components.css'

interface Props {
  onSessionCreated: (id: string) => void
}

export default function ChatPanel({ onSessionCreated }: Props) {
  const [input, setInput] = useState('')
  const [loading, setLoading] = useState(false)
  const [messages, setMessages] = useState<{ role: 'user' | 'agent'; text: string }[]>([])

  const handleSend = async () => {
    if (!input.trim()) return
    setMessages((prev) => [...prev, { role: 'user', text: input }])
    setLoading(true)
    try {
      const { session_id } = await sendAlert(input)
      onSessionCreated(session_id)
      setMessages((prev) => [...prev, { role: 'agent', text: `Session: ${session_id}` }])
    } catch (err) {
      setMessages((prev) => [...prev, { role: 'agent', text: 'Error: failed to create session' }])
    } finally {
      setLoading(false)
      setInput('')
    }
  }

  return (
    <div className="chat-panel">
      <div className="chat-messages">
        {messages.map((m, i) => (
          <div key={i} className={`chat-message ${m.role}`}>
            {m.text}
          </div>
        ))}
      </div>
      <div className="chat-input-area">
        <input
          value={input}
          onChange={(e) => setInput(e.target.value)}
          onKeyDown={(e) => e.key === 'Enter' && handleSend()}
          placeholder="输入告警，例如：Redis 容器内存过高"
          disabled={loading}
        />
        <button onClick={handleSend} disabled={loading}>
          {loading ? '发送中' : '发送'}
        </button>
      </div>
    </div>
  )
}
