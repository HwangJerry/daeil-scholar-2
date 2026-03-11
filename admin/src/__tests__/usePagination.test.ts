// usePagination.test — unit tests for pagination state management hook
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { usePagination } from '../hooks/usePagination'

describe('usePagination', () => {
  describe('initial values', () => {
    it('defaults to page 1 and pageSize 20', () => {
      const { result } = renderHook(() => usePagination())

      expect(result.current.page).toBe(1)
      expect(result.current.pageSize).toBe(20)
    })

    it('accepts custom initial page and pageSize', () => {
      const { result } = renderHook(() =>
        usePagination({ initialPage: 3, initialPageSize: 50 }),
      )

      expect(result.current.page).toBe(3)
      expect(result.current.pageSize).toBe(50)
    })

    it('accepts partial options (only initialPage)', () => {
      const { result } = renderHook(() =>
        usePagination({ initialPage: 5 }),
      )

      expect(result.current.page).toBe(5)
      expect(result.current.pageSize).toBe(20)
    })

    it('accepts partial options (only initialPageSize)', () => {
      const { result } = renderHook(() =>
        usePagination({ initialPageSize: 10 }),
      )

      expect(result.current.page).toBe(1)
      expect(result.current.pageSize).toBe(10)
    })
  })

  describe('setPage', () => {
    it('updates the current page', () => {
      const { result } = renderHook(() => usePagination())

      act(() => result.current.setPage(5))

      expect(result.current.page).toBe(5)
    })

    it('can set page to any positive number', () => {
      const { result } = renderHook(() => usePagination())

      act(() => result.current.setPage(100))

      expect(result.current.page).toBe(100)
    })
  })

  describe('resetPage', () => {
    it('resets the page to 1', () => {
      const { result } = renderHook(() => usePagination())

      act(() => result.current.setPage(10))
      expect(result.current.page).toBe(10)

      act(() => result.current.resetPage())
      expect(result.current.page).toBe(1)
    })

    it('keeps pageSize unchanged when resetting page', () => {
      const { result } = renderHook(() =>
        usePagination({ initialPageSize: 50 }),
      )

      act(() => result.current.setPage(10))
      act(() => result.current.resetPage())

      expect(result.current.page).toBe(1)
      expect(result.current.pageSize).toBe(50)
    })
  })

  describe('handlePageSizeChange', () => {
    it('updates pageSize and resets page to 1', () => {
      const { result } = renderHook(() => usePagination())

      act(() => result.current.setPage(5))
      expect(result.current.page).toBe(5)

      act(() => result.current.handlePageSizeChange(50))

      expect(result.current.pageSize).toBe(50)
      expect(result.current.page).toBe(1)
    })

    it('resets page to 1 even when already on page 1', () => {
      const { result } = renderHook(() => usePagination())

      act(() => result.current.handlePageSizeChange(10))

      expect(result.current.pageSize).toBe(10)
      expect(result.current.page).toBe(1)
    })
  })
})
