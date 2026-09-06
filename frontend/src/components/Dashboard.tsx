import { useEffect, useState } from 'react'
import './components.css'

interface AlertStats {
  total: number
  warning: number
  critical: number
  detected: number
  processing: number
  resolved: number
  failed: number
  from_docker: number
  from_prometheus: number
  avg_duration_ms: number
}

export default function Dashboard() {
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

  if (!stats) {
    return (
      <div className="dashboard">
        <div className="panel-header">
          <h3>运维总览</h3>
        </div>
        <div className="dash-loading">加载中...</div>
      </div>
    )
  }

  const selfHealRate = stats.total > 0
    ? Math.round((stats.resolved / stats.total) * 100)
    : 0

  return (
    <div className="dashboard">
      <div className="panel-header">
        <h3>运维总览</h3>
        <span className="dash-subtitle">实时统计</span>
      </div>

      <div className="dash-stats">
        <div className="stat-card critical">
          <div className="stat-value">{stats.critical}</div>
          <div className="stat-label">严重告警</div>
        </div>
        <div className="stat-card warning">
          <div className="stat-value">{stats.warning}</div>
          <div className="stat-label">警告</div>
        </div>
        <div className="stat-card processing">
          <div className="stat-value">{stats.processing}</div>
          <div className="stat-label">处理中</div>
        </div>
        <div className="stat-card resolved">
          <div className="stat-value">{stats.resolved}</div>
          <div className="stat-label">已自愈</div>
        </div>
      </div>

      <div className="dash-charts">
        <div className="chart-card">
          <div className="chart-title">自愈率</div>
          <div className="heal-rate">
            <div className="heal-ring">
              <svg viewBox="0 0 100 100">
                <circle
                  cx="50" cy="50" r="40"
                  fill="none" stroke="#f0f0f0" strokeWidth="10"
                />
                <circle
                  cx="50" cy="50" r="40"
                  fill="none"
                  stroke="#52c41a"
                  strokeWidth="10"
                  strokeDasharray={`${selfHealRate * 2.51} 251`}
                  strokeLinecap="round"
                  transform="rotate(-90 50 50)"
                />
              </svg>
              <div className="heal-text">
                <span className="heal-num">{selfHealRate}%</span>
                <span className="heal-label">成功率</span>
              </div>
            </div>
            <div className="heal-detail">
              <div className="heal-row">
                <span>平均处理耗时</span>
                <span className="heal-val">
                  {stats.avg_duration_ms > 0
                    ? `${(stats.avg_duration_ms / 1000).toFixed(1)}s`
                    : '-'}
                </span>
              </div>
            </div>
          </div>
        </div>

        <div className="chart-card">
          <div className="chart-title">告警来源分布</div>
          <div className="source-bars">
            <div className="source-row">
              <span className="source-name">🐳 Docker</span>
              <div className="bar-wrap">
                <div
                  className="bar-fill docker"
                  style={{ width: pct(stats.from_docker, stats.total) }}
                />
              </div>
              <span className="bar-num">{stats.from_docker}</span>
            </div>
            <div className="source-row">
              <span className="source-name">📊 Prometheus</span>
              <div className="bar-wrap">
                <div
                  className="bar-fill prom"
                  style={{ width: pct(stats.from_prometheus, stats.total) }}
                />
              </div>
              <span className="bar-num">{stats.from_prometheus}</span>
            </div>
          </div>
        </div>
      </div>

      <div className="dash-summary">
        共 <strong>{stats.total}</strong> 条告警，
        已自愈 <strong style={{ color: '#52c41a' }}>{stats.resolved}</strong> 条，
        {stats.failed > 0 && (
          <>失败 <strong style={{ color: '#ff4d4f' }}>{stats.failed}</strong> 条</>
        )}
      </div>
    </div>
  )
}

function pct(val: number, total: number): string {
  if (total === 0) return '0%'
  return `${Math.max(5, Math.round((val / total) * 100))}%`
}
