import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { describe, expect, it, vi } from 'vitest';
import { MemberStatusControl } from '../components/member/MemberStatusControl.tsx';

describe('MemberStatusControl', () => {
  it('offers only 탈퇴/휴면/정지 and shows a verification status as the disabled current value', () => {
    render(<MemberStatusControl status="CCC" disabled={false} onChange={vi.fn()} />);
    const options = screen.getAllByRole('option') as HTMLOptionElement[];
    expect(options.map((o) => o.textContent)).toEqual(['승인회원 (현재)', '탈퇴', '휴면', '정지']);
    expect(options[0].disabled).toBe(true);
  });

  it('reports a newly selected status', async () => {
    const onChange = vi.fn();
    const user = userEvent.setup();
    render(<MemberStatusControl status="ABA" disabled={false} onChange={onChange} />);

    await user.selectOptions(screen.getByLabelText('상태 변경'), 'ACA');

    expect(onChange).toHaveBeenCalledWith('ACA');
  });
});
