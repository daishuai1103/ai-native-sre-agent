import { useEffect, useState } from 'react'
import './components.css'

interface Props {
  category: string
  source: string
  confidence: number
  alert: string
}

interface RouterStatus {
  onnx_available: boolean
  threshold: number
  model_path: string
}

const categoryLabels: Record<string, string> = {
  container: '容器资源',
  database: '数据库',
  cache: '缓存服务',
  network: '网络',
  web: 'Web 服务',
  unknown: '未知',
}

const sourceLabels: Record<string, { label: string; color: string }> = {
  rule: { label: '规则路由', color: '#6b7280' },
  'hybrid-rule': { label: '混合(规则)', color: '#f59e0b' },
  'onnx-lowconf': { label: 'ONNX(低置信)', color: '#ef4444' },
  onnx: { label: 'ONNX 模型', color: '#7c3aed' },
}

export default function RouterInline({ category, source, confidence, alert }: Props) {
  const [status, setStatus] = useState<RouterStatus | null>(null)

  useEffect(() => {
    fetch('/api/router/status')
      .then((r) => r.json())
      .then(setStatus)
      .catch(() => {})
  }, [])

  const src = sourceLabels[source] || { label: source, color: '#6b7280' }
  const cat = categoryLabels[category] || category || '分析中...'
  const confPercent = Math.round(confidence * 100)
  const confColor =
    confPercent >= 80 ? '#10b981' : confPercent >= 60 ? '#f59e0b' : '#ef4444'

  return (
    <div className="router-inline">
      <div className="router-inline-left">
        <div className="router-alert">
          <span className="router-alert-icon">🚨</span>
          <span className="router-alert-text" title={alert}>
            {alert || '等待告警...'}
          </span>
        </div>
        <div className="router-inline-row">
          <span className="router-tag category">
            <span className="tag-dot" style={{ background: confColor }} />
            {cat}
          </span>
          <span
            className="router-tag source"
            style={{ color: src.color, borderColor: src.color }}
          >
            {status?.onnx_available ? 'ONNX' : '规则模式'} · {src.label}
          </span>
          <div className="router-conf-wrap">
            <div className="router-conf-bar">
              <div
                className="router-conf-fill"
                style={{ width: `${confPercent}%`, backgroundColor: confColor }}
              />
            </div>
            <span className="router-conf-text" style={{ color: confColor }}>
              {confPercent}%
            </span>
          </div>
        </div>
      </div>
    </div>
  )
}
