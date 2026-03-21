import ky from 'ky'
import type { ApiError } from '@/types'
import { apiBaseUrl } from '@/lib/apiBase'

export class ApiRequestError extends Error {
  code: string
  details: Record<string, unknown>
  status: number

  constructor(error: ApiError, status: number) {
    super(error.message)
    this.name = 'ApiRequestError'
    this.code = error.code
    this.details = error.details
    this.status = status
  }
}

export const api = ky.create({
  prefixUrl: apiBaseUrl,
  hooks: {
    beforeRequest: [
      (request) => {
        const token = localStorage.getItem('auth-token')
        if (token) {
          request.headers.set('Authorization', `Bearer ${token}`)
        }
      },
    ],
    afterResponse: [
      async (_request, _options, response) => {
        if (!response.ok) {
          if (response.status === 401) {
            localStorage.removeItem('auth-token')
            localStorage.removeItem('auth-storage')
            if (window.location.pathname !== '/login' && window.location.pathname !== '/register') {
              window.location.href = '/login'
            }
          }
          let apiError: ApiError | undefined
          try {
            const body = await response.json<{ error: ApiError }>()
            apiError = body.error
          } catch {
            // response body is not valid JSON or not in expected format
          }
          throw new ApiRequestError(
            apiError ?? { code: 'UNKNOWN', message: `请求失败 (${response.status})`, details: {} },
            response.status,
          )
        }
      },
    ],
  },
})
