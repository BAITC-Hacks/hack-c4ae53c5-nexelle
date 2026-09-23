export default function ProgressBar({ value, tone = 'blue', ariaLabel }) {
  const safeValue = Math.max(0, Math.min(100, value))

  return (
    <div className={`progress-track progress-track--${tone}`} role="progressbar" aria-label={ariaLabel} aria-valuenow={safeValue} aria-valuemin="0" aria-valuemax="100">
      <span className="progress-track__fill" style={{ width: `${safeValue}%` }} />
    </div>
  )
}
