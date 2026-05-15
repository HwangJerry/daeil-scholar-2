// useSocialLinkExistingMember — Merge-mode prefill: fetches existing member info immediately by token+phone, no debounce.
import { useQuery } from '@tanstack/react-query';
import { getSocialLinkPhoneMatch } from '../api/auth';

export function useSocialLinkExistingMember(token: string, phone: string) {
  const digits = phone.replace(/\D/g, '');
  return useQuery({
    queryKey: ['social-link-existing-member', token, phone],
    queryFn: () => getSocialLinkPhoneMatch(token, phone),
    enabled: token.length > 0 && digits.length >= 9,
    staleTime: Infinity,
    retry: false,
  });
}
