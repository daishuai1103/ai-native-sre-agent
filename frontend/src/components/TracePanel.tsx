import type { StepEvent } from '../App'
import './components.css'

interface Props {
  events: StepEvent[]
}

const agentLabels: Record<string, string> = {
  observer: '观测者',
  diagnosis: '诊断师',
  planner: '规划师',
  executor: '执行者',
  reporter: '报告员',
}

const stepTypeLabels: Record<string, { label: string; icon: string; color: string }> = {
  thought: { label: '思考', icon: '💭', color: '#722ed1' },
  tool_call: { label: '调用工具', icon: '🔧', color: '#1890ff' },
  tool_result: { label: '工具返回', icon: '📦', color: '#13c2c2' },
  final: { label: '结论', icon: '✅', color: '#52c41a' },
  fallback: { label: '降级输出', icon: '⚠️', color: '#faad14' },
  error: { label: '错误', icon: '❌', color: '#ff4d4f' },
}

export default function TracePanel({ events }: Props) {
  return (
    <div className="trace-panel">
      <div className="panel-header">
        <h3>Agent 执行链路</h3>
        <span className="trace-count">{events.length} 步</span>
      </div>
      <div className="trace-list">
        {events.length === 0 && <p className="trace-empty">暂无执行步骤，发送告警后开始...</p>}
        {events.map((ev, i) => {
          // Agent lifecycle events
          if (ev.type === 'step_started') {
            return (
              <div key={i} className="trace-agent-header">
                <span className="trace-agent-dot" />
                <span className="trace-agent-name">
                  {agentLabels[ev.agent || ''] || ev.agent}
                </span>
                <span className="trace-agent-status">启动中...</span>
              </div>
            )
          }
          if (ev.type === 'agent_completed') {
            return (
              <div key={i} className="trace-agent-header done">
                <span className="trace-agent-dot done" />
                <span className="trace-agent-name">
                  {agentLabels[ev.agent || ''] || ev.agent}
                </span>
                <span className="trace-agent-status done">完成 ✓</span>
              </div>
            )
          }
          if (ev.type === 'agent_failed') {
            return (
              <div key={i} className="trace-agent-header failed">
                <span className="trace-agent-dot failed" />
                <span className="trace-agent-name">
                  {agentLabels[ev.agent || ''] || ev.agent}
                </span>
                <span className="trace-agent-status failed">失败</span>
              </div>
            )
          }
          if (ev.type === 'report_generated') {
            return (
              <div key={i} className="trace-report-banner">
                📄 报告已生成
              </div>
            )
          }

          // ReAct step events
          if (ev.type === 'react_step' && ev.payload) {
            const stepType = ev.payload.step_type as string
            const info = stepTypeLabels[stepType] || { label: stepType, icon: '•', color: '#666' }
            const content = ev.payload.content as string
            const toolName = ev.payload.tool_name as string
            const toolArgs = ev.payload.tool_args as string
            const toolResult = ev.payload.tool_result as string

            return (
              <div key={i} className={`trace-step-item step-${stepType}`}>
                <div className="step-header" style={{ borderLeftColor: info.color }}>
                  <span className="step-icon">{info.icon}</span>
                  <span className="step-label" style={{ color: info.color }}>
                    {info.label}
                  </span>
                  {toolName && <span className="step-tool-name">{toolName}</span>}
                </div>
                {content && stepType === 'thought' && (
                  <div className="step-content thought">{content}</div>
                )}
                {content && stepType === 'final' && (
                  <div className="step-content final">{content}</div>
                )}
                {content && stepType === 'fallback' && (
                  <div className="step-content fallback">{content}</div>
                )}
                {toolArgs && stepType === 'tool_call' && (
                  <div className="step-code">
                    <span className="code-label">参数</span>
                    <pre>{formatJSON(toolArgs)}</pre>
                  </div>
                )}
                {toolResult && stepType === 'tool_result' && (
                  <div className="step-code result">
                    <span className="code-label">结果</span>
                    <pre>{truncate(toolResult, 500)}</pre>
                  </div>
                )}
              </div>
            )
          }

          // Fallback: show raw event
          return (
            <div key={i} className="trace-item">
              <div className="trace-meta">
                <span className="trace-agent">{ev.agent || 'system'}</span>
                <span className="trace-type">{ev.type}</span>
              </div>
              <div className="trace-step">{ev.step}</div>
            </div>
          )
        })}
      </div>
    </div>
  )
}

function formatJSON(args: string): string {
  try {
    const obj = JSON.parse(args)
    return JSON.stringify(obj, null, 2)
  } catch {
    return args
  }
}

function truncate(s: string, max: number): string {
  if (s.length <= max) return s
  return s.slice(0, max) + `\n... (truncated, total ${s.length} chars)`
}
