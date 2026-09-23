export default function MetricCard({ label, value, detail, tone = 'blue', icon }) {
  return (
    <article className={`metric-card metric-card--${tone}`}>
      <div className="metric-card__topline">
        <span>{label}</span>
        {icon && <span className="metric-card__icon" aria-hidden="true">{icon}</span>}
      </div>
      <strong>{value}</strong>
      {detail && <p>{detail}</p>}
    </article>
  )
}
