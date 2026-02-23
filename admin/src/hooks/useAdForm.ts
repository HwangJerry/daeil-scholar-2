// useAdForm — manages form state for creating or editing an advertisement
import { useState } from 'react';
import type { AdminAdListItem } from '../types/api.ts';

interface AdFormState {
  maName: string;
  maUrl: string;
  maImg: string;
  maStatus: string;
  adTier: string;
  adTitleLabel: string;
  maIndx: number;
  setMaName: (v: string) => void;
  setMaUrl: (v: string) => void;
  setMaImg: (v: string) => void;
  setMaStatus: (v: string) => void;
  setAdTier: (v: string) => void;
  setAdTitleLabel: (v: string) => void;
  setMaIndx: (v: number) => void;
  isValid: boolean;
}

export function useAdForm(ad: AdminAdListItem | undefined): AdFormState {
  const [maName, setMaName] = useState(ad?.maName ?? '');
  const [maUrl, setMaUrl] = useState(ad?.maUrl ?? '');
  const [maImg, setMaImg] = useState(ad?.maImg ?? '');
  const [maStatus, setMaStatus] = useState(ad?.maStatus ?? 'Y');
  const [adTier, setAdTier] = useState<string>(ad?.adTier ?? 'NORMAL');
  const [adTitleLabel, setAdTitleLabel] = useState(ad?.adTitleLabel ?? '추천 동문 소식');
  const [maIndx, setMaIndx] = useState(ad?.maIndx ?? 0);

  const isValid = maName.trim().length > 0 && maUrl.trim().length > 0;

  return {
    maName, maUrl, maImg, maStatus, adTier, adTitleLabel, maIndx,
    setMaName, setMaUrl, setMaImg, setMaStatus, setAdTier, setAdTitleLabel, setMaIndx,
    isValid,
  };
}
