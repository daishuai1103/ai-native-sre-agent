import { useEffect, useState } from 'react'
import './components.css'

interface AlertStats {
  total: number
  warning: number
  critical: number
  detected: number
  processing: number
  processed: number
  resolved: number
  failed: number
  from_docker: number
  from_prometheus: number
  avg_duration_ms: number
}

export default function StatusBar() {
  const [stats, setStats] = useState<AlertStats | null>(null)

  useEffect(() => {
    const fetchStats = async () => {
      try {
        const res = await fetch('/api/alerts/stats')
        const data = await res.json()
        setStats(data)
      } catch (e) {
        // silently ignore
      }
    }

    fetchStats()
    const timer = setInterval(fetchStats, 5000)
    return () => clearInterval(timer)
  }, [])

  const healRate = stats && stats.total > 0
    ? Math.round(((stats.resolved + stats.processed) / stats.total) * 100)
    : 0

  return (
    <header className="status-bar">
      <div className="status-left">
        <span className="status-logo">🛡️</span>
        <span className="status-title">AI Native SRE Agent</span>
        <span className="status-version">v1.0</span>
      </div>

      <div className="status-center">
        <div className="status-item critical">
          <span className="status-dot" />
          <span className="status-label">严重</span>
          <span className="status-value">{stats?.critical ?? 0}</span>
        </div>
        <div className="status-item warning">
          <span className="status-dot" />
          <span className="status-label">警告</span>
          <span className="status-value">{stats?.warning ?? 0}</span>
        </div>
        <div className="status-item processing">
          <span className="status-dot pulse" />
          <span className="status-label">处理中</span>
          <span className="status-value">{stats?.processing ?? 0}</span>
        </div>
        <div className="status-item resolved">
          <span className="status-dot" />
          <span className="status-label">已自愈</span>
          <span className="status-value">{stats?.resolved ?? 0}</span>
        </div>
        <div className="status-item processed">
          <span className="status-dot" />
          <span className="status-label">已诊断</span>
          <span className="status-value">{stats?.processed ?? 0}</span>
        </div>
        <div className="status-divider" />
        <div className="status-item heal">
          <span className="status-heal-icon">💚</span>
          <span className="status-label">自愈率</span>
          <span className="status-value heal-val">{healRate}%</span>
        </div>
        {stats && stats.avg_duration_ms > 0 && (
          <div className="status-item duration">
            <span className="status-label">平均耗时</span>
            <span className="status-value">
              {(stats.avg_duration_ms / 1000).toFixed(1)}s
            </span>
          </div>
        )}
      </div>

      <div className="status-right">
        <span className="status-source">
          🐳 Docker {stats?.from_docker ?? 0}
        </span>
        <span className="status-source">
          📊 Prom {stats?.from_prometheus ?? 0}
        </span>
      </div>
    </header>
  )
}
