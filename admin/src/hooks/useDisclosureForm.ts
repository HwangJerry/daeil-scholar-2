// useDisclosureForm — manages form state for creating or editing a disclosure (no pin)
import { useState } from 'react';
import type { NoticeDetail } from '../types/api.ts';

interface DisclosureFormState {
  subject: string;
  contentMd: string;
  setSubject: (v: string) => void;
  setContentMd: (v: string) => void;
  isValid: boolean;
}

export function useDisclosureForm(notice: NoticeDetail | undefined): DisclosureFormState {
  const [subject, setSubject] = useState(notice?.subject ?? '');
  const [contentMd, setContentMd] = useState(notice?.contentMd ?? '');

  const isValid = subject.trim().length > 0 && contentMd.trim().length > 0;

  return { subject, contentMd, setSubject, setContentMd, isValid };
}
