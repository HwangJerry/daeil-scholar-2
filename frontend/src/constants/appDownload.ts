// appDownload — Validated app marketplace links sourced from Vite environment variables
export interface AppDownloadLink {
  readonly url: string | null;
  readonly available: boolean;
}

export interface AppDownloadLinks {
  readonly appStore: AppDownloadLink;
  readonly googlePlay: AppDownloadLink;
}

const HTTPS_PROTOCOL = 'https:';

export function createAppDownloadLink(rawUrl: string | undefined): AppDownloadLink {
  const trimmedUrl = rawUrl?.trim();

  if (!trimmedUrl) {
    return { url: null, available: false };
  }

  try {
    const parsedUrl = new URL(trimmedUrl);
    if (parsedUrl.protocol !== HTTPS_PROTOCOL) {
      return { url: null, available: false };
    }

    return { url: parsedUrl.toString(), available: true };
  } catch {
    return { url: null, available: false };
  }
}

export const APP_DOWNLOAD_LINKS: AppDownloadLinks = {
  appStore: createAppDownloadLink(import.meta.env.VITE_APP_STORE_URL),
  googlePlay: createAppDownloadLink(import.meta.env.VITE_GOOGLE_PLAY_URL),
};
