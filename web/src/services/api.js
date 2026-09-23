import * as mockApi from './mockApi'
import { requestJson } from './apiClient'

const configuredMode = (import.meta.env.VITE_API_MODE ?? 'mock').toLowerCase()
export const apiMode = configuredMode === 'real' ? 'real' : 'mock'

const mockDemoEmployeeId = 'emp-003'
const configuredRealDemoEmployeeId = (import.meta.env.VITE_DEMO_EMPLOYEE_ID ?? 'E0001').trim()
const realDemoEmployeeId = configuredRealDemoEmployeeId || 'E0001'

// A future authenticated identity provider can replace this demo value without
// changing components or API call sites.
export const demoEmployeeId = apiMode === 'real' ? realDemoEmployeeId : mockDemoEmployeeId

const encodeId = (value) => encodeURIComponent(value)
const toDisplayDate = (value) => typeof value === 'string' ? value.slice(0, 10) : ''

function frontendFormat(format) {
  if (format === 'self_paced') return 'Course'
  if (format === 'online') return 'Workshop'
  return 'Event'
}

function employeeHeaders(employeeId) {
  return {
    'X-Demo-Role': 'employee',
    'X-Employee-ID': employeeId,
  }
}

const hrHeaders = { 'X-Demo-Role': 'hr' }
const profileRequests = new Map()

function getBackendProfile(employeeId) {
  const existingRequest = profileRequests.get(employeeId)
  if (existingRequest) return existingRequest

  const request = requestJson(`/api/v1/employees/${encodeId(employeeId)}/profile`, {
    headers: employeeHeaders(employeeId),
  }).finally(() => profileRequests.delete(employeeId))

  profileRequests.set(employeeId, request)
  return request
}

function normalizeActivity(activity, metadata = {}) {
  return {
    activity_id: activity.recordId ?? activity.record_id,
    title: metadata.eventTitle ?? activity.eventTitle ?? activity.event_id ?? activity.eventId,
    format: frontendFormat(metadata.eventFormat ?? activity.eventFormat),
    date: toDisplayDate(activity.date),
    status: activity.status,
  }
}

function normalizeEmployee(payload) {
  const employee = payload.employee ?? {}
  const progress = payload.progress ?? {}
  const target = progress.target ?? employee.careerGoal ?? {}
  const activities = payload.activities ?? []
  const completedActivities = activities.filter((activity) => activity.status === 'completed').length

  return {
    employee_id: employee.employeeId,
    name: employee.fullName,
    role: employee.role,
    grade: employee.grade,
    career_goal: {
      target_role: target.role ?? target.targetRole ?? employee.role,
      target_grade: target.grade ?? target.targetGrade ?? employee.grade,
      target_date: '—',
    },
    skills: (progress.skillGaps ?? []).map((gap) => ({
      name: gap.skillName ?? gap.skillId,
      current_level: gap.currentLevel,
      required_level: gap.requiredLevel,
      gap: gap.gap,
    })),
    progress: {
      completed_activities: completedActivities,
      total_activities: activities.length,
      percentage: Math.round(progress.readinessPercent ?? 0),
    },
    history: activities.map((activity) => normalizeActivity(activity, activity)),
  }
}

function normalizeRecommendation(recommendation) {
  const skillEvidence = recommendation.evidence?.skills ?? []
  const history = recommendation.evidence?.history ?? {}
  const availability = recommendation.nextSessionDate || (recommendation.format === 'self_paced' ? 'Self-paced' : '—')
  const evidence = skillEvidence.slice(0, 1).map((skill) => ({
    factor: 'skill_gaps',
    detail: `${skill.skillName ?? skill.skillId}: ${skill.currentLevel}/${skill.requiredLevel}${skill.isCritical ? ' · critical' : ''}`,
  }))

  if (history.completedCount > 0 || history.negativeCount > 0) {
    evidence.push({
      factor: 'activity_history',
      detail: `completed: ${history.completedCount ?? 0}; negative: ${history.negativeCount ?? 0}`,
    })
  }

  evidence.push({ factor: 'availability', detail: availability })

  return {
    event_id: recommendation.eventId,
    title: recommendation.title,
    event_type: frontendFormat(recommendation.format),
    score_breakdown: {
      skill_gaps: recommendation.score?.gapScore ?? 0,
      career_goal: recommendation.score?.goalMatchScore ?? 0,
      activity_history: recommendation.score?.historyScore ?? 0,
      availability: recommendation.score?.feasibilityScore ?? 0,
    },
    skill_gaps: skillEvidence.map((skill) => skill.skillName ?? skill.skillId),
    prerequisites: [],
    eligibility: { status: 'eligible' },
    history_signals: [],
    expected_gain: skillEvidence.map((skill) => ({
      skill: skill.skillName ?? skill.skillId,
      level_delta: skill.gain,
    })),
    availability: { date: availability, status: 'available' },
    explanation: {
      why: recommendation.explanations?.ru ?? recommendation.explanation,
      evidence,
    },
  }
}

function normalizeHRDashboard(payload) {
  const activityParticipation = (payload.participationByEvent ?? []).reduce(
    (result, event) => ({
      completed: result.completed + (event.completed ?? 0),
      no_show: result.no_show + (event.noShow ?? 0),
      dropped: result.dropped + (event.dropped ?? 0),
      declined: result.declined + (event.declined ?? 0),
    }),
    { completed: 0, no_show: 0, dropped: 0, declined: 0 },
  )
  const withoutRecommendations = payload.employeesWithoutRecommendation ?? []
  const totalEmployees = payload.employeeCount ?? withoutRecommendations.length

  return {
    total_employees: totalEmployees,
    employees_in_development: Math.max(0, totalEmployees - withoutRecommendations.length),
    employees_without_recommendation: withoutRecommendations.length,
    skill_gaps: (payload.topSkillGaps ?? []).map((gap) => ({
      skill: gap.skillName ?? gap.skillId,
      employees: gap.employeeCount,
    })),
    activity_participation: activityParticipation,
  }
}

const realApi = {
  async getEmployeeProfile(employeeId) {
    return normalizeEmployee(await getBackendProfile(employeeId))
  },

  async getEmployeeHistory(employeeId) {
    const [history, profile] = await Promise.all([
      requestJson(`/employees/${encodeId(employeeId)}/history`, { headers: employeeHeaders(employeeId) }),
      getBackendProfile(employeeId),
    ])
    const activitiesById = new Map((profile.activities ?? []).map((activity) => [activity.recordId, activity]))

    return history.map((activity) => normalizeActivity(activity, activitiesById.get(activity.record_id)))
  },

  async getRecommendations(employeeId) {
    const profile = await getBackendProfile(employeeId)
    return (profile.recommendations ?? []).map(normalizeRecommendation)
  },

  async completeActivity(employeeId, eventId) {
    await requestJson(`/employees/${encodeId(employeeId)}/complete`, {
      method: 'POST',
      body: JSON.stringify({ eventId }),
      headers: employeeHeaders(employeeId),
    })
  },

  async getHRDashboard() {
    return normalizeHRDashboard(await requestJson('/hr/dashboard', { headers: hrHeaders }))
  },
}

const activeApi = apiMode === 'real' ? realApi : mockApi

export const getEmployeeProfile = (employeeId) => activeApi.getEmployeeProfile(employeeId)
export const getEmployeeHistory = (employeeId) => activeApi.getEmployeeHistory(employeeId)
export const getRecommendations = (employeeId) => activeApi.getRecommendations(employeeId)
export const completeActivity = (employeeId, eventId) => activeApi.completeActivity(employeeId, eventId)
export const getHRDashboard = () => activeApi.getHRDashboard()
