// uploadImage — uploads an image file to the admin upload endpoint and returns the URL
import { api } from '../../api/client.ts';
import type { UploadResponse } from '../../types/api.ts';

export async function uploadImage(file: File): Promise<string> {
  const formData = new FormData();
  formData.append('file', file);
  const res = await api.upload<UploadResponse>('/api/admin/upload?type=notice', formData);
  return res.url;
}
