export const employeeProfileDto = {
  employee_id: 'emp-003',
  name: 'Айым Сейтова',
  role: 'Product Analyst',
  grade: 'G2 · Middle',
  career_goal: {
    target_role: 'Senior Product Analyst',
    target_grade: 'G3 · Senior',
    target_date: 'Q4 2026',
  },
  skills: [
    { name: 'SQL', current_level: 2, required_level: 4, gap: 2 },
    { name: 'Product analytics', current_level: 3, required_level: 4, gap: 1 },
    { name: 'Experiment design', current_level: 2, required_level: 3, gap: 1 },
    { name: 'Stakeholder management', current_level: 3, required_level: 3, gap: 0 },
  ],
  history: [
    {
      activity_id: 'hist-1',
      title: 'Product Metrics Foundations',
      format: 'Course',
      date: '18.08.2026',
      status: 'completed',
    },
    {
      activity_id: 'hist-2',
      title: 'Analytics Community Meetup',
      format: 'Event',
      date: '02.09.2026',
      status: 'completed',
    },
    {
      activity_id: 'hist-3',
      title: 'SQL Window Functions Lab',
      format: 'Workshop',
      date: '12.09.2026',
      status: 'registered',
    },
  ],
  progress: {
    completed_activities: 3,
    total_activities: 7,
    percentage: 43,
  },
}

export const recommendationsDto = [
  {
    event_id: 'event-sql-lab',
    title: 'Advanced SQL: Window Functions Lab',
    event_type: 'Workshop',
    score_breakdown: {
      grade: 24,
      skill_gaps: 30,
      career_goal: 22,
      activity_history: 12,
      availability: 12,
    },
    skill_gaps: ['SQL'],
    prerequisites: ['Базовый SQL', 'Ноутбук с доступом к учебной среде'],
    eligibility: { status: 'eligible' },
    history_signals: ['Продолжает тему SQL после базового курса'],
    expected_gain: [{ skill: 'SQL', level_delta: 1 }],
    availability: {
      date: '26.09.2026 · 15:00',
      status: 'available',
    },
    explanation: {
      why: 'Практическая лаборатория закрывает самый большой skill gap на пути к G3.',
      evidence: [
        { factor: 'grade', detail: 'Соответствует уровню G2 и целевому G3.' },
        { factor: 'skill_gaps', detail: 'SQL — разрыв 2 уровня из требуемых 4.' },
        { factor: 'career_goal', detail: 'Навык указан в траектории Senior Product Analyst.' },
        { factor: 'activity_history', detail: 'Продолжает пройденный курс по метрикам.' },
        { factor: 'availability', detail: 'Есть свободное место в ближайшем потоке.' },
      ],
    },
  },
  {
    event_id: 'event-experiment',
    title: 'Designing Trustworthy A/B Tests',
    event_type: 'Course',
    score_breakdown: {
      grade: 22,
      skill_gaps: 25,
      career_goal: 25,
      activity_history: 16,
      availability: 12,
    },
    skill_gaps: ['Experiment design', 'Product analytics'],
    prerequisites: ['Знание продуктовых метрик'],
    eligibility: { status: 'eligible' },
    history_signals: ['Закрыл основы продуктовой аналитики'],
    expected_gain: [
      { skill: 'Experiment design', level_delta: 1 },
      { skill: 'Product analytics', level_delta: 1 },
    ],
    availability: {
      date: '01–08.10.2026',
      status: 'available',
    },
    explanation: {
      why: 'Курс укрепляет два навыка, необходимых для самостоятельного ведения экспериментов на следующем грейде.',
      evidence: [
        { factor: 'grade', detail: 'Рассчитан на специалистов уровня Middle.' },
        { factor: 'skill_gaps', detail: 'Закрывает два gap по карьерной матрице.' },
        { factor: 'career_goal', detail: 'Помогает перейти к ownership продуктовых решений.' },
        { factor: 'activity_history', detail: 'Следующий шаг после курса по метрикам.' },
        { factor: 'availability', detail: 'Регистрация открыта до 29 сентября.' },
      ],
    },
  },
  {
    event_id: 'event-mentor',
    title: 'Analytics Leadership Mentoring Circle',
    event_type: 'Mentoring',
    score_breakdown: {
      grade: 20,
      skill_gaps: 18,
      career_goal: 28,
      activity_history: 18,
      availability: 16,
    },
    skill_gaps: ['Stakeholder management'],
    prerequisites: ['Не менее одного завершённого аналитического проекта'],
    eligibility: { status: 'eligible' },
    history_signals: ['Уже завершены два профильных формата развития'],
    expected_gain: [{ skill: 'Stakeholder management', level_delta: 1 }],
    availability: {
      date: 'Старт 04.10.2026',
      status: 'limited',
    },
    explanation: {
      why: 'Менторский круг помогает подготовиться к росту ответственности и работе со стейкхолдерами.',
      evidence: [
        { factor: 'grade', detail: 'Подходит сотрудникам, которые готовятся к переходу G2 → G3.' },
        { factor: 'skill_gaps', detail: 'Поддерживает развитие ключевого поведенческого навыка.' },
        { factor: 'career_goal', detail: 'Развивает лидерский контекст будущей роли Senior.' },
        { factor: 'activity_history', detail: 'Дополняет технические форматы развития.' },
        { factor: 'availability', detail: 'В текущем наборе осталось два места.' },
      ],
    },
  },
]

export const hrDashboardDto = {
  total_employees: 128,
  employees_in_development: 83,
  employees_without_recommendation: 14,
  skill_gaps: [
    { skill: 'SQL', employees: 36 },
    { skill: 'Experiment design', employees: 29 },
    { skill: 'Product analytics', employees: 24 },
    { skill: 'Stakeholder management', employees: 18 },
  ],
  activity_participation: {
    completed: 67,
    no_show: 8,
    dropped: 11,
    declined: 19,
  },
}
