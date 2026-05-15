// SearchFilter.test — Search input submit, debounce, and IME composition behavior
import { act, fireEvent, render, screen } from '@testing-library/react'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { SearchFilter } from '../components/alumni/SearchFilter'
import { createWrapper } from './test-utils'

vi.mock('../api/client', () => ({
  api: {
    get: vi.fn(() => new Promise(() => {})),
  },
}))

function renderSearchFilter(onSearch = vi.fn()) {
  const Wrapper = createWrapper()
  render(<SearchFilter onSearch={onSearch} />, { wrapper: Wrapper })
  return { onSearch }
}

describe('SearchFilter', () => {
  beforeEach(() => {
    vi.useFakeTimers()
  })

  afterEach(() => {
    vi.runOnlyPendingTimers()
    vi.useRealTimers()
    vi.clearAllMocks()
  })

  it('submits the trimmed search term immediately and blurs the input', () => {
    const { onSearch } = renderSearchFilter()
    const input = screen.getByPlaceholderText('이름 또는 태그로 검색')
    const form = input.closest('form')

    input.focus()
    fireEvent.change(input, { target: { value: '  홍길동  ' } })
    expect(onSearch).not.toHaveBeenCalled()

    fireEvent.submit(form!)

    expect(onSearch).toHaveBeenCalledTimes(1)
    expect(onSearch).toHaveBeenLastCalledWith({
      fn: '',
      dept: '',
      name: '홍길동',
      jobCat: '',
    })
    expect(input).not.toHaveFocus()

    act(() => {
      vi.advanceTimersByTime(400)
    })
    expect(onSearch).toHaveBeenCalledTimes(1)
  })

  it('debounces name search and skips identical trimmed terms', () => {
    const { onSearch } = renderSearchFilter()
    const input = screen.getByPlaceholderText('이름 또는 태그로 검색')

    fireEvent.change(input, { target: { value: '김철수' } })
    act(() => {
      vi.advanceTimersByTime(399)
    })
    expect(onSearch).not.toHaveBeenCalled()

    act(() => {
      vi.advanceTimersByTime(1)
    })
    expect(onSearch).toHaveBeenCalledTimes(1)
    expect(onSearch).toHaveBeenLastCalledWith({
      fn: '',
      dept: '',
      name: '김철수',
      jobCat: '',
    })

    fireEvent.change(input, { target: { value: '  김철수  ' } })
    act(() => {
      vi.advanceTimersByTime(400)
    })
    expect(onSearch).toHaveBeenCalledTimes(1)
  })

  it('does not debounce while Korean IME composition is active', () => {
    const { onSearch } = renderSearchFilter()
    const input = screen.getByPlaceholderText('이름 또는 태그로 검색')

    fireEvent.compositionStart(input)
    fireEvent.change(input, { target: { value: 'ㅎ' } })
    act(() => {
      vi.advanceTimersByTime(1000)
    })
    expect(onSearch).not.toHaveBeenCalled()

    fireEvent.change(input, { target: { value: '한' } })
    fireEvent.compositionEnd(input)
    act(() => {
      vi.advanceTimersByTime(400)
    })

    expect(onSearch).toHaveBeenCalledTimes(1)
    expect(onSearch).toHaveBeenLastCalledWith({
      fn: '',
      dept: '',
      name: '한',
      jobCat: '',
    })
  })
})
