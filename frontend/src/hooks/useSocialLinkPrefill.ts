// useSocialLinkPrefill — Fetches cached social-provider data (email/nickname/profileImage) for the /login/link signup form.
import { useQuery } from '@tanstack/react-query';
import { getSocialLinkPrefill } from '../api/auth';

export function useSocialLinkPrefill(token: string, enabled = true) {
  return useQuery({
    queryKey: ['social-link-prefill', token],
    queryFn: () => getSocialLinkPrefill(token),
    enabled: enabled && token.length > 0,
    staleTime: Infinity,
    retry: false,
  });
}
