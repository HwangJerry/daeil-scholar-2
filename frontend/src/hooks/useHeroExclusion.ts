// Derives hero exclusion state for the feed query — separates hero filtering from pagination
import { useHeroNotice } from './useHeroNotice';

export function useHeroExclusion() {
  const { data, isSuccess, isError } = useHeroNotice();
  const heroSeq = data?.seq;

  return {
    /** seq to exclude from feed, or undefined if hero hasn't loaded */
    heroSeq,
    /** Stable query key segment that changes when hero state changes */
    queryKeySegment: heroSeq ?? ('no-hero' as const),
    /** true once the hero query has settled (success or error) */
    isHeroLoaded: isSuccess || isError,
  };
}
