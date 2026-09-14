// AccountDeletionRequestGuide.test — The web page offers a deletion request path without the app.
import { render, screen } from '@testing-library/react';
import { describe, expect, it } from 'vitest';
import { AccountDeletionRequestGuide } from '../components/accountDeletion/AccountDeletionRequestGuide';
import { PRIVACY_CONTACT } from '../domains/privacy/policyContent';

describe('AccountDeletionRequestGuide', () => {
  it('shows the app name, request steps, deleted data and retention periods', () => {
    render(<AccountDeletionRequestGuide />);
    expect(screen.getByRole('heading', { name: '앱 없이 계정 삭제 요청하기' })).toBeInTheDocument();
    expect(screen.getByText(/대일외고 장학회 앱/)).toBeInTheDocument();
    const mail = screen.getByRole('link', { name: PRIVACY_CONTACT.email });
    expect(mail.getAttribute('href')).toMatch(new RegExp(`^mailto:${PRIVACY_CONTACT.email}\\?subject=`));
    expect(decodeURIComponent(mail.getAttribute('href') ?? '')).toContain('계정 삭제 요청');
    expect(screen.getByRole('heading', { name: '삭제하는 정보' })).toBeInTheDocument();
    expect(screen.getByRole('heading', { name: '보관하는 정보와 기간' })).toBeInTheDocument();
    expect(screen.getByText('발급일부터 5년. 기부자명·기부일·금액·거래 증빙만 암호화해 분리 보관합니다.')).toBeInTheDocument();
    expect(screen.getByText('삭제 완료 후 35일')).toBeInTheDocument();
  });
});
