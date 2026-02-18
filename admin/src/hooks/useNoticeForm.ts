// useNoticeForm — manages form state for creating or editing a notice
import { useState } from 'react';
import type { NoticeDetail } from '../types/api.ts';

interface NoticeFormState {
  subject: string;
  contentMd: string;
  isPinned: boolean;
  setSubject: (v: string) => void;
  setContentMd: (v: string) => void;
  setIsPinned: (v: boolean) => void;
  isValid: boolean;
}

export function useNoticeForm(notice: NoticeDetail | undefined): NoticeFormState {
  const [subject, setSubject] = useState(notice?.subject ?? '');
  const [contentMd, setContentMd] = useState(notice?.contentMd ?? '');
  const [isPinned, setIsPinned] = useState(notice?.isPinned === 'Y');

  const isValid = subject.trim().length > 0 && contentMd.trim().length > 0;

  return { subject, contentMd, isPinned, setSubject, setContentMd, setIsPinned, isValid };
}
