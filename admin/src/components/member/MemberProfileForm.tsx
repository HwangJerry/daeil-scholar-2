// MemberProfileForm — dialog for administrators to correct a member's name, phone, email, cohort and department
import * as Dialog from '@radix-ui/react-dialog';
import { Loader2, X } from 'lucide-react';
import { useState } from 'react';
import { ApiClientError } from '../../api/client.ts';
import { DEPARTMENTS } from '../../constants/departments.ts';
import { useMemberProfileUpdate } from '../../hooks/useMemberProfileUpdate.ts';
import type { AdminMemberDetail } from '../../types/api.ts';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';
import { Select } from '../ui/Select.tsx';
import {
  createMemberProfileFormValues,
  toMemberProfileInput,
  validateMemberProfileForm,
  type MemberProfileFormErrors,
  type MemberProfileFormValues,
} from './memberProfileForm.ts';

interface MemberProfileFormProps {
  open: boolean;
  member: AdminMemberDetail;
  onOpenChange: (open: boolean) => void;
}

interface FieldProps {
  htmlFor: string;
  label: string;
  hint?: string;
  error?: string;
  required?: boolean;
  children: React.ReactNode;
}

function Field({ htmlFor, label, hint, error, required, children }: FieldProps) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={htmlFor} className="block text-sm font-medium text-dark-slate">
        {label} {required && <span className="text-error-text">*</span>}
      </label>
      {children}
      {error ? <p className="text-xs text-error-text">{error}</p> : hint && <p className="text-xs text-cool-gray">{hint}</p>}
    </div>
  );
}

function getRequestErrorMessage(error: unknown) {
  if (error instanceof ApiClientError) return error.message;
  return '네트워크 상태를 확인하고 다시 시도해 주세요.';
}

export function MemberProfileForm(props: MemberProfileFormProps) {
  if (!props.open) return null;
  return <MemberProfileFormDialog {...props} />;
}

function MemberProfileFormDialog({ open, member, onOpenChange }: MemberProfileFormProps) {
  const mutation = useMemberProfileUpdate(String(member.usrSeq));
  const [values, setValues] = useState(() => createMemberProfileFormValues(member));
  const [errors, setErrors] = useState<MemberProfileFormErrors>({});
  const [requestError, setRequestError] = useState('');
  const isSubmitting = mutation.isPending;

  const setValue = (name: keyof MemberProfileFormValues, value: string) => {
    setValues((current) => ({ ...current, [name]: value }));
    setErrors((current) => ({ ...current, [name]: undefined }));
    setRequestError('');
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const validationErrors = validateMemberProfileForm(values);
    setErrors(validationErrors);
    if (Object.keys(validationErrors).length > 0) return;
    mutation.mutate(toMemberProfileInput(values), {
      onSuccess: () => onOpenChange(false),
      onError: (error) => setRequestError(getRequestErrorMessage(error)),
    });
  };

  return (
    <Dialog.Root open={open} onOpenChange={(nextOpen) => { if (!isSubmitting) onOpenChange(nextOpen); }}>
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[calc(100%-2rem)] max-w-lg -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl bg-white p-6 shadow-xl">
          <div className="flex items-start justify-between gap-4">
            <div>
              <Dialog.Title className="text-lg font-bold text-dark-slate">회원 정보 수정</Dialog.Title>
              <Dialog.Description className="mt-1 text-sm text-cool-gray">
                승인된 회원은 기수·학과가 동문 인증 정보에도 함께 반영됩니다.
              </Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <button type="button" aria-label="닫기" disabled={isSubmitting} className="rounded-lg p-1 text-cool-gray hover:bg-background disabled:opacity-50">
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>

          <form className="mt-6 space-y-4" noValidate onSubmit={handleSubmit}>
            <Field htmlFor="member-name" label="이름" required error={errors.usrName}>
              <Input id="member-name" value={values.usrName} disabled={isSubmitting} onChange={(e) => setValue('usrName', e.target.value)} />
            </Field>
            <Field htmlFor="member-phone" label="휴대폰" error={errors.usrPhone} hint="다른 회원이 쓰는 번호는 저장되지 않습니다. 비우면 기존 번호를 유지합니다.">
              <Input id="member-phone" inputMode="tel" value={values.usrPhone} disabled={isSubmitting} onChange={(e) => setValue('usrPhone', e.target.value)} />
            </Field>
            <Field htmlFor="member-email" label="연락용 이메일" error={errors.usrEmail} hint="로그인 이메일은 바뀌지 않습니다.">
              <Input id="member-email" type="email" value={values.usrEmail} disabled={isSubmitting} onChange={(e) => setValue('usrEmail', e.target.value)} />
            </Field>
            <div className="grid gap-4 sm:grid-cols-2">
              <Field htmlFor="member-cohort" label="기수" error={errors.usrFn}>
                <Input id="member-cohort" inputMode="numeric" value={values.usrFn} disabled={isSubmitting} onChange={(e) => setValue('usrFn', e.target.value)} />
              </Field>
              <Field htmlFor="member-dept" label="학과" error={errors.usrDept}>
                <Select id="member-dept" value={values.usrDept} disabled={isSubmitting} onChange={(e) => setValue('usrDept', e.target.value)}>
                  <option value="">변경 안 함</option>
                  {DEPARTMENTS.map((dept) => <option key={dept} value={dept}>{dept}</option>)}
                </Select>
              </Field>
            </div>

            {requestError && <p role="alert" className="rounded-lg bg-red-50 px-3 py-2 text-sm text-error-text">{requestError}</p>}

            <div className="flex justify-end gap-2 pt-2">
              <Dialog.Close asChild>
                <Button type="button" variant="ghost" disabled={isSubmitting}>취소</Button>
              </Dialog.Close>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                저장
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
