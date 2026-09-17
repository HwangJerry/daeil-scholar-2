// useSaveAppUpdatePolicy — mutation hook that refreshes policies and history after a save
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { saveAppUpdatePolicy } from '../api/appUpdate.ts';
import { policyErrorMessage } from '../lib/appUpdatePolicyErrors.ts';
import type { AppPlatform } from '../types/appUpdate.ts';
import { APP_UPDATE_POLICIES_QUERY_KEY } from './useAppUpdatePolicies.ts';
import { appUpdatePolicyHistoryQueryKey } from './useAppUpdatePolicyHistory.ts';
import { useToast } from './useToast.ts';

const PLATFORM_LABELS: Record<AppPlatform, string> = { ios: 'iOS', android: 'Android' };

export function useSaveAppUpdatePolicy(platform: AppPlatform) {
  const queryClient = useQueryClient();
  const addToast = useToast((state) => state.addToast);

  return useMutation({
    mutationFn: saveAppUpdatePolicy,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: APP_UPDATE_POLICIES_QUERY_KEY });
      await queryClient.invalidateQueries({ queryKey: appUpdatePolicyHistoryQueryKey(platform) });
      addToast({
        variant: 'success',
        title: `${PLATFORM_LABELS[platform]} 업데이트 정책이 저장되었습니다.`,
      });
    },
    onError: (error) => {
      addToast({
        variant: 'error',
        title: `${PLATFORM_LABELS[platform]} 정책 저장 실패`,
        description: policyErrorMessage(error),
      });
    },
  });
}
