import { useEffect, useState } from 'react'
import './components.css'

export interface AlertItem {
  id: string
  source: string
  severity: string
  message: string
  container: string
  metric: string
  value: number
  threshold: number
  detected_at: string
  resolved: boolean
  session_id?: string
  status?: 'detected' | 'processing' | 'resolved' | 'failed'
  duration_ms?: number
  error_msg?: string
}

interface AlertListProps {
  onSelectSession: (sessionId: string) => void
  selectedSessionId: string
}

export default function AlertList({ onSelectSession, selectedSessionId }: AlertListProps) {
  const [alerts, setAlerts] = useState<AlertItem[]>([])

  useEffect(() => {
    const fetchAlerts = async () => {
      try {
        const res = await fetch('/api/alerts')
        const data = await res.json()
        setAlerts(data.alerts || [])
      } catch (e) {
        console.error('failed to fetch alerts', e)
      }
    }

    fetchAlerts()
    const timer = setInterval(fetchAlerts, 5000)
    return () => clearInterval(timer)
  }, [])

  const severityColor = (s: string) => {
    if (s === 'critical') return '#ff4d4f'
    if (s === 'warning') return '#faad14'
    return '#52c41a'
  }

  const sourceLabel = (s: string) => {
    if (s === 'docker') return 'Docker'
    if (s === 'prometheus') return 'Prometheus'
    return s
  }

  const statusInfo = (status?: string) => {
    switch (status) {
      case 'processing':
        return { label: '处理中', color: '#1890ff', bg: '#e6f7ff' }
      case 'processed':
        return { label: '已诊断', color: '#722ed1', bg: '#f9f0ff' }
      case 'resolved':
        return { label: '已自愈', color: '#52c41a', bg: '#f6ffed' }
      case 'failed':
        return { label: '失败', color: '#ff4d4f', bg: '#fff1f0' }
      default:
        return { label: '待处理', color: '#faad14', bg: '#fffbe6' }
    }
  }

  const formatDuration = (ms?: number) => {
    if (!ms) return ''
    if (ms < 1000) return `${ms}ms`
    return `${(ms / 1000).toFixed(1)}s`
  }

  const [injecting, setInjecting] = useState(false)

  const handleInject = async () => {
    if (injecting) return
    setInjecting(true)
    try {
      const res = await fetch('/api/alerts/inject', { method: 'POST' })
      await res.json()
      // 立即刷新一次
      const r = await fetch('/api/alerts')
      const d = await r.json()
      setAlerts(d.alerts || [])
    } catch (e) {
      console.error('inject failed', e)
    } finally {
      setTimeout(() => setInjecting(false), 1000)
    }
  }

  return (
    <div className="alert-list">
      <div className="panel-header">
        <h3>自动告警</h3>
        <div className="panel-header-right">
          <button
            className="inject-btn"
            onClick={handleInject}
            disabled={injecting}
            title="注入一批模拟告警用于演示"
          >
            {injecting ? '注入中...' : '⚡ 模拟告警'}
          </button>
          <span className="alert-count">{alerts.length}</span>
        </div>
      </div>
      <div className="alert-items">
        {alerts.length === 0 && (
          <div className="empty">暂无告警，系统巡检中...</div>
        )}
        {alerts.map((alert) => {
          const st = statusInfo(alert.status)
          return (
            <div
              key={alert.id}
              className={`alert-item ${alert.session_id === selectedSessionId ? 'selected' : ''}`}
              onClick={() => alert.session_id && onSelectSession(alert.session_id)}
            >
              <div className="alert-top">
                <span className="severity-badge" style={{ backgroundColor: severityColor(alert.severity) }}>
                  {alert.severity === 'critical' ? '严重' : '警告'}
                </span>
                <span className="source-tag">{sourceLabel(alert.source)}</span>
                <span
                  className="status-tag"
                  style={{ backgroundColor: st.bg, color: st.color, borderColor: st.color }}
                >
                  {st.label}
                </span>
              </div>
              <div className="alert-message">{alert.message}</div>
              <div className="alert-meta">
                <span>容器: {alert.container}</span>
                {alert.value > 0 && (
                  <span>{alert.value.toFixed(1)}% / {alert.threshold}%</span>
                )}
              </div>
              <div className="alert-time">
                <span>{new Date(alert.detected_at).toLocaleTimeString('zh-CN')}</span>
                <span className="alert-time-right">
                  {alert.duration_ms ? (
                    <span className="duration-tag">耗时 {formatDuration(alert.duration_ms)}</span>
                  ) : alert.status === 'processing' ? (
                    <span className="processing-dot" />
                  ) : null}
                  {alert.session_id && <span className="session-tag">会话</span>}
                </span>
              </div>
            </div>
          )
        })}
      </div>
    </div>
  )
}
