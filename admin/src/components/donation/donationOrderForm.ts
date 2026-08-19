import type {
  DonationOrder,
  DonationOrderInput,
  DonationOrderUpdateInput,
  DonationPaymentMethod,
  DonationSource,
  DonationStatus,
  DonationType,
} from '../../types/api.ts';

export interface DonationOrderFormValues {
  source: DonationSource;
  accountUsrSeq: string;
  transactionNumber: string;
  donationDate: string;
  donorName: string;
  donorCohort: string;
  donorDepartment: string;
  donorPhone: string;
  donationType: DonationType;
  grossAmount: string;
  refundedAmount: string;
  status: DonationStatus;
  paymentMethod: DonationPaymentMethod;
  memo: string;
}

export type DonationOrderFormErrors = Partial<Record<keyof DonationOrderFormValues, string>>;

const DONOR_PHONE_PATTERN = /^010(?:-[0-9]{4}-[0-9]{4}|[0-9]{8})$/;

function isASCII(value: string) {
  return [...value].every((character) => character.charCodeAt(0) <= 127);
}

function getToday() {
  const now = new Date();
  const timezoneOffsetMs = now.getTimezoneOffset() * 60_000;
  return new Date(now.getTime() - timezoneOffsetMs).toISOString().slice(0, 10);
}

export function createDonationOrderFormValues(order?: DonationOrder): DonationOrderFormValues {
  return {
    source: order?.source ?? 'bank_transfer',
    accountUsrSeq: order?.accountUsrSeq == null ? '' : String(order.accountUsrSeq),
    transactionNumber: order?.transactionNumber ?? '',
    donationDate: order?.donationDate ?? getToday(),
    donorName: order?.donor.name ?? '',
    donorCohort: order?.donor.cohort ?? '',
    donorDepartment: order?.donor.department ?? '',
    donorPhone: order?.donor.phone ?? '',
    donationType: order?.donationType ?? 'one_time',
    grossAmount: order ? String(order.grossAmount) : '',
    refundedAmount: String(order?.refundedAmount ?? 0),
    status: order?.status ?? 'completed',
    paymentMethod: order?.paymentMethod ?? 'bank',
    memo: order?.memo ?? '',
  };
}

export function validateDonationOrderForm(values: DonationOrderFormValues) {
  const errors: DonationOrderFormErrors = {};
  const grossAmount = Number(values.grossAmount);
  const refundedAmount = Number(values.refundedAmount);
  const accountUsrSeq = Number(values.accountUsrSeq);

  if (!values.donationDate) errors.donationDate = '기부일자를 선택해 주세요.';
  if (!values.donorName.trim()) errors.donorName = '기부자명을 입력해 주세요.';
  if (!values.donorCohort.trim()) errors.donorCohort = '기수를 입력해 주세요.';
  if (!values.donorDepartment.trim()) errors.donorDepartment = '학과를 입력해 주세요.';
  if (!DONOR_PHONE_PATTERN.test(values.donorPhone.trim())) {
    errors.donorPhone = '010-1234-5678 또는 01012345678 형식으로 입력해 주세요.';
  }
  if (values.accountUsrSeq !== '' && (!Number.isInteger(accountUsrSeq) || accountUsrSeq <= 0)) {
    errors.accountUsrSeq = '회원 번호는 1 이상의 정수여야 합니다.';
  }
  const transactionNumber = values.transactionNumber.trim();
  if (transactionNumber && (transactionNumber.length > 191 || !isASCII(transactionNumber))) {
    errors.transactionNumber = '거래번호는 191자 이하의 영문, 숫자, 기호만 사용할 수 있습니다.';
  }
  if (values.grossAmount.trim() === '' || !Number.isInteger(grossAmount) || grossAmount < 0) {
    errors.grossAmount = '총액은 0 이상의 정수여야 합니다.';
  }
  if (values.refundedAmount.trim() === '' || !Number.isInteger(refundedAmount) || refundedAmount < 0 || refundedAmount > grossAmount) {
    errors.refundedAmount = '환불액은 총액 이하의 0 이상의 정수여야 합니다.';
  } else if (values.status === 'partially_refunded' && (
    refundedAmount === 0 || refundedAmount === grossAmount
  )) {
    errors.refundedAmount = '부분 환불 상태는 0원보다 크고 총액보다 작은 환불액이 필요합니다.';
  } else if (values.status === 'fully_refunded' && refundedAmount !== grossAmount) {
    errors.refundedAmount = '전액 환불 상태의 환불액은 총액과 같아야 합니다.';
  } else if (
    !['partially_refunded', 'fully_refunded'].includes(values.status)
    && refundedAmount !== 0
  ) {
    errors.refundedAmount = '선택한 상태에서는 환불액이 0원이어야 합니다.';
  }

  return errors;
}

function toDonationOrderInputWithoutAccount(
  values: DonationOrderFormValues,
): Omit<DonationOrderInput, 'accountUsrSeq'> {
  const transactionNumber = values.transactionNumber.trim();
  const memo = values.memo.trim();

  return {
    source: values.source,
    transactionNumber: transactionNumber || null,
    donationDate: values.donationDate,
    donor: {
      name: values.donorName.trim(),
      cohort: values.donorCohort.trim(),
      department: values.donorDepartment.trim(),
      phone: values.donorPhone.trim(),
    },
    donationType: values.donationType,
    grossAmount: Number(values.grossAmount),
    refundedAmount: Number(values.refundedAmount),
    status: values.status,
    paymentMethod: values.paymentMethod,
    memo: memo || null,
  };
}

function parseAccountUsrSeq(value: string) {
  const accountUsrSeq = value.trim();
  return accountUsrSeq ? Number(accountUsrSeq) : null;
}

export function toDonationOrderInput(values: DonationOrderFormValues): DonationOrderInput {
  return {
    ...toDonationOrderInputWithoutAccount(values),
    accountUsrSeq: parseAccountUsrSeq(values.accountUsrSeq),
  };
}

export function toDonationOrderUpdateInput(
  values: DonationOrderFormValues,
  initialAccountUsrSeq: number | null,
): DonationOrderUpdateInput {
  const input = toDonationOrderInputWithoutAccount(values);
  const accountUsrSeq = parseAccountUsrSeq(values.accountUsrSeq);
  if (accountUsrSeq === initialAccountUsrSeq) return input;
  return { ...input, accountUsrSeq };
}
