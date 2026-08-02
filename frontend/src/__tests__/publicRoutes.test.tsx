import { describe, expect, it } from 'vitest';
import { PUBLIC_ROUTE_PATHS } from '../routes';

describe('public MVP route allowlist', () => {
  it('contains only notices and foundation information routes', () => {
    expect(PUBLIC_ROUTE_PATHS).toEqual([
      '/',
      '/post/:seq',
      '/about',
      '/greetings',
      '/vision',
      '/history',
      '/organization',
      '/business',
      '/disclosure',
      '/disclosure/:seq',
      '*',
    ]);

    for (const forbidden of [
      '/ad/:maSeq',
      '/alumni',
      '/messages',
      '/login',
      '/register',
      '/me',
      '/mypage',
    ]) {
      expect(PUBLIC_ROUTE_PATHS).not.toContain(forbidden);
    }
  });
});
