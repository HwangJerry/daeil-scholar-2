// useUpdateAppSetting — mutation hook with cache refresh and user feedback
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { updateAppSetting } from '../api/appSettings.ts';
import { useToast } from './useToast.ts';
import { APP_SETTINGS_QUERY_KEY } from './useAppSettings.ts';

export function useUpdateAppSetting() {
  const queryClient = useQueryClient();
  const addToast = useToast((state) => state.addToast);

  return useMutation({
    mutationFn: updateAppSetting,
    onSuccess: async () => {
      await queryClient.invalidateQueries({ queryKey: APP_SETTINGS_QUERY_KEY });
      addToast({ variant: 'success', title: '앱 설정이 저장되었습니다.' });
    },
    onError: () => {
      addToast({
        variant: 'error',
        title: '앱 설정 저장 실패',
        description: '오류 내용을 확인하고 다시 시도해 주세요.',
      });
    },
  });
}
