import { useEffect } from 'react'
import type { StepEvent } from '../App'

interface UseSSEOptions {
  sessionId: string
  onEvent: (ev: StepEvent) => void
  onClear: () => void
}

export function useSSE({ sessionId, onEvent, onClear }: UseSSEOptions) {
  useEffect(() => {
    if (!sessionId) return

    onClear()
    let es: EventSource | null = null
    let closed = false

    const connect = () => {
      if (closed) return
      es = new EventSource(`/api/sessions/${sessionId}/events`)

      es.onmessage = (e) => {
        try {
          const data = JSON.parse(e.data) as StepEvent
          onEvent(data)
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
  }, [sessionId, onEvent, onClear])
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

// Convert backend Step array to frontend StepEvent array
export function stepsToEvents(steps: Array<{
  agent: string
  type: string
  content: string
  tool_name?: string
  tool_args?: string
  tool_result?: string
}>): StepEvent[] {
  const events: StepEvent[] = []
  let lastAgent = ''

  for (const step of steps) {
    // Add agent header when agent changes
    if (step.agent !== lastAgent) {
      if (lastAgent) {
        events.push({
          type: 'agent_completed',
          agent: lastAgent,
          step: lastAgent + ' 完成',
          payload: {},
          timestamp: new Date().toISOString(),
        })
      }
      events.push({
        type: 'step_started',
        agent: step.agent,
        step: step.agent + ' 开始执行',
        payload: {},
        timestamp: new Date().toISOString(),
      })
      lastAgent = step.agent
    }

    // Skip llm_response and retry internal steps (not meaningful for display)
    if (step.type === 'llm_response' || step.type === 'retry') {
      continue
    }

    events.push({
      type: 'react_step',
      agent: step.agent,
      step: step.type,
      payload: {
        step_type: step.type,
        content: step.content,
        tool_name: step.tool_name,
        tool_args: step.tool_args,
        tool_result: step.tool_result,
      },
      timestamp: new Date().toISOString(),
    })
  }

  // Close last agent
  if (lastAgent) {
    events.push({
      type: 'agent_completed',
      agent: lastAgent,
      step: lastAgent + ' 完成',
      payload: {},
      timestamp: new Date().toISOString(),
    })
  }

  return events
}
