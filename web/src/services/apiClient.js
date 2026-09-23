const configuredBaseUrl = (import.meta.env.VITE_API_BASE_URL ?? '').replace(/\/$/, '')

export class ApiClientError extends Error {
  constructor(message, status) {
    super(message)
    this.name = 'ApiClientError'
    this.status = status
  }
}

function buildUrl(path) {
  if (!configuredBaseUrl) {
    throw new ApiClientError('VITE_API_BASE_URL is required when VITE_API_MODE=real')
  }

  return `${configuredBaseUrl}${path}`
}

export async function requestJson(path, options = {}) {
  const response = await fetch(buildUrl(path), {
    ...options,
    headers: {
      Accept: 'application/json',
      ...(options.body ? { 'Content-Type': 'application/json' } : {}),
      ...options.headers,
    },
  })

  if (response.status === 204) return null

  const payload = await response.json().catch(() => null)
  if (!response.ok) {
    throw new ApiClientError(payload?.message ?? `Request failed with status ${response.status}`, response.status)
  }

  return payload
}
