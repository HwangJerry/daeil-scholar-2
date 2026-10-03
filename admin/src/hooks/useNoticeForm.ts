// useNoticeForm — manages form state for creating or editing a notice
import { useState } from 'react';
import { useFeedCategories } from './useFeedCategories.ts';
import { defaultCategorySeq } from '../components/notice/noticeCategoryOptions.ts';
import type { NoticeDetail } from '../types/api.ts';

interface NoticeFormState {
  subject: string;
  contentMd: string;
  isPinned: boolean;
  /** Chosen category, else the post's saved one, else the default once loaded. */
  categorySeq: number | null;
  setCategorySeq: (v: number) => void;
  setSubject: (v: string) => void;
  setContentMd: (v: string) => void;
  setIsPinned: (v: boolean) => void;
  /** Publish under the official profile; on by default for new posts, the stored value on edit. */
  officialProfile: boolean;
  setOfficialProfile: (v: boolean) => void;
  isValid: boolean;
}

export function useNoticeForm(notice: NoticeDetail | undefined): NoticeFormState {
  const [subject, setSubject] = useState(notice?.subject ?? '');
  const [contentMd, setContentMd] = useState(notice?.contentMd ?? '');
  const [isPinned, setIsPinned] = useState(notice?.isPinned === 'Y');
  const [officialProfile, setOfficialProfile] = useState(notice ? notice.officialProfile === true : true);
  const [chosenCategorySeq, setCategorySeq] = useState<number | null>(null);
  const { data: categories = [] } = useFeedCategories();
  const categorySeq = chosenCategorySeq ?? notice?.categorySeq ?? defaultCategorySeq(categories);

  const isValid = subject.trim().length > 0 && contentMd.trim().length > 0 && categorySeq !== null;

  return {
    subject, contentMd, isPinned, categorySeq, setCategorySeq, setSubject, setContentMd, setIsPinned,
    officialProfile, setOfficialProfile, isValid,
  };
}
