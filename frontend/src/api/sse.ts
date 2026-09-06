import { useEffect, useState } from 'react'
import type { StepEvent } from '../App'

export function useSSE(sessionId: string) {
  const [events, setEvents] = useState<StepEvent[]>([])

  useEffect(() => {
    if (!sessionId) return

    setEvents([])
    let es: EventSource | null = null
    let closed = false

    const connect = () => {
      if (closed) return
      es = new EventSource(`/api/sessions/${sessionId}/events`)

      es.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data) as StepEvent
          setEvents((prev) => [...prev, data])
        } catch (err) {
          console.error('failed to parse sse event', err)
        }
      }

      es.onerror = () => {
        es?.close()
        if (!closed) {
          setTimeout(connect, 1000)
        }
      }
    }

    connect()

    return () => {
      closed = true
      es?.close()
    }
  }, [sessionId])

  return { events }
}

export async function sendAlert(alert: string): Promise<{ session_id: string }> {
  const res = await fetch('/api/alerts', {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ alert }),
  })
  if (!res.ok) throw new Error('failed to send alert')
  return res.json()
}
