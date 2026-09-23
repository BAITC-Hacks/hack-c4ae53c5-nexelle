import { employeeProfileDto, hrDashboardDto, recommendationsDto } from '../mocks/data'

const clone = (value) => JSON.parse(JSON.stringify(value))
const wait = (milliseconds = 380) => new Promise((resolve) => setTimeout(resolve, milliseconds))
const storageKey = 'career-quest.mock-api.v1'

const createInitialDatabase = () => ({
  employee: clone(employeeProfileDto),
  recommendations: clone(recommendationsDto),
  hr: clone(hrDashboardDto),
})

function loadMockDatabase() {
  if (typeof window === 'undefined') return createInitialDatabase()

  try {
    const savedDatabase = window.localStorage.getItem(storageKey)
    if (savedDatabase) return JSON.parse(savedDatabase)
  } catch {
    // An invalid or unavailable browser storage must not break the demo.
  }

  return createInitialDatabase()
}

function persistMockDatabase() {
  if (typeof window === 'undefined') return

  try {
    window.localStorage.setItem(storageKey, JSON.stringify(mockDatabase))
  } catch {
    // The in-memory mock remains functional when storage is unavailable.
  }
}

// This module mirrors the boundary of a future HTTP client. Components only use api.js.
const mockDatabase = loadMockDatabase()

export async function getEmployeeProfile(_employeeId) {
  await wait()
  return clone(mockDatabase.employee)
}

export async function getEmployeeHistory(_employeeId) {
  await wait()
  return clone(mockDatabase.employee.history)
}

export async function getRecommendations(_employeeId) {
  await wait()
  return clone(mockDatabase.recommendations.filter((item) => !item.completed))
}

export async function completeActivity(_employeeId, eventId) {
  await wait(520)

  const recommendation = mockDatabase.recommendations.find((item) => item.event_id === eventId)
  if (!recommendation || recommendation.completed) {
    throw new Error('Activity is unavailable')
  }

  recommendation.completed = true
  mockDatabase.employee.history.unshift({
    activity_id: eventId,
    title: recommendation.title,
    format: recommendation.event_type,
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
  persistMockDatabase()

  return {
    employee: clone(mockDatabase.employee),
    completed_activity: clone(recommendation),
  }
}

export async function getHRDashboard() {
  await wait()
  return clone(mockDatabase.hr)
}
