// EasyPayBridge — hidden form that loads the EasyPay SDK and initiates payment
import { useEffect, useRef } from 'react';
import type { PaymentParams } from '../../types/donate';

type SdkFunction = (
  form: HTMLFormElement,
  relayUrl: string,
  target: string,
  width: string,
  height: string,
  windowType: string,
  timeout: number,
) => void;

declare global {
  interface Window {
    easypay_card_webpay: SdkFunction;
  }
}

const SDK_URL = 'https://sp.easypay.co.kr/webpay/EasypayCard_Web.js';
const SDK_LOAD_TIMEOUT_MS = 10_000;

interface EasyPayBridgeProps {
  params: PaymentParams;
  onError?: (message: string) => void;
}

export function EasyPayBridge({ params, onError }: EasyPayBridgeProps) {
  const formRef = useRef<HTMLFormElement>(null);
  const hasTriggeredRef = useRef(false);

  useEffect(() => {
    if (hasTriggeredRef.current) return;

    const script = document.createElement('script');
    script.src = SDK_URL;
    script.async = true;

    const timeoutId = setTimeout(() => {
      hasTriggeredRef.current = true;
      console.error('[EasyPayBridge] SDK load timed out');
      onError?.('결제 모듈 로드 시간이 초과되었습니다.');
    }, SDK_LOAD_TIMEOUT_MS);

    script.onload = () => {
      clearTimeout(timeoutId);

      if (!formRef.current) return;
      if (typeof window.easypay_card_webpay !== 'function') {
        console.error('[EasyPayBridge] easypay_card_webpay not available after script load');
        onError?.('결제 모듈 초기화에 실패했습니다.');
        return;
      }

      hasTriggeredRef.current = true;
      window.easypay_card_webpay(
        formRef.current,
        params.relayUrl,
        '_self',
        '0',
        '0',
        params.windowType,
        30,
      );
    };

    script.onerror = () => {
      clearTimeout(timeoutId);
      console.error('[EasyPayBridge] Failed to load SDK script');
      onError?.('결제 모듈을 불러올 수 없습니다.');
    };

    document.head.appendChild(script);

    return () => {
      clearTimeout(timeoutId);
      if (script.parentNode) {
        script.parentNode.removeChild(script);
      }
    };
  }, [params, onError]);

  return (
    <form ref={formRef} name="payment" method="post" style={{ display: 'none' }}>
      {/* Request fields */}
      <input type="hidden" name="sp_mall_id" value={params.mallId} />
      <input type="hidden" name="sp_mall_nm" value={params.mallName} />
      <input type="hidden" name="sp_order_no" value={params.orderNo} />
      <input type="hidden" name="sp_currency" value={params.currency} />
      <input type="hidden" name="sp_return_url" value={params.returnUrl} />
      <input type="hidden" name="sp_lang_flag" value={params.langFlag} />
      <input type="hidden" name="sp_charset" value={params.charset} />
      <input type="hidden" name="sp_user_nm" value={params.userName} />
      <input type="hidden" name="sp_product_amt" value={params.productAmt} />
      <input type="hidden" name="sp_product_nm" value={params.productName} />
      <input type="hidden" name="sp_pay_type" value={params.payType} />
      <input type="hidden" name="sp_window_type" value={params.windowType} />
      {/* Response fields (populated by EasyPay SDK) */}
      <input type="hidden" name="sp_res_cd" />
      <input type="hidden" name="sp_res_msg" />
      <input type="hidden" name="sp_tr_cd" />
      <input type="hidden" name="sp_ret_pay_type" />
      <input type="hidden" name="sp_trace_no" />
      <input type="hidden" name="sp_sessionkey" />
      <input type="hidden" name="sp_encrypt_data" />
      <input type="hidden" name="sp_card_code" />
      <input type="hidden" name="sp_card_req_type" />
    </form>
  );
}
