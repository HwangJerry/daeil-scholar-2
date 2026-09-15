// appUpdate — API contract types for per-platform app update policies
export type AppPlatform = 'ios' | 'android';

export interface AppUpdatePolicy {
  forceEnabled: boolean;
  minBuild: number;
  recommendEnabled: boolean;
  recommendedBuild: number;
  minOsVersion: string;
  storeUrl?: string;
}

export interface AppUpdatePolicyRecord {
  platform: AppPlatform;
  policy: AppUpdatePolicy;
  updatedAt: string;
  updatedBy: number | null;
  /** Set when the stored row is missing or undecodable; the form is read-only. */
  unavailable?: boolean;
}

export interface AppUpdatePolicyListResponse {
  policies: AppUpdatePolicyRecord[];
}

export interface SaveAppUpdatePolicyRequest {
  platform: AppPlatform;
  policy: AppUpdatePolicy;
  /** The updatedAt last read, so a concurrent edit fails instead of being overwritten. */
  updatedAt: string;
  /** The policy being edited, which catches a concurrent save in the same second. */
  expectedPolicy: AppUpdatePolicy;
  /** Confirms a threshold the API has never observed from this platform. */
  allowUnobservedBuild: boolean;
}

export interface AppUpdatePolicyHistoryEntry {
  seq: number;
  platform: AppPlatform;
  before: string;
  after: string;
  changedBy: number | null;
  changedAt: string;
}

export interface AppUpdatePolicyHistoryResponse {
  entries: AppUpdatePolicyHistoryEntry[];
}

export interface AppClientBuild {
  platform: AppPlatform;
  build: number;
  versionName: string;
  firstSeenAt: string;
  lastSeenAt: string;
}

export interface AppClientBuildListResponse {
  builds: AppClientBuild[];
}

export interface AppUpdatePolicyFieldError {
  field: string;
  reason: string;
}
