// useDonationImport — previews Excel donations and commits approved import rows
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { api } from '../api/client.ts';
import type {
  DonationImportCommitResult,
  DonationImportCommitRequest,
  DonationImportPreviewResult,
} from '../types/api.ts';

export function usePreviewDonationImport() {
  return useMutation({
    mutationFn: ({ file, donationDate }: { file: File; donationDate: string }) => {
      const formData = new FormData();
      formData.append('file', file);
      formData.append('donationDate', donationDate);
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
    mutationFn: (request: DonationImportCommitRequest) =>
      api.post<DonationImportCommitResult>('/api/admin/donation/import/commit', request),
    onSuccess: () => {
      void queryClient.invalidateQueries({ queryKey: ['admin', 'donation', 'orders'] });
      void queryClient.invalidateQueries({ queryKey: ['admin', 'donation', 'history'] });
      void queryClient.invalidateQueries({ queryKey: ['donation', 'summary'] });
      void queryClient.invalidateQueries({ queryKey: ['admin', 'dashboard'] });
    },
  });
}
