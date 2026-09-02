// appSettings API — list and update remotely managed application settings
import { api } from './client.ts';
import type { AppSetting, UpdateAppSettingRequest } from '../types/appSettings.ts';

const APP_SETTINGS_ENDPOINT = '/api/admin/settings';

export function fetchAppSettings() {
  return api.get<AppSetting[]>(APP_SETTINGS_ENDPOINT);
}

export function updateAppSetting({ key, value }: UpdateAppSettingRequest) {
  const encodedKey = encodeURIComponent(key);
  return api.put<void>(`${APP_SETTINGS_ENDPOINT}/${encodedKey}`, { value });
}
