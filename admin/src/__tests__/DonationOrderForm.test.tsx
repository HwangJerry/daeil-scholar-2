import { QueryClient, QueryClientProvider } from '@tanstack/react-query';
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { afterEach, describe, expect, it, vi } from 'vitest';
import { DonationOrderForm } from '../components/donation/DonationOrderForm.tsx';

describe('DonationOrderForm', () => {
  afterEach(() => vi.unstubAllGlobals());

  it('shows canonical field errors before sending an invalid order', async () => {
    const queryClient = new QueryClient();
    const fetchMock = vi.fn();
    vi.stubGlobal('fetch', fetchMock);
    const user = userEvent.setup();

    render(
      <QueryClientProvider client={queryClient}>
        <DonationOrderForm open onOpenChange={vi.fn()} />
      </QueryClientProvider>,
    );

    await user.click(screen.getByRole('button', { name: '등록' }));

    expect(screen.getByText('기부자명을 입력해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('기수를 입력해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('학과를 입력해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('010-1234-5678 또는 01012345678 형식으로 입력해 주세요.')).toBeInTheDocument();
    expect(screen.getByText('총액은 0 이상의 정수여야 합니다.')).toBeInTheDocument();
    expect(fetchMock).not.toHaveBeenCalled();
  });
});
