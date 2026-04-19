// useAdForm — manages form state for creating or editing an advertisement
import { useState } from 'react';
import type { AdminAdListItem } from '../types/api.ts';

const TIER_LABELS: Record<string, string> = {
  PREMIUM: '가장 핫한 동문 소식',
  GOLD: '추천 동문 소식',
  NORMAL: '동문 소식',
};

interface AdFormState {
  maName: string;
  maUrl: string;
  maImg: string;
  maStatus: string;
  adTier: string;
  adTitleLabel: string;
  maIndx: number;
  adStartDate: string;
  adEndDate: string;
  setMaName: (v: string) => void;
  setMaUrl: (v: string) => void;
  setMaImg: (v: string) => void;
  setMaStatus: (v: string) => void;
  setAdTierAndLabel: (v: string) => void;
  setAdTier: (v: string) => void;
  setAdTitleLabel: (v: string) => void;
  setMaIndx: (v: number) => void;
  setAdStartDate: (v: string) => void;
  setAdEndDate: (v: string) => void;
  isValid: boolean;
}

// utcToDatetimeLocal: "2026-03-17 00:00:00" (UTC DB string) → "2026-03-17T09:00" (KST datetime-local)
function utcToDatetimeLocal(utcStr: string | null | undefined): string {
  if (!utcStr) return '';
  const d = new Date(utcStr.replace(' ', 'T') + 'Z');
  if (isNaN(d.getTime())) return '';
  // Format as KST (UTC+9)
  const kst = new Date(d.getTime() + 9 * 60 * 60 * 1000);
  return kst.toISOString().slice(0, 16);
}

export function useAdForm(ad: AdminAdListItem | undefined): AdFormState {
  const [maName, setMaName] = useState(ad?.maName ?? '');
  const [maUrl, setMaUrl] = useState(ad?.maUrl ?? '');
  const [maImg, setMaImg] = useState(ad?.maImg ?? '');
  const [maStatus, setMaStatus] = useState(ad?.maStatus ?? 'Y');
  const [adTier, setAdTier] = useState<string>(ad?.adTier ?? 'NORMAL');
  const [adTitleLabel, setAdTitleLabel] = useState(ad?.adTitleLabel ?? TIER_LABELS['NORMAL']);
  const [maIndx, setMaIndx] = useState(ad?.maIndx ?? 0);
  const [adStartDate, setAdStartDate] = useState(utcToDatetimeLocal(ad?.adStartDate));
  const [adEndDate, setAdEndDate] = useState(utcToDatetimeLocal(ad?.adEndDate));

  const setAdTierAndLabel = (v: string) => {
    setAdTier(v);
    setAdTitleLabel(TIER_LABELS[v] ?? v);
  };

  const isValid = maName.trim().length > 0 && maUrl.trim().length > 0;

  return {
    maName, maUrl, maImg, maStatus, adTier, adTitleLabel, maIndx, adStartDate, adEndDate,
    setMaName, setMaUrl, setMaImg, setMaStatus, setAdTierAndLabel, setAdTier, setAdTitleLabel,
    setMaIndx, setAdStartDate, setAdEndDate,
    isValid,
  };
}
