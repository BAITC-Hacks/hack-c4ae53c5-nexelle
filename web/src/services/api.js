const API_BASE_URL = import.meta.env.VITE_API_BASE_URL ?? 'http://localhost:8080'
export const DEMO_EMPLOYEE_ID = import.meta.env.VITE_DEMO_EMPLOYEE_ID ?? 'E0001'

export class ApiError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiError'
    this.status = status
  }
}

async function request(path, { role, employeeId, method = 'GET', body } = {}) {
  const headers = { 'X-Demo-Role': role }
  if (role === 'employee') headers['X-Employee-ID'] = employeeId
  if (body) headers['Content-Type'] = 'application/json'

  const response = await fetch(`${API_BASE_URL}${path}`, {
    method,
    headers,
    body: body ? JSON.stringify(body) : undefined,
  })
  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    throw new ApiError(payload?.error?.message ?? `HTTP ${response.status}`, response.status)
  }
  return payload
}

function humanSkillName(skill) {
  return skill.skillName || skill.skillId
}

function normalizeRecommendation(item, locale) {
  const skills = item.evidence?.skills ?? []
  const history = item.evidence?.history ?? {}
  const availability = item.nextSessionDate || (item.format === 'self_paced' ? 'Self-paced' : '—')
  const evidence = skills.slice(0, 1).map((skill) => ({
    factor: 'skill_gaps',
    detail: `${humanSkillName(skill)}: ${skill.currentLevel}/${skill.requiredLevel}${skill.isCritical ? ' · critical' : ''}`,
  }))
  if (history.completedCount > 0 || history.negativeCount > 0) {
    evidence.push({
      factor: 'activity_history',
      detail: `completed: ${history.completedCount ?? 0}; negative: ${history.negativeCount ?? 0}`,
    })
  }
  evidence.push({ factor: 'availability', detail: availability })

  return {
    event_id: item.eventId,
    title: item.title,
    format: item.format,
    score_breakdown: {
      skill_gaps: item.score?.gapScore ?? 0,
      career_goal: item.score?.goalMatchScore ?? 0,
      activity_history: item.score?.historyScore ?? 0,
      availability: item.score?.feasibilityScore ?? 0,
    },
    skill_gaps: skills.map(humanSkillName),
    eligibility: { status: 'eligible', prerequisites: [] },
    expected_gain: skills.map((skill) => ({ skill: humanSkillName(skill), level_delta: skill.gain })),
    availability: { date: availability, status: 'available' },
    explanation: {
      why: item.explanations?.[locale] ?? item.explanation,
      evidence,
    },
  }
}

function normalizeProfile(payload, locale) {
  const target = payload.progress?.target ?? {}
  const employee = payload.employee ?? {}
  const activities = payload.activities ?? []
  const skillGaps = payload.progress?.skillGaps ?? []
  const completedActivities = activities.filter((item) => item.status === 'completed').length

  return {
    employee: {
      employee_id: employee.employeeId,
      name: employee.fullName,
      role: employee.role,
      grade: employee.grade,
      career_goal: {
        target_role: target.role ?? employee.careerGoal?.targetRole ?? employee.role,
        target_grade: target.grade ?? employee.careerGoal?.targetGrade ?? employee.grade,
        target_date: '—',
      },
      skills: skillGaps.map((gap) => ({
        name: humanSkillName(gap),
        current_level: gap.currentLevel,
        required_level: gap.requiredLevel,
        gap: gap.gap,
      })),
      history: activities.map((activity) => ({
        activity_id: activity.recordId,
        title: activity.eventTitle || activity.eventId,
        format: activity.eventFormat,
        date: activity.date,
        status: activity.status,
      })),
      progress: {
        completed_activities: completedActivities,
        total_activities: activities.length,
        percentage: Math.round(payload.progress?.readinessPercent ?? 0),
      },
    },
    recommendations: (payload.recommendations ?? []).map((item) => normalizeRecommendation(item, locale)),
  }
}

function normalizeHRDashboard(payload) {
  const activityParticipation = (payload.participationByEvent ?? []).reduce(
    (result, item) => ({
      completed: result.completed + (item.completed ?? 0),
      no_show: result.no_show + (item.noShow ?? 0),
      dropped: result.dropped + (item.dropped ?? 0),
      declined: result.declined + (item.declined ?? 0),
    }),
    { completed: 0, no_show: 0, dropped: 0, declined: 0 },
  )
  const withoutNextStep = payload.employeesWithoutRecommendation ?? []
  const totalEmployees = payload.employeeCount ?? withoutNextStep.length

  return {
    total_employees: totalEmployees,
    employees_in_development: Math.max(0, totalEmployees - withoutNextStep.length),
    employees_without_recommendation: withoutNextStep.length,
    skill_gaps: (payload.topSkillGaps ?? []).map((item) => ({
      skill: item.skillName || item.skillId,
      employees: item.employeeCount,
    })),
    activity_participation: activityParticipation,
  }
}

export async function getEmployeeProfile(employeeId = DEMO_EMPLOYEE_ID, locale = 'ru') {
  const payload = await request(`/api/employees/${encodeURIComponent(employeeId)}`, {
    role: 'employee',
    employeeId,
  })
  return normalizeProfile(payload, locale)
}

export async function completeActivity(employeeId, eventId, locale = 'ru') {
  const payload = await request(`/api/employees/${encodeURIComponent(employeeId)}/complete`, {
    role: 'employee',
    employeeId,
    method: 'POST',
    body: { eventId },
  })
  return normalizeProfile(payload.profile, locale)
}

export async function getHRDashboard() {
  const payload = await request('/api/hr/analytics', { role: 'hr' })
  return normalizeHRDashboard(payload)
}
