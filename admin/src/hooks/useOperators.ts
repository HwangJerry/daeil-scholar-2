// useOperators — operator list query plus grant/change/revoke mutations with toast feedback
import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ApiClientError } from '../api/client.ts';
import { fetchOperators, revokeOperator, setOperatorRole, type AdminRole } from '../api/operators.ts';
import { useToast } from './useToast.ts';

const OPERATORS_KEY = ['admin', 'operators'] as const;

function errorMessage(error: unknown) {
  if (error instanceof ApiClientError) return error.message;
  return '네트워크 상태를 확인하고 다시 시도해 주세요.';
}

export function useOperators(enabled: boolean) {
  const queryClient = useQueryClient();
  const addToast = useToast((s) => s.addToast);
  const query = useQuery({ queryKey: OPERATORS_KEY, queryFn: fetchOperators, enabled });

  const onSettled = () => void queryClient.invalidateQueries({ queryKey: OPERATORS_KEY });
  const onError = (error: unknown) => addToast({ variant: 'error', title: '권한 변경 실패', description: errorMessage(error) });

  const setRole = useMutation({
    mutationFn: ({ usrSeq, role }: { usrSeq: number; role: AdminRole }) => setOperatorRole(usrSeq, role),
    onSuccess: () => addToast({ variant: 'success', title: '관리자 권한을 저장했습니다.' }),
    onError,
    onSettled,
  });

  const revoke = useMutation({
    mutationFn: (usrSeq: number) => revokeOperator(usrSeq),
    onSuccess: () => addToast({ variant: 'success', title: '관리자 권한을 해제했습니다.' }),
    onError,
    onSettled,
  });

  return { ...query, operators: query.data?.items ?? [], setRole, revoke };
}
