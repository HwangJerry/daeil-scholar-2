// DonationOrderForm — shared create/edit dialog for one canonical donation order
import * as Dialog from '@radix-ui/react-dialog';
import { Loader2, X } from 'lucide-react';
import { useState } from 'react';
import { ApiClientError } from '../../api/client.ts';
import {
  useCreateDonationOrder,
  useUpdateDonationOrder,
} from '../../hooks/useDonationOrders.ts';
import type { DonationOrder } from '../../types/api.ts';
import { Button } from '../ui/Button.tsx';
import { Input } from '../ui/Input.tsx';
import { Select } from '../ui/Select.tsx';
import { Textarea } from '../ui/Textarea.tsx';
import {
  createDonationOrderFormValues,
  toDonationOrderInput,
  toDonationOrderUpdateInput,
  validateDonationOrderForm,
  type DonationOrderFormErrors,
  type DonationOrderFormValues,
} from './donationOrderForm.ts';

interface DonationOrderFormProps {
  open: boolean;
  order?: DonationOrder;
  onOpenChange: (open: boolean) => void;
}

interface FormFieldProps {
  htmlFor: string;
  label: string;
  error?: string;
  required?: boolean;
  children: React.ReactNode;
}

function FormField({ htmlFor, label, error, required, children }: FormFieldProps) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={htmlFor} className="block text-sm font-medium text-dark-slate">
        {label} {required && <span className="text-error-text">*</span>}
      </label>
      {children}
      {error && <p className="text-xs text-error-text">{error}</p>}
    </div>
  );
}

function getRequestErrorMessage(error: unknown) {
  if (error instanceof ApiClientError && error.status === 409 && error.code === 'DONATION_ORDER_STALE') {
    return '다른 관리자가 먼저 수정했습니다. 새로고침 후 다시 시도해주세요.';
  }
  if (error instanceof ApiClientError) return error.message;
  return '네트워크 상태를 확인하고 다시 시도해 주세요.';
}

export function DonationOrderForm(props: DonationOrderFormProps) {
  if (!props.open) return null;
  return <DonationOrderFormDialog {...props} />;
}

