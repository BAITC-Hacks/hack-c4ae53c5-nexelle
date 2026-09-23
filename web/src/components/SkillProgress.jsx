import ProgressBar from './ProgressBar'

export default function SkillProgress({ skills, t, compact = false }) {
  return (
    <div className={`skill-list ${compact ? 'skill-list--compact' : ''}`}>
      {skills.map((skill) => {
        const value = Math.round((skill.current_level / skill.required_level) * 100)
        return (
          <article className="skill-row" key={skill.name}>
            <div className="skill-row__header">
              <strong>{skill.name}</strong>
              <span className={skill.gap === 0 ? 'gap-pill gap-pill--covered' : 'gap-pill'}>
                {skill.gap === 0 ? t('covered') : `${t('skillGap')}: ${skill.gap}`}
              </span>
            </div>
            <ProgressBar value={value} ariaLabel={skill.name} />
            <div className="skill-row__levels">
              <span>{t('current')}: <b>{skill.current_level}</b></span>
              <span>{t('required')}: <b>{skill.required_level}</b></span>
            </div>
          </article>
        )
      })}
    </div>
  )
}
