// api-client-msw.test.ts — MSW-based integration tests for the API client (network-level mocking)
import { describe, it, expect, beforeAll, afterAll, afterEach } from 'vitest'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import { api, ApiClientError } from '../api/client'

const BASE = 'http://localhost:3000'

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'error' }))
afterEach(() => server.resetHandlers())
afterAll(() => server.close())

describe('api client (MSW integration)', () => {
  describe('401 unauthorized handling', () => {
    it('throws ApiClientError with status 401 and parsed error code', async () => {
      server.use(
        http.get(`${BASE}/api/protected`, () => {
          return HttpResponse.json(
            { code: 'UNAUTHORIZED', message: 'Login required' },
            { status: 401 },
          )
        }),
      )

      try {
        await api.get(`${BASE}/api/protected`)
        expect.fail('Should have thrown')
      } catch (err) {
        expect(err).toBeInstanceOf(ApiClientError)
        const apiErr = err as ApiClientError
        expect(apiErr.status).toBe(401)
        expect(apiErr.code).toBe('UNAUTHORIZED')
        expect(apiErr.message).toBe('Login required')
      }
    })
  })

  describe('500 server error handling', () => {
    it('throws ApiClientError with status 500 and parsed error body', async () => {
      server.use(
        http.get(`${BASE}/api/broken`, () => {
          return HttpResponse.json(
            { code: 'INTERNAL_ERROR', message: 'Something went wrong' },
            { status: 500 },
          )
        }),
      )

      try {
        await api.get(`${BASE}/api/broken`)
        expect.fail('Should have thrown')
      } catch (err) {
        expect(err).toBeInstanceOf(ApiClientError)
        const apiErr = err as ApiClientError
        expect(apiErr.status).toBe(500)
        expect(apiErr.code).toBe('INTERNAL_ERROR')
        expect(apiErr.message).toBe('Something went wrong')
      }
    })

    it('falls back to UNKNOWN code when 500 body is not JSON', async () => {
      server.use(
        http.get(`${BASE}/api/crash`, () => {
          return new HttpResponse('Internal Server Error', {
            status: 500,
            headers: { 'Content-Type': 'text/plain' },
          })
        }),
      )

      try {
        await api.get(`${BASE}/api/crash`)
        expect.fail('Should have thrown')
      } catch (err) {
        expect(err).toBeInstanceOf(ApiClientError)
        const apiErr = err as ApiClientError
        expect(apiErr.status).toBe(500)
        expect(apiErr.code).toBe('UNKNOWN')
      }
    })
  })

  describe('204 empty response', () => {
    it('returns undefined for 204 No Content', async () => {
      server.use(
        http.delete(`${BASE}/api/items/1`, () => {
          return new HttpResponse(null, { status: 204 })
        }),
      )

      const result = await api.del(`${BASE}/api/items/1`)
      expect(result).toBeUndefined()
    })
  })

  describe('network error handling', () => {
    it('propagates network errors as-is (not ApiClientError)', async () => {
      server.use(
        http.get(`${BASE}/api/offline`, () => {
          return HttpResponse.error()
        }),
      )

      try {
        await api.get(`${BASE}/api/offline`)
        expect.fail('Should have thrown')
      } catch (err) {
        // Network errors are raw fetch errors, not ApiClientError
        expect(err).not.toBeInstanceOf(ApiClientError)
        expect(err).toBeInstanceOf(Error)
      }
    })
  })

  describe('successful responses', () => {
    it('returns parsed JSON for a successful GET', async () => {
      const mockData = { id: 42, name: 'Test Item' }
      server.use(
        http.get(`${BASE}/api/items/42`, () => {
          return HttpResponse.json(mockData)
        }),
      )

      const result = await api.get<{ id: number; name: string }>(`${BASE}/api/items/42`)
      expect(result).toEqual(mockData)
    })

    it('sends JSON body for a successful POST', async () => {
      const payload = { name: 'New Item' }
      const response = { id: 1, name: 'New Item' }

      server.use(
        http.post(`${BASE}/api/items`, async ({ request }) => {
          const body = await request.json()
          expect(body).toEqual(payload)
          return HttpResponse.json(response, { status: 201 })
        }),
      )

      const result = await api.post<{ id: number; name: string }>(`${BASE}/api/items`, payload)
      expect(result).toEqual(response)
    })
  })
})
