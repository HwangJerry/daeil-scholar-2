// donate — Type definitions for donation domain
export interface CreateOrderRequest {
  amount: number;
  payType: 'CARD' | 'BANK' | 'HP';
  gate: 'immediately' | 'profile';
}

export interface PaymentParams {
  mallId: string;
  orderNo: string;
  productAmt: string;
  productName: string;
  payType: string;
  returnUrl: string;
  relayUrl: string;
  windowType: string;
  userName: string;
  mallName: string;
  currency: string;
  charset: string;
  langFlag: string;
}

export interface CreateOrderResponse {
  orderSeq: number;
  paymentParams: PaymentParams | null;
}

export interface MyDonationItem {
  orderSeq: number;
  amount: number;
  payType: string;
  paidAt: string;
}

export interface MyDonationResponse {
  items: MyDonationItem[];
  totalAmount: number;
  totalCount: number;
}
