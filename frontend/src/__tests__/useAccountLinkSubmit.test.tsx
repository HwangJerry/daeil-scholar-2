// useAccountLinkSubmit.test — Verifies canonical social-link submission outcomes.
import { act, renderHook } from '@testing-library/react'
import type { ReactNode } from 'react'
import { MemoryRouter } from 'react-router-dom'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { useAccountLinkSubmit } from '../hooks/useAccountLinkSubmit'

const fetchUser = vi.fn(async () => undefined)

vi.mock('../hooks/useAuth', () => ({
  useAuth: (selector: (state: { fetchUser: typeof fetchUser }) => unknown) =>
    selector({ fetchUser }),
}))

function wrapper({ children }: { children: ReactNode }) {
  return <MemoryRouter>{children}</MemoryRouter>
}

describe('useAccountLinkSubmit', () => {
  afterEach(() => {
    vi.restoreAllMocks()
    fetchUser.mockClear()
  })

  it('posts the canonical reauthentication payload to the web cookie bridge', async () => {
    const fetchMock = vi
      .spyOn(globalThis, 'fetch')
      .mockResolvedValue(new Response(null, { status: 204 }))
    const { result } = renderHook(() => useAccountLinkSubmit(), { wrapper })

    await act(async () => {
      await result.current.submit({
        linkToken: 'fixture-link-token',
        email: 'member@example.com',
        password: 'fixture-password',
      })
    })

    expect(fetchMock).toHaveBeenCalledWith(
      '/api/auth/social/link/web',
      expect.objectContaining({
        method: 'POST',
        body: JSON.stringify({
          linkToken: 'fixture-link-token',
          email: 'member@example.com',
          password: 'fixture-password',
        }),
      }),
    )
    expect(fetchUser).toHaveBeenCalledTimes(1)
  })
})
