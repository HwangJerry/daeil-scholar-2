import { useState } from 'react';
import type { AdminBannerAdRow, BannerAdOpenYn } from '../types/api.ts';

const KST_OFFSET_MINUTES = 9 * 60;
const TIME_ZONE_SUFFIX_PATTERN = /(?:Z|[+-]\d{2}:\d{2})$/i;

interface BannerAdFormState {
  bnName: string;
  bnUrl: string;
  openYn: BannerAdOpenYn;
  indx: number;
  bnStartDate: string;
  bnEndDate: string;
  imageUrls: string[];
  setBnName: (value: string) => void;
  setBnUrl: (value: string) => void;
  setOpenYn: (value: BannerAdOpenYn) => void;
  setIndx: (value: number) => void;
  setBnStartDate: (value: string) => void;
  setBnEndDate: (value: string) => void;
  setImageUrls: (value: string[]) => void;
  isValid: boolean;
}

export function utcToDatetimeLocal(utcStr: string | null | undefined): string {
  if (!utcStr) return '';
  const normalizedDateTime = utcStr.replace(' ', 'T');
  const dateTimeWithTimeZone = TIME_ZONE_SUFFIX_PATTERN.test(normalizedDateTime)
    ? normalizedDateTime
    : `${normalizedDateTime}Z`;
  const date = new Date(dateTimeWithTimeZone);
  if (Number.isNaN(date.getTime())) return '';
  const kstDate = new Date(date.getTime() + KST_OFFSET_MINUTES * 60 * 1000);
  return kstDate.toISOString().slice(0, 16);
}

export function useBannerAdForm(ad: AdminBannerAdRow | undefined): BannerAdFormState {
  const [bnName, setBnName] = useState(ad?.bnName ?? '');
  const [bnUrl, setBnUrl] = useState(ad?.bnUrl ?? '');
  const [openYn, setOpenYn] = useState(ad?.openYn ?? 'Y');
  const [indx, setIndx] = useState(ad?.indx ?? 0);
  const [bnStartDate, setBnStartDate] = useState(utcToDatetimeLocal(ad?.bnStartDate));
  const [bnEndDate, setBnEndDate] = useState(utcToDatetimeLocal(ad?.bnEndDate));
  const [imageUrls, setImageUrls] = useState(
    [...(ad?.images ?? [])]
      .sort((left, right) => left.sortOrder - right.sortOrder)
      .map((image) => image.imageUrl),
  );

  const isValid = bnName.trim().length > 0 && bnUrl.trim().length > 0 && imageUrls.length > 0;

  return {
    bnName,
    bnUrl,
    openYn,
    indx,
    bnStartDate,
    bnEndDate,
    imageUrls,
    setBnName,
    setBnUrl,
    setOpenYn,
    setIndx,
    setBnStartDate,
    setBnEndDate,
    setImageUrls,
    isValid,
  };
}
