// useTableSort.test — unit tests for table sort state management and client-side sorting
import { renderHook, act } from '@testing-library/react'
import { describe, it, expect } from 'vitest'
import { useTableSort } from '../hooks/useTableSort'

describe('useTableSort', () => {
  describe('initial state', () => {
    it('starts with null column and null direction by default', () => {
      const { result } = renderHook(() => useTableSort())

      expect(result.current.sort.column).toBeNull()
      expect(result.current.sort.direction).toBeNull()
    })

    it('accepts an initial column', () => {
      const { result } = renderHook(() => useTableSort('name'))

      expect(result.current.sort.column).toBe('name')
      expect(result.current.sort.direction).toBeNull()
    })
  })

  describe('toggleSort — sort cycle', () => {
    it('sets direction to asc on first toggle when no column is active', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('name'))

      expect(result.current.sort).toEqual({ column: 'name', direction: 'asc' })
    })

    it('cycles asc -> desc on second toggle of same column', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('name'))
      act(() => result.current.toggleSort('name'))

      expect(result.current.sort).toEqual({ column: 'name', direction: 'desc' })
    })

    it('cycles desc -> null on third toggle of same column', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('name'))
      act(() => result.current.toggleSort('name'))
      act(() => result.current.toggleSort('name'))

      expect(result.current.sort).toEqual({ column: null, direction: null })
    })

    it('resets to asc when switching to a different column', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('name'))
      act(() => result.current.toggleSort('name')) // now desc

      expect(result.current.sort.direction).toBe('desc')

      act(() => result.current.toggleSort('email'))

      expect(result.current.sort).toEqual({ column: 'email', direction: 'asc' })
    })
  })

  describe('getSortedItems', () => {
    const accessors = {
      name: (item: { name: string; age: number | null }) => item.name,
      age: (item: { name: string; age: number | null }) => item.age,
    }

    const items = [
      { name: 'Charlie', age: 30 },
      { name: 'Alice', age: 25 },
      { name: 'Bob', age: 35 },
    ]

    it('returns items unchanged when no sort is active', () => {
      const { result } = renderHook(() => useTableSort())

      const sorted = result.current.getSortedItems(items, accessors)

      expect(sorted).toEqual(items)
    })

    it('sorts strings ascending with Korean locale', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('name'))

      const sorted = result.current.getSortedItems(items, accessors)

      expect(sorted.map((i) => i.name)).toEqual(['Alice', 'Bob', 'Charlie'])
    })

    it('sorts strings descending', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('name'))
      act(() => result.current.toggleSort('name'))

      const sorted = result.current.getSortedItems(items, accessors)

      expect(sorted.map((i) => i.name)).toEqual(['Charlie', 'Bob', 'Alice'])
    })

    it('sorts numbers ascending', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('age'))

      const sorted = result.current.getSortedItems(items, accessors)

      expect(sorted.map((i) => i.age)).toEqual([25, 30, 35])
    })

    it('sorts numbers descending', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('age'))
      act(() => result.current.toggleSort('age'))

      const sorted = result.current.getSortedItems(items, accessors)

      expect(sorted.map((i) => i.age)).toEqual([35, 30, 25])
    })

    it('pushes null values to the end regardless of direction', () => {
      const itemsWithNull = [
        { name: 'Alice', age: null },
        { name: 'Bob', age: 20 },
        { name: 'Charlie', age: null },
        { name: 'Dave', age: 10 },
      ]

      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('age'))

      const sortedAsc = result.current.getSortedItems(itemsWithNull, accessors)
      expect(sortedAsc.map((i) => i.age)).toEqual([10, 20, null, null])

      act(() => result.current.toggleSort('age'))

      const sortedDesc = result.current.getSortedItems(itemsWithNull, accessors)
      expect(sortedDesc.map((i) => i.age)).toEqual([20, 10, null, null])
    })

    it('returns items unchanged when accessor for column is not found', () => {
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('unknown'))

      const sorted = result.current.getSortedItems(items, accessors)

      expect(sorted).toEqual(items)
    })

    it('sorts Korean strings correctly', () => {
      const koreanItems = [
        { name: '다', age: 1 },
        { name: '가', age: 2 },
        { name: '나', age: 3 },
      ]

      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('name'))

      const sorted = result.current.getSortedItems(koreanItems, accessors)

      expect(sorted.map((i) => i.name)).toEqual(['가', '나', '다'])
    })

    it('does not mutate the original array', () => {
      const original = [...items]
      const { result } = renderHook(() => useTableSort())

      act(() => result.current.toggleSort('name'))

      result.current.getSortedItems(items, accessors)

      expect(items).toEqual(original)
    })
  })
})
