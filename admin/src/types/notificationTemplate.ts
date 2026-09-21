// notificationTemplate — API contract types for admin-editable notification texts
export type NotificationChannel = 'sms' | 'push';

/** One `{name}` token a template may use, with the sample used for previews. */
export interface NotificationTemplatePlaceholder {
  name: string;
  required: boolean;
  description: string;
  sample: string;
}

/** Channel caps mirrored from the server; only the channel's fields are set. */
export interface NotificationChannelLimits {
  maxBodyEucKrBytes?: number;
  maxTitleRunes?: number;
  maxBodyRunes?: number;
}

export interface NotificationTemplateView {
  key: string;
  channel: NotificationChannel;
  displayName: string;
  description: string;
  title: string;
  body: string;
  defaultTitle: string;
  defaultBody: string;
  /** True while nothing is stored and the default text is what gets sent. */
  isDefault: boolean;
  /** True when the stored text no longer validates and the default is sent. */
  invalid: boolean;
  version: number;
  updatedAt: string | null;
  updatedBy: number | null;
  placeholders: NotificationTemplatePlaceholder[];
  limits: NotificationChannelLimits;
}

export interface SaveNotificationTemplateRequest {
  key: string;
  title: string;
  body: string;
  /** The version the card was loaded with, so a concurrent edit is rejected. */
  expectedVersion: number;
}

/** The editable half of a template; what the card's form holds. */
export interface NotificationTemplateDraft {
  title: string;
  body: string;
}

/** Per-field messages, keyed by the field the server or the form rejected. */
export type NotificationTemplateFieldErrors = Partial<Record<'title' | 'body', string>>;
