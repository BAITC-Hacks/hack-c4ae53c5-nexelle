import { employeeProfileDto, hrDashboardDto, recommendationsDto } from '../mocks/data'

const clone = (value) => JSON.parse(JSON.stringify(value))
const wait = (milliseconds = 380) => new Promise((resolve) => setTimeout(resolve, milliseconds))

// This module mirrors the boundary of a future HTTP client. Components only use these functions.
const mockDatabase = {
  employee: clone(employeeProfileDto),
  recommendations: clone(recommendationsDto),
  hr: clone(hrDashboardDto),
}

export async function getEmployeeProfile() {
  await wait()
  return clone(mockDatabase.employee)
}

export async function getRecommendations() {
  await wait()
  return clone(mockDatabase.recommendations.filter((item) => !item.completed))
}

export async function completeActivity(eventId) {
  await wait(520)

  const recommendation = mockDatabase.recommendations.find((item) => item.event_id === eventId)
  if (!recommendation || recommendation.completed) {
    throw new Error('Activity is unavailable')
  }

  recommendation.completed = true
  mockDatabase.employee.history.unshift({
    activity_id: eventId,
    title: recommendation.title,
    format: recommendation.format,
    date: 'today',
    status: 'completed',
  })

  recommendation.expected_gain.forEach(({ skill, level_delta }) => {
    const employeeSkill = mockDatabase.employee.skills.find((item) => item.name === skill)
    if (!employeeSkill) return
    employeeSkill.current_level = Math.min(employeeSkill.required_level, employeeSkill.current_level + level_delta)
    employeeSkill.gap = Math.max(0, employeeSkill.required_level - employeeSkill.current_level)
  })

  const progress = mockDatabase.employee.progress
  progress.completed_activities = Math.min(progress.total_activities, progress.completed_activities + 1)
  progress.percentage = Math.round((progress.completed_activities / progress.total_activities) * 100)
  mockDatabase.hr.activity_participation.completed += 1

  return {
    employee: clone(mockDatabase.employee),
    completed_activity: clone(recommendation),
  }
}

export async function getHRDashboard() {
  await wait()
  return clone(mockDatabase.hr)
}
