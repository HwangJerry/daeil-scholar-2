// publicRoutes.test — Public information and app support route boundaries
import { describe, expect, it } from 'vitest';
import { PUBLIC_ROUTE_PATHS } from '../routes';

describe('public MVP route allowlist', () => {
  it('includes mobile signup while keeping authenticated community routes disabled', () => {
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
      '/support',
      '/privacy',
      '/account-deletion',
      '/register',
      '/register/complete',
      '*',
    ]);

    for (const forbidden of [
      '/ad/:maSeq',
      '/alumni',
      '/messages',
      '/login',
      '/me',
      '/mypage',
    ]) {
      expect(PUBLIC_ROUTE_PATHS).not.toContain(forbidden);
    }
  });
});
