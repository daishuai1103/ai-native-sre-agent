import { useEffect, useRef } from 'react'
import './components.css'

interface Props {
  report: string
  sessionId: string
}

function renderParagraphs(text: string) {
  return text
    .split(/\n+/)
    .map((line) => line.trim())
    .filter(Boolean)
    .map((line, index) => <p key={index}>{line}</p>)
}

export default function ReportView({ report, sessionId }: Props) {
  const viewRef = useRef<HTMLDivElement>(null)

  useEffect(() => {
    if (report && viewRef.current) {
      viewRef.current.scrollTop = viewRef.current.scrollHeight
    }
  }, [report])

  const sections = report
    .split(/\n(?=##\s+)/)
    .map((section) => section.trim())
    .filter(Boolean)

  return (
    <div className="report-view" ref={viewRef}>
      <div className="report-header">
        <div>
          <h2>Final Report</h2>
          <p>{sessionId ? `Session: ${sessionId}` : 'Waiting for an alert to generate a report'}</p>
        </div>
        <span className={`report-status ${report ? 'ready' : 'idle'}`}>{report ? 'Ready' : 'Waiting'}</span>
      </div>
      {sections.length === 0 ? (
        <div className="report-empty">No report available.</div>
      ) : (
        sections.map((section, index) => {
          const lines = section.split('\n').map((line) => line.trim())
          const title = lines[0]?.replace(/^##\s*/, '') || `Section ${index + 1}`
          const body = lines.slice(1).join('\n').trim()

          return (
            <section key={index} className="report-section">
              <h2>{title}</h2>
              {body ? renderParagraphs(body) : null}
            </section>
          )
        })
      )}
    </div>
  )
}
