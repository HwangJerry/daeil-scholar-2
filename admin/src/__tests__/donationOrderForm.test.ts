import { describe, expect, it } from 'vitest';
import {
  createDonationOrderFormValues,
  toDonationOrderInput,
  toDonationOrderUpdateInput,
  validateDonationOrderForm,
} from '../components/donation/donationOrderForm.ts';

describe('donation order form helpers', () => {
  it('validates the canonical required fields and refund rules', () => {
    const values = createDonationOrderFormValues();
    values.donationDate = '';
    values.donorName = '';
    values.donorCohort = '';
    values.donorDepartment = '';
    values.donorPhone = '010-123-4567';
    values.grossAmount = '10000';
    values.refundedAmount = '10000';
    values.status = 'partially_refunded';

    expect(validateDonationOrderForm(values)).toMatchObject({
      donationDate: expect.any(String),
      donorName: expect.any(String),
      donorCohort: expect.any(String),
      donorDepartment: expect.any(String),
      donorPhone: expect.any(String),
      refundedAmount: expect.any(String),
    });
  });

  it('serializes optional inputs as explicit null values', () => {
    const values = createDonationOrderFormValues();
    Object.assign(values, {
      donorName: '홍길동',
      donorCohort: '30기',
      donorDepartment: '경영학과',
      donorPhone: '010-1234-5678',
      grossAmount: '50000',
      accountUsrSeq: '',
      transactionNumber: '   ',
      memo: '',
    });

    expect(toDonationOrderInput(values)).toMatchObject({
      accountUsrSeq: null,
      transactionNumber: null,
      memo: null,
      grossAmount: 50000,
      refundedAmount: 0,
      donor: {
        name: '홍길동',
        cohort: '30기',
        department: '경영학과',
        phone: '010-1234-5678',
      },
    });
  });

  it('omits accountUsrSeq from an update when the loaded value is unchanged', () => {
    const values = createDonationOrderFormValues();
    values.accountUsrSeq = '42';

    expect(toDonationOrderUpdateInput(values, 42)).not.toHaveProperty('accountUsrSeq');
  });

  it('omits accountUsrSeq from an update when an unlinked order stays unchanged', () => {
    const values = createDonationOrderFormValues();

    expect(toDonationOrderUpdateInput(values, null)).not.toHaveProperty('accountUsrSeq');
  });

  it('serializes an explicitly cleared accountUsrSeq as null for an update', () => {
    const values = createDonationOrderFormValues();
    values.accountUsrSeq = '';

    expect(toDonationOrderUpdateInput(values, 42)).toHaveProperty('accountUsrSeq', null);
  });

  it('serializes a newly entered accountUsrSeq value for an update', () => {
    const values = createDonationOrderFormValues();
    values.accountUsrSeq = '73';

    expect(toDonationOrderUpdateInput(values, null)).toHaveProperty('accountUsrSeq', 73);
  });
});