function DonationOrderFormDialog({ open, order, onOpenChange }: DonationOrderFormProps) {
  const createMutation = useCreateDonationOrder();
  const updateMutation = useUpdateDonationOrder();
  const [values, setValues] = useState(() => createDonationOrderFormValues(order));
  const [errors, setErrors] = useState<DonationOrderFormErrors>({});
  const [requestError, setRequestError] = useState('');
  const isEditing = order !== undefined;
  const isSubmitting = createMutation.isPending || updateMutation.isPending;

  const setValue = <Key extends keyof DonationOrderFormValues>(
    name: Key,
    value: DonationOrderFormValues[Key],
  ) => {
    setValues((current) => ({ ...current, [name]: value }));
    setErrors((current) => ({ ...current, [name]: undefined }));
    setRequestError('');
  };

  const handleSubmit = (event: React.FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const validationErrors = validateDonationOrderForm(values);
    setErrors(validationErrors);
    if (Object.keys(validationErrors).length > 0) return;

    const mutationOptions = {
      onSuccess: () => {
        setValues(createDonationOrderFormValues());
        setRequestError('');
        onOpenChange(false);
      },
      onError: (error: unknown) => setRequestError(getRequestErrorMessage(error)),
    };

    if (order) {
      const input = toDonationOrderUpdateInput(values, order.accountUsrSeq, order.lastEditedAt);
      updateMutation.mutate({ orderSeq: order.orderSeq, input }, mutationOptions);
      return;
    }
    const input = toDonationOrderInput(values);
    createMutation.mutate(input, mutationOptions);
  };

  return (
    <Dialog.Root
      open={open}
      onOpenChange={(nextOpen) => { if (!isSubmitting) onOpenChange(nextOpen); }}
    >
      <Dialog.Portal>
        <Dialog.Overlay className="fixed inset-0 z-50 bg-black/40" />
        <Dialog.Content className="fixed left-1/2 top-1/2 z-50 max-h-[90vh] w-[calc(100%-2rem)] max-w-4xl -translate-x-1/2 -translate-y-1/2 overflow-y-auto rounded-2xl bg-white p-6 shadow-xl">
          <div className="flex items-start justify-between gap-4">
            <div>
              <Dialog.Title className="text-lg font-bold text-dark-slate">
                {isEditing ? '기부 주문 수정' : '새 기부 등록'}
              </Dialog.Title>
              <Dialog.Description className="mt-1 text-sm text-cool-gray">
                기부 거래와 기부자 정보를 입력해 주세요.
              </Dialog.Description>
            </div>
            <Dialog.Close asChild>
              <button
                type="button"
                aria-label="닫기"
                disabled={isSubmitting}
                className="rounded-lg p-1 text-cool-gray hover:bg-background disabled:opacity-50"
              >
                <X className="h-5 w-5" />
              </button>
            </Dialog.Close>
          </div>

          <form className="mt-6 space-y-6" noValidate onSubmit={handleSubmit}>
            <div className="grid gap-4 md:grid-cols-3">
              <FormField htmlFor="donation-source" label="유입 경로" required>
                <Select id="donation-source" value={values.source} disabled={isSubmitting} onChange={(event) => setValue('source', event.target.value as DonationOrderFormValues['source'])}>
                  <option value="happy_nanum">해피나눔</option>
                  <option value="bank_transfer">계좌이체</option>
                  <option value="other">기타</option>
                </Select>
              </FormField>
              <FormField htmlFor="donation-account" label="회원 번호" error={errors.accountUsrSeq}>
                <Input id="donation-account" type="number" min="1" step="1" placeholder="연결하지 않으면 비워두기" value={values.accountUsrSeq} disabled={isSubmitting} onChange={(event) => setValue('accountUsrSeq', event.target.value)} />
              </FormField>
              <FormField htmlFor="donation-transaction" label="거래번호" error={errors.transactionNumber}>
                <Input id="donation-transaction" maxLength={191} placeholder="없으면 비워두기" value={values.transactionNumber} disabled={isSubmitting} onChange={(event) => setValue('transactionNumber', event.target.value)} />
              </FormField>
              <FormField htmlFor="donation-date" label="기부일자" error={errors.donationDate} required>
                <Input id="donation-date" type="date" value={values.donationDate} disabled={isSubmitting} onChange={(event) => setValue('donationDate', event.target.value)} />
              </FormField>
              <FormField htmlFor="donation-type" label="기부유형" required>
                <Select id="donation-type" value={values.donationType} disabled={isSubmitting} onChange={(event) => setValue('donationType', event.target.value as DonationOrderFormValues['donationType'])}>
                  <option value="recurring">정기기부</option>
                  <option value="one_time">일시기부</option>
                  <option value="sponsorship">후원</option>
                </Select>
              </FormField>
              <FormField htmlFor="donation-payment-method" label="결제방식" required>
                <Select id="donation-payment-method" value={values.paymentMethod} disabled={isSubmitting} onChange={(event) => setValue('paymentMethod', event.target.value as DonationOrderFormValues['paymentMethod'])}>
                  <option value="card">카드</option>
                  <option value="bank">계좌이체</option>
                  <option value="virtual_bank">가상계좌</option>
                  <option value="mobile">휴대폰</option>
                  <option value="admin">관리자 등록</option>
                  <option value="other">기타</option>
                </Select>
              </FormField>
            </div>

            <fieldset>
              <legend className="mb-3 text-sm font-semibold text-dark-slate">기부자 정보</legend>
              <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-4">
                <FormField htmlFor="donor-name" label="기부자명" error={errors.donorName} required>
                  <Input id="donor-name" value={values.donorName} disabled={isSubmitting} onChange={(event) => setValue('donorName', event.target.value)} />
                </FormField>
                <FormField htmlFor="donor-cohort" label="기수" error={errors.donorCohort} required>
                  <Input id="donor-cohort" value={values.donorCohort} disabled={isSubmitting} onChange={(event) => setValue('donorCohort', event.target.value)} />
                </FormField>
                <FormField htmlFor="donor-department" label="학과" error={errors.donorDepartment} required>
                  <Input id="donor-department" value={values.donorDepartment} disabled={isSubmitting} onChange={(event) => setValue('donorDepartment', event.target.value)} />
                </FormField>
                <FormField htmlFor="donor-phone" label="전화번호" error={errors.donorPhone} required>
                  <Input id="donor-phone" type="tel" placeholder="010-1234-5678" value={values.donorPhone} disabled={isSubmitting} onChange={(event) => setValue('donorPhone', event.target.value)} />
                </FormField>
              </div>
            </fieldset>

            <div className="grid gap-4 md:grid-cols-3">
              <FormField htmlFor="donation-gross" label="총액" error={errors.grossAmount} required>
                <Input id="donation-gross" type="number" min="0" step="1" value={values.grossAmount} disabled={isSubmitting} onChange={(event) => setValue('grossAmount', event.target.value)} />
              </FormField>
              <FormField htmlFor="donation-refunded" label="환불액" error={errors.refundedAmount} required>
                <Input id="donation-refunded" type="number" min="0" step="1" value={values.refundedAmount} disabled={isSubmitting} onChange={(event) => setValue('refundedAmount', event.target.value)} />
              </FormField>
              <FormField htmlFor="donation-status" label="상태" required>
                <Select id="donation-status" value={values.status} disabled={isSubmitting} onChange={(event) => setValue('status', event.target.value as DonationOrderFormValues['status'])}>
                  <option value="scheduled">예약</option>
                  <option value="pending">대기</option>
                  <option value="completed">완료</option>
                  <option value="partially_refunded">부분 환불</option>
                  <option value="cancelled">취소</option>
                  <option value="fully_refunded">전액 환불</option>
                </Select>
              </FormField>
            </div>

            <FormField htmlFor="donation-memo" label="메모">
              <Textarea id="donation-memo" rows={3} placeholder="선택 입력" value={values.memo} disabled={isSubmitting} onChange={(event) => setValue('memo', event.target.value)} />
            </FormField>

            {requestError && (
              <p role="alert" className="rounded-xl bg-error-subtle px-4 py-3 text-sm text-error-text">
                {requestError}
              </p>
            )}

            <div className="flex justify-end gap-3">
              <Dialog.Close asChild>
                <Button type="button" variant="outline" disabled={isSubmitting}>취소</Button>
              </Dialog.Close>
              <Button type="submit" disabled={isSubmitting}>
                {isSubmitting && <Loader2 className="mr-2 h-4 w-4 animate-spin" />}
                {isSubmitting ? '저장 중...' : isEditing ? '수정 저장' : '등록'}
              </Button>
            </div>
          </form>
        </Dialog.Content>
      </Dialog.Portal>
    </Dialog.Root>
  );
}
