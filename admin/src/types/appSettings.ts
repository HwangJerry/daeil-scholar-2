// appSettings — API contract types for remotely managed application settings
export interface AppSetting {
  key: string;
  value: string;
  description: string;
  public: 'Y' | 'N';
  updatedAt: string;
  updatedBy: number | null;
}

export interface UpdateAppSettingRequest {
  key: string;
  value: string;
}
