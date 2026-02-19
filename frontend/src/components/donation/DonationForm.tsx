// DonationForm — donation form with multi-payType support and EasyPay PG integration
import { useState, useCallback } from 'react';
import { useNavigate } from 'react-router-dom';
import { CreditCard, Smartphone, Building2, Loader2 } from 'lucide-react';
import { Button } from '../ui/Button';
import { Input } from '../ui/Input';
import { Card } from '../ui/Card';
import { cn } from '../../lib/utils';
import { useAuth } from '../../hooks/useAuth';
import { useAuthRedirect } from '../../hooks/useAuthRedirect';
import { useDonateOrder, useCreateSubscription } from '../../hooks/useDonateOrder';
import { EasyPayBridge } from './EasyPayBridge';
import { BankTransferConfirm } from './BankTransferConfirm';
import type { PaymentParams, CreateOrderResponse } from '../../types/donate';

// --- Constants ---

const AMOUNT_PRESETS = [
  { value: 10_000, label: '1만' },
  { value: 30_000, label: '3만' },
  { value: 50_000, label: '5만' },
  { value: 100_000, label: '10만' },
  { value: 300_000, label: '30만' },
  { value: 500_000, label: '50만' },
  { value: 1_000_000, label: '100만' },
  { value: 2_000_000, label: '200만' },
] as const;

const MIN_DONATION_AMOUNT = 10_000;
const MAX_DONATION_AMOUNT = 10_000_000;

type GateType = 'immediately' | 'recurring';
type PayType = 'CARD' | 'BANK' | 'PHONE';

const PAY_TYPE_LABELS: Record<PayType, string> = {
  CARD: '신용카드(체크카드)',
  BANK: '계좌이체',
  PHONE: '휴대폰',
};

const PAY_TYPE_ICONS: Record<PayType, typeof CreditCard> = {
  CARD: CreditCard,
  BANK: Building2,
  PHONE: Smartphone,
};

interface ValidationErrors {
  amount?: string;
  payType?: string;
}

// --- Helpers ---

function formatWithCommas(value: string): string {
  const digits = value.replace(/\D/g, '');
  if (digits === '') return '';
  return Number(digits).toLocaleString();
}

function parseDigits(value: string): number {
  return Number(value.replace(/\D/g, '')) || 0;
}

// --- Sub-components ---

function GateSelector({
  selected,
  onChange,
}: {
  selected: GateType;
  onChange: (gate: GateType) => void;
}) {
  return (
    <div className="flex gap-2">
      <button
        type="button"
        onClick={() => onChange('immediately')}
        className={cn(
          'flex-1 rounded-lg py-2.5 text-sm font-medium transition-all duration-150',
          selected === 'immediately'
            ? 'bg-primary text-white shadow-sm'
            : 'bg-background text-text-secondary border border-border-subtle',
        )}
      >
        일시후원
      </button>
      <button
        type="button"
        onClick={() => onChange('recurring')}
        className={cn(
          'flex-1 rounded-lg py-2.5 text-sm font-medium transition-all duration-150',
          selected === 'recurring'
            ? 'bg-primary text-white shadow-sm'
            : 'bg-background text-text-secondary border border-border-subtle',
        )}
      >
        월정기후원
      </button>
    </div>
  );
}

function AmountPresetGrid({
  selectedAmount,
  onSelect,
}: {
  selectedAmount: number | null;
  onSelect: (amount: number) => void;
}) {
  return (
    <div className="grid grid-cols-2 gap-2 md:grid-cols-4">
      {AMOUNT_PRESETS.map(({ value, label }) => (
        <button
          key={value}
          type="button"
          onClick={() => onSelect(value)}
          className={cn(
            'rounded-lg py-2.5 text-sm font-medium transition-all duration-150',
            selectedAmount === value
              ? 'bg-primary text-white shadow-sm'
              : 'bg-background text-text-secondary border border-border-subtle hover:border-primary/30',
          )}
        >
          {label}원
        </button>
      ))}
    </div>
  );
}

function PayTypeSelector({
  selected,
  onChange,
  error,
  gate,
}: {
  selected: PayType | null;
  onChange: (type: PayType) => void;
  error?: string;
  gate: GateType;
}) {
  const options: { value: PayType; label: string; disabled: boolean }[] = [
    { value: 'CARD', label: '신용카드(체크카드)', disabled: false },
    { value: 'BANK', label: gate === 'recurring' ? '계좌이체 (카드만 가능)' : '계좌이체', disabled: gate === 'recurring' },
    { value: 'PHONE', label: gate === 'recurring' ? '휴대폰 (카드만 가능)' : '휴대폰', disabled: gate === 'recurring' },
  ];

  return (
    <div className="space-y-2">
      {options.map(({ value, label, disabled }) => {
        const Icon = PAY_TYPE_ICONS[value];
        return (
          <label
            key={value}
            className={cn(
              'flex items-center gap-3 rounded-lg border p-3 transition-all duration-150 cursor-pointer',
              disabled && 'opacity-50 cursor-not-allowed',
              selected === value
                ? 'border-primary/40 bg-primary-light/30'
                : 'border-border-subtle bg-surface',
              error && !selected && 'border-error-border',
            )}
          >
            <input
              type="radio"
              name="payType"
              value={value}
              checked={selected === value}
              onChange={() => onChange(value)}
              disabled={disabled}
              className="h-4 w-4 text-primary accent-primary"
            />
            <span
              className={cn(
                'text-sm',
                disabled ? 'text-text-placeholder' : 'text-text-primary',
              )}
            >
              {label}
            </span>
            <Icon size={16} className="ml-auto text-text-tertiary" />
          </label>
        );
      })}
      {error && <p className="text-xs text-error-text">{error}</p>}
    </div>
  );
}

