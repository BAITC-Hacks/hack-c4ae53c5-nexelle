import ProgressBar from './ProgressBar'

const formatKey = {
  Course: 'course',
  Workshop: 'workshop',
  Event: 'event',
  Mentoring: 'mentoring',
}

export default function RecommendationCard({ recommendation, t, onComplete, completing }) {
  const availabilityKey = recommendation.availability.status === 'limited' ? 'limited' : 'available'
  const scoreEntries = Object.entries(recommendation.score_breakdown)
  const eventType = recommendation.event_type ?? recommendation.format
  const prerequisites = recommendation.prerequisites ?? recommendation.eligibility?.prerequisites ?? []

  return (
    <article className="recommendation-card">
      <div className="recommendation-card__accent" />
      <div className="recommendation-card__header">
        <div>
          <div className="eyebrow-row">
            <span className="format-chip">{t(formatKey[eventType] ?? 'event')}</span>
            <span className="eligible-chip">● {t(recommendation.eligibility.status)}</span>
          </div>
          <h3>{recommendation.title}</h3>
        </div>
        <span className={`availability availability--${recommendation.availability.status}`}>
          {t(availabilityKey)}
        </span>
      </div>

      <div className="recommendation-card__availability">
        <span className="inline-label">{t('availability')}</span>
        <strong>{recommendation.availability.date}</strong>
      </div>

      <div className="recommendation-card__chips">
        <div>
          <span className="inline-label">{t('skillGap')}</span>
          <div className="chip-row">{recommendation.skill_gaps.map((gap) => <span className="tag tag--gap" key={gap}>{gap}</span>)}</div>
        </div>
        <div>
          <span className="inline-label">{t('expectedGain')}</span>
          <div className="chip-row">{recommendation.expected_gain.map(({ skill, level_delta }) => <span className="tag tag--gain" key={skill}>{skill} +{level_delta}</span>)}</div>
        </div>
      </div>

      <div className="prerequisites">
        <span className="inline-label">{t('prerequisites')}</span>
        <span>{prerequisites.join(' · ')}</span>
      </div>

      <section className="why-panel">
        <span className="section-kicker">✦ {t('whyThis')}</span>
        <p>{recommendation.explanation.why}</p>
        <div className="evidence-list">
          {recommendation.explanation.evidence.map((item) => (
            <div className="evidence-item" key={item.factor}>
              <span>{t(`factor_${item.factor}`)}</span>
              <p>{item.detail}</p>
            </div>
          ))}
        </div>
      </section>

      <section className="score-panel">
        <span className="inline-label">{t('scoreFactors')}</span>
        <div className="score-grid">
          {scoreEntries.map(([factor, score]) => (
            <div className="score-factor" key={factor}>
              <div><span>{t(`factor_${factor}`)}</span><b>{score}</b></div>
              <ProgressBar value={score * 3.33} tone="violet" ariaLabel={t(`factor_${factor}`)} />
            </div>
          ))}
        </div>
      </section>

      <button className="button button--primary recommendation-card__action" type="button" onClick={() => onComplete(recommendation.event_id)} disabled={completing}>
        {completing ? t('completing') : t('completeActivity')}
        <span aria-hidden="true">→</span>
      </button>
    </article>
  )
}
