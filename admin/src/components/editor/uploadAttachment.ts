// uploadAttachment — uploads a notice attachment file and returns persisted file metadata
import { api } from '../../api/client.ts';

export interface AttachmentUploadResponse {
  fSeq: number;
  url: string;
  fileName: string;
  filePath: string;
  fileOrgName: string;
  fileSize: string;
}

export async function uploadAttachment(file: File): Promise<AttachmentUploadResponse> {
  const formData = new FormData();
  formData.append('file', file);
  return api.upload<AttachmentUploadResponse>('/api/admin/upload/attachment', formData);
}
