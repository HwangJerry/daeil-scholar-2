// profileFieldValues — Shared type and default-value constant for the profile fields shown in signup/link forms.
export interface ProfileFieldValues {
  name: string;
  phone: string;
  email: string;
  fn: string;
  fmDept: string;
  jobCat: number | null;
  bizName: string;
  bizDesc: string;
  bizAddr: string;
  position: string;
  tags: string[];
  usrPhonePublic: 'Y' | 'N';
  usrEmailPublic: 'Y' | 'N';
}

export const defaultProfileFieldValues: ProfileFieldValues = {
  name: '',
  phone: '',
  email: '',
  fn: '',
  fmDept: '',
  jobCat: null,
  bizName: '',
  bizDesc: '',
  bizAddr: '',
  position: '',
  tags: [],
  usrPhonePublic: 'N',
  usrEmailPublic: 'N',
};
