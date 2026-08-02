// InfoPageShell.test — Foundation detail return-link visibility contracts
import { render, screen } from '@testing-library/react';
import { HelmetProvider } from 'react-helmet-async';
import { MemoryRouter } from 'react-router-dom';
import { describe, expect, it } from 'vitest';
import { InfoPageShell } from '../components/info/InfoPageShell';

function renderShell(canonicalPath: string) {
  render(
    <HelmetProvider>
      <MemoryRouter>
        <InfoPageShell title="테스트 페이지" canonicalPath={canonicalPath}>
          <p>본문</p>
        </InfoPageShell>
      </MemoryRouter>
    </HelmetProvider>,
  );
}

describe('InfoPageShell', () => {
  it('links foundation detail pages directly back to the foundation information hub', () => {
    renderShell('/vision');

    expect(screen.getByRole('link', { name: '장학회 소개로 돌아가기' })).toHaveAttribute(
      'href',
      '/about',
    );
  });

  it('does not show a redundant back link on the foundation information hub', () => {
    renderShell('/about');

    expect(screen.queryByRole('link', { name: '장학회 소개로 돌아가기' })).not.toBeInTheDocument();
  });
});
