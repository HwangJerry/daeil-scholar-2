// useDonationImport — previews Excel donations and commits approved import rows
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type {
  DonationImportCommitResult,
  DonationImportCommitRow,
  DonationImportPreviewResult,
} from '../types/api.ts';

export function usePreviewDonationImport() {
  return useMutation({
    mutationFn: (file: File) => {
      const formData = new FormData();
      formData.append('file', file);
      return api.upload<DonationImportPreviewResult>(
        '/api/admin/donation/import/preview',
        formData,
      );
    },
  });
}

export function useCommitDonationImport() {
  const queryClient = useQueryClient();

  return useMutation({
    mutationFn: (rows: DonationImportCommitRow[]) =>
      api.post<DonationImportCommitResult>('/api/admin/donation/import/commit', rows),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'donation', 'orders'] });
      void queryClient.invalidateQueries({ queryKey: ['admin', 'donation', 'history'] });
      void queryClient.invalidateQueries({ queryKey: ['donation', 'summary'] });
      void queryClient.invalidateQueries({ queryKey: ['admin', 'dashboard'] });
    },
  });
}
