import RecommendationCard from '../components/RecommendationCard'
import SkillProgress from '../components/SkillProgress'

export default function CareerPath({ employee, recommendations, t, onComplete, completingId }) {
  return (
    <div className="page-stack">
      <section className="page-title">
        <p className="page-overline">{t('careerPath')}</p>
        <h1>{t('careerTrajectory')}</h1>
        <p>{employee.role} · {employee.career_goal.target_role}</p>
      </section>

      <section className="surface-card trajectory-card">
        <div className="trajectory-card__line" aria-hidden="true" />
        <div className="trajectory-step trajectory-step--current">
          <span className="trajectory-step__marker">●</span>
          <p>{t('current')}</p>
          <h2>{employee.grade}</h2>
          <strong>{employee.role}</strong>
        </div>
        <div className="trajectory-arrow" aria-hidden="true">→</div>
        <div className="trajectory-step trajectory-step--target">
          <span className="trajectory-step__marker">✦</span>
          <p>{t('nextStep')}</p>
          <h2>{employee.career_goal.target_grade}</h2>
          <strong>{employee.career_goal.target_role}</strong>
          <em>{employee.career_goal.target_date}</em>
        </div>
      </section>

      <section className="content-grid content-grid--path">
        <article className="surface-card path-skills">
          <div className="section-heading"><div><p className="section-kicker">{t('careerGoal')}</p><h2>{t('requiredSkills')}</h2></div></div>
          <SkillProgress skills={employee.skills} t={t} />
        </article>
        <article className="surface-card path-guide">
          <p className="section-kicker">{t('nextStep')}</p>
          <h2>{employee.career_goal.target_grade}</h2>
          <p>{employee.career_goal.target_role}</p>
          <div className="path-guide__rule" />
          <span>{t('targetDate')}</span>
          <strong>{employee.career_goal.target_date}</strong>
          <p className="muted">{t('focusText')}</p>
        </article>
      </section>

      {recommendations[0] && <section className="career-recommendation">
        <div className="section-heading"><div><p className="section-kicker">✦ {t('aiRecommendation')}</p><h2>{t('nextStep')}</h2></div></div>
        <div className="recommendation-grid recommendation-grid--single">
          <RecommendationCard recommendation={recommendations[0]} t={t} onComplete={onComplete} completing={completingId === recommendations[0].event_id} />
        </div>
      </section>}
    </div>
  )
}
