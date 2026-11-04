// useHallOfFameImport — previews a Hall of Fame spreadsheet and commits the reviewed diff
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { commitHallImport, previewHallImport } from '../api/hallOfFame.ts';
import type { HallImportCommitRequest } from '../types/hallOfFame.ts';
import { hallOfFameQueryKey } from './useHallOfFameEntries.ts';

export function usePreviewHallImport() {
  return useMutation({ mutationFn: (file: File) => previewHallImport(file) });
}

export function useCommitHallImport() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (request: HallImportCommitRequest) => commitHallImport(request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: hallOfFameQueryKey });
    },
  });
}
