// useRegisterFormValidation.test.ts — Unit tests for registration form validation hook.
import { describe, it, expect } from 'vitest'
import { renderHook } from '@testing-library/react'
import { useRegisterFormValidation } from '../hooks/useRegisterFormValidation'

function makeFields(overrides: Partial<{
  usrId: string
  password: string
  passwordConfirm: string
  email: string
}> = {}) {
  return {
    usrId: 'validUser1',
    password: 'secret123',
    passwordConfirm: 'secret123',
    email: 'test@example.com',
    ...overrides,
  }
}

describe('useRegisterFormValidation', () => {
  it('returns null when all fields are valid', () => {
    const { result } = renderHook(() => useRegisterFormValidation())
    const error = result.current.validate(makeFields())
    expect(error).toBeNull()
  })

  describe('usrId validation', () => {
    it('rejects IDs shorter than 4 characters', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ usrId: 'abc' }))
      expect(error).not.toBeNull()
      expect(error).toContain('4~20')
    })

    it('rejects IDs longer than 20 characters', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ usrId: 'a'.repeat(21) }))
      expect(error).not.toBeNull()
    })

    it('rejects IDs with special characters', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ usrId: 'user@name' }))
      expect(error).not.toBeNull()
    })

    it('rejects IDs with spaces', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ usrId: 'user name' }))
      expect(error).not.toBeNull()
    })

    it('accepts alphanumeric IDs within 4-20 chars', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      expect(result.current.validate(makeFields({ usrId: 'abcd' }))).toBeNull()
      expect(result.current.validate(makeFields({ usrId: 'A1b2C3d4' }))).toBeNull()
      expect(result.current.validate(makeFields({ usrId: 'a'.repeat(20) }))).toBeNull()
    })
  })

  describe('password validation', () => {
    it('rejects passwords shorter than 6 characters', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ password: '12345', passwordConfirm: '12345' }))
      expect(error).not.toBeNull()
      expect(error).toContain('6')
    })

    it('accepts passwords of exactly 6 characters', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ password: '123456', passwordConfirm: '123456' }))
      expect(error).toBeNull()
    })
  })

  describe('password confirmation', () => {
    it('rejects mismatched passwords', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ password: 'abcdef', passwordConfirm: 'ghijkl' }))
      expect(error).not.toBeNull()
      expect(error).toContain('일치')
    })
  })

  describe('email validation', () => {
    it('rejects email without @ symbol', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ email: 'nope.com' }))
      expect(error).not.toBeNull()
      expect(error).toContain('이메일')
    })

    it('accepts email with @ symbol', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ email: 'a@b' }))
      expect(error).toBeNull()
    })
  })

  describe('validation order (first failing rule wins)', () => {
    it('returns usrId error before password error', () => {
      const { result } = renderHook(() => useRegisterFormValidation())
      const error = result.current.validate(makeFields({ usrId: '!', password: '1', passwordConfirm: '1' }))
      expect(error).toContain('아이디')
    })
  })
})
