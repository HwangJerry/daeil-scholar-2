// useHallOfFameEntries — Hall of Fame entry list query and entry mutations
import { keepPreviousData, useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { ApiClientError } from '../api/client.ts';
import {
  createHallEntry,
  deleteHallEntry,
  fetchHallEntries,
  updateHallEntry,
} from '../api/hallOfFame.ts';
import type {
  HallEntryCreateRequest,
  HallEntryFilters,
  HallEntryPatch,
} from '../types/hallOfFame.ts';
import { useToast } from './useToast.ts';

export const hallOfFameQueryKey = ['admin', 'hallOfFame'] as const;

function describeError(error: unknown) {
  return error instanceof ApiClientError && error.message
    ? error.message
    : '네트워크 상태를 확인하고 다시 시도해 주세요.';
}

export function useHallOfFameList(filters: HallEntryFilters, page: number, size: number) {
  return useQuery({
    queryKey: [...hallOfFameQueryKey, filters, page, size],
    queryFn: () => fetchHallEntries(filters, page, size),
    placeholderData: keepPreviousData,
  });
}

export function useHallOfFameMutations() {
  const queryClient = useQueryClient();
  const addToast = useToast((state) => state.addToast);

  const invalidate = () => queryClient.invalidateQueries({ queryKey: hallOfFameQueryKey });
  const reportFailure = (title: string) => (error: unknown) => {
    addToast({ variant: 'error', title, description: describeError(error) });
  };

  const create = useMutation({
    mutationFn: (input: HallEntryCreateRequest) => createHallEntry(input),
    onSuccess: () => {
      void invalidate();
      addToast({ variant: 'success', title: '명예의 전당 항목을 추가했습니다.' });
    },
    onError: reportFailure('항목 추가 실패'),
  });

  const update = useMutation({
    mutationFn: ({ id, patch }: { id: number; patch: HallEntryPatch }) => (
      updateHallEntry(id, patch)
    ),
    onSuccess: () => void invalidate(),
    onError: reportFailure('항목 수정 실패'),
  });

  const remove = useMutation({
    mutationFn: (id: number) => deleteHallEntry(id),
    onSuccess: () => {
      void invalidate();
      addToast({ variant: 'success', title: '항목을 삭제했습니다.' });
    },
    onError: reportFailure('항목 삭제 실패'),
  });

  return { create, update, remove };
}
