import { fireEvent, render, screen } from '@testing-library/react'
import { afterEach, describe, expect, it, vi } from 'vitest'
import { BottomSheet } from '../components/ui/BottomSheet'

describe('BottomSheet', () => {
  afterEach(() => {
    vi.restoreAllMocks()
  })

  it('closes when the drag handle is pulled downward past the dismiss threshold', () => {
    const onClose = vi.fn()

    render(
      <BottomSheet onClose={onClose}>
        <p>sheet content</p>
      </BottomSheet>,
    )

    const handle = screen.getByTestId('bottom-sheet-drag-handle')
    fireEvent.pointerDown(handle, { clientY: 10, pointerId: 1, pointerType: 'touch' })
    fireEvent.pointerMove(handle, { clientY: 180, pointerId: 1, pointerType: 'touch' })
    fireEvent.pointerUp(handle, { clientY: 180, pointerId: 1, pointerType: 'touch' })

    expect(onClose).toHaveBeenCalledTimes(1)
  })

  it('snaps back without closing after a short slow drag', () => {
    const onClose = vi.fn()
    let now = 0
    vi.spyOn(Date, 'now').mockImplementation(() => now)

    render(
      <BottomSheet onClose={onClose}>
        <p>sheet content</p>
      </BottomSheet>,
    )

    const handle = screen.getByTestId('bottom-sheet-drag-handle')
    fireEvent.pointerDown(handle, { clientY: 10, pointerId: 1, pointerType: 'touch' })
    now = 1000
    fireEvent.pointerMove(handle, { clientY: 80, pointerId: 1, pointerType: 'touch' })
    fireEvent.pointerUp(handle, { clientY: 80, pointerId: 1, pointerType: 'touch' })

    expect(onClose).not.toHaveBeenCalled()
  })
})
