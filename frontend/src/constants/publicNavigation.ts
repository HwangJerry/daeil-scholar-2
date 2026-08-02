// publicNavigation — Canonical parent navigation grouping for public routes
export type PublicNavigationSection = 'news' | 'about';

const ABOUT_ROUTE_ROOTS = [
  '/about',
  '/greetings',
  '/vision',
  '/history',
  '/organization',
  '/business',
  '/disclosure',
] as const;

export function getPublicNavigationSection(pathname: string): PublicNavigationSection | null {
  if (pathname === '/' || pathname.startsWith('/post/')) {
    return 'news';
  }

  if (ABOUT_ROUTE_ROOTS.some((root) => pathname === root || pathname.startsWith(`${root}/`))) {
    return 'about';
  }

  return null;
}
