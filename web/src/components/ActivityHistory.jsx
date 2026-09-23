const formatKey = { Course: 'course', Workshop: 'workshop', Event: 'event', Mentoring: 'mentoring' }

export default function ActivityHistory({ history, t }) {
  return (
    <div className="history-list">
      {history.map((item) => (
        <article className="history-item" key={item.activity_id}>
          <span className={`history-item__dot history-item__dot--${item.status}`} aria-hidden="true" />
          <div className="history-item__body">
            <strong>{item.title}</strong>
            <span>{t(formatKey[item.format] ?? 'event')} · {item.date === 'today' ? t('today') : item.date}</span>
          </div>
          <span className={`status status--${item.status}`}>{t(item.status)}</span>
        </article>
      ))}
    </div>
  )
}
