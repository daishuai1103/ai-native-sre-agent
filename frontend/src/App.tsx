import { useCallback, useEffect, useState } from 'react'
import './App.css'
import AlertList from './components/AlertList'
import ChatPanel from './components/ChatPanel'
import StatusBar from './components/StatusBar'
import ReportView from './components/ReportView'
import RouterInline from './components/RouterInline'
import TracePanel from './components/TracePanel'
import { useSSE, stepsToEvents } from './api/sse'

export interface StepEvent {
  type: string
  agent?: string
  step: string
  payload?: any
  timestamp: string
}

export default function App() {
  const [sessionId, setSessionId] = useState('')
  const [report, setReport] = useState('')
  const [routeCategory, setRouteCategory] = useState('')
  const [routeSource, setRouteSource] = useState('')
  const [routeConfidence, setRouteConfidence] = useState(0)
  const [currentAlert, setCurrentAlert] = useState('')
  const [events, setEvents] = useState<StepEvent[]>([])

  const handleSSEEvent = useCallback((ev: StepEvent) => {
    setEvents((prev) => [...prev, ev])
    // Also catch report from live SSE
    if (ev.type === 'report_generated') {
      setReport(ev.step || '')
    }
  }, [])

  const handleSSEClear = useCallback(() => {
    setEvents([])
    setReport('')
  }, [])

  useSSE({ sessionId, onEvent: handleSSEEvent, onClear: handleSSEClear })

  // Fetch session details on session change
  useEffect(() => {
    if (!sessionId) return
    fetchSession(sessionId)
  }, [sessionId])

  async function fetchSession(id: string) {
    try {
      const res = await fetch(`/api/sessions/${id}`)
      const data = await res.json()
      if (data.task) {
        setRouteCategory(data.task.route_category || data.task.RouteCategory || '')
        setRouteSource(data.task.route_source || data.task.RouteSource || '')
        setRouteConfidence(data.task.route_confidence || data.task.RouteConfidence || 0)
        setCurrentAlert(data.task.alert || data.task.Alert || '')
      }
      // Load historical steps if session is already done (or partially done)
      if (data.steps && data.steps.length > 0) {
        const historical = stepsToEvents(data.steps)
        // Add report banner if report exists
        if (data.report) {
          historical.push({
            type: 'report_generated',
            agent: 'reporter',
            step: data.report,
            payload: {},
            timestamp: new Date().toISOString(),
          })
          setReport(data.report)
        }
        setEvents(historical)
      } else if (data.report) {
        setReport(data.report)
      }
    } catch (e) {
      console.error('fetch session failed', e)
    }
  }

  const handleSelectSession = (sid: string) => {
    setSessionId(sid)
    setReport('')
    setEvents([])
  }

  const handleSessionCreated = (sid: string) => {
    handleSelectSession(sid)
  }

  return (
    <div className="app-container">
      <StatusBar />

      <div className="main-layout">
        {/* Left column: Alert list + Chat */}
        <aside className="left-col">
          <AlertList
            onSelectSession={handleSelectSession}
            selectedSessionId={sessionId}
          />
          <ChatPanel onSessionCreated={handleSessionCreated} />
        </aside>

        {/* Middle column: Agent trace (main focus) */}
        <main className="center-col">
          {sessionId ? (
            <>
              <RouterInline
                category={routeCategory}
                source={routeSource}
                confidence={routeConfidence}
                alert={currentAlert}
              />
              <TracePanel events={events} />
            </>
          ) : (
            <div className="empty-center">
              <div className="empty-center-icon">🤖</div>
              <h2>AI Native SRE Agent</h2>
              <p>选择左侧告警，或手动输入问题，开始智能运维分析</p>
              <div className="empty-features">
                <div className="empty-feature">
                  <span>🔍</span>
                  <span>自动告警采集</span>
                </div>
                <div className="empty-feature">
                  <span>🧠</span>
                  <span>智能路由分类</span>
                </div>
                <div className="empty-feature">
                  <span>⚡</span>
                  <span>多 Agent 协同</span>
                </div>
                <div className="empty-feature">
                  <span>✅</span>
                  <span>自动处置闭环</span>
                </div>
              </div>
            </div>
          )}
        </main>

        {/* Right column: Report */}
        <aside className="right-col">
          <ReportView report={report} sessionId={sessionId} />
        </aside>
      </div>
    </div>
  )
}
