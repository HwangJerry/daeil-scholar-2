// appUpdate API — per-platform update policies, their history, and observed builds
import { api } from './client.ts';
import type {
  AppClientBuildListResponse,
  AppPlatform,
  AppUpdatePolicyHistoryResponse,
  AppUpdatePolicyListResponse,
  SaveAppUpdatePolicyRequest,
} from '../types/appUpdate.ts';

const POLICIES_ENDPOINT = '/api/admin/app-update-policies';
const CLIENT_BUILDS_ENDPOINT = '/api/admin/app-client-builds';

export function fetchAppUpdatePolicies() {
  return api.get<AppUpdatePolicyListResponse>(POLICIES_ENDPOINT);
}

export function saveAppUpdatePolicy({
  platform,
  policy,
  updatedAt,
  expectedPolicy,
  allowUnobservedBuild,
}: SaveAppUpdatePolicyRequest) {
  return api.put<void>(`${POLICIES_ENDPOINT}/${platform}`, {
    policy,
    updatedAt,
    expectedPolicy,
    allowUnobservedBuild,
  });
}

export function fetchAppUpdatePolicyHistory(platform: AppPlatform) {
  return api.get<AppUpdatePolicyHistoryResponse>(`${POLICIES_ENDPOINT}/${platform}/history`);
}

export function fetchAppClientBuilds(platform: AppPlatform) {
  return api.get<AppClientBuildListResponse>(`${CLIENT_BUILDS_ENDPOINT}?platform=${platform}`);
}
