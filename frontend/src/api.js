const API_ROOT = '/api/v1'

class APIError extends Error {
  constructor(status, payload) {
    super(payload?.error?.message || 'Не удалось выполнить запрос')
    this.name = 'APIError'
    this.status = status
    this.code = payload?.error?.code || 'unknown_error'
    this.requestId = payload?.error?.requestId || ''
  }
}

async function request(path, options = {}) {
  const { body, headers, ...requestOptions } = options
  const response = await fetch(`${API_ROOT}${path}`, {
    credentials: 'include',
    ...requestOptions,
    headers: {
      ...(body ? { 'Content-Type': 'application/json' } : {}),
      ...headers,
    },
    body: body ? JSON.stringify(body) : undefined,
  })

  if (response.status === 204) return null

  const payload = await response.json().catch(() => null)
  if (!response.ok) throw new APIError(response.status, payload)
  return payload
}

const period = (daysBack = 30, daysForward = 120) => {
  const from = new Date(Date.now() - daysBack * 86400000).toISOString()
  const to = new Date(Date.now() + daysForward * 86400000).toISOString()
  return `?from=${encodeURIComponent(from)}&to=${encodeURIComponent(to)}`
}

export const api = {
  login: (login, password) => request('/auth/login', { method: 'POST', body: { login, password } }),
  logout: () => request('/auth/logout', { method: 'POST' }),
  me: () => request('/me'),
  updateProfile: (payload) => request('/me', { method: 'PATCH', body: payload }),
  changePassword: (currentPassword, newPassword) => request('/auth/password', {
    method: 'POST',
    body: { currentPassword, newPassword },
  }),
  teachers: () => request('/teachers'),
  students: () => request('/students'),
  bookings: () => request(`/bookings${period()}`),
  availability: (teacherId) => request(`/teachers/${teacherId}/availability${period()}`),
  slots: (teacherId) => request(`/teachers/${teacherId}/slots${period(0, 30)}`),
  createBooking: (payload) => request('/bookings', {
    method: 'POST',
    headers: { 'Idempotency-Key': crypto.randomUUID() },
    body: payload,
  }),
  bookingAction: (id, action) => request(`/bookings/${id}/${action}`, { method: 'POST' }),
  createAvailability: (payload) => request('/availability', { method: 'POST', body: payload }),
  conferenceLinks: (teacherId) => request(`/teachers/${teacherId}/conference-links`),
  updateConferenceLink: (teacherId, linkId, payload) => request(
    `/teachers/${teacherId}/conference-links/${linkId}`,
    { method: 'PATCH', body: payload },
  ),
  createConferenceLink: (teacherId, payload) => request(
    `/teachers/${teacherId}/conference-links`,
    { method: 'POST', body: payload },
  ),
  users: () => request('/admin/users'),
  createUser: (payload) => request('/admin/users', { method: 'POST', body: payload }),
}

export { APIError }
