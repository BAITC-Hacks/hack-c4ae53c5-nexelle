import MetricCard from '../components/MetricCard'
import ProgressBar from '../components/ProgressBar'

export default function HRDashboard({ data, t }) {
  const participation = data.activity_participation
  const participationRows = [
    { key: 'completed', value: participation.completed, tone: 'blue' },
    { key: 'no_show', value: participation.no_show, tone: 'amber' },
    { key: 'dropped', value: participation.dropped, tone: 'rose' },
    { key: 'declined', value: participation.declined, tone: 'slate' },
  ]
  const participationTotal = participationRows.reduce((total, item) => total + item.value, 0)
  const completedRate = Math.round((participation.completed / participationTotal) * 100)
  const highestGap = Math.max(...data.skill_gaps.map((item) => item.employees))

  return (
    <div className="page-stack">
      <section className="page-title page-title--hr">
        <div>
          <p className="page-overline">{t('developmentSnapshot')}</p>
          <h1>{t('hrDashboard')}</h1>
          <p>{t('dataScope')}</p>
        </div>
        <span className="demo-badge">{t('demoMode')}</span>
      </section>

      <section className="metric-grid metric-grid--hr">
        <MetricCard label={t('totalEmployees')} value={data.total_employees} detail={t('developmentSnapshot')} tone="blue" icon="◉" />
        <MetricCard label={t('inDevelopment')} value={data.employees_in_development} detail={`${Math.round((data.employees_in_development / data.total_employees) * 100)}% ${t('totalEmployees').toLowerCase()}`} tone="teal" icon="↗" />
        <MetricCard label={t('withoutNextStep')} value={data.employees_without_recommendation} detail={t('focusArea')} tone="amber" icon="!" />
      </section>

      <section className="content-grid content-grid--hr">
        <article className="surface-card gaps-card">
          <div className="section-heading"><div><p className="section-kicker">{t('developmentSnapshot')}</p><h2>{t('commonSkillGaps')}</h2></div></div>
          <div className="gap-ranking">
            {data.skill_gaps.map((item, index) => (
              <div className="gap-ranking__row" key={item.skill}>
                <span className="gap-ranking__index">0{index + 1}</span>
                <div className="gap-ranking__skill">
                  <div><strong>{item.skill}</strong><span>{item.employees} {t('employees')}</span></div>
                  <ProgressBar value={(item.employees / highestGap) * 100} ariaLabel={item.skill} />
                </div>
              </div>
            ))}
          </div>
        </article>

        <article className="surface-card participation-card">
          <div className="section-heading"><div><p className="section-kicker">{t('activityParticipation')}</p><h2>{t('participationBreakdown')}</h2></div></div>
          <div className="participation-card__body">
            <div className="donut" style={{ background: `conic-gradient(#2f6bff 0 ${completedRate}%, #e5a837 ${completedRate}% ${completedRate + Math.round((participation.no_show / participationTotal) * 100)}%, #d35d71 ${completedRate + Math.round((participation.no_show / participationTotal) * 100)}% ${completedRate + Math.round(((participation.no_show + participation.dropped) / participationTotal) * 100)}%, #9da9bc ${completedRate + Math.round(((participation.no_show + participation.dropped) / participationTotal) * 100)}% 100%)` }}>
              <div><strong>{completedRate}%</strong><span>{t('completionRate')}</span></div>
            </div>
            <div className="participation-legend">
              {participationRows.map((item) => (
                <div key={item.key} className="participation-legend__item">
                  <span className={`legend-dot legend-dot--${item.tone}`} />
                  <span>{t(item.key === 'no_show' ? 'noShow' : item.key)}</span>
                  <strong>{item.value}</strong>
                </div>
              ))}
            </div>
          </div>
        </article>
      </section>

      <section className="hr-insight">
        <div className="hr-insight__icon" aria-hidden="true">✦</div>
        <div><p className="section-kicker">{t('hrInsight')}</p><h2>{t('focusArea')}</h2><p>{t('focusText')}</p></div>
        <div className="hr-insight__stat"><span>{t('withoutNextStep')}</span><strong>{data.employees_without_recommendation}</strong></div>
      </section>
    </div>
  )
}
