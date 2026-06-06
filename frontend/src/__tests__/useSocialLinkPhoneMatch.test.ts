import { act, renderHook } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'

vi.mock('../api/auth', () => ({
  getSocialLinkPhoneMatch: vi.fn(),
}))

const { getSocialLinkPhoneMatch } = await import('../api/auth')
const { useSocialLinkPhoneMatch } = await import('../hooks/useSocialLinkPhoneMatch')

const mockGetSocialLinkPhoneMatch = vi.mocked(getSocialLinkPhoneMatch)

const matchedProfile = {
  name: '홍길동',
  email: 'hong@example.com',
  fn: '10',
  fmDept: '영어',
  jobCat: null,
  bizName: '',
  bizDesc: '',
  bizAddr: '',
  position: '',
  tags: [],
  usrPhonePublic: 'Y' as const,
  usrEmailPublic: 'N' as const,
}

describe('useSocialLinkPhoneMatch', () => {
  beforeEach(() => {
    vi.useFakeTimers()
    mockGetSocialLinkPhoneMatch.mockResolvedValue({
      matched: true,
      profile: matchedProfile,
    })
  })

  afterEach(() => {
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('keeps a matched result visible when only phone formatting changes', async () => {
    const { result, rerender } = renderHook(
      ({ phone }) => useSocialLinkPhoneMatch({ token: 'token', phone }),
      { initialProps: { phone: '010-1234-5678' } },
    )

    await act(async () => {
      await result.current.refetch()
    })

    expect(result.current.status).toBe('matched')
    expect(result.current.profile).toEqual(matchedProfile)

    rerender({ phone: '01012345678' })

    expect(result.current.status).toBe('matched')
    expect(result.current.profile).toEqual(matchedProfile)
  })

  it('hides a stale matched result as soon as the phone key changes', async () => {
    const { result, rerender } = renderHook(
      ({ phone }) => useSocialLinkPhoneMatch({ token: 'token', phone }),
      { initialProps: { phone: '010-1234-5678' } },
    )

    await act(async () => {
      await result.current.refetch()
    })

    expect(result.current.status).toBe('matched')
    expect(result.current.profile).toEqual(matchedProfile)

    rerender({ phone: '010-1234-567' })

    expect(result.current.status).toBe('idle')
    expect(result.current.profile).toBeNull()
    expect(mockGetSocialLinkPhoneMatch).toHaveBeenCalledTimes(1)
  })
})
