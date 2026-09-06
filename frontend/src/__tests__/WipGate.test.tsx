// WipGate.test — App support remains public without unlocking the rest of the site
import { render, screen } from '@testing-library/react';
import userEvent from '@testing-library/user-event';
import { Link, MemoryRouter } from 'react-router-dom';
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest';

beforeEach(() => {
  vi.resetModules();
  vi.stubEnv('VITE_WIP_ADMIN_CODE', 'test-maintenance-code');
  sessionStorage.clear();
});

afterEach(() => {
  vi.unstubAllEnvs();
});

describe('public support during maintenance', () => {
  it.each(['/support', '/support/'])('opens %s without granting access to other pages', async (path) => {
    const { WipGate } = await import('../components/common/WipGate');
    const user = userEvent.setup();

    render(
      <MemoryRouter initialEntries={[path]}>
        <WipGate>
          <h1>앱 이용 문의</h1>
          <Link to="/">홈으로</Link>
        </WipGate>
      </MemoryRouter>,
    );

    expect(screen.getByRole('heading', { name: '앱 이용 문의' })).toBeInTheDocument();
    expect(sessionStorage.getItem('wip-unlock')).toBeNull();
    await user.click(screen.getByRole('link', { name: '홈으로' }));
    expect(screen.getByRole('heading', { name: '사이트 점검 중' })).toBeInTheDocument();
    expect(screen.queryByRole('heading', { name: '앱 이용 문의' })).not.toBeInTheDocument();
  });
});
