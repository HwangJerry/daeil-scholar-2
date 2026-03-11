// api-client.test.ts — Unit tests for the API client (fetch wrapper, error handling, 204 responses).
import { describe, it, expect, vi, beforeEach, afterEach } from 'vitest'
import { api, ApiClientError } from '../api/client'

// Helper to create a mock Response
function mockResponse(status: number, body?: unknown, ok?: boolean): Response {
  const isOk = ok ?? (status >= 200 && status < 300)
  return {
    ok: isOk,
    status,
    statusText: status === 204 ? 'No Content' : 'OK',
    json: vi.fn().mockResolvedValue(body),
    headers: new Headers(),
    redirected: false,
    type: 'basic' as ResponseType,
    url: '',
    clone: vi.fn(),
    body: null,
    bodyUsed: false,
    arrayBuffer: vi.fn(),
    blob: vi.fn(),
    formData: vi.fn(),
    text: vi.fn(),
    bytes: vi.fn(),
  } as unknown as Response
}

describe('ApiClientError', () => {
  it('stores status, code, and message', () => {
    const err = new ApiClientError(404, 'NOT_FOUND', 'Resource not found')
    expect(err).toBeInstanceOf(Error)
    expect(err.name).toBe('ApiClientError')
    expect(err.status).toBe(404)
    expect(err.code).toBe('NOT_FOUND')
    expect(err.message).toBe('Resource not found')
  })

  it('is catchable as an Error', () => {
    const err = new ApiClientError(500, 'INTERNAL', 'Server error')
    expect(err instanceof Error).toBe(true)
  })
})

describe('api client', () => {
  const originalFetch = globalThis.fetch

  beforeEach(() => {
    globalThis.fetch = vi.fn()
  })

  afterEach(() => {
    globalThis.fetch = originalFetch
  })

  describe('api.get', () => {
    it('calls fetch with GET method and credentials', async () => {
      const data = { id: 1, name: 'Test' }
      vi.mocked(globalThis.fetch).mockResolvedValue(mockResponse(200, data))

      const result = await api.get('/api/test')

      expect(globalThis.fetch).toHaveBeenCalledWith('/api/test', expect.objectContaining({
        method: 'GET',
        credentials: 'include',
      }))
      expect(result).toEqual(data)
    })
  })

  describe('api.post', () => {
    it('sends JSON body with POST method', async () => {
      const payload = { name: 'New Item' }
      const response = { id: 1, name: 'New Item' }
      vi.mocked(globalThis.fetch).mockResolvedValue(mockResponse(200, response))

      const result = await api.post('/api/items', payload)

      expect(globalThis.fetch).toHaveBeenCalledWith('/api/items', expect.objectContaining({
        method: 'POST',
        body: JSON.stringify(payload),
      }))
      expect(result).toEqual(response)
    })
  })

  describe('204 No Content handling', () => {
    it('returns undefined for 204 responses', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue(mockResponse(204, undefined))

      const result = await api.del('/api/items/1')

      expect(result).toBeUndefined()
    })
  })

  describe('error handling', () => {
    it('throws ApiClientError with parsed error body on non-ok response', async () => {
      const errorBody = { code: 'FORBIDDEN', message: 'Access denied' }
      vi.mocked(globalThis.fetch).mockResolvedValue(mockResponse(403, errorBody, false))

      await expect(api.get('/api/secret')).rejects.toThrow(ApiClientError)

      try {
        await api.get('/api/secret')
      } catch (err) {
        expect(err).toBeInstanceOf(ApiClientError)
        const apiErr = err as InstanceType<typeof ApiClientError>
        expect(apiErr.status).toBe(403)
        expect(apiErr.code).toBe('FORBIDDEN')
        expect(apiErr.message).toBe('Access denied')
      }
    })

    it('falls back to UNKNOWN code when error body is not JSON', async () => {
      const res = mockResponse(500, undefined, false)
      vi.mocked(res.json).mockRejectedValue(new SyntaxError('Unexpected token'))
      vi.mocked(globalThis.fetch).mockResolvedValue(res)

      try {
        await api.get('/api/broken')
      } catch (err) {
        const apiErr = err as InstanceType<typeof ApiClientError>
        expect(apiErr.code).toBe('UNKNOWN')
      }
    })
  })

  describe('api.put', () => {
    it('sends JSON body with PUT method', async () => {
      const payload = { name: 'Updated' }
      vi.mocked(globalThis.fetch).mockResolvedValue(mockResponse(200, { ok: true }))

      await api.put('/api/items/1', payload)

      expect(globalThis.fetch).toHaveBeenCalledWith('/api/items/1', expect.objectContaining({
        method: 'PUT',
        body: JSON.stringify(payload),
      }))
    })
  })

  describe('api.del', () => {
    it('sends DELETE request', async () => {
      vi.mocked(globalThis.fetch).mockResolvedValue(mockResponse(200, { ok: true }))

      await api.del('/api/items/1')

      expect(globalThis.fetch).toHaveBeenCalledWith('/api/items/1', expect.objectContaining({
        method: 'DELETE',
      }))
    })
  })
})
