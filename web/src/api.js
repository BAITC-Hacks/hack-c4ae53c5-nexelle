import { createMockApi } from './mock.js';

// No backend DTOs or endpoint definitions exist yet. Keep mock mode enabled
// until the backend team publishes its contract; do not infer business rules.
const USE_MOCK = new URLSearchParams(location.search).get('mock') !== 'false';
const mock = createMockApi();

export const api = USE_MOCK ? mock : {
  async getCareerPath() { return request('/api/me/career-path'); },
  async getHRDashboard() { return request('/api/hr/dashboard'); },
  async completeActivity(id) { return request(`/api/activities/${encodeURIComponent(id)}/complete`, { method: 'POST' }); },
  async resetDemo() { return mock.resetDemo(); },
};

async function request(path, options = {}) {
  const response = await fetch(path, { ...options, headers: { Accept: 'application/json', ...options.headers } });
  if (!response.ok) throw new Error(`Request failed (${response.status})`);
  return response.json();
}
