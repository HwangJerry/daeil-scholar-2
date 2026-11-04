// hallOfFame — admin API for Hall of Fame entries and the preview/commit Excel import
import { api } from './client.ts';
import type {
  HallEntry,
  HallEntryCreateRequest,
  HallEntryFilters,
  HallEntryPage,
  HallEntryPatch,
  HallImportCommitRequest,
  HallImportCommitResult,
  HallImportPreview,
} from '../types/hallOfFame.ts';

const HALL_OF_FAME_URL = '/api/admin/hall-of-fame';

export function fetchHallEntries(filters: HallEntryFilters, page: number, size: number) {
  const params = new URLSearchParams({ page: String(page), size: String(size) });
  if (filters.tier) params.set('tier', String(filters.tier));
  if (filters.kind) params.set('kind', filters.kind);
  const q = filters.q.trim();
  if (q) params.set('q', q);
  return api.get<HallEntryPage>(`${HALL_OF_FAME_URL}?${params.toString()}`);
}

export function createHallEntry(input: HallEntryCreateRequest) {
  return api.post<HallEntry>(HALL_OF_FAME_URL, input);
}

export function updateHallEntry(id: number, patch: HallEntryPatch) {
  return api.patch<HallEntry>(`${HALL_OF_FAME_URL}/${id}`, patch);
}

export function deleteHallEntry(id: number) {
  return api.del<void>(`${HALL_OF_FAME_URL}/${id}`);
}

export function previewHallImport(file: File) {
  const formData = new FormData();
  formData.append('file', file);
  return api.upload<HallImportPreview>(`${HALL_OF_FAME_URL}/import/preview`, formData);
}

export function commitHallImport(request: HallImportCommitRequest) {
  return api.post<HallImportCommitResult>(`${HALL_OF_FAME_URL}/import/commit`, request);
}
