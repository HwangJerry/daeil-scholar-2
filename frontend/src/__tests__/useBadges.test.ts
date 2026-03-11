// useBadges.test.ts — MSW-based integration tests for the useBadges React Query hook
import { describe, it, expect, beforeAll, afterAll, afterEach, vi } from 'vitest'
import { renderHook, waitFor } from '@testing-library/react'
import { setupServer } from 'msw/node'
import { http, HttpResponse } from 'msw'
import { createWrapper } from './test-utils'

// Mock useAuth to control authentication state without hitting a real store.
// The mock must be declared before importing the hook under test so that
// Vitest's module hoisting applies the mock first.
const mockUseAuth = vi.fn()

vi.mock('../hooks/useAuth', () => ({
  useAuth: (...args: unknown[]) => mockUseAuth(...args),
}))

// Import the hook *after* the mock is registered
const { useBadges } = await import('../hooks/useBadges')

const BASE = 'http://localhost:3000'

const server = setupServer()

beforeAll(() => server.listen({ onUnhandledRequest: 'bypass' }))
afterEach(() => {
  server.resetHandlers()
  vi.clearAllMocks()
})
afterAll(() => server.close())

describe('useBadges', () => {
  describe('when authenticated', () => {
    it('returns badge counts from the API', async () => {
      mockUseAuth.mockReturnValue({
        user: { usrSeq: 1, usrId: 'test', name: 'Tester' },
      })

      server.use(
        http.get(`${BASE}/api/badges`, () => {
          return HttpResponse.json({
            unreadMessages: 3,
            unreadNotifications: 7,
          })
        }),
      )

      const { result } = renderHook(() => useBadges(), {
        wrapper: createWrapper(),
      })

      // Initially returns defaults (0, 0) before the query resolves
      expect(result.current.unreadMessages).toBe(0)
      expect(result.current.unreadNotifications).toBe(0)

      await waitFor(() => {
        expect(result.current.unreadMessages).toBe(3)
        expect(result.current.unreadNotifications).toBe(7)
      })
    })
  })

  describe('when not authenticated', () => {
    it('returns zeroed badge counts without calling the API', async () => {
      mockUseAuth.mockReturnValue({ user: null })

      // Register a handler that should never be called
      let apiCalled = false
      server.use(
        http.get(`${BASE}/api/badges`, () => {
          apiCalled = true
          return HttpResponse.json({ unreadMessages: 5, unreadNotifications: 5 })
        }),
      )

      const { result } = renderHook(() => useBadges(), {
        wrapper: createWrapper(),
      })

      // Give React Query a tick to potentially fire (it should not)
      await new Promise((r) => setTimeout(r, 50))

      expect(result.current.unreadMessages).toBe(0)
      expect(result.current.unreadNotifications).toBe(0)
      expect(apiCalled).toBe(false)
    })
  })

  describe('error response handling', () => {
    it('returns default zero counts when the API returns an error', async () => {
      mockUseAuth.mockReturnValue({
        user: { usrSeq: 1, usrId: 'test', name: 'Tester' },
      })

      server.use(
        http.get(`${BASE}/api/badges`, () => {
          return HttpResponse.json(
            { code: 'UNAUTHORIZED', message: 'Session expired' },
            { status: 401 },
          )
        }),
      )

      const { result } = renderHook(() => useBadges(), {
        wrapper: createWrapper(),
      })

      // Wait for the query to settle (it will error, hook returns defaults)
      await waitFor(() => {
        expect(result.current.unreadMessages).toBe(0)
        expect(result.current.unreadNotifications).toBe(0)
      })
    })
  })
})
