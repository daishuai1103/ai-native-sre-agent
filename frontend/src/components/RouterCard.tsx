import { useEffect, useState } from 'react'
import './components.css'

interface RouterStatus {
  onnx_available: boolean
  threshold: number
  model_path: string
}

interface RouterCardProps {
  category: string
  source: string
  confidence: number
}

const categoryLabels: Record<string, string> = {
  cache: '缓存服务',
  container: '容器资源',
  db: '数据库',
  network: '网络',
  web: 'Web 服务',
  unknown: '未知',
}

const sourceLabels: Record<string, { label: string; color: string }> = {
  onnx: { label: 'ONNX 模型', color: '#4f46e5' },
  'onnx-lowconf': { label: 'ONNX(低置信)', color: '#f59e0b' },
  rule: { label: '规则路由', color: '#6b7280' },
  'hybrid-rule': { label: '混合(规则)', color: '#0891b2' },
}

export default function RouterCard({ category, source, confidence }: RouterCardProps) {
  const [status, setStatus] = useState<RouterStatus | null>(null)

  useEffect(() => {
    fetch('/api/router/status')
      .then((r) => r.json())
      .then(setStatus)
      .catch(() => {})
  }, [])

  const src = sourceLabels[source] || { label: source, color: '#6b7280' }
  const cat = categoryLabels[category] || category

  const confPercent = Math.round(confidence * 100)
  const confColor =
    confPercent >= 80 ? '#10b981' : confPercent >= 60 ? '#f59e0b' : '#ef4444'

  return (
    <div className="router-card">
      <div className="panel-header">
        <h3>智能路由</h3>
        {status?.onnx_available ? (
          <span className="badge onnx">ONNX</span>
        ) : (
          <span className="badge rule">规则模式</span>
        )}
      </div>

      <div className="router-body">
        <div className="router-row">
          <span className="router-label">分类结果</span>
          <span className="router-category">{cat}</span>
        </div>

        <div className="router-row">
          <span className="router-label">路由来源</span>
          <span className="router-source" style={{ color: src.color }}>
            {src.label}
          </span>
        </div>

        <div className="router-row">
          <span className="router-label">置信度</span>
          <div className="confidence-bar-wrap">
            <div
              className="confidence-bar"
              style={{ width: `${confPercent}%`, backgroundColor: confColor }}
            />
            <span className="confidence-text" style={{ color: confColor }}>
              {confPercent}%
            </span>
          </div>
        </div>

        {status && (
          <div className="router-meta">
            <span>阈值: {Math.round(status.threshold * 100)}%</span>
          </div>
        )}
      </div>
    </div>
  )
}