function OrderSummary({
  amount,
  gate,
  payType,
}: {
  amount: number;
  gate: GateType;
  payType: PayType;
}) {
  const gateLabel = gate === 'recurring' ? '정기후원' : '일시후원';

  return (
    <div className="rounded-lg bg-background p-4 space-y-2">
      <div className="flex justify-between text-sm">
        <span className="text-text-tertiary">기부 유형</span>
        <span className="text-text-primary font-medium">{gateLabel}</span>
      </div>
      <div className="flex justify-between text-sm">
        <span className="text-text-tertiary">결제 수단</span>
        <span className="text-text-primary font-medium">{PAY_TYPE_LABELS[payType]}</span>
      </div>
      <div className="border-t border-border-subtle my-2" />
      <div className="flex justify-between text-sm">
        <span className="text-text-secondary font-medium">결제 금액</span>
        <span className="text-primary font-bold text-base">
          {amount.toLocaleString()}원
        </span>
      </div>
    </div>
  );
}

// --- Main Component ---

export function DonationForm() {
  const navigate = useNavigate();
  const { isLoggedIn, isLoading: authLoading } = useAuth();
  const onAuthError = useAuthRedirect();
  const donateOrder = useDonateOrder({ onAuthError });
  const createSubscription = useCreateSubscription({ onAuthError });

  // Form state
  const [gate, setGate] = useState<GateType>('immediately');
  const [selectedPreset, setSelectedPreset] = useState<number | null>(null);
  const [customAmountInput, setCustomAmountInput] = useState('');
  const [payType, setPayType] = useState<PayType | null>(null);
  const [errors, setErrors] = useState<ValidationErrors>({});

  // Payment bridge state
  const [paymentParams, setPaymentParams] = useState<PaymentParams | null>(null);
  const [bridgeError, setBridgeError] = useState<string | null>(null);
  const [bankTransferOrder, setBankTransferOrder] = useState<{ orderSeq: number; amount: number } | null>(null);

  const isCustomAmount = customAmountInput !== '' && selectedPreset === null;
  const finalAmount = isCustomAmount ? parseDigits(customAmountInput) : (selectedPreset ?? 0);
  const isFormComplete = finalAmount >= MIN_DONATION_AMOUNT && payType !== null;

  const handleGateChange = useCallback((newGate: GateType) => {
    setGate(newGate);
    if (newGate === 'recurring' && (payType === 'BANK' || payType === 'PHONE')) {
      setPayType('CARD');
    }
  }, [payType]);

  const handlePresetSelect = useCallback((amount: number) => {
    setSelectedPreset(amount);
    setCustomAmountInput('');
    setErrors((prev) => ({ ...prev, amount: undefined }));
    donateOrder.reset();
    createSubscription.reset();
  }, [donateOrder, createSubscription]);

  const handleCustomAmountChange = useCallback(
    (e: React.ChangeEvent<HTMLInputElement>) => {
      const raw = e.target.value.replace(/\D/g, '');
      setCustomAmountInput(formatWithCommas(raw));
      setSelectedPreset(null);
      setErrors((prev) => ({ ...prev, amount: undefined }));
      donateOrder.reset();
      createSubscription.reset();
    },
    [donateOrder, createSubscription],
  );

  const handlePayTypeChange = useCallback((type: PayType) => {
    setPayType(type);
    setErrors((prev) => ({ ...prev, payType: undefined }));
    donateOrder.reset();
    createSubscription.reset();
  }, [donateOrder, createSubscription]);

  const validate = useCallback((): boolean => {
    const newErrors: ValidationErrors = {};

    if (finalAmount < MIN_DONATION_AMOUNT) {
      newErrors.amount = `최소 ${MIN_DONATION_AMOUNT.toLocaleString()}원 이상 입력해 주세요.`;
    } else if (finalAmount > MAX_DONATION_AMOUNT) {
      newErrors.amount = `최대 ${MAX_DONATION_AMOUNT.toLocaleString()}원까지 기부할 수 있습니다.`;
    }

    if (!payType) {
      newErrors.payType = '결제 수단을 선택해 주세요.';
    }

    setErrors(newErrors);
    return Object.keys(newErrors).length === 0;
  }, [finalAmount, payType]);

  const handleSubmit = useCallback(
    async (e: React.FormEvent) => {
      e.preventDefault();
      if (!validate() || !payType) return;

      const apiPayType = payType === 'PHONE' ? 'HP' : payType;

      const onSuccess = (data: CreateOrderResponse) => {
        if (data.paymentParams === null) {
          setBankTransferOrder({ orderSeq: data.orderSeq, amount: finalAmount });
        } else {
          setPaymentParams(data.paymentParams);
        }
      };

      if (gate === 'recurring') {
        createSubscription.mutate(
          { amount: finalAmount, payType: apiPayType as 'CARD' | 'BANK' | 'HP' },
          { onSuccess },
        );
      } else {
        donateOrder.mutate(
          { amount: finalAmount, payType: apiPayType as 'CARD' | 'BANK' | 'HP', gate: 'immediately' as const },
          { onSuccess },
        );
      }
    },
    [validate, finalAmount, payType, gate, donateOrder, createSubscription],
  );

  const handleBridgeError = useCallback((message: string) => {
    setPaymentParams(null);
    setBridgeError(message);
  }, []);

  // Bank transfer confirmation view
  if (bankTransferOrder) {
    return <BankTransferConfirm orderSeq={bankTransferOrder.orderSeq} amount={bankTransferOrder.amount} />;
  }

  // If payment bridge is active, show processing state
  if (paymentParams) {
    return (
      <Card padding="lg" className="space-y-4 text-center">
        <Loader2 size={32} className="mx-auto text-primary animate-spin" />
        <p className="text-sm text-text-secondary">결제 페이지로 이동 중입니다...</p>
        <p className="text-xs text-text-placeholder">
          잠시만 기다려 주세요. 자동으로 이동하지 않으면 새로고침 해 주세요.
        </p>
        <EasyPayBridge
          params={paymentParams}
          onError={handleBridgeError}
        />
      </Card>
    );
  }

  // Auth loading guard
  if (authLoading) {
    return <div className="h-60 rounded-xl skeleton-shimmer" />;
  }

  const hasAmountError = Boolean(errors.amount);
  const submitLabel = gate === 'recurring' ? '정기후원 등록하기' : '기부하기';

  return (
    <form onSubmit={handleSubmit} className="space-y-6">
      {/* Gate selector */}
      <Card padding="lg" className="space-y-5">
        <div className="space-y-3">
          <h3 className="font-bold text-text-primary">후원 유형</h3>
          <GateSelector selected={gate} onChange={handleGateChange} />
        </div>

        {/* Amount selection */}
        <div className="space-y-3">
          <h3 className="font-bold text-text-primary">기부 금액</h3>
          <AmountPresetGrid
            selectedAmount={selectedPreset}
            onSelect={handlePresetSelect}
          />
          <div className="relative">
            <Input
              inputMode="numeric"
              placeholder="직접 입력"
              value={customAmountInput}
              onChange={handleCustomAmountChange}
              className={cn(
                'pr-8',
                hasAmountError && 'border-error-border focus:border-error-border focus:ring-error-subtle',
              )}
            />
            {customAmountInput && (
              <span className="absolute right-3 top-1/2 -translate-y-1/2 text-sm text-text-tertiary">
                원
              </span>
            )}
          </div>
          {errors.amount && (
            <p className="text-xs text-error-text">{errors.amount}</p>
          )}
        </div>

        {/* Payment method */}
        <div className="space-y-3">
          <h3 className="font-bold text-text-primary">결제 수단</h3>
          <PayTypeSelector
            selected={payType}
            onChange={handlePayTypeChange}
            error={errors.payType}
            gate={gate}
          />
        </div>
      </Card>

      {/* Order summary */}
      {isFormComplete && payType && (
        <div className="animate-fade-in">
          <OrderSummary amount={finalAmount} gate={gate} payType={payType} />
        </div>
      )}

      {/* Submit / Login button */}
      {isLoggedIn ? (
        <Button
          type="submit"
          size="lg"
          className="w-full shadow-primary-glow"
          disabled={donateOrder.isPending || createSubscription.isPending}
        >
          {donateOrder.isPending || createSubscription.isPending ? (
            <>
              <Loader2 size={18} className="animate-spin" />
              처리 중...
            </>
          ) : (
            submitLabel
          )}
        </Button>
      ) : (
        <Button
          type="button"
          size="lg"
          className="w-full shadow-primary-glow"
          onClick={() => navigate('/login')}
        >
          로그인 후 기부하기
        </Button>
      )}

      {/* Mutation error */}
      {(donateOrder.isError || createSubscription.isError) && (
        <p className="text-center text-xs text-error-text">
          {(donateOrder.error ?? createSubscription.error)?.message ?? '주문 생성에 실패했습니다. 잠시 후 다시 시도해 주세요.'}
        </p>
      )}

      {/* Bridge error */}
      {bridgeError && (
        <p className="text-center text-xs text-error-text">{bridgeError}</p>
      )}
    </form>
  );
}
