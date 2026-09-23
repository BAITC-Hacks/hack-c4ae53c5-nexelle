import * as mockApi from './mockApi'
import { requestJson } from './apiClient'

const configuredMode = (import.meta.env.VITE_API_MODE ?? 'mock').toLowerCase()
export const apiMode = configuredMode === 'real' ? 'real' : 'mock'

const encodeId = (value) => encodeURIComponent(value)

// These paths are a proposed integration surface. They are called only when
// VITE_API_MODE=real; mock mode remains the default for the standalone demo.
const realApi = {
  getEmployeeProfile: (employeeId) => requestJson(`/employees/${encodeId(employeeId)}/profile`),
  getEmployeeHistory: (employeeId) => requestJson(`/employees/${encodeId(employeeId)}/history`),
  getRecommendations: (employeeId) => requestJson(`/employees/${encodeId(employeeId)}/recommendations`),
  completeActivity: (employeeId, eventId) => requestJson(`/employees/${encodeId(employeeId)}/activities/${encodeId(eventId)}/complete`, { method: 'POST' }),
  getHRDashboard: () => requestJson('/hr/dashboard'),
}

const activeApi = apiMode === 'real' ? realApi : mockApi

export const getEmployeeProfile = (employeeId) => activeApi.getEmployeeProfile(employeeId)
export const getEmployeeHistory = (employeeId) => activeApi.getEmployeeHistory(employeeId)
export const getRecommendations = (employeeId) => activeApi.getRecommendations(employeeId)
export const completeActivity = (employeeId, eventId) => activeApi.completeActivity(employeeId, eventId)
export const getHRDashboard = () => activeApi.getHRDashboard()
