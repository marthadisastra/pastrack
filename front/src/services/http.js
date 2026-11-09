// Central transport adapter. Backend routes and response envelopes can be changed here.
export const apiBaseURL = (
  import.meta.env.VITE_API_BASE_URL || 'http://localhost:8080/api'
).replace(/\/$/, '')
export class ApiError extends Error {
  constructor(message, status, details) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.details = details
  }
}
export async function request(
  path,
  { body, headers, baseURL = apiBaseURL, unauthorized = true, ...options } = {},
) {
  const response = await fetch(`${baseURL}${path}`, {
    credentials: 'include',
    signal: options.signal || AbortSignal.timeout(15000),
    ...options,
    headers: {
      Accept: 'application/json',
      ...(body !== undefined ? { 'Content-Type': 'application/json' } : {}),
      ...headers,
    },
    ...(body !== undefined ? { body: JSON.stringify(body) } : {}),
  })
  const payload = response.status === 204 ? null : await response.text()
  let data
  try {
    data = payload ? JSON.parse(payload) : null
  } catch {
    data = { message: payload }
  }
  if (!response.ok) {
    if (response.status === 401 && unauthorized)
      window.dispatchEvent(new Event('pastrack:session-expired'))
    const messages = {
      invalid_credentials: 'Email, username or password is incorrect.',
      authentication_required: 'Your session has expired. Sign in again.',
      password_recovery_unavailable:
        'Password recovery is temporarily unavailable. Please contact your administrator.',
      invalid_or_expired_reset_token:
        'This reset link has expired or has already been used. Request a new link.',
      password_policy_not_met:
        'Use 12–72 bytes, uppercase and lowercase letters, a number and a special character.',
      too_many_attempts: 'Too many attempts. Please wait before trying again.',
    }
    throw new ApiError(
      messages[data?.error?.code] ||
        data?.message ||
        data?.error?.message ||
        `Request failed (${response.status})`,
      response.status,
      data,
    )
  }
  return data
}
export const useMock = import.meta.env.VITE_USE_MOCK !== 'false'
