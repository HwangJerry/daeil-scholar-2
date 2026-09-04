// LandingHeader — Landing-page section state wired to the shared public-site header
import { PublicSiteHeader, type PublicSiteNavItem } from '../layout/PublicSiteHeader';
import { useActiveLandingSection } from './useActiveLandingSection';

const LANDING_NAV_ITEMS: readonly PublicSiteNavItem[] = [
  { id: 'download', label: '다운로드', href: '#download' },
  { id: 'news', label: '최근 소식', href: '#news' },
  { id: 'about', label: '장학회 소개', href: '#about' },
  { id: 'business', label: '장학사업', href: '#business' },
];

const ABOUT_SECTION_IDS = ['about', 'greeting', 'vision', 'history', 'organization'];
const LANDING_SECTION_IDS = [
  'download',
  'news',
  ...ABOUT_SECTION_IDS,
  'business',
];

export function LandingHeader() {
  const observedSectionId = useActiveLandingSection(LANDING_SECTION_IDS);
  const activeSectionId =
    observedSectionId !== null && ABOUT_SECTION_IDS.includes(observedSectionId)
      ? 'about'
      : observedSectionId;

  return (
    <PublicSiteHeader
      activeSectionId={activeSectionId}
      items={LANDING_NAV_ITEMS}
      navigationLabel="랜딩 페이지"
    />
  );
}
