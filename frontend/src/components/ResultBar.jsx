export default function ResultBar({ result, maxVotes }) {
  const { option, votes, percentage } = result
  const pct = Math.round(percentage ?? 0)
  const width = maxVotes > 0 ? Math.round(((votes ?? 0) / maxVotes) * 100) : 0
  const barWidth = percentage != null && percentage >= 0 ? pct : width

  return (
    <div className="result-bar">
      <div className="result-bar-meta">
        <span className="result-bar-option-label">{option}</span>
        <span className="result-bar-stats">
          <strong>{votes ?? 0}</strong> vote{votes === 1 ? '' : 's'} · {pct}%
        </span>
      </div>
      <div className="result-bar-track">
        <div
          className="result-bar-fill"
          style={{ width: `${barWidth}%` }}
        />
      </div>
    </div>
  )
}