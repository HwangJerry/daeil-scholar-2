// public.ts — Public (unauthenticated) reference data API client functions.
import { api } from './client';
import type { JobCategory } from '../types/api';

/** Fetch all active job categories (no auth required, used on registration forms). */
export function fetchPublicJobCategories(): Promise<JobCategory[]> {
  return api.get<JobCategory[]>('/api/public/job-categories');
}
