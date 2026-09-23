import ActivityHistory from '../components/ActivityHistory'
import MetricCard from '../components/MetricCard'
import ProgressBar from '../components/ProgressBar'
import RecommendationCard from '../components/RecommendationCard'
import SkillProgress from '../components/SkillProgress'

export default function EmployeeDashboard({ employee, recommendations, t, onComplete, completingId, refreshing, onViewPath }) {
  const progress = employee.progress

  return (
    <div className="page-stack">
      <section className="profile-hero">
        <div className="profile-hero__identity">
          <div className="avatar" aria-hidden="true">АС</div>
          <div>
            <p className="page-overline">{t('employeeDashboard')}</p>
            <h1>{employee.name}</h1>
            <p>{employee.role} <span>·</span> {employee.grade}</p>
          </div>
        </div>
        <div className="goal-card">
          <span>{t('careerGoal')}</span>
          <strong>{employee.career_goal.target_role}</strong>
          <p>{employee.career_goal.target_grade} · {employee.career_goal.target_date}</p>
        </div>
      </section>

      <section className="metric-grid metric-grid--employee">
        <MetricCard label={t('overallProgress')} value={`${progress.percentage}%`} detail={`${progress.completed_activities}/${progress.total_activities} ${t('completedActivities').toLowerCase()}`} tone="blue" icon="↗" />
        <MetricCard label={t('currentGrade')} value={employee.grade.split(' · ')[0]} detail={employee.role} tone="violet" icon="◆" />
        <MetricCard label={t('nextStep')} value={employee.career_goal.target_grade.split(' · ')[0]} detail={employee.career_goal.target_role} tone="teal" icon="→" />
      </section>

      <section className="content-grid content-grid--overview">
        <article className="surface-card progress-summary">
          <div className="section-heading">
            <div><p className="section-kicker">{t('developmentPlan')}</p><h2>{t('overallProgress')}</h2></div>
            <strong className="progress-number">{progress.percentage}%</strong>
          </div>
          <ProgressBar value={progress.percentage} ariaLabel={t('overallProgress')} />
          <div className="progress-summary__footer">
            <span>{progress.completed_activities} {t('completed')}</span>
            <span>{progress.total_activities} {t('developmentPlan').toLowerCase()}</span>
          </div>
          <button className="text-button" onClick={onViewPath} type="button">{t('seeCareerPath')} <span>→</span></button>
        </article>

        <article className="surface-card skills-card">
          <div className="section-heading"><div><p className="section-kicker">{t('developmentSnapshot')}</p><h2>{t('skills')}</h2></div></div>
          <SkillProgress skills={employee.skills} t={t} compact />
        </article>
      </section>

      <section className="recommendations-section" id="recommendations">
        <div className="section-heading section-heading--wide">
          <div><p className="section-kicker">✦ {t('recommendations')}</p><h2>{t('recommendedForYou')}</h2></div>
          {refreshing && <span className="refreshing"><i />{t('refreshing')}</span>}
        </div>
        {recommendations.length > 0 ? (
          <div className="recommendation-grid">
            {recommendations.map((recommendation) => (
              <RecommendationCard key={recommendation.event_id} recommendation={recommendation} t={t} onComplete={onComplete} completing={completingId === recommendation.event_id} />
            ))}
          </div>
        ) : <div className="empty-state">✦ {t('noRecommendations')}</div>}
      </section>

      <section className="surface-card history-card">
        <div className="section-heading"><div><p className="section-kicker">{t('developmentPlan')}</p><h2>{t('history')}</h2></div></div>
        <ActivityHistory history={employee.history} t={t} />
      </section>
    </div>
  )
}
